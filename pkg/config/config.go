package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// Validate rejects a production configuration that would silently fall back
// to the local JSON store. Local JSON mode remains an explicit supported mode.
func (c Config) Validate() error {
	if c.RequirePostgres && c.StorageDriver != "postgres" {
		return fmt.Errorf("TRACKER_REQUIRE_POSTGRES=true 时 TRACKER_STORAGE 必须为 postgres")
	}
	return nil
}

func (c Config) Address(port int) string {
	return fmt.Sprintf("%s:%d", c.Host, port)
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envPort(key string, fallback int) int {
	if port, ok := parsePort(os.Getenv(key)); ok {
		return port
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

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

func parsePort(value string) (int, bool) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return port, port > 0 && port <= 65535
}

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
