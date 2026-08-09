package config

import "testing"

// TestLoadParsesFlags 在配置层中验证对应场景的行为与边界条件。
func TestLoadParsesFlags(t *testing.T) {
	cfg := Load([]string{"--port", "8010", "--host", "0.0.0.0", "--no-browser"})
	if cfg.Port != 8010 {
		t.Fatalf("expected port 8010, got %d", cfg.Port)
	}
	if cfg.Host != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %q", cfg.Host)
	}
	if !cfg.NoBrowser {
		t.Fatal("expected no-browser to be enabled")
	}

	cfg = Load([]string{"--port=8020", "--host=localhost"})
	if cfg.Port != 8020 {
		t.Fatalf("expected port 8020, got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Fatalf("expected host localhost, got %q", cfg.Host)
	}
}

// TestLoadReadsEnvironment 在配置层中验证对应场景的行为与边界条件。
func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("TRACKER_PORT", "8030")
	t.Setenv("TRACKER_HOST", "localhost")
	t.Setenv("TRACKER_NO_BROWSER", "true")
	t.Setenv("TRACKER_STORAGE", "Postgres")
	t.Setenv("TRACKER_DATABASE_URL", "postgres://example")
	t.Setenv("TRACKER_JWT_SECRET", "test-secret")
	t.Setenv("TRACKER_REGISTRATION_ENABLED", "false")
	t.Setenv("TRACKER_EMAIL_VERIFICATION_ENABLED", "true")
	t.Setenv("TRACKER_PUBLIC_URL", "https://study.example.com")
	t.Setenv("TRACKER_SMTP_HOST", "smtp.example.com")
	t.Setenv("TRACKER_SMTP_PORT", "587")
	t.Setenv("TRACKER_SMTP_USERNAME", "mailer")
	t.Setenv("TRACKER_SMTP_PASSWORD", "smtp-secret")
	t.Setenv("TRACKER_SMTP_FROM", "mailer@example.com")
	t.Setenv("TRACKER_SMTP_TLS_MODE", "starttls")

	cfg := Load(nil)
	if cfg.Port != 8030 {
		t.Fatalf("expected env port 8030, got %d", cfg.Port)
	}
	if cfg.Host != "localhost" {
		t.Fatalf("expected env host localhost, got %q", cfg.Host)
	}
	if !cfg.NoBrowser {
		t.Fatal("expected no-browser from env")
	}
	if cfg.StorageDriver != "postgres" {
		t.Fatalf("expected normalized storage driver, got %q", cfg.StorageDriver)
	}
	if cfg.DatabaseURL != "postgres://example" {
		t.Fatalf("expected database url from env, got %q", cfg.DatabaseURL)
	}
	if !cfg.AuthEnabled {
		t.Fatal("expected auth to be enabled for postgres storage")
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("expected jwt secret from env, got %q", cfg.JWTSecret)
	}
	if cfg.RegistrationEnabled {
		t.Fatal("expected registration to be disabled from env")
	}
	if !cfg.EmailVerificationEnabled || cfg.PublicURL != "https://study.example.com" || cfg.SMTPPort != 587 || cfg.SMTPTLSMode != "starttls" {
		t.Fatalf("expected email verification settings from environment, got %#v", cfg)
	}
}

// TestLoadKeepsJSONAuthDisabled 在配置层中验证对应场景的行为与边界条件。
func TestLoadKeepsJSONAuthDisabled(t *testing.T) {
	t.Setenv("TRACKER_STORAGE", "json")

	cfg := Load(nil)
	if cfg.AuthEnabled {
		t.Fatal("expected auth to be disabled for json storage")
	}
}

// TestValidateRejectsJSONWhenPostgresIsRequired 在配置层中验证对应场景的行为与边界条件。
func TestValidateRejectsJSONWhenPostgresIsRequired(t *testing.T) {
	t.Setenv("TRACKER_STORAGE", "json")
	t.Setenv("TRACKER_REQUIRE_POSTGRES", "true")

	if err := Load(nil).Validate(); err == nil {
		t.Fatal("expected a required PostgreSQL configuration to reject JSON storage")
	}
}

// TestValidateRejectsIncompletePostgresConfiguration 在配置层中验证对应场景的行为与边界条件。
func TestValidateRejectsIncompletePostgresConfiguration(t *testing.T) {
	tests := []Config{
		{Host: "127.0.0.1", Port: 8000, StorageDriver: "postgres", JWTSecret: "01234567890123456789012345678901"},
		{Host: "127.0.0.1", Port: 8000, StorageDriver: "postgres", DatabaseURL: "mysql://db/study", JWTSecret: "01234567890123456789012345678901"},
		{Host: "127.0.0.1", Port: 8000, StorageDriver: "postgres", DatabaseURL: "postgres://db/study", JWTSecret: "short"},
	}
	for _, cfg := range tests {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected invalid PostgreSQL configuration to fail: %#v", cfg)
		}
	}
}

// TestValidateAcceptsPostgresConfiguration 在配置层中验证对应场景的行为与边界条件。
func TestValidateAcceptsPostgresConfiguration(t *testing.T) {
	cfg := Config{
		Host:          "0.0.0.0",
		Port:          8000,
		GinMode:       "release",
		StorageDriver: "postgres",
		DatabaseURL:   "postgres://user:password@db:5432/study?sslmode=disable",
		JWTSecret:     "01234567890123456789012345678901",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid PostgreSQL configuration, got %v", err)
	}
}
