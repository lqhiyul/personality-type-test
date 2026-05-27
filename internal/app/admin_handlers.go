package app

import (
	"encoding/csv"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lqhiyul/personality-type-test/internal/sessions"
)

type loginRequest struct {
	Password string `json:"password"`
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "РќРµРєРѕСЂРµРєС‚РЅРёР№ JSON Сѓ Р·Р°РїРёС‚С–")
		return
	}

	limitKey := a.loginRateLimitKey(r)
	if retryAfter, ok := a.loginLimiter.allow(limitKey); !ok {
		writeRateLimitError(w, retryAfter)
		return
	}

	if !subtleCompare(req.Password, a.adminPassword) {
		if retryAfter, limited := a.loginLimiter.recordFailure(limitKey); limited {
			writeRateLimitError(w, retryAfter)
			return
		}
		writeJSONError(w, http.StatusUnauthorized, "РќРµРІС–СЂРЅРёР№ РїР°СЂРѕР»СЊ")
		return
	}

	a.loginLimiter.recordSuccess(limitKey)
	token, _, err := a.sessionStore.CreateAdmin(r.Context(), sessionTTL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create admin session")
		return
	}
	setAdminCookie(w, a.sessionName, token, a.cookieSecure, int(sessionTTL.Seconds()))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if cookie, err := r.Cookie(a.sessionName); err == nil {
		if err := a.sessionStore.Revoke(r.Context(), sessions.KindAdmin, cookie.Value); err != nil {
			log.Printf("admin session revoke error: %v", err)
		}
	}
	setAdminCookie(w, a.sessionName, "", a.cookieSecure, -1)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleResults(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "РџРѕС‚СЂС–Р±РЅРѕ СѓРІС–Р№С‚Рё")
		return
	}

	switch r.Method {
	case http.MethodGet:
		results, err := a.store.All()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "РќРµ РІРґР°Р»РѕСЃСЏ РїСЂРѕС‡РёС‚Р°С‚Рё СЂРµР·СѓР»СЊС‚Р°С‚Рё")
			return
		}
		sortResults(results)
		writeJSON(w, http.StatusOK, map[string]any{"results": results})
	case http.MethodDelete:
		if err := a.store.Clear(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "РќРµ РІРґР°Р»РѕСЃСЏ РѕС‡РёСЃС‚РёС‚Рё СЂРµР·СѓР»СЊС‚Р°С‚Рё")
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
		writeJSONError(w, http.StatusUnauthorized, "РџРѕС‚СЂС–Р±РЅРѕ СѓРІС–Р№С‚Рё")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/results/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		writeJSONError(w, http.StatusBadRequest, "РќРµРєРѕСЂРµРєС‚РЅРёР№ ID СЂРµР·СѓР»СЊС‚Р°С‚Сѓ")
		return
	}
	if err := a.store.DeleteByID(id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "Р РµР·СѓР»СЊС‚Р°С‚ РЅРµ Р·РЅР°Р№РґРµРЅРѕ")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "РќРµ РІРґР°Р»РѕСЃСЏ РІРёРґР°Р»РёС‚Рё СЂРµР·СѓР»СЊС‚Р°С‚")
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
		writeJSONError(w, http.StatusUnauthorized, "РџРѕС‚СЂС–Р±РЅРѕ СѓРІС–Р№С‚Рё")
		return
	}

	results, err := a.store.All()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "РќРµ РІРґР°Р»РѕСЃСЏ РїСЂРѕС‡РёС‚Р°С‚Рё СЂРµР·СѓР»СЊС‚Р°С‚Рё")
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

func (a *App) authorized(r *http.Request) bool {
	cookie, err := r.Cookie(a.sessionName)
	if err != nil || cookie.Value == "" {
		return false
	}
	ok, err := a.sessionStore.ValidateAdmin(r.Context(), cookie.Value)
	if err != nil {
		log.Printf("admin session validation error: %v", err)
		return false
	}
	return ok
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
