package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMigrationsSorted(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected embedded migrations")
	}
	for i, migration := range migrations {
		if migration.Version != i+1 {
			t.Fatalf("expected migration %d to have version %d, got %+v", i, i+1, migration)
		}
		if migration.Name == "" || migration.SQL == "" {
			t.Fatalf("migration should include name and SQL: %+v", migration)
		}
	}
}

func TestRunMigrationsRecordsAppliedVersionsAndIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", DSN(filepath.Join(t.TempDir(), "app.db")))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := Ping(ctx, db); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations() second run error = %v", err)
	}

	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != len(migrations) {
		t.Fatalf("expected %d applied migrations, got %d", len(migrations), count)
	}
	for _, table := range []string{"users", "user_test_results", "friendships", "profile_comments", "conversations", "conversation_participants", "messages", "user_blocks", "user_reports", "sessions", "admin_audit_logs"} {
		if !sqliteObjectExists(t, db, "table", table) {
			t.Fatalf("expected table %q to exist", table)
		}
	}

	now := "2026-05-28T10:00:00Z"
	result, err := db.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "migration_user", "migration@example.com", "hash", "Migration User", now, now)
	if err != nil {
		t.Fatalf("insert user after migrations: %v", err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read inserted user id: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO user_test_results (user_id, mbti_type, scores_json, answers_json, duration_seconds, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, "INTJ", "{}", "[]", 180, 1, now); err != nil {
		t.Fatalf("insert user result after migrations: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO sessions (kind, token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, "user", "token-hash", userID, now, now); err != nil {
		t.Fatalf("insert user session after migrations: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO admin_audit_logs (action, target_type, target_id, ip, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "admin_login_success", "result", "one", "127.0.0.1", "test", now); err != nil {
		t.Fatalf("insert admin audit log after migrations: %v", err)
	}
}

func TestRepositoryMigrationsMatchEmbeddedCopies(t *testing.T) {
	repoDir := filepath.Join("..", "..", "..", "migrations")
	repoEntries, err := os.ReadDir(repoDir)
	if err != nil {
		t.Fatalf("read repository migrations: %v", err)
	}
	embedded, err := LoadMigrations()
	if err != nil {
		t.Fatalf("LoadMigrations() error = %v", err)
	}

	repoNames := make([]string, 0, len(repoEntries))
	for _, entry := range repoEntries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			repoNames = append(repoNames, entry.Name())
		}
	}
	embeddedNames := make([]string, 0, len(embedded))
	for _, migration := range embedded {
		embeddedNames = append(embeddedNames, migration.Name)
		repoBody, err := os.ReadFile(filepath.Join(repoDir, migration.Name))
		if err != nil {
			t.Fatalf("read repository migration %s: %v", migration.Name, err)
		}
		if string(repoBody) != migration.SQL {
			t.Fatalf("repository migration %s does not match embedded copy", migration.Name)
		}
	}
	if !reflect.DeepEqual(repoNames, embeddedNames) {
		t.Fatalf("repository migrations mismatch:\nwant %+v\n got %+v", embeddedNames, repoNames)
	}
}

func sqliteObjectExists(t *testing.T, db *sql.DB, objectType, name string) bool {
	t.Helper()

	var found string
	err := db.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type = ? AND name = ?
	`, objectType, name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("query sqlite object %s %s: %v", objectType, name, err)
	}
	return found == name
}
