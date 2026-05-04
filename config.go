package main

import (
	"net"
	"os"
	"strings"
)

const defaultAdminPassword = "change-me"

type Config struct {
	Addr                     string
	AdminPassword            string
	DataFile                 string
	CookieSecure             bool
	UsesDefaultAdminPassword bool
}

func ConfigFromEnv() Config {
	adminPassword := envOr("ADMIN_PASSWORD", defaultAdminPassword)

	return Config{
		Addr:                     resolveAddr(),
		AdminPassword:            adminPassword,
		DataFile:                 envOr("DATA_FILE", "data/results.json"),
		CookieSecure:             envBool("COOKIE_SECURE", false),
		UsesDefaultAdminPassword: adminPassword == defaultAdminPassword,
	}
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
