package app

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

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

	switch path.Ext(r.URL.Path) {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
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
	if strings.Contains(name, "/") && !strings.HasPrefix(name, "assets/") && !strings.HasPrefix(name, "js/") {
		return false
	}
	switch path.Ext(name) {
	case ".css", ".js", ".svg", ".png", ".webp", ".ico", ".txt":
		return true
	default:
		return false
	}
}
