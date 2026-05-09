package main

import "testing"

func TestConfigFromEnvUsesPortByDefault(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("HOST", "")
	t.Setenv("PORT", "18080")

	cfg := ConfigFromEnv()
	if cfg.Addr != ":18080" {
		t.Fatalf("expected :18080, got %q", cfg.Addr)
	}
}

func TestConfigFromEnvUsesHostForLocalNetwork(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("PORT", "18080")

	cfg := ConfigFromEnv()
	if cfg.Addr != "0.0.0.0:18080" {
		t.Fatalf("expected 0.0.0.0:18080, got %q", cfg.Addr)
	}
}

func TestConfigFromEnvAddrOverridesHostAndPort(t *testing.T) {
	t.Setenv("ADDR", "127.0.0.1:19090")
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("PORT", "18080")

	cfg := ConfigFromEnv()
	if cfg.Addr != "127.0.0.1:19090" {
		t.Fatalf("expected ADDR override, got %q", cfg.Addr)
	}
}

func TestConfigFromEnvUsesDefaultDatabasePath(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")

	cfg := ConfigFromEnv()
	if cfg.DatabasePath != defaultDatabasePath {
		t.Fatalf("expected default database path %q, got %q", defaultDatabasePath, cfg.DatabasePath)
	}
}

func TestConfigFromEnvUsesDatabasePath(t *testing.T) {
	t.Setenv("DATABASE_PATH", "data/custom.db")

	cfg := ConfigFromEnv()
	if cfg.DatabasePath != "data/custom.db" {
		t.Fatalf("expected configured database path, got %q", cfg.DatabasePath)
	}
}
