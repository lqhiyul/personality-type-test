package config

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

func TestConfigValidateRejectsProductionDefaults(t *testing.T) {
	cfg := Config{
		AdminPassword:            DefaultAdminPassword,
		CookieSecure:             false,
		Production:               true,
		UsesDefaultAdminPassword: true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config with default admin password to fail")
	}

	cfg.AdminPassword = "strong-password"
	cfg.UsesDefaultAdminPassword = false
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production config without secure cookies to fail")
	}

	cfg.CookieSecure = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected hardened production config to pass, got %v", err)
	}
}

func TestConfigFromEnvReadsTrustedProxyCIDRs(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 192.168.0.0/16")

	cfg := ConfigFromEnv()
	if len(cfg.TrustedProxyCIDRs) != 2 {
		t.Fatalf("expected two trusted proxy CIDRs, got %+v", cfg.TrustedProxyCIDRs)
	}
}
