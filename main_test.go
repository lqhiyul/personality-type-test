package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestApp(t *testing.T) *App {
	t.Helper()

	store, err := NewStore(filepath.Join(t.TempDir(), "results.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	db, err := OpenAppDB(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenAppDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	sub, err := fs.Sub(staticFiles, "web/static")
	if err != nil {
		t.Fatalf("fs.Sub() error = %v", err)
	}

	return &App{
		store:            store,
		userStore:        NewUserStore(db),
		adminPassword:    "secret",
		sessionName:      "test_admin_session_" + newID(),
		baseTemplateFS:   sub,
		loginLimiter:     newLoginRateLimiter(defaultLoginFailureLimit, defaultLoginCooldown),
		userLoginLimiter: newLoginRateLimiter(defaultLoginFailureLimit, defaultLoginCooldown),
		userSessions:     newUserSessionStore(userSessionTTL),
	}
}

func answersForType(code string) []string {
	answers := make([]string, len(questions))
	for i, question := range questions {
		if strings.Contains(code, question.A) {
			answers[i] = question.A
			continue
		}
		answers[i] = question.B
	}
	return answers
}

func lowerAnswersForType(code string) []string {
	answers := answersForType(code)
	for i := range answers {
		answers[i] = strings.ToLower(answers[i])
	}
	return answers
}

func performJSON(app *App, method, target string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, app *App) []*http.Cookie {
	t.Helper()
	rec := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "secret"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected login cookie")
	}
	return cookies
}

func addStoredResult(t *testing.T, app *App, id, code string) {
	t.Helper()
	if err := app.store.Add(Result{ID: id, Name: "Yehor", Type: code, Answers: strings.Join(answersForType(code), "")}); err != nil {
		t.Fatalf("store.Add() error = %v", err)
	}
}

func TestComputeProfileBreakdown(t *testing.T) {
	t.Parallel()

	profile, err := computeProfile(answersForType("INTJ"))
	if err != nil {
		t.Fatalf("computeProfile() error = %v", err)
	}
	if profile.Type != "INTJ" {
		t.Fatalf("expected INTJ, got %q", profile.Type)
	}
	if len(profile.Dimensions) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(profile.Dimensions))
	}
	for _, dim := range profile.Dimensions {
		if dim.Percent != 100 {
			t.Fatalf("expected full preference for %s, got %d%%", dim.Key, dim.Percent)
		}
	}
}

func TestHealthz(t *testing.T) {
	app := newTestApp(t)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected healthz 200, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("unexpected healthz body: %q", rec.Body.String())
	}
}

