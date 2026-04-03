package handlers

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"

	"pixelbeast/src/config"
)

// isValidUTF8 检查字符串是否是有效的 UTF-8
func isValidUTF8(s string) bool {
	return utf8.ValidString(s)
}

// PasswordValidator 密码验证器接口
type PasswordValidator interface {
	ValidateFTPUser(username, password string) bool
	GetFTPUserConfig(username string) *config.FTPUser
}

// FTPServer 简易FTP服务器
type FTPServer struct {
	Config       *config.FTPConfig
	validator    PasswordValidator // 密码验证器（支持加密密码）
	listener     net.Listener
	clients      map[net.Conn]bool
	mu           sync.Mutex
	running      bool
	rootDir      string
	userConns    map[string]int // 每用户当前连接数
}

// SetValidator 更新密码验证器（配置重载后同步）
func (s *FTPServer) SetValidator(v PasswordValidator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.validator = v
}

// NewFTPServer 创建FTP服务器
func NewFTPServer(cfg *config.FTPConfig, rootDir string) (*FTPServer, error) {
	return NewFTPServerWithValidator(cfg, nil, rootDir)
}

// NewFTPServerWithValidator 创建FTP服务器（带密码验证器）
func NewFTPServerWithValidator(cfg *config.FTPConfig, validator PasswordValidator, rootDir string) (*FTPServer, error) {
	// 确保FTP根目录存在
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}

	return &FTPServer{
		Config:    cfg,
		validator: validator,
		clients:   make(map[net.Conn]bool),
		userConns: make(map[string]int),
		rootDir:   rootDir,
	}, nil
}

// Start 启动FTP服务器
func (s *FTPServer) Start() error {
	addr := fmt.Sprintf(":%d", s.Config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = listener
	s.running = true

	Log(LogCategoryFTP, "access", LogLevelInfo, "[FTP] 服务器启动在端口 %d", s.Config.Port)
	Log(LogCategoryFTP, "access", LogLevelInfo, "[FTP] 根目录: %s", s.rootDir)

	go s.acceptConnections()
	return nil
}

// Stop 停止FTP服务器
func (s *FTPServer) Stop() error {
	s.mu.Lock()
	s.running = false
	for conn := range s.clients {
		conn.Close()
	}
	s.mu.Unlock()

	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections 接受连接
func (s *FTPServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.running {
				LogFTPError("", "", "接受连接", err.Error())
			}
			return
		}

		s.mu.Lock()
		s.clients[conn] = true
		s.mu.Unlock()

		go s.handleClient(conn)
	}
}

// handleClient 处理客户端连接
func (s *FTPServer) handleClient(conn net.Conn) {
	client := &FTPClient{
		server:   s,
		conn:     conn,
		reader:   bufio.NewReader(conn),
		rootDir:  s.rootDir,
		cwd:      "/",
		loggedIn: false,
	}

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		// 释放用户连接计数
		if client.connTracked && client.username != "" {
			s.userConns[client.username]--
			if s.userConns[client.username] <= 0 {
				delete(s.userConns, client.username)
			}
		}
		s.mu.Unlock()
		conn.Close()
	}()

	LogFTPConnection(conn.RemoteAddr().String(), true)
	client.sendMessage("220 轻羽 FTP Server Ready")

	for {
		line, err := client.reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		cmd := strings.ToUpper(parts[0])
		var args string
		if len(parts) > 1 {
			args = parts[1]
		}

		LogFTPCommand(client.username, conn.RemoteAddr().String(), cmd, args)
		client.handleCommand(cmd, args)
	}

	LogFTPConnection(conn.RemoteAddr().String(), false)
}

// FTPClient FTP客户端
type FTPClient struct {
	server         *FTPServer
	conn           net.Conn
	reader         *bufio.Reader
	rootDir        string
	cwd            string
	loggedIn       bool
	username       string
	dataConn       net.Conn
	dataPort       string
	passive        bool
	passiveListener net.Listener
	transferMu      sync.Mutex
	utf8Enabled     bool // UTF-8 编码支持
	connTracked     bool // 连接计数已计入 userConns
}

