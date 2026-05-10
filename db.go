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
			profile_visibility TEXT NOT NULL DEFAULT 'public',
			show_primary_result INTEGER NOT NULL DEFAULT 1,
			show_completed_count INTEGER NOT NULL DEFAULT 1,
			show_friends INTEGER NOT NULL DEFAULT 1,
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
		`CREATE TABLE IF NOT EXISTS friendships (
			id INTEGER PRIMARY KEY,
			requester_id INTEGER NOT NULL,
			addressee_id INTEGER NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'rejected')),
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(requester_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(addressee_id) REFERENCES users(id) ON DELETE CASCADE,
			CHECK (requester_id <> addressee_id)
		)`,
		`CREATE TABLE IF NOT EXISTS profile_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile_user_id INTEGER NOT NULL,
			author_user_id INTEGER NOT NULL,
			body TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(profile_user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(author_user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_user_test_results_user_id ON user_test_results(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_test_results_user_primary ON user_test_results(user_id, is_primary)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_requester_id ON friendships(requester_id)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_addressee_id ON friendships(addressee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_friendships_status ON friendships(status)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_comments_profile_user_id_created_at ON profile_comments(profile_user_id, created_at, id)`,
		`CREATE INDEX IF NOT EXISTS idx_profile_comments_author_user_id ON profile_comments(author_user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_friendships_pair_unique ON friendships (
			CASE WHEN requester_id < addressee_id THEN requester_id ELSE addressee_id END,
			CASE WHEN requester_id < addressee_id THEN addressee_id ELSE requester_id END
		)`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("run sqlite migration: %w", err)
		}
	}
	if err := ensureUserPrivacyColumns(ctx, db); err != nil {
		return err
	}
	return nil
}

func ensureUserPrivacyColumns(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(users)`)
	if err != nil {
		return fmt.Errorf("inspect users columns: %w", err)
	}
	defer rows.Close()

	existing := map[string]struct{}{}
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan users column: %w", err)
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect users columns rows: %w", err)
	}

	columns := []struct {
		name       string
		definition string
	}{
		{name: "profile_visibility", definition: "TEXT NOT NULL DEFAULT 'public'"},
		{name: "show_primary_result", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "show_completed_count", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "show_friends", definition: "INTEGER NOT NULL DEFAULT 1"},
	}
	for _, column := range columns {
		if _, ok := existing[column.name]; ok {
			continue
		}
		statement := fmt.Sprintf("ALTER TABLE users ADD COLUMN %s %s", column.name, column.definition)
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("add users.%s column: %w", column.name, err)
		}
	}
	return nil
}