func TestHandleSubmitReturnsProfileAndClampsDuration(t *testing.T) {
	app := newTestApp(t)
	rec := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Yehor",
		"answers":  answersForType("INFJ"),
		"duration": -20,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Type    string      `json:"type"`
		Profile TypeProfile `json:"profile"`
		Result  Result      `json:"result"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Type != "INFJ" || res.Profile.Type != "INFJ" {
		t.Fatalf("expected INFJ profile, got type=%q profile=%q", res.Type, res.Profile.Type)
	}
	if res.Result.Duration != 0 {
		t.Fatalf("expected clamped duration 0, got %d", res.Result.Duration)
	}
}

func TestSubmitAcceptsUnicodeNameAndStoresNormalizedAnswers(t *testing.T) {
	app := newTestApp(t)
	rec := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Єгор",
		"answers":  lowerAnswersForType("INTJ"),
		"duration": 42,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	results, err := app.store.All()
	if err != nil {
		t.Fatalf("store.All() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one stored result, got %d", len(results))
	}
	if results[0].Name != "Єгор" {
		t.Fatalf("expected unicode name to be preserved, got %q", results[0].Name)
	}
	if results[0].Answers != strings.Join(answersForType("INTJ"), "") {
		t.Fatalf("expected uppercase normalized answers, got %q", results[0].Answers)
	}
}

func TestSubmitRejectsInvalidPayloads(t *testing.T) {
	app := newTestApp(t)

	cases := []struct {
		name   string
		body   any
		raw    string
		status int
	}{
		{name: "invalid json", raw: `{`, status: http.StatusBadRequest},
		{name: "empty name", body: map[string]any{"name": "  ", "answers": answersForType("INTJ"), "duration": 1}, status: http.StatusBadRequest},
		{name: "control character in name", body: map[string]any{"name": "Yehor\nAdmin", "answers": answersForType("INTJ"), "duration": 1}, status: http.StatusBadRequest},
		{name: "unknown field", body: map[string]any{"name": "Yehor", "answers": answersForType("INTJ"), "duration": 1, "extra": true}, status: http.StatusBadRequest},
		{name: "invalid answer count", body: map[string]any{"name": "Yehor", "answers": []string{"I"}, "duration": 1}, status: http.StatusBadRequest},
		{name: "invalid answer value", body: map[string]any{"name": "Yehor", "answers": append([]string{"X"}, answersForType("INTJ")[1:]...), "duration": 1}, status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.raw != "" {
				req = httptest.NewRequest(http.MethodPost, "/api/submit", strings.NewReader(tc.raw))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				app.ServeHTTP(rec, req)
				if rec.Code != tc.status {
					t.Fatalf("expected %d, got %d: %s", tc.status, rec.Code, rec.Body.String())
				}
				return
			}
			rec := performJSON(app, http.MethodPost, "/api/submit", tc.body)
			if rec.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminRequiresLoginExportsAndLogout(t *testing.T) {
	app := newTestApp(t)
	addStoredResult(t, app, "one", "INTJ")

	unauthorized := httptest.NewRecorder()
	app.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/results", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized results access, got %d", unauthorized.Code)
	}

	cookies := login(t, app)
	exportRec := performJSON(app, http.MethodGet, "/api/results/export", nil, cookies...)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d", exportRec.Code)
	}
	if ct := exportRec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected CSV content type, got %q", ct)
	}
	csvBody := exportRec.Body.String()
	if !strings.Contains(csvBody, "id,name,type") || !strings.Contains(csvBody, "INTJ") {
		t.Fatalf("unexpected csv body: %s", csvBody)
	}

	logoutRec := performJSON(app, http.MethodPost, "/api/logout", nil, cookies...)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected logout 200, got %d", logoutRec.Code)
	}
	afterLogout := performJSON(app, http.MethodGet, "/api/results", nil, cookies...)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("expected session to be invalid after logout, got %d", afterLogout.Code)
	}
}

func TestAdminLoginSuccessAndFailure(t *testing.T) {
	app := newTestApp(t)

	failed := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "wrong"})
	if failed.Code != http.StatusUnauthorized {
		t.Fatalf("expected failed login 401, got %d: %s", failed.Code, failed.Body.String())
	}

	success := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "secret"})
	if success.Code != http.StatusOK {
		t.Fatalf("expected successful login 200, got %d: %s", success.Code, success.Body.String())
	}
	if len(success.Result().Cookies()) == 0 {
		t.Fatal("expected successful login to set a session cookie")
	}
}

func TestAdminLoginRateLimitDoesNotBlockOtherRoutes(t *testing.T) {
	app := newTestApp(t)
	now := time.Unix(1000, 0).UTC()
	limiter := newLoginRateLimiter(3, time.Minute)
	limiter.now = func() time.Time { return now }
	app.loginLimiter = limiter

	for i := 0; i < 2; i++ {
		rec := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "wrong"})
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: expected 401 before limit, got %d", i+1, rec.Code)
		}
	}

	limited := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "wrong"})
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limited login 429, got %d: %s", limited.Code, limited.Body.String())
	}
	if limited.Header().Get("Retry-After") != "60" {
		t.Fatalf("expected Retry-After 60, got %q", limited.Header().Get("Retry-After"))
	}

	blockedSuccess := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "secret"})
	if blockedSuccess.Code != http.StatusTooManyRequests {
		t.Fatalf("expected correct password to remain blocked during cooldown, got %d", blockedSuccess.Code)
	}

	health := httptest.NewRecorder()
	app.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("expected healthz to keep working while login is limited, got %d", health.Code)
	}

	submit := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Yehor",
		"answers":  answersForType("INTJ"),
		"duration": 10,
	})
	if submit.Code != http.StatusOK {
		t.Fatalf("expected submit to keep working while login is limited, got %d: %s", submit.Code, submit.Body.String())
	}

	now = now.Add(time.Minute + time.Second)
	afterCooldown := performJSON(app, http.MethodPost, "/api/login", map[string]string{"password": "secret"})
	if afterCooldown.Code != http.StatusOK {
		t.Fatalf("expected login after cooldown 200, got %d: %s", afterCooldown.Code, afterCooldown.Body.String())
	}
}

func TestAdminJSONExportDeleteAndClear(t *testing.T) {
	app := newTestApp(t)
	addStoredResult(t, app, "one", "INTJ")
	addStoredResult(t, app, "two", "ENFP")
	cookies := login(t, app)

	jsonRec := performJSON(app, http.MethodGet, "/api/results/export?format=json", nil, cookies...)
	if jsonRec.Code != http.StatusOK {
		t.Fatalf("expected json export 200, got %d", jsonRec.Code)
	}
	if ct := jsonRec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON content type, got %q", ct)
	}
	var exported struct {
		Results []Result `json:"results"`
	}
	if err := json.NewDecoder(jsonRec.Body).Decode(&exported); err != nil {
		t.Fatalf("decode json export: %v", err)
	}
	if len(exported.Results) != 2 {
		t.Fatalf("expected two exported results, got %d", len(exported.Results))
	}

	deleteRec := performJSON(app, http.MethodDelete, "/api/results/one", nil, cookies...)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected delete one 200, got %d", deleteRec.Code)
	}
	results, _ := app.store.All()
	if len(results) != 1 || results[0].ID != "two" {
		t.Fatalf("expected only result two after delete, got %+v", results)
	}

	clearRec := performJSON(app, http.MethodDelete, "/api/results", nil, cookies...)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("expected clear 200, got %d", clearRec.Code)
	}
	results, _ = app.store.All()
	if len(results) != 0 {
		t.Fatalf("expected no results after clear, got %+v", results)
	}
}

func TestStatsEndpointRequiresLoginAndHandlesEmptyResults(t *testing.T) {
	app := newTestApp(t)

	unauthorized := performJSON(app, http.MethodGet, "/api/stats", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected stats to require login, got %d", unauthorized.Code)
	}

	rec := performJSON(app, http.MethodGet, "/api/stats", nil, login(t, app)...)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected stats 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var stats statsResponse
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Total != 0 || stats.AverageDurationSeconds != 0 {
		t.Fatalf("unexpected empty totals: %+v", stats)
	}
	if len(stats.ByType) != 0 || len(stats.TopTypes) != 0 {
		t.Fatalf("expected empty type stats, got %+v", stats)
	}
	if stats.LatestResultAt != nil {
		t.Fatalf("expected no latest timestamp for empty stats, got %v", stats.LatestResultAt)
	}
	for _, code := range []string{"E", "I", "S", "N", "T", "F", "J", "P"} {
		if stats.AxisDistribution[code] != 0 {
			t.Fatalf("expected empty axis %s to be 0, got %d", code, stats.AxisDistribution[code])
		}
	}
}

func TestBuildStatsAggregatesOneResult(t *testing.T) {
	created := time.Unix(100, 0).UTC()
	stats := buildStats([]Result{{Type: "infj", Duration: 42, Created: created}})

	if stats.Total != 1 {
		t.Fatalf("expected total 1, got %d", stats.Total)
	}
	if stats.AverageDurationSeconds != 42 {
		t.Fatalf("expected average duration 42, got %d", stats.AverageDurationSeconds)
	}
	if !reflect.DeepEqual(stats.ByType, map[string]int{"INFJ": 1}) {
		t.Fatalf("unexpected byType: %+v", stats.ByType)
	}
	if !reflect.DeepEqual(stats.TopTypes, []typeCount{{Type: "INFJ", Count: 1}}) {
		t.Fatalf("unexpected topTypes: %+v", stats.TopTypes)
	}
	wantAxis := map[string]int{"E": 0, "I": 1, "S": 0, "N": 1, "T": 0, "F": 1, "J": 1, "P": 0}
	if !reflect.DeepEqual(stats.AxisDistribution, wantAxis) {
		t.Fatalf("unexpected axis distribution: %+v", stats.AxisDistribution)
	}
	if stats.LatestResultAt == nil || !stats.LatestResultAt.Equal(created) {
		t.Fatalf("unexpected latest timestamp: %v", stats.LatestResultAt)
	}
}

func TestBuildStatsAggregatesMultipleResults(t *testing.T) {
	older := time.Unix(100, 0).UTC()
	newer := time.Unix(200, 0).UTC()
	stats := buildStats([]Result{
		{Type: "INTJ", Duration: 10, Created: older},
		{Type: "ENFP", Duration: 20, Created: newer},
		{Type: "INTJ", Duration: 30, Created: older},
		{Type: "INFP", Duration: -5},
	})

	if stats.Total != 4 {
		t.Fatalf("expected total 4, got %d", stats.Total)
	}
	if stats.AverageDurationSeconds != 15 {
		t.Fatalf("expected average duration 15, got %d", stats.AverageDurationSeconds)
	}
	wantByType := map[string]int{"INTJ": 2, "ENFP": 1, "INFP": 1}
	if !reflect.DeepEqual(stats.ByType, wantByType) {
		t.Fatalf("unexpected byType: %+v", stats.ByType)
	}
	wantTopTypes := []typeCount{{Type: "INTJ", Count: 2}, {Type: "ENFP", Count: 1}, {Type: "INFP", Count: 1}}
	if !reflect.DeepEqual(stats.TopTypes, wantTopTypes) {
		t.Fatalf("unexpected topTypes: %+v", stats.TopTypes)
	}
	wantAxis := map[string]int{"E": 1, "I": 3, "S": 0, "N": 4, "T": 2, "F": 2, "J": 2, "P": 2}
	if !reflect.DeepEqual(stats.AxisDistribution, wantAxis) {
		t.Fatalf("unexpected axis distribution: %+v", stats.AxisDistribution)
	}
	if stats.LatestResultAt == nil || !stats.LatestResultAt.Equal(newer) {
		t.Fatalf("unexpected latest timestamp: %v", stats.LatestResultAt)
	}
}

func TestStaticAssetsServed(t *testing.T) {
	app := newTestApp(t)
	targets := []string{"/", "/compatibility-engine.js", "/content-uk.js", "/content-ru.js", "/content-en.js", "/content-author.js", "/content-profiles-uk.js", "/content-profiles-ru.js", "/content-profiles-en.js", "/style.css", "/types-data.js"}
	for _, target := range []string{"/js/api.js", "/js/state.js", "/js/dom.js", "/js/utils.js", "/js/i18n.js", "/js/ui.js", "/js/results.js", "/js/compatibility.js", "/js/quiz.js", "/js/types.js", "/js/admin.js", "/js/auth.js", "/js/profile.js", "/js/friends.js", "/js/share.js", "/js/events.js", "/js/app.js"} {
		targets = append(targets, target)
	}
	for _, code := range []string{"intj", "intp", "entj", "entp", "infj", "infp", "enfj", "enfp", "istj", "isfj", "estj", "esfj", "istp", "isfp", "estp", "esfp"} {
		targets = append(targets, "/assets/share-cards/"+code+".png")
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("expected %s to return 200, got %d", target, rec.Code)
			}
		})
	}
}
