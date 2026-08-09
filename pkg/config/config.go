package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host        string
	Port        int
	NoBrowser   bool
	GinMode     string
	FrontendDir string

	StorageDriver   string
	DatabaseURL     string
	RequirePostgres bool

	AuthEnabled         bool
	RegistrationEnabled bool
	JWTSecret           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	CookieSecure        bool

	EmailVerificationEnabled bool
	PublicURL                string
	SMTPHost                 string
	SMTPPort                 int
	SMTPUsername             string
	SMTPPassword             string
	SMTPFrom                 string
	SMTPTLSMode              string
	EmailVerificationTTL     time.Duration
}

// Load 在配置层中读取并整理所需数据。
func Load(args []string) Config {
	cfg := Config{
		Host:        envString("TRACKER_HOST", "127.0.0.1"),
		Port:        envPort("TRACKER_PORT", 8000),
		NoBrowser:   envBool("TRACKER_NO_BROWSER", false),
		GinMode:     envString("GIN_MODE", ""),
		FrontendDir: envString("TRACKER_FRONTEND_DIR", "frontend/dist"),

		StorageDriver:   strings.ToLower(envString("TRACKER_STORAGE", "json")),
		DatabaseURL:     envString("TRACKER_DATABASE_URL", ""),
		RequirePostgres: envBool("TRACKER_REQUIRE_POSTGRES", false),

		JWTSecret:           envString("TRACKER_JWT_SECRET", ""),
		RegistrationEnabled: envBool("TRACKER_REGISTRATION_ENABLED", true),
		AccessTokenTTL:      envDuration("TRACKER_ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:     envDuration("TRACKER_REFRESH_TOKEN_TTL", 30*24*time.Hour),
		CookieSecure:        envBool("TRACKER_COOKIE_SECURE", false),

		EmailVerificationEnabled: envBool("TRACKER_EMAIL_VERIFICATION_ENABLED", false),
		PublicURL:                envString("TRACKER_PUBLIC_URL", ""),
		SMTPHost:                 envString("TRACKER_SMTP_HOST", ""),
		SMTPPort:                 envPort("TRACKER_SMTP_PORT", 465),
		SMTPUsername:             envString("TRACKER_SMTP_USERNAME", ""),
		SMTPPassword:             envString("TRACKER_SMTP_PASSWORD", ""),
		SMTPFrom:                 envString("TRACKER_SMTP_FROM", ""),
		SMTPTLSMode:              strings.ToLower(envString("TRACKER_SMTP_TLS_MODE", "implicit")),
		EmailVerificationTTL:     envDuration("TRACKER_EMAIL_VERIFICATION_TTL", 24*time.Hour),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--no-browser":
			cfg.NoBrowser = true
		case arg == "--port" || arg == "-p":
			if i+1 < len(args) {
				if port, ok := parsePort(args[i+1]); ok {
					cfg.Port = port
				}
				i++
			}
		case strings.HasPrefix(arg, "--port="):
			if port, ok := parsePort(strings.TrimPrefix(arg, "--port=")); ok {
				cfg.Port = port
			}
		case arg == "--host":
			if i+1 < len(args) {
				cfg.Host = strings.TrimSpace(args[i+1])
				i++
			}
		case strings.HasPrefix(arg, "--host="):
			cfg.Host = strings.TrimSpace(strings.TrimPrefix(arg, "--host="))
		}
	}

	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	cfg.StorageDriver = strings.ToLower(strings.TrimSpace(cfg.StorageDriver))
	if cfg.StorageDriver == "" {
		cfg.StorageDriver = "json"
	}
	cfg.AuthEnabled = cfg.StorageDriver == "postgres"
	if cfg.AuthEnabled && strings.TrimSpace(cfg.JWTSecret) == "" {
		cfg.JWTSecret = persistentJWTSecret()
	}
	if strings.HasPrefix(strings.ToLower(cfg.PublicURL), "https://") {
		cfg.CookieSecure = true
	}
	return cfg
}

