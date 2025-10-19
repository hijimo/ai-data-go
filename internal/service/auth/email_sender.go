package auth

import (
	"fmt"
	"log"
)

// consoleEmailSender 控制台邮件发送器（用于开发和测试）
type consoleEmailSender struct{}

// NewConsoleEmailSender 创建控制台邮件发送器实例
func NewConsoleEmailSender() EmailSender {
	return &consoleEmailSender{}
}

// SendEmail 发送邮件（输出到控制台）
func (s *consoleEmailSender) SendEmail(to, subject, body string) error {
	log.Printf("=== 发送邮件 ===")
	log.Printf("收件人: %s", to)
	log.Printf("主题: %s", subject)
	log.Printf("内容:\n%s", body)
	log.Printf("===============")
	return nil
}

// smtpEmailSender SMTP邮件发送器（用于生产环境）
type smtpEmailSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// SMTPConfig SMTP配置
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// NewSMTPEmailSender 创建SMTP邮件发送器实例
func NewSMTPEmailSender(config SMTPConfig) EmailSender {
	return &smtpEmailSender{
		host:     config.Host,
		port:     config.Port,
		username: config.Username,
		password: config.Password,
		from:     config.From,
	}
}

// SendEmail 发送邮件（通过SMTP）
func (s *smtpEmailSender) SendEmail(to, subject, body string) error {
	// TODO: 实现真实的SMTP邮件发送
	// 可以使用 net/smtp 包或第三方库如 gomail
	// 这里仅作为示例，实际实现需要根据具体的SMTP服务配置
	
	log.Printf("通过SMTP发送邮件: %s -> %s", s.from, to)
	
	// 示例实现（需要替换为真实的SMTP发送逻辑）
	return fmt.Errorf("SMTP邮件发送功能尚未实现，请配置真实的SMTP服务")
}
