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
	for _, table := range []string{"users", "user_test_results", "friendships", "profile_comments", "conversations", "conversation_participants", "messages", "user_blocks", "user_reports", "sessions"} {
		if !sqliteObjectExists(t, db, "table", table) {
			t.Fatalf("expected table %q to exist", table)
		}
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
