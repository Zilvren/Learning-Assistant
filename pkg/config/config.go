package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
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

	StorageDriver string
	DatabaseURL   string

	AuthEnabled         bool
	RegistrationEnabled bool
	JWTSecret           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	CookieSecure        bool
}

func Load(args []string) Config {
	cfg := Config{
		Host:        envString("TRACKER_HOST", "127.0.0.1"),
		Port:        envPort("TRACKER_PORT", 8000),
		NoBrowser:   envBool("TRACKER_NO_BROWSER", false),
		GinMode:     envString("GIN_MODE", ""),
		FrontendDir: envString("TRACKER_FRONTEND_DIR", "frontend/dist"),

		StorageDriver: strings.ToLower(envString("TRACKER_STORAGE", "json")),
		DatabaseURL:   envString("TRACKER_DATABASE_URL", ""),

		JWTSecret:           envString("TRACKER_JWT_SECRET", ""),
		RegistrationEnabled: envBool("TRACKER_REGISTRATION_ENABLED", true),
		AccessTokenTTL:      envDuration("TRACKER_ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:     envDuration("TRACKER_REFRESH_TOKEN_TTL", 30*24*time.Hour),
		CookieSecure:        envBool("TRACKER_COOKIE_SECURE", false),
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
		cfg.JWTSecret = randomSecret()
	}
	return cfg
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