// sendMessage 发送消息
func (c *FTPClient) sendMessage(msg string) {
	c.conn.Write([]byte(msg + "\r\n"))
}

// handleCommand 处理命令
func (c *FTPClient) handleCommand(cmd, args string) {
	switch cmd {
	case "USER":
		c.handleUSER(args)
	case "PASS":
		c.handlePASS(args)
	case "QUIT":
		c.sendMessage("221 Goodbye")
		c.conn.Close()
	case "PWD", "XPWD":
		if c.checkLogin() {
			c.sendMessage(fmt.Sprintf("257 \"%s\" is current directory", c.cwd))
		}
	case "EPSV":
		if c.checkLogin() {
			c.handleEPSV()
		}
	case "CWD":
		if c.checkLogin() {
			c.handleCWD(args)
		}
	case "CDUP":
		if c.checkLogin() {
			c.handleCDUP()
		}
	case "TYPE":
		if c.checkLogin() {
			c.sendMessage("200 Type set to " + args)
		}
	case "PASV":
		if c.checkLogin() {
			c.handlePASV()
		}
	case "PORT":
		if c.checkLogin() {
			c.handlePORT(args)
		}
	case "LIST", "NLST":
		if c.checkLogin() {
			c.handleLIST(args)
		}
	case "RETR":
		if c.checkLogin() {
			c.handleRETR(args)
		}
	case "STOR":
		if c.checkLogin() {
			c.handleSTOR(args)
		}
	case "MKD", "XMKD":
		if c.checkLogin() {
			c.handleMKD(args)
		}
	case "RMD", "XRMD":
		if c.checkLogin() {
			c.handleRMD(args)
		}
	case "DELE":
		if c.checkLogin() {
			c.handleDELE(args)
		}
	case "SIZE":
		if c.checkLogin() {
			c.handleSIZE(args)
		}
	case "FEAT":
		c.sendMessage("211-Features:")
		c.sendMessage(" PASV")
		c.sendMessage(" EPSV")
		c.sendMessage(" PORT")
		c.sendMessage(" TYPE")
		c.sendMessage(" SIZE")
		c.sendMessage("211 End")
	case "SYST":
		c.sendMessage("215 UNIX Type: L8")
	case "NOOP":
		c.sendMessage("200 OK")
	case "OPTS":
		c.handleOPTS(args)
	default:
		c.sendMessage("500 Unknown command: " + cmd)
	}
}

// checkLogin 检查登录状态（同时验证用户是否仍处于启用状态）
func (c *FTPClient) checkLogin() bool {
	if !c.loggedIn {
		c.sendMessage("530 Please login with USER and PASS")
		return false
	}
	// 实时检查用户是否被禁用
	if c.server.validator != nil {
		userCfg := c.server.validator.GetFTPUserConfig(c.username)
		if userCfg == nil {
			c.sendMessage("530 Account has been disabled")
			c.conn.Close()
			return false
		}
	}
	return true
}

// getUserConfig è·åå½åç¨æ·çéç½®
func (c *FTPClient) getUserConfig() *config.FTPUser {
	if c.username == "" || c.server.validator == nil {
		return nil
	}
	return c.server.validator.GetFTPUserConfig(c.username)
}

// countFiles è®¡ç®ç¨æ·ç®å½ä¸çæä»¶æ»æ°
func (c *FTPClient) countFiles(root string) int {
	count := 0
	filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		count++
		return nil
	})
	return count
}

// rateLimitedWrite ä»¥ééæ¹å¼åå¥æ°æ®
func (c *FTPClient) rateLimitedWrite(conn net.Conn, data []byte, bytesPerSec int64) {
	if bytesPerSec <= 0 {
		conn.Write(data)
		return
	}

	chunkSize := bytesPerSec / 10
	if chunkSize < 1024 {
		chunkSize = 1024
	}
	interval := time.Duration(float64(time.Second) * float64(chunkSize) / float64(bytesPerSec))

	for offset := 0; offset < len(data); {
		end := offset + int(chunkSize)
		if end > len(data) {
			end = len(data)
		}
		if _, err := conn.Write(data[offset:end]); err != nil {
			return
		}
		offset = end
		if offset < len(data) {
			time.Sleep(interval)
		}
	}
}