// Validate fails fast for unsupported or incomplete startup configuration.
// Local JSON mode remains an explicit supported mode, while PostgreSQL mode
// must always include a usable database URL.
// Validate 检查配置是否足以安全启动当前存储模式。
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("TRACKER_HOST 不能为空")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("TRACKER_PORT 必须在 1 到 65535 之间")
	}
	switch c.StorageDriver {
	case "json", "postgres":
	default:
		return fmt.Errorf("TRACKER_STORAGE 仅支持 json 或 postgres")
	}
	switch c.GinMode {
	case "", ginModeDebug, ginModeRelease, ginModeTest:
	default:
		return fmt.Errorf("GIN_MODE 仅支持 debug、release 或 test")
	}
	if c.RequirePostgres && c.StorageDriver != "postgres" {
		return fmt.Errorf("TRACKER_REQUIRE_POSTGRES=true 时 TRACKER_STORAGE 必须为 postgres")
	}
	if c.StorageDriver == "postgres" {
		if err := validatePostgresURL(c.DatabaseURL); err != nil {
			return err
		}
		if len(strings.TrimSpace(c.JWTSecret)) < 32 {
			return fmt.Errorf("PostgreSQL 模式下 TRACKER_JWT_SECRET 至少需要 32 个字符")
		}
	}
	return nil
}

const (
	ginModeDebug   = "debug"
	ginModeRelease = "release"
	ginModeTest    = "test"
)

// validatePostgresURL 在配置层中校验输入或判断当前条件。
func validatePostgresURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil {
		return fmt.Errorf("TRACKER_DATABASE_URL 不是有效的 PostgreSQL 连接地址")
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("TRACKER_DATABASE_URL 必须使用 postgres:// 或 postgresql://")
	}
	if parsed.Host == "" || strings.Trim(parsed.Path, "/") == "" {
		return fmt.Errorf("TRACKER_DATABASE_URL 必须包含数据库主机和数据库名")
	}
	return nil
}

// Address 在配置层中创建或更新相应状态。
func (c Config) Address(port int) string {
	return fmt.Sprintf("%s:%d", c.Host, port)
}

// envString 在配置层中完成本文件定义的局部处理。
func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// envPort 在配置层中完成本文件定义的局部处理。
func envPort(key string, fallback int) int {
	if port, ok := parsePort(os.Getenv(key)); ok {
		return port
	}
	return fallback
}

// envBool 在配置层中完成本文件定义的局部处理。
func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// envDuration 在配置层中完成本文件定义的局部处理。
func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}

// parsePort 在配置层中解析外部输入为内部数据。
func parsePort(value string) (int, bool) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return port, port > 0 && port <= 65535
}

// randomSecret 在配置层中完成本文件定义的局部处理。
func randomSecret() string {
	var buffer [32]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return fmt.Sprintf("study-tracker-dev-secret-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer[:])
}

// persistentJWTSecret keeps local PostgreSQL sessions valid across restarts
// without requiring a development-only environment variable. Production
// deployments should still set TRACKER_JWT_SECRET explicitly and keep it in a
// proper secret manager.
// persistentJWTSecret 读取或生成开发环境可复用的 JWT 密钥。
func persistentJWTSecret() string {
	dir := strings.TrimSpace(os.Getenv("TRACKER_DATA_DIR"))
	if dir == "" {
		dir = "data"
	}
	path := filepath.Join(dir, ".tracker-jwt-secret")
	if existing, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(existing)) != "" {
		return strings.TrimSpace(string(existing))
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return randomSecret()
	}
	secret := randomSecret()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		if existing, readErr := os.ReadFile(path); readErr == nil && strings.TrimSpace(string(existing)) != "" {
			return strings.TrimSpace(string(existing))
		}
		return secret
	}
	if err != nil {
		return secret
	}
	if _, err = file.WriteString(secret); err == nil {
		err = file.Sync()
	}
	_ = file.Close()
	if err != nil {
		_ = os.Remove(path)
	}
	return secret
}
