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

func TestRunMigrationsEnforcesImportantConstraints(t *testing.T) {
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

	aliceID := insertMigrationUser(t, db, "constraint_alice", "constraint-alice@example.com")
	bobID := insertMigrationUser(t, db, "constraint_bob", "constraint-bob@example.com")
	now := "2026-05-28T10:00:00Z"

	assertExecFails(t, db, "duplicate username", `
		INSERT INTO users (username, email, password_hash, display_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "constraint_alice", "other@example.com", "hash", "Other", now, now)
	assertExecFails(t, db, "duplicate email", `
		INSERT INTO users (username, email, password_hash, display_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "other_user", "constraint-alice@example.com", "hash", "Other", now, now)

	assertExecFails(t, db, "self friendship", `
		INSERT INTO friendships (requester_id, addressee_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, aliceID, aliceID, "pending", now, now)
	if _, err := db.Exec(`
		INSERT INTO friendships (requester_id, addressee_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, aliceID, bobID, "pending", now, now); err != nil {
		t.Fatalf("insert friendship: %v", err)
	}
	assertExecFails(t, db, "duplicate reverse friendship", `
		INSERT INTO friendships (requester_id, addressee_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, bobID, aliceID, "pending", now, now)
	assertExecFails(t, db, "invalid friendship status", `
		INSERT INTO friendships (requester_id, addressee_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, bobID, aliceID, "unknown", now, now)

	assertExecFails(t, db, "self block", `
		INSERT INTO user_blocks (blocker_user_id, blocked_user_id, created_at)
		VALUES (?, ?, ?)
	`, aliceID, aliceID, now)
	if _, err := db.Exec(`
		INSERT INTO user_blocks (blocker_user_id, blocked_user_id, created_at)
		VALUES (?, ?, ?)
	`, aliceID, bobID, now); err != nil {
		t.Fatalf("insert user block: %v", err)
	}
	assertExecFails(t, db, "duplicate user block", `
		INSERT INTO user_blocks (blocker_user_id, blocked_user_id, created_at)
		VALUES (?, ?, ?)
	`, aliceID, bobID, now)

	assertExecFails(t, db, "invalid report target type", `
		INSERT INTO user_reports (reporter_user_id, target_type, reason, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, aliceID, "account", "spam", "open", now)
	assertExecFails(t, db, "invalid report status", `
		INSERT INTO user_reports (reporter_user_id, target_type, reason, status, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, aliceID, "profile", "spam", "pending", now)

	if _, err := db.Exec(`
		INSERT INTO sessions (kind, token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, "user", "token-hash", aliceID, now, now); err != nil {
		t.Fatalf("insert user session: %v", err)
	}
	assertExecFails(t, db, "duplicate session token hash", `
		INSERT INTO sessions (kind, token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, "user", "token-hash", bobID, now, now)
	assertExecFails(t, db, "user session without user", `
		INSERT INTO sessions (kind, token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, "user", "missing-user-id", nil, now, now)
	assertExecFails(t, db, "admin session with user", `
		INSERT INTO sessions (kind, token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, "admin", "admin-with-user", aliceID, now, now)
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

func insertMigrationUser(t *testing.T, db *sql.DB, username, email string) int64 {
	t.Helper()

	now := "2026-05-28T10:00:00Z"
	res, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, display_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, username, email, "hash", username, now, now)
	if err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("read inserted user id for %s: %v", username, err)
	}
	return id
}

func assertExecFails(t *testing.T, db *sql.DB, name, query string, args ...any) {
	t.Helper()

	if _, err := db.Exec(query, args...); err == nil {
		t.Fatalf("%s: expected database constraint failure", name)
	}
}