// rateLimitedRead ä»¥ééæ¹å¼è¯»åæ°æ®
func (c *FTPClient) rateLimitedRead(src io.Reader, dst io.Writer, bytesPerSec int64) (int64, error) {
	if bytesPerSec <= 0 {
		return io.Copy(dst, src)
	}

	chunkSize := bytesPerSec / 10
	if chunkSize < 1024 {
		chunkSize = 1024
	}
	interval := time.Duration(float64(time.Second) * float64(chunkSize) / float64(bytesPerSec))

	buf := make([]byte, chunkSize)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total, werr
			}
			total += int64(n)
		}
		if err != nil {
			return total, err
		}
		if int64(n) >= chunkSize {
			time.Sleep(interval)
		}
	}
}

// handleUSER 处理USER命令
func (c *FTPClient) handleUSER(username string) {
	c.username = username
	c.sendMessage("331 User name okay, need password")
}

// handlePASS 处理PASS命令
func (c *FTPClient) handlePASS(password string) {
	// 使用验证器验证（加密密码）
	if c.server.validator == nil {
		c.sendMessage("500 Server configuration error")
		LogFTPLogin(c.username, c.conn.RemoteAddr().String(), false, "验证器未配置")
		c.username = ""
		return
	}

	if !c.server.validator.ValidateFTPUser(c.username, password) {
		c.sendMessage("530 Login incorrect")
		LogFTPLogin(c.username, c.conn.RemoteAddr().String(), false, "密码错误")
		c.username = ""
		return
	}

	// 通过验证器（带锁）获取用户配置，确保读取最新状态
	userCfg := c.server.validator.GetFTPUserConfig(c.username)
	if userCfg == nil {
		c.sendMessage("530 User not found or disabled")
		LogFTPLogin(c.username, c.conn.RemoteAddr().String(), false, "用户不存在或已禁用")
		c.username = ""
		return
	}

	// 检查账号是否过期
	if userCfg.ExpiryDate != "" {
		if expiryTime, err := time.Parse("2006-01-02", userCfg.ExpiryDate); err == nil {
			if time.Now().After(expiryTime) {
				c.sendMessage("530 Account expired")
				LogFTPLogin(c.username, c.conn.RemoteAddr().String(), false, "账号已过期")
				c.username = ""
				return
			}
		}
	}

	// 检查最大连接数
	if userCfg.MaxConnections > 0 {
		c.server.mu.Lock()
		current := c.server.userConns[c.username]
		if current >= userCfg.MaxConnections {
			c.server.mu.Unlock()
			c.sendMessage(fmt.Sprintf("530 Too many connections (max %d)", userCfg.MaxConnections))
			LogFTPLogin(c.username, c.conn.RemoteAddr().String(), false, "超过最大连接数")
			c.username = ""
			return
		}
		c.server.userConns[c.username] = current + 1
		c.server.mu.Unlock()
		c.connTracked = true
	}

	c.rootDir = userCfg.RootPath
	c.cwd = "/"
	// 确保用户目录存在
	os.MkdirAll(c.rootDir, 0755)

	c.loggedIn = true
	c.sendMessage("230 Login successful, welcome " + c.username)
	LogFTPLogin(c.username, c.conn.RemoteAddr().String(), true, "登录成功")
}


// handleCWD 处理CWD命令
func (c *FTPClient) handleCWD(path string) {
	// 转换文件名编码
	path = c.toUTF8(path)
	newPath := c.resolvePath(path)
	fullPath := filepath.Join(c.rootDir, newPath)

	info, err := os.Stat(fullPath)
	if err != nil || !info.IsDir() {
		c.sendMessage("550 Directory not found")
		return
	}

	c.cwd = newPath
	c.sendMessage("250 Directory changed to " + c.cwd)
}

// handleCDUP 处理CDUP命令
func (c *FTPClient) handleCDUP() {
	if c.cwd != "/" {
		c.cwd = filepath.Dir(c.cwd)
		if c.cwd == "." {
			c.cwd = "/"
		}
	}
	c.sendMessage("250 Directory changed to " + c.cwd)
}

