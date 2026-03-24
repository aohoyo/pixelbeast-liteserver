package handlers

import (
	"bufio"
	"fmt"
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
}

// FTPServer 简易FTP服务器
type FTPServer struct {
	Config    *config.FTPConfig
	validator PasswordValidator // 密码验证器（支持加密密码）
	listener  net.Listener
	clients   map[net.Conn]bool
	mu        sync.Mutex
	running   bool
}

// NewFTPServer 创建FTP服务器
func NewFTPServer(cfg *config.FTPConfig) (*FTPServer, error) {
	return NewFTPServerWithValidator(cfg, nil)
}

// NewFTPServerWithValidator 创建FTP服务器（带密码验证器）
func NewFTPServerWithValidator(cfg *config.FTPConfig, validator PasswordValidator) (*FTPServer, error) {
	// 确保FTP根目录存在
	if err := os.MkdirAll(cfg.Root, 0755); err != nil {
		return nil, err
	}

	return &FTPServer{
		Config:    cfg,
		validator: validator,
		clients:   make(map[net.Conn]bool),
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
	Log(LogCategoryFTP, "access", LogLevelInfo, "[FTP] 根目录: %s", s.Config.Root)

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
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	client := &FTPClient{
		server:   s,
		conn:     conn,
		reader:   bufio.NewReader(conn),
		rootDir:  s.Config.Root,
		cwd:      "/",
		loggedIn: false,
	}

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

// checkLogin 检查登录状态
func (c *FTPClient) checkLogin() bool {
	if !c.loggedIn {
		c.sendMessage("530 Please login with USER and PASS")
		return false
	}
	return true
}

// handleUSER 处理USER命令
func (c *FTPClient) handleUSER(username string) {
	c.username = username
	c.sendMessage("331 User name okay, need password")
}

// handlePASS 处理PASS命令
func (c *FTPClient) handlePASS(password string) {
	// 使用验证器验证（支持加密密码）
	if c.server.validator != nil {
		if c.server.validator.ValidateFTPUser(c.username, password) {
			c.loggedIn = true
			c.sendMessage("230 Login successful")
			LogFTPLogin(c.username, c.conn.RemoteAddr().String(), true, "登录成功")
			return
		}
	} else {
		// 兼容旧模式：明文密码比较
		for _, user := range c.server.Config.Users {
			if user.Username == c.username && user.Password == password {
				c.loggedIn = true
				c.sendMessage("230 Login successful")
				LogFTPLogin(c.username, c.conn.RemoteAddr().String(), true, "登录成功")
				return
			}
		}
	}
	c.sendMessage("530 Login incorrect")
	LogFTPLogin(c.username, c.conn.RemoteAddr().String(), false, "密码错误")
	c.username = ""
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

// handleRETR 处理RETR命令
func (c *FTPClient) handleRETR(filename string) {
	// 转换文件名编码
	filename = c.toUTF8(filename)
	path := c.resolvePath(filename)
	fullPath := filepath.Join(c.rootDir, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "下载 "+filename, "文件不存在")
		c.sendMessage("550 File not found")
		return
	}

	c.sendMessage("150 Opening data connection for file transfer")

	dataConn, err := c.getDataConnection()
	if err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "下载 "+filename, err.Error())
		c.sendMessage("425 Cannot open data connection")
		return
	}
	defer dataConn.Close()

	start := time.Now()
	dataConn.Write(data)
	duration := time.Since(start)
	LogFTPTransfer(c.username, c.conn.RemoteAddr().String(), filename, "下载", int64(len(data)), duration, true)
	c.sendMessage("226 Transfer complete")
}

// handleSTOR 处理STOR命令
func (c *FTPClient) handleSTOR(filename string) {
	// 转换文件名编码
	filename = c.toUTF8(filename)
	path := c.resolvePath(filename)
	fullPath := filepath.Join(c.rootDir, path)

	c.sendMessage("150 Opening data connection for file transfer")

	dataConn, err := c.getDataConnection()
	if err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "上传 "+filename, err.Error())
		c.sendMessage("425 Cannot open data connection")
		return
	}
	defer dataConn.Close()

	start := time.Now()
	data := make([]byte, 0)
	buf := make([]byte, 4096)
	for {
		n, err := dataConn.Read(buf)
		if err != nil {
			break
		}
		data = append(data, buf[:n]...)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		LogFTPError(c.username, c.conn.RemoteAddr().String(), "上传 "+filename, err.Error())
		c.sendMessage("553 Cannot store file")
		return
	}

	duration := time.Since(start)
	LogFTPTransfer(c.username, c.conn.RemoteAddr().String(), filename, "上传", int64(len(data)), duration, true)
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
