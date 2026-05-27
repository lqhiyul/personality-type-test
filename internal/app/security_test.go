package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpmiddleware "github.com/lqhiyul/personality-type-test/internal/http/middleware"
)

func TestSecurityHeadersAndRequestID(t *testing.T) {
	app := newTestApp(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(httpmiddleware.RequestIDHeader, "test-request-id")
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected healthz 200, got %d", rec.Code)
	}
	for name, want := range map[string]string{
		"X-Content-Type-Options":       "nosniff",
		"X-Frame-Options":              "DENY",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		httpmiddleware.RequestIDHeader: "test-request-id",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Fatalf("expected %s=%q, got %q", name, want, got)
		}
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
}

func TestCSRFMiddlewareRejectsUnsafeRequestsWithoutToken(t *testing.T) {
	app := newTestApp(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/submit", nil)
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected missing CSRF token to return 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if findCookie(rec.Result().Cookies(), httpmiddleware.CSRFCookieName) == nil {
		t.Fatal("expected CSRF rejection to set a token cookie for retry")
	}
}

func TestCSRFMiddlewareAllowsValidUnsafeRequest(t *testing.T) {
	app := newTestApp(t)

	rec := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Yehor",
		"answers":  answersForType("INTJ"),
		"duration": 10,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected valid CSRF request to succeed, got %d: %s", rec.Code, rec.Body.String())
	}
}