// handleOPTS 处理OPTS命令 (选项设置)
func (c *FTPClient) handleOPTS(args string) {
	parts := strings.Fields(strings.ToUpper(args))
	if len(parts) >= 2 && parts[0] == "UTF8" && parts[1] == "ON" {
		c.utf8Enabled = true
		c.sendMessage("200 UTF8 enabled")
	} else if len(parts) >= 1 && parts[0] == "UTF8" {
		if len(parts) >= 2 {
			if parts[1] == "ON" {
				c.utf8Enabled = true
				c.sendMessage("200 UTF8 enabled")
			} else {
				c.sendMessage("200 UTF8 disabled")
			}
		} else {
			c.sendMessage("200 UTF8 " + map[bool]string{true: "ON", false: "OFF"}[c.utf8Enabled])
		}
	} else {
		c.sendMessage("500 Unknown option")
	}
}

// toUTF8 将字符串转换为 UTF-8
func (c *FTPClient) toUTF8(s string) string {
	if c.utf8Enabled {
		return s
	}
	// 如果不是 UTF-8 模式，尝试从 GBK 转换（Windows 客户端常用）
	if !isValidUTF8(s) {
		// 尝试 GBK 解码
		decoder := simplifiedchinese.GBK.NewDecoder()
		if utf8Str, err := decoder.String(s); err == nil {
			return utf8Str
		}
	}
	return s
}

// fromUTF8 将 UTF-8 转换为客户端编码
func (c *FTPClient) fromUTF8(s string) string {
	if c.utf8Enabled {
		return s
	}
	// 如果不是 UTF-8 模式，转换为 GBK（Windows 客户端）
	encoder := simplifiedchinese.GBK.NewEncoder()
	if gbkStr, err := encoder.String(s); err == nil {
		return gbkStr
	}
	return s
}

// handlePASV 处理PASV命令
func (c *FTPClient) handlePASV() {
	// 创建临时监听器
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		c.sendMessage("425 Cannot open passive connection")
		return
	}

	// 获取端口号
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	// 获取服务器IP
	localAddr := c.conn.LocalAddr().(*net.TCPAddr)
	ip := localAddr.IP
	if ip.IsUnspecified() {
		ip = net.ParseIP("127.0.0.1")
	}

	// 格式化PASV响应
	p1 := port / 256
	p2 := port % 256
	ipStr := strings.ReplaceAll(ip.String(), ".", ",")

	c.passive = true
	c.passiveListener = listener
	c.sendMessage(fmt.Sprintf("227 Entering Passive Mode (%s,%d,%d)", ipStr, p1, p2))

	// 等待数据连接
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		c.transferMu.Lock()
		c.dataConn = conn
		c.transferMu.Unlock()
	}()
}

// handleEPSV 处理EPSV命令 (扩展被动模式)
func (c *FTPClient) handleEPSV() {
	// 创建临时监听器
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		c.sendMessage("425 Cannot open passive connection")
		return
	}

	// 获取端口号
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	c.passive = true
	c.passiveListener = listener
	c.sendMessage(fmt.Sprintf("229 Entering Extended Passive Mode (|||%d|)", port))

	// 等待数据连接
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		c.transferMu.Lock()
		c.dataConn = conn
		c.transferMu.Unlock()
	}()
}

// handlePORT 处理PORT命令
func (c *FTPClient) handlePORT(args string) {
	parts := strings.Split(args, ",")
	if len(parts) != 6 {
		c.sendMessage("500 Invalid PORT command")
		return
	}

	ip := strings.Join(parts[0:4], ".")
	p1, _ := strconv.Atoi(parts[4])
	p2, _ := strconv.Atoi(parts[5])
	port := p1*256 + p2

	c.dataPort = fmt.Sprintf("%s:%d", ip, port)
	c.passive = false
	c.sendMessage("200 PORT command successful")
}

