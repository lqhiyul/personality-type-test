package app

import (
	"net/http/httptest"
	"testing"
)

func TestLoginRateLimitKeyIgnoresForwardedHeadersWithoutTrustedProxy(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")

	if got := app.loginRateLimitKey(req); got != "203.0.113.10" {
		t.Fatalf("expected remote address to be used, got %q", got)
	}
}

func TestLoginRateLimitKeyUsesForwardedForFromTrustedProxy(t *testing.T) {
	proxies, err := parseTrustedProxyCIDRs([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("parseTrustedProxyCIDRs() error = %v", err)
	}
	app := &App{trustedProxies: proxies}
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "10.1.2.3:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 10.9.8.7")

	if got := app.loginRateLimitKey(req); got != "198.51.100.20" {
		t.Fatalf("expected client IP from trusted X-Forwarded-For chain, got %q", got)
	}
}
