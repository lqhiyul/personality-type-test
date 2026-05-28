package app

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lqhiyul/personality-type-test/internal/config"
	requestlog "github.com/lqhiyul/personality-type-test/internal/platform/logging"
	"github.com/lqhiyul/personality-type-test/internal/sessions"
)

type Options struct {
	Config   config.Config
	StaticFS fs.FS
	Logger   *log.Logger
}

func New(ctx context.Context, opts Options) (*App, func() error, error) {
	cfg := opts.Config
	staticFS := opts.StaticFS
	if staticFS == nil {
		staticFS = os.DirFS(cfg.StaticDir)
	}

	store, err := NewStore(cfg.DataFile)
	if err != nil {
		return nil, nil, err
	}

	db, err := OpenAppDB(ctx, cfg.DatabasePath)
	if err != nil {
		return nil, nil, err
	}
	trustedProxies, err := parseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	app := &App{
		store:            store,
		userStore:        NewUserStore(db),
		adminPassword:    cfg.AdminPassword,
		cookieSecure:     cfg.CookieSecure,
		sessionName:      "mbti_admin_session",
		baseTemplateFS:   staticFS,
		loginLimiter:     newLoginRateLimiter(defaultLoginFailureLimit, defaultLoginCooldown),
		userLoginLimiter: newLoginRateLimiter(defaultLoginFailureLimit, defaultLoginCooldown),
		sessionStore:     sessions.NewStore(db),
		auditStore:       NewAdminAuditStore(db),
		trustedProxies:   trustedProxies,
	}

	cleanup := func() error {
		return db.Close()
	}
	return app, cleanup, nil
}

func Run(ctx context.Context, cfg config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if cfg.UsesDefaultAdminPassword {
		log.Print("warning: ADMIN_PASSWORD uses the default value; change it before deployment")
	}

	app, cleanup, err := New(ctx, Options{Config: cfg})
	if err != nil {
		return err
	}
	defer func() {
		if err := cleanup(); err != nil {
			log.Printf("database close error: %v", err)
		}
	}()

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           requestlog.Middleware(app.routes(), log.Default()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		errCh <- server.ListenAndServe()
	}()

	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-shutdownCtx.Done():
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(timeoutCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
