package config

import "testing"

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

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("TRACKER_PORT", "8030")
	t.Setenv("TRACKER_HOST", "localhost")
	t.Setenv("TRACKER_NO_BROWSER", "true")
	t.Setenv("TRACKER_STORAGE", "Postgres")
	t.Setenv("TRACKER_DATABASE_URL", "postgres://example")
	t.Setenv("TRACKER_JWT_SECRET", "test-secret")
	t.Setenv("TRACKER_REGISTRATION_ENABLED", "false")

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
}

func TestLoadKeepsJSONAuthDisabled(t *testing.T) {
	t.Setenv("TRACKER_STORAGE", "json")

	cfg := Load(nil)
	if cfg.AuthEnabled {
		t.Fatal("expected auth to be disabled for json storage")
	}
}
