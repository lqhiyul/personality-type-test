package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()

	store, err := NewStore(filepath.Join(t.TempDir(), "results.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	sub, err := fs.Sub(staticFiles, "web/static")
	if err != nil {
		t.Fatalf("fs.Sub() error = %v", err)
	}

	return &App{
		store:          store,
		adminPassword:  "secret",
		sessionName:    "test_admin_session_" + newID(),
		baseTemplateFS: sub,
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

func TestStaticAssetsServed(t *testing.T) {
	app := newTestApp(t)
	targets := []string{"/", "/app.js", "/compatibility-engine.js", "/content-uk.js", "/content-ru.js", "/content-en.js", "/content-author.js", "/content-profiles-uk.js", "/content-profiles-ru.js", "/content-profiles-en.js", "/style.css", "/types-data.js"}
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