// handleLIST 处理LIST命令
func (c *FTPClient) handleLIST(args string) {
	path := c.cwd
	if args != "" && !strings.HasPrefix(args, "-") {
		path = c.resolvePath(args)
	}
	fullPath := filepath.Join(c.rootDir, path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.sendMessage("550 Cannot list directory")
		return
	}

	c.sendMessage("150 Opening data connection for directory listing")

	dataConn, err := c.getDataConnection()
	if err != nil {
		c.sendMessage("425 Cannot open data connection")
		return
	}
	defer dataConn.Close()

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		mode := "-rw-r--r--"
		if entry.IsDir() {
			mode = "drwxr-xr-x"
		}

		line := fmt.Sprintf("%s 1 owner group %13d %s %s\r\n",
			mode,
			info.Size(),
			info.ModTime().Format("Jan 02 15:04"),
			entry.Name(),
		)
		dataConn.Write([]byte(line))
	}

	c.sendMessage("226 Transfer complete")
}

// handleRETR å¤çRETRå½ä»¤
func (c *FTPClient) handleRETR(filename string) {
	// è½¬æ¢æä»¶åç¼ç 
	filename = c.toUTF8(filename)
	path := c.resolvePath(filename)
	fullPath := filepath.Join(c.rootDir, path)

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸è½½ "+filename, "æä»¶ä¸å­å¨")
		c.sendMessage("550 File not found")
		return
	}

	// æ£æ¥åæä»¶å¤§å°éå¶
	if userCfg := c.getUserConfig(); userCfg != nil && userCfg.MaxFileSize > 0 {
		if info.Size() > userCfg.MaxFileSize*1024*1024 {
			LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸è½½ "+filename, "è¶è¿åæä»¶å¤§å°éå¶")
			c.sendMessage("550 File too large")
			return
		}
	}

	c.sendMessage("150 Opening data connection for file transfer")

	dataConn, err := c.getDataConnection()
	if err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸è½½ "+filename, err.Error())
		c.sendMessage("425 Cannot open data connection")
		return
	}
	defer dataConn.Close()

	data, err := os.ReadFile(fullPath)
	if err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸è½½ "+filename, err.Error())
		c.sendMessage("550 Cannot read file")
		return
	}

	start := time.Now()
	// ä¸è½½éé
	if userCfg := c.getUserConfig(); userCfg != nil && userCfg.SpeedLimit > 0 {
		c.rateLimitedWrite(dataConn, data, userCfg.SpeedLimit*1024)
	} else {
		dataConn.Write(data)
	}
	duration := time.Since(start)
	LogFTPTransfer(c.username, c.conn.RemoteAddr().String(), filename, "ä¸è½½", int64(len(data)), duration, true)
	c.sendMessage("226 Transfer complete")
}

