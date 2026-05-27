package config

import (
	"errors"
	"net"
	"os"
	"strings"
)

const (
	DefaultAdminPassword = "change-me"
	DefaultDatabasePath  = "data/app.db"
	DefaultDataFile      = "data/results.json"
	DefaultStaticDir     = "web/static"
)

const defaultAdminPassword = DefaultAdminPassword
const defaultDatabasePath = DefaultDatabasePath

type Config struct {
	Addr                     string
	AdminPassword            string
	DataFile                 string
	DatabasePath             string
	StaticDir                string
	CookieSecure             bool
	Production               bool
	TrustedProxyCIDRs        []string
	UsesDefaultAdminPassword bool
}

func ConfigFromEnv() Config {
	adminPassword := envOr("ADMIN_PASSWORD", defaultAdminPassword)
	production := envBool("PRODUCTION", false)
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		production = true
	}

	return Config{
		Addr:                     resolveAddr(),
		AdminPassword:            adminPassword,
		DataFile:                 envOr("DATA_FILE", DefaultDataFile),
		DatabasePath:             envOr("DATABASE_PATH", defaultDatabasePath),
		StaticDir:                envOr("STATIC_DIR", DefaultStaticDir),
		CookieSecure:             envBool("COOKIE_SECURE", false),
		Production:               production,
		TrustedProxyCIDRs:        envList("TRUSTED_PROXY_CIDRS"),
		UsesDefaultAdminPassword: adminPassword == defaultAdminPassword,
	}
}

func FromEnv() Config {
	return ConfigFromEnv()
}

func (c Config) Validate() error {
	if c.Production && c.UsesDefaultAdminPassword {
		return errors.New("ADMIN_PASSWORD must be changed when PRODUCTION=true or APP_ENV=production")
	}
	if c.Production && !c.CookieSecure {
		return errors.New("COOKIE_SECURE=true is required when PRODUCTION=true or APP_ENV=production")
	}
	return nil
}

func resolveAddr() string {
	if addr := strings.TrimSpace(os.Getenv("ADDR")); addr != "" {
		return addr
	}

	host := strings.TrimSpace(os.Getenv("HOST"))
	port := strings.TrimSpace(envOr("PORT", "8080"))
	if strings.Contains(port, ":") && host == "" {
		return port
	}

	return net.JoinHostPort(host, strings.TrimPrefix(port, ":"))
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func envList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
