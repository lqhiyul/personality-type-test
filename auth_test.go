package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthRegisterSuccessAutoLoginAndStoresPasswordHash(t *testing.T) {
	app := newTestApp(t)

	rec := performJSON(app, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "Example_User",
		"email":    "USER@Example.COM",
		"password": "StrongPassword123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected register 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "password") {
		t.Fatalf("response leaked password data: %s", rec.Body.String())
	}

	var user authUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if user.ID == 0 || user.Username != "example_user" || user.Email != "user@example.com" || user.DisplayName != "example_user" {
		t.Fatalf("unexpected registered user: %+v", user)
	}

	cookie := findCookie(rec.Result().Cookies(), userSessionCookieName)
	if cookie == nil {
		t.Fatal("expected regular user session cookie")
	}
	if !cookie.HttpOnly || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge <= 0 {
		t.Fatalf("unexpected user session cookie settings: %+v", cookie)
	}

	var storedHash string
	if err := app.userStore.db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, user.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read password_hash: %v", err)
	}
	if storedHash == "StrongPassword123" {
		t.Fatal("password_hash must not equal plaintext password")
	}
	if !CheckPasswordHash("StrongPassword123", storedHash) {
		t.Fatal("stored password_hash should verify the original password")
	}

	me := performJSON(app, http.MethodGet, "/api/auth/me", nil, cookie)
	if me.Code != http.StatusOK {
		t.Fatalf("expected auto-login /me 200, got %d: %s", me.Code, me.Body.String())
	}
}

func TestAuthRegisterRejectsDuplicatesAndInvalidInput(t *testing.T) {
	app := newTestApp(t)

	seed := performJSON(app, http.MethodPost, "/api/auth/register", map[string]string{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "StrongPassword123",
	})
	if seed.Code != http.StatusCreated {
		t.Fatalf("seed register got %d: %s", seed.Code, seed.Body.String())
	}

	cases := []struct {
		name   string
		body   map[string]string
		status int
	}{
		{name: "duplicate username", body: map[string]string{"username": "Alice", "email": "alice2@example.com", "password": "StrongPassword123"}, status: http.StatusConflict},
		{name: "duplicate email", body: map[string]string{"username": "alice2", "email": "ALICE@example.com", "password": "StrongPassword123"}, status: http.StatusConflict},
		{name: "invalid email", body: map[string]string{"username": "bob", "email": "not-an-email", "password": "StrongPassword123"}, status: http.StatusBadRequest},
		{name: "invalid username", body: map[string]string{"username": "bad name!", "email": "bad@example.com", "password": "StrongPassword123"}, status: http.StatusBadRequest},
		{name: "weak password", body: map[string]string{"username": "weak", "email": "weak@example.com", "password": "short"}, status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSON(app, http.MethodPost, "/api/auth/register", tc.body)
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthLoginByEmailAndUsername(t *testing.T) {
	app := newTestApp(t)
	registerAccountForTest(t, app, "alice", "alice@example.com", "StrongPassword123")

	for _, body := range []map[string]string{
		{"emailOrUsername": "alice@example.com", "password": "StrongPassword123"},
		{"emailOrUsername": "Alice", "password": "StrongPassword123"},
	} {
		rec := performJSON(app, http.MethodPost, "/api/auth/login", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected login 200 for %+v, got %d: %s", body, rec.Code, rec.Body.String())
		}
		if findCookie(rec.Result().Cookies(), userSessionCookieName) == nil {
			t.Fatal("expected login to set regular user session cookie")
		}
	}
}

func TestAuthLoginRejectsWrongPasswordAndUnknownUserGenerically(t *testing.T) {
	app := newTestApp(t)
	registerAccountForTest(t, app, "alice", "alice@example.com", "StrongPassword123")

	wrongPassword := performJSON(app, http.MethodPost, "/api/auth/login", map[string]string{
		"emailOrUsername": "alice",
		"password":        "WrongPassword123",
	})
	unknownUser := performJSON(app, http.MethodPost, "/api/auth/login", map[string]string{
		"emailOrUsername": "unknown",
		"password":        "WrongPassword123",
	})

	for name, rec := range map[string]*httptest.ResponseRecorder{
		"wrong password": wrongPassword,
		"unknown user":   unknownUser,
	} {
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d: %s", name, rec.Code, rec.Body.String())
		}
		var body map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("%s: decode body: %v", name, err)
		}
		if body["error"] != "invalid credentials" {
			t.Fatalf("%s: expected generic invalid credentials, got %+v", name, body)
		}
	}
}

func TestAuthLoginRateLimit(t *testing.T) {
	app := newTestApp(t)
	now := time.Unix(1000, 0).UTC()
	limiter := newLoginRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }
	app.userLoginLimiter = limiter
	registerAccountForTest(t, app, "alice", "alice@example.com", "StrongPassword123")

	first := performJSON(app, http.MethodPost, "/api/auth/login", map[string]string{
		"emailOrUsername": "alice",
		"password":        "WrongPassword123",
	})
	if first.Code != http.StatusUnauthorized {
		t.Fatalf("expected first failure 401, got %d: %s", first.Code, first.Body.String())
	}

	limited := performJSON(app, http.MethodPost, "/api/auth/login", map[string]string{
		"emailOrUsername": "alice",
		"password":        "WrongPassword123",
	})
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second failure to rate limit, got %d: %s", limited.Code, limited.Body.String())
	}
	if limited.Header().Get("Retry-After") != "60" {
		t.Fatalf("expected Retry-After 60, got %q", limited.Header().Get("Retry-After"))
	}

	blockedCorrect := performJSON(app, http.MethodPost, "/api/auth/login", map[string]string{
		"emailOrUsername": "alice",
		"password":        "StrongPassword123",
	})
	if blockedCorrect.Code != http.StatusTooManyRequests {
		t.Fatalf("expected correct password to remain blocked during cooldown, got %d", blockedCorrect.Code)
	}

	now = now.Add(time.Minute + time.Second)
	afterCooldown := performJSON(app, http.MethodPost, "/api/auth/login", map[string]string{
		"emailOrUsername": "alice",
		"password":        "StrongPassword123",
	})
	if afterCooldown.Code != http.StatusOK {
		t.Fatalf("expected login after cooldown 200, got %d: %s", afterCooldown.Code, afterCooldown.Body.String())
	}
}