// handleSTOR å¤çSTORå½ä»¤
func (c *FTPClient) handleSTOR(filename string) {
	// è½¬æ¢æä»¶åç¼ç 
	filename = c.toUTF8(filename)
	path := c.resolvePath(filename)
	fullPath := filepath.Join(c.rootDir, path)

	userCfg := c.getUserConfig()

	// æ£æ¥æä»¶æ°ééå¶
	if userCfg != nil && userCfg.MaxFiles > 0 {
		if c.countFiles(c.rootDir) >= userCfg.MaxFiles {
			LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸ä¼  "+filename, "è¶è¿æä»¶æ°ééå¶")
			c.sendMessage("550 Too many files")
			return
		}
	}

	c.sendMessage("150 Opening data connection for file transfer")

	dataConn, err := c.getDataConnection()
	if err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸ä¼  "+filename, err.Error())
		c.sendMessage("425 Cannot open data connection")
		return
	}
	defer dataConn.Close()

	start := time.Now()
	var buf bytes.Buffer
	var totalSize int64

	// ä¸ä¼ éé + åæä»¶å¤§å°æ£æ¥
	maxFileSize := int64(0)
	if userCfg != nil {
		maxFileSize = userCfg.MaxFileSize * 1024 * 1024
	}

	if userCfg != nil && userCfg.Bandwidth > 0 {
		// ééè¯»å
		var tmpBuf bytes.Buffer
		totalSize, _ = c.rateLimitedRead(dataConn, &tmpBuf, userCfg.Bandwidth*1024)
		if maxFileSize > 0 && totalSize > maxFileSize {
			LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸ä¼  "+filename, "è¶è¿åæä»¶å¤§å°éå¶")
			c.sendMessage("553 File too large")
			return
		}
		buf = tmpBuf
	} else {
		// ä¸ééï¼ä½æ£æ¥æä»¶å¤§å°
		tmpBuf := make([]byte, 0, 64*1024)
		readBuf := make([]byte, 32*1024)
		for {
			n, err := dataConn.Read(readBuf)
			if n > 0 {
				totalSize += int64(n)
				if maxFileSize > 0 && totalSize > maxFileSize {
					LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸ä¼  "+filename, "è¶è¿åæä»¶å¤§å°éå¶")
					c.sendMessage("553 File too large")
					return
				}
				tmpBuf = append(tmpBuf, readBuf[:n]...)
			}
			if err != nil {
				break
			}
		}
		buf = *bytes.NewBuffer(tmpBuf)
	}

	if err := os.WriteFile(fullPath, buf.Bytes(), 0644); err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "ä¸ä¼  "+filename, err.Error())
		c.sendMessage("553 Cannot store file")
		return
	}

	duration := time.Since(start)
	LogFTPTransfer(c.username, c.conn.RemoteAddr().String(), filename, "ä¸ä¼ ", totalSize, duration, true)
	c.sendMessage("226 Transfer complete")
}

// handleMKD 处理MKD命令
func (c *FTPClient) handleMKD(dirname string) {
	// 转换文件名编码
	dirname = c.toUTF8(dirname)
	path := c.resolvePath(dirname)
	fullPath := filepath.Join(c.rootDir, path)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		c.sendMessage("550 Cannot create directory")
		return
	}

	c.sendMessage(fmt.Sprintf("257 \"%s\" created", path))
}

// handleRMD 处理RMD命令
func (c *FTPClient) handleRMD(dirname string) {
	// 转换文件名编码
	dirname = c.toUTF8(dirname)
	path := c.resolvePath(dirname)
	fullPath := filepath.Join(c.rootDir, path)

	if err := os.Remove(fullPath); err != nil {
		c.sendMessage("550 Cannot remove directory")
		return
	}

	c.sendMessage("250 Directory removed")
}

// handleDELE 处理DELE命令
func (c *FTPClient) handleDELE(filename string) {
	path := c.resolvePath(filename)
	fullPath := filepath.Join(c.rootDir, path)

	if err := os.Remove(fullPath); err != nil {
		c.sendMessage("550 Cannot delete file")
		return
	}

	c.sendMessage("250 File deleted")
}

// handleSIZE 处理SIZE命令
func (c *FTPClient) handleSIZE(filename string) {
	// 转换文件名编码
	filename = c.toUTF8(filename)
	path := c.resolvePath(filename)
	fullPath := filepath.Join(c.rootDir, path)

	info, err := os.Stat(fullPath)
	if err != nil {
		c.sendMessage("550 File not found")
		return
	}

	c.sendMessage(fmt.Sprintf("213 %d", info.Size()))
}

// resolvePath 解析路径
func (c *FTPClient) resolvePath(path string) string {
	if strings.HasPrefix(path, "/") {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(c.cwd, path))
}

// getDataConnection 获取数据连接
func (c *FTPClient) getDataConnection() (net.Conn, error) {
	c.transferMu.Lock()
	defer c.transferMu.Unlock()

	if c.passive {
		// 等待被动连接
		for i := 0; i < 100; i++ {
			if c.dataConn != nil {
				conn := c.dataConn
				c.dataConn = nil
				return conn, nil
			}
			c.transferMu.Unlock()
			time.Sleep(10 * time.Millisecond)
			c.transferMu.Lock()
		}
		// 超时，关闭监听器
		if c.passiveListener != nil {
			c.passiveListener.Close()
		}
		return nil, fmt.Errorf("timeout waiting for passive connection")
	}

	// 主动连接
	return net.Dial("tcp", c.dataPort)
}
