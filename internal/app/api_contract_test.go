package app

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAPIErrorResponsesUseConsistentShape(t *testing.T) {
	app := newTestApp(t)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
		status int
	}{
		{name: "auth me unauthorized", method: http.MethodGet, path: "/api/auth/me", status: http.StatusUnauthorized},
		{name: "method not allowed", method: http.MethodGet, path: "/api/auth/login", status: http.StatusMethodNotAllowed},
		{name: "validation error", method: http.MethodPost, path: "/api/auth/register", body: map[string]string{
			"username": "x",
			"email":    "bad-email",
			"password": "short",
		}, status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSON(app, tc.method, tc.path, tc.body)
			if rec.Code != tc.status {
				t.Fatalf("expected status %d, got %d: %s", tc.status, rec.Code, rec.Body.String())
			}
			var payload map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if payload["error"] == "" {
				t.Fatalf("expected non-empty error field, got %+v", payload)
			}
		})
	}
}

func TestAdminExportRequiresAdminSession(t *testing.T) {
	app := newTestApp(t)

	rec := performJSON(app, http.MethodGet, "/api/results/export", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin export to require admin auth, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload["error"] != "admin authentication required" {
		t.Fatalf("unexpected export error: %+v", payload)
	}
}