func TestAuthMeAndLogout(t *testing.T) {
	app := newTestApp(t)

	loggedOut := performJSON(app, http.MethodGet, "/api/auth/me", nil)
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out /me 401, got %d", loggedOut.Code)
	}

	register := registerAccountForTest(t, app, "alice", "alice@example.com", "StrongPassword123")
	cookie := findCookie(register.Result().Cookies(), userSessionCookieName)
	if cookie == nil {
		t.Fatal("expected register to set user session cookie")
	}

	me := performJSON(app, http.MethodGet, "/api/auth/me", nil, cookie)
	if me.Code != http.StatusOK {
		t.Fatalf("expected /me 200, got %d: %s", me.Code, me.Body.String())
	}
	if strings.Contains(strings.ToLower(me.Body.String()), "password") {
		t.Fatalf("/me leaked password data: %s", me.Body.String())
	}

	logout := performJSON(app, http.MethodPost, "/api/auth/logout", nil, cookie)
	if logout.Code != http.StatusOK {
		t.Fatalf("expected logout 200, got %d: %s", logout.Code, logout.Body.String())
	}
	cleared := findCookie(logout.Result().Cookies(), userSessionCookieName)
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Fatalf("expected logout to clear user session cookie, got %+v", cleared)
	}

	afterLogout := performJSON(app, http.MethodGet, "/api/auth/me", nil, cookie)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected /me 401 after logout, got %d", afterLogout.Code)
	}
}

func TestRegularAuthStaysSeparateFromAdminAuth(t *testing.T) {
	app := newTestApp(t)
	userRegister := registerAccountForTest(t, app, "alice", "alice@example.com", "StrongPassword123")
	userCookie := findCookie(userRegister.Result().Cookies(), userSessionCookieName)
	if userCookie == nil {
		t.Fatal("expected user session cookie")
	}

	adminOnly := performJSON(app, http.MethodGet, "/api/results", nil, userCookie)
	if adminOnly.Code != http.StatusUnauthorized {
		t.Fatalf("regular user cookie should not authorize admin results, got %d", adminOnly.Code)
	}

	adminCookies := login(t, app)
	if findCookie(adminCookies, userSessionCookieName) != nil {
		t.Fatal("admin login must not set regular user session cookie")
	}
	adminMe := performJSON(app, http.MethodGet, "/api/auth/me", nil, adminCookies...)
	if adminMe.Code != http.StatusUnauthorized {
		t.Fatalf("admin cookie should not authorize regular /me, got %d", adminMe.Code)
	}

	allCookies := append([]*http.Cookie{userCookie}, adminCookies...)
	userLogout := performJSON(app, http.MethodPost, "/api/auth/logout", nil, allCookies...)
	if userLogout.Code != http.StatusOK {
		t.Fatalf("expected regular logout 200, got %d: %s", userLogout.Code, userLogout.Body.String())
	}

	adminResults := performJSON(app, http.MethodGet, "/api/results", nil, adminCookies...)
	if adminResults.Code != http.StatusOK {
		t.Fatalf("regular logout should not clear admin session, got %d: %s", adminResults.Code, adminResults.Body.String())
	}
}

func registerAccountForTest(t *testing.T, app *App, username, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	rec := performJSON(app, http.MethodPost, "/api/auth/register", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register account: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	return rec
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
