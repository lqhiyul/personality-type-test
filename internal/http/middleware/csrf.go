package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

type CSRF struct {
	SecureCookie bool
}

func (c CSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := csrfTokenFromCookie(r)
		if token == "" {
			token = newCSRFToken()
		}
		setCSRFCookie(w, token, c.SecureCookie)

		if unsafeMethod(r.Method) && !validCSRFToken(token, r.Header.Get(CSRFHeaderName)) {
			writeCSRFError(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func unsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func csrfTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(cookie.Value)
	if !validCSRFFormat(value) {
		return ""
	}
	return value
}

func validCSRFToken(cookieToken, headerToken string) bool {
	headerToken = strings.TrimSpace(headerToken)
	if !validCSRFFormat(cookieToken) || !validCSRFFormat(headerToken) {
		return false
	}
	if len(cookieToken) != len(headerToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) == 1
}

func validCSRFFormat(token string) bool {
	if len(token) != 64 {
		return false
	}
	for _, r := range token {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}

func newCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", 64)
	}
	return hex.EncodeToString(b)
}

func setCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   24 * 60 * 60,
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeCSRFError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		_, _ = w.Write([]byte(`{"error":"csrf token is missing or invalid"}` + "\n"))
		return
	}
	_, _ = w.Write([]byte("csrf token is missing or invalid\n"))
}
