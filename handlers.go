package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type App struct {
	store          *Store
	adminPassword  string
	cookieSecure   bool
	sessionName    string
	baseTemplateFS fs.FS
}

type submitRequest struct {
	Name     string   `json:"name"`
	Answers  []string `json:"answers"`
	Duration int      `json:"duration"`
}

type loginRequest struct {
	Password string `json:"password"`
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(started).Round(time.Millisecond))
	})
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.handleHealthz)
	mux.HandleFunc("/api/submit", a.handleSubmit)
	mux.HandleFunc("/api/login", a.handleLogin)
	mux.HandleFunc("/api/logout", a.handleLogout)
	mux.HandleFunc("/api/results/export", a.handleExportResults)
	mux.HandleFunc("/api/results/", a.handleResultByID)
	mux.HandleFunc("/api/results", a.handleResults)
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/", a.handleStatic)
	return mux
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.routes().ServeHTTP(w, r)
}

func (a *App) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (a *App) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}

	if r.URL.Path == "/" {
		a.serveIndex(w, r)
		return
	}
	if !isStaticAssetRequest(r.URL.Path) {
		http.NotFound(w, r)
		return
	}

	http.FileServer(http.FS(a.baseTemplateFS)).ServeHTTP(w, r)
}

func (a *App) serveIndex(w http.ResponseWriter, r *http.Request) {
	body, err := fs.ReadFile(a.baseTemplateFS, "index.html")
	if err != nil {
		http.Error(w, "index page is not available", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(body))
}

func isStaticAssetRequest(requestPath string) bool {
	name := strings.TrimPrefix(path.Clean(requestPath), "/")
	if name == "." || strings.HasPrefix(name, "../") {
		return false
	}
	if strings.Contains(name, "/") && !strings.HasPrefix(name, "assets/") {
		return false
	}
	switch path.Ext(name) {
	case ".css", ".js", ".svg", ".png", ".webp", ".ico", ".txt":
		return true
	default:
		return false
	}
}

func (a *App) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req submitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Некоректний JSON у запиті")
		return
	}

	name := strings.TrimSpace(req.Name)
	if !validName(name) {
		writeJSONError(w, http.StatusBadRequest, "Ім'я має містити від 1 до 64 символів")
		return
	}

	normalizedAnswers, err := normalizeAnswers(req.Answers)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	profile := buildProfile(normalizedAnswers)

	result := Result{
		ID:       newID(),
		Name:     name,
		Type:     profile.Type,
		Answers:  strings.Join(normalizedAnswers, ""),
		Duration: clampDuration(req.Duration),
		Created:  time.Now().UTC(),
	}
	if err := a.store.Add(result); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Не вдалося зберегти результат")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"type":    profile.Type,
		"profile": profile,
		"result":  result,
	})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Некоректний JSON у запиті")
		return
	}
	if !subtleCompare(req.Password, a.adminPassword) {
		writeJSONError(w, http.StatusUnauthorized, "Невірний пароль")
		return
	}

	token := newSessionToken()
	storeSession(token)
	setAdminCookie(w, a.sessionName, token, a.cookieSecure, int(sessionTTL.Seconds()))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if cookie, err := r.Cookie(a.sessionName); err == nil {
		deleteSession(cookie.Value)
	}
	setAdminCookie(w, a.sessionName, "", a.cookieSecure, -1)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleResults(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "Потрібно увійти")
		return
	}

	switch r.Method {
	case http.MethodGet:
		results, err := a.store.All()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Не вдалося прочитати результати")
			return
		}
		sortResults(results)
		writeJSON(w, http.StatusOK, map[string]any{"results": results})
	case http.MethodDelete:
		if err := a.store.Clear(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Не вдалося очистити результати")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodDelete)
	}
}

func (a *App) handleResultByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "Потрібно увійти")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/results/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		writeJSONError(w, http.StatusBadRequest, "Некоректний ID результату")
		return
	}
	if err := a.store.DeleteByID(id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "Результат не знайдено")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Не вдалося видалити результат")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleExportResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "Потрібно увійти")
		return
	}

	results, err := a.store.All()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Не вдалося прочитати результати")
		return
	}
	sortResults(results)

	if strings.EqualFold(r.URL.Query().Get("format"), "json") {
		w.Header().Set("Content-Disposition", `attachment; filename="mbti-results.json"`)
		writeJSON(w, http.StatusOK, map[string]any{
			"results":     results,
			"generatedAt": time.Now().UTC(),
		})
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="mbti-results.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"id", "name", "type", "answers", "duration", "created"})
	for _, result := range results {
		_ = writer.Write([]string{
			result.ID,
			result.Name,
			result.Type,
			result.Answers,
			strconv.Itoa(result.Duration),
			result.Created.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("csv export error: %v", err)
	}
}

func (a *App) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "Потрібно увійти")
		return
	}

	results, err := a.store.All()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Не вдалося прочитати статистику")
		return
	}
	byType := map[string]int{}
	for _, result := range results {
		byType[result.Type]++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total":  len(results),
		"byType": byType,
	})
}

func (a *App) authorized(r *http.Request) bool {
	cookie, err := r.Cookie(a.sessionName)
	if err != nil || cookie.Value == "" {
		return false
	}
	return checkSession(cookie.Value)
}

func setAdminCookie(w http.ResponseWriter, name, value string, secure bool, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSONError(w, http.StatusMethodNotAllowed, "Метод не підтримується")
}

func sortResults(results []Result) {
	sort.Slice(results, func(i, j int) bool {
		return results[i].Created.After(results[j].Created)
	})
}

func validName(name string) bool {
	count := utf8.RuneCountInString(name)
	if count < 1 || count > 64 {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func clampDuration(seconds int) int {
	if seconds < 0 {
		return 0
	}
	if seconds > 24*60*60 {
		return 24 * 60 * 60
	}
	return seconds
}
