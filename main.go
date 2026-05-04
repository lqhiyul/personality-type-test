package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:embed web/static
var staticFiles embed.FS

func main() {
	cfg := ConfigFromEnv()
	if cfg.UsesDefaultAdminPassword {
		log.Print("warning: ADMIN_PASSWORD uses the default value; change it before deployment")
	}

	store, err := NewStore(cfg.DataFile)
	if err != nil {
		log.Fatalf("store init: %v", err)
	}

	staticFS, err := fs.Sub(staticFiles, "web/static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	app := &App{
		store:          store,
		adminPassword:  cfg.AdminPassword,
		cookieSecure:   cfg.CookieSecure,
		sessionName:    "mbti_admin_session",
		baseTemplateFS: staticFS,
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           logging(app.routes()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(server.ListenAndServe())
}
