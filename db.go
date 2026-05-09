package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func OpenAppDB(ctx context.Context, databasePath string) (*sql.DB, error) {
	path := strings.TrimSpace(databasePath)
	if path == "" {
		path = defaultDatabasePath
	}
	if err := ensureSQLiteParentDir(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := pingSQLite(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := RunMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func ensureSQLiteParentDir(databasePath string) error {
	if databasePath == ":memory:" || strings.HasPrefix(databasePath, "file:") {
		return nil
	}
	dir := filepath.Dir(databasePath)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}
	return nil
}

func sqliteDSN(databasePath string) string {
	separator := "?"
	if strings.Contains(databasePath, "?") {
		separator = "&"
	}
	return databasePath + separator + "_pragma=busy_timeout%3d5000&_pragma=foreign_keys%3d1"
}

func pingSQLite(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite database: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	return nil
}

func RunMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("nil database")
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			bio TEXT,
			avatar_key TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_test_results (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			mbti_type TEXT NOT NULL,
			scores_json TEXT,
			answers_json TEXT,
			duration_seconds INTEGER,
			is_primary INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_user_test_results_user_id ON user_test_results(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_test_results_user_primary ON user_test_results(user_id, is_primary)`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("run sqlite migration: %w", err)
		}
	}
	return nil
}
