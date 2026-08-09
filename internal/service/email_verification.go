package service

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	models "study-tracker-go/internal/model"
	"study-tracker-go/internal/repository"
	"study-tracker-go/pkg/config"
)

// validateEmailVerificationConfig 在业务层中校验输入或判断当前条件。
func validateEmailVerificationConfig(cfg config.Config) error {
	if !cfg.EmailVerificationEnabled {
		return nil
	}
	if !cfg.AuthEnabled {
		return fmt.Errorf("邮箱验证只能在 PostgreSQL 登录模式下启用")
	}
	publicURL, err := url.Parse(cfg.PublicURL)
	if err != nil || publicURL.Host == "" || (publicURL.Scheme != "https" && publicURL.Scheme != "http") {
		return fmt.Errorf("TRACKER_PUBLIC_URL 必须是可访问的 http(s) 地址")
	}
	if publicURL.Scheme != "https" && !isLocalHost(publicURL.Hostname()) {
		return fmt.Errorf("公网邮箱验证必须使用 HTTPS，请将 TRACKER_PUBLIC_URL 设置为 HTTPS 域名")
	}
	if strings.TrimSpace(cfg.SMTPHost) == "" || strings.TrimSpace(cfg.SMTPFrom) == "" {
		return fmt.Errorf("启用邮箱验证时必须设置 TRACKER_SMTP_HOST 和 TRACKER_SMTP_FROM")
	}
	from, err := mail.ParseAddress(cfg.SMTPFrom)
	if err != nil || from.Address != strings.TrimSpace(cfg.SMTPFrom) {
		return fmt.Errorf("TRACKER_SMTP_FROM 格式不正确")
	}
	if cfg.SMTPPort <= 0 || cfg.SMTPPort > 65535 {
		return fmt.Errorf("TRACKER_SMTP_PORT 必须是有效端口")
	}
	if cfg.EmailVerificationTTL <= 0 {
		return fmt.Errorf("TRACKER_EMAIL_VERIFICATION_TTL 必须大于 0")
	}
	if (strings.TrimSpace(cfg.SMTPUsername) == "") != (strings.TrimSpace(cfg.SMTPPassword) == "") {
		return fmt.Errorf("TRACKER_SMTP_USERNAME 与 TRACKER_SMTP_PASSWORD 必须同时设置")
	}
	switch cfg.SMTPTLSMode {
	case "implicit", "starttls", "none":
	default:
		return fmt.Errorf("TRACKER_SMTP_TLS_MODE 必须为 implicit、starttls 或 none")
	}
	if cfg.SMTPTLSMode == "none" && strings.TrimSpace(cfg.SMTPUsername) != "" {
		return fmt.Errorf("使用 SMTP 账号时必须启用 TLS")
	}
	return nil
}

// isLocalHost 在业务层中校验输入或判断当前条件。
func isLocalHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validateEmailAddress 在业务层中校验输入或判断当前条件。
func validateEmailAddress(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || !strings.Contains(parsed.Address, "@") {
		return "", fmt.Errorf("邮箱格式不正确")
	}
	return email, nil
}

// createEmailVerification 在业务层中创建或更新相应状态。
func createEmailVerification(ctx context.Context, repo repository.AuthRepository, cfg config.Config, user models.User) error {
	token, err := randomToken(32)
	if err != nil {
		return err
	}
	if err := repo.CreateEmailVerificationToken(ctx, user.ID, hashRefreshToken(token), timeNow().Add(cfg.EmailVerificationTTL)); err != nil {
		return err
	}
	if err := sendVerificationEmail(ctx, cfg, user.Email, token); err != nil {
		return fmt.Errorf("验证邮件发送失败，请稍后重试")
	}
	return nil
}

var timeNow = func() time.Time { return time.Now() }

// sendVerificationEmail 在业务层中完成本文件定义的局部处理。
func sendVerificationEmail(ctx context.Context, cfg config.Config, recipient string, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	link := strings.TrimRight(cfg.PublicURL, "/") + "/verify-email?token=" + url.QueryEscape(token)
	body := "你好，\n\n请打开下面的链接完成学习空间账号验证：\n\n" + link + "\n\n此链接将在 " + cfg.EmailVerificationTTL.String() + " 后失效。如果不是你本人注册，请忽略此邮件。\n"
	message := buildEmailMessage(cfg.SMTPFrom, recipient, "验证你的学习空间账号", body)
	return smtpSend(cfg, recipient, message)
}

var smtpSend = sendSMTP

// buildEmailMessage 在业务层中构造、编码或标准化数据。
func buildEmailMessage(from string, recipient string, subject string, body string) []byte {
	encodedSubject := mime.QEncoding.Encode("UTF-8", subject)
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	return []byte("From: " + from + "\r\n" +
		"To: " + recipient + "\r\n" +
		"Subject: " + encodedSubject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\n" +
		encodedBody + "\r\n")
}

// sendSMTP 在业务层中完成本文件定义的局部处理。
func sendSMTP(cfg config.Config, recipient string, message []byte) error {
	address := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
	var (
		client *smtp.Client
		err    error
	)
	if cfg.SMTPTLSMode == "implicit" {
		conn, dialErr := tls.Dial("tcp", address, &tls.Config{ServerName: cfg.SMTPHost, MinVersion: tls.VersionTLS12})
		if dialErr != nil {
			return dialErr
		}
		client, err = smtp.NewClient(conn, cfg.SMTPHost)
	} else {
		client, err = smtp.Dial(address)
		if err == nil && cfg.SMTPTLSMode == "starttls" {
			err = client.StartTLS(&tls.Config{ServerName: cfg.SMTPHost, MinVersion: tls.VersionTLS12})
		}
	}
	if err != nil {
		return err
	}
	defer client.Close()
	if cfg.SMTPUsername != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)); err != nil {
			return err
		}
	}
	if err := client.Mail(cfg.SMTPFrom); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
