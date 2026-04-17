package notify

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Send 发送通知到指定渠道
func Send(ch Channel, title, message string) error {
	switch ch.Type {
	case "email":
		return sendEmail(ch, title, message)
	default:
		return fmt.Errorf("不支持的渠道类型: %s", ch.Type)
	}
}

// sendEmail 通过 SMTP 发送邮件通知
func sendEmail(ch Channel, title, message string) error {
	if ch.SMTPHost == "" || ch.To == "" {
		return fmt.Errorf("邮件 SMTP 或收件人未配置")
	}

	// 纯文本格式消息
	plain := fmt.Sprintf("%s\n\n%s\n\n时间: %s\n\n--\nPixelBeast LiteServer 告警通知",
		title, strings.ReplaceAll(message, "**", ""), time.Now().Format("2006-01-02 15:04:05"))

	header := make(map[string]string)
	header["From"] = ch.SMTPUser
	header["To"] = ch.To
	header["Subject"] = "[PixelBeast] " + title
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=UTF-8"

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(plain)

	addr := fmt.Sprintf("%s:%d", ch.SMTPHost, ch.SMTPPort)
	auth := smtp.PlainAuth("", ch.SMTPUser, ch.SMTPPass, ch.SMTPHost)

	// TLS 连接
	if ch.SMTPPort == 465 {
		return sendEmailTLS(addr, auth, ch.SMTPUser, ch.To, msg.String())
	}
	return smtp.SendMail(addr, auth, ch.SMTPUser, []string{ch.To}, []byte(msg.String()))
}

// sendEmailTLS 通过 TLS 465 端口发送邮件
func sendEmailTLS(addr string, auth smtp.Auth, from, to string, msg string) error {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, addr)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取写入器失败: %w", err)
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}

	return client.Quit()
}
