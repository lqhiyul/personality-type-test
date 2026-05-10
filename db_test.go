package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestOpenAppDBCreatesParentDirAndRunsMigrations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "nested", "app.db")
	db, err := OpenAppDB(ctx, databasePath)
	if err != nil {
		t.Fatalf("OpenAppDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := os.Stat(databasePath); err != nil {
		t.Fatalf("expected database file to exist at %s: %v", databasePath, err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations() should be idempotent: %v", err)
	}

	for _, table := range []string{"users", "user_test_results", "friendships", "profile_comments"} {
		t.Run(table, func(t *testing.T) {
			if !sqliteObjectExists(t, db, "table", table) {
				t.Fatalf("expected table %q to exist", table)
			}
		})
	}
	for _, column := range []string{"profile_visibility", "show_primary_result", "show_completed_count", "show_friends"} {
		t.Run("users."+column, func(t *testing.T) {
			if !sqliteColumnExists(t, db, "users", column) {
				t.Fatalf("expected users.%s column to exist", column)
			}
		})
	}

	for _, index := range []string{
		"idx_users_username",
		"idx_users_email",
		"idx_user_test_results_user_id",
		"idx_user_test_results_user_primary",
		"idx_friendships_requester_id",
		"idx_friendships_addressee_id",
		"idx_friendships_status",
		"idx_friendships_pair_unique",
		"idx_profile_comments_profile_user_id_created_at",
		"idx_profile_comments_author_user_id",
	} {
		t.Run(index, func(t *testing.T) {
			if !sqliteObjectExists(t, db, "index", index) {
				t.Fatalf("expected index %q to exist", index)
			}
		})
	}
}

func TestUserStoreCreateAndReadUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, store := newTestUserStore(t)
	fixedNow := time.Date(2026, 5, 9, 12, 30, 0, 0, time.UTC)
	store.now = func() time.Time { return fixedNow }

	user, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash-for-future-auth",
		DisplayName:  "Alice",
		Bio:          "Testing the foundation",
		AvatarKey:    "avatar-1",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	want := User{
		ID:                 user.ID,
		Username:           "alice",
		Email:              "alice@example.com",
		DisplayName:        "Alice",
		Bio:                "Testing the foundation",
		AvatarKey:          "avatar-1",
		ProfileVisibility:  profileVisibilityPublic,
		ShowPrimaryResult:  true,
		ShowCompletedCount: true,
		ShowFriends:        true,
		CreatedAt:          fixedNow,
		UpdatedAt:          fixedNow,
	}
	if !reflect.DeepEqual(user, want) {
		t.Fatalf("created user mismatch:\nwant: %+v\n got: %+v", want, user)
	}

	for name, read := range map[string]func(context.Context) (User, error){
		"id":       func(ctx context.Context) (User, error) { return store.GetUserByID(ctx, user.ID) },
		"username": func(ctx context.Context) (User, error) { return store.GetUserByUsername(ctx, "alice") },
		"email":    func(ctx context.Context) (User, error) { return store.GetUserByEmail(ctx, "alice@example.com") },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := read(ctx)
			if err != nil {
				t.Fatalf("read user by %s: %v", name, err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("read user mismatch:\nwant: %+v\n got: %+v", want, got)
			}
		})
	}

	var storedHash string
	if err := db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = ?`, user.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read stored password hash: %v", err)
	}
	if storedHash != "hash-for-future-auth" {
		t.Fatalf("expected password hash to be stored internally, got %q", storedHash)
	}
}

func TestRunMigrationsAddsPrivacyColumnsToExistingUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", sqliteDSN(filepath.Join(t.TempDir(), "legacy.db")))
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := pingSQLite(ctx, db); err != nil {
		t.Fatalf("pingSQLite() error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		display_name TEXT,
		bio TEXT,
		avatar_key TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy users table: %v", err)
	}
	now := formatDBTime(time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC))
	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, bio, avatar_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "legacy", "legacy@example.com", "hash", "Legacy", "", "", now, now); err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	for _, column := range []string{"profile_visibility", "show_primary_result", "show_completed_count", "show_friends"} {
		if !sqliteColumnExists(t, db, "users", column) {
			t.Fatalf("expected users.%s column after migration", column)
		}
	}

	store := NewUserStore(db)
	user, err := store.GetUserByUsername(ctx, "legacy")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user.ProfileVisibility != profileVisibilityPublic || !user.ShowPrimaryResult || !user.ShowCompletedCount || !user.ShowFriends {
		t.Fatalf("expected legacy user to keep public defaults, got %+v", user)
	}
}

func TestUserStoreUpdateProfilePrivacy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	user, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "privacy",
		Email:        "privacy@example.com",
		PasswordHash: "hash",
		DisplayName:  "Privacy",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	showPrimary := false
	showCompleted := false
	showFriends := false
	updated, err := store.UpdateUserProfile(ctx, user.ID, UpdateUserProfileParams{
		DisplayName:        "Privacy Updated",
		Bio:                "Quiet profile",
		AvatarKey:          "gradient-green",
		ProfileVisibility:  profileVisibilityPrivate,
		ShowPrimaryResult:  &showPrimary,
		ShowCompletedCount: &showCompleted,
		ShowFriends:        &showFriends,
	})
	if err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}
	if updated.ProfileVisibility != profileVisibilityPrivate || updated.ShowPrimaryResult || updated.ShowCompletedCount || updated.ShowFriends {
		t.Fatalf("unexpected privacy update: %+v", updated)
	}
	if updated.Username != user.Username || updated.Email != user.Email {
		t.Fatalf("privacy update changed identity fields: %+v", updated)
	}
}

func TestUserStoreRejectsDuplicateUsernameAndEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)

	if _, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
	}); err != nil {
		t.Fatalf("CreateUser(seed) error = %v", err)
	}

	if _, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "alice",
		Email:        "second@example.com",
		PasswordHash: "hash",
	}); err == nil {
		t.Fatal("expected duplicate username to fail")
	}

	if _, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "second",
		Email:        "alice@example.com",
		PasswordHash: "hash",
	}); err == nil {
		t.Fatal("expected duplicate email to fail")
	}
}

func TestUserStoreCreateAndListUserTestResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	firstTime := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	secondTime := firstTime.Add(time.Minute)

	store.now = func() time.Time { return firstTime }
	user, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	first, err := store.CreateUserTestResult(ctx, CreateUserTestResultParams{
		UserID:          user.ID,
		MBTIType:        "INTJ",
		ScoresJSON:      `{"I":7,"N":8,"T":6,"J":9}`,
		AnswersJSON:     `["I","N","T","J"]`,
		DurationSeconds: 120,
		IsPrimary:       true,
	})
	if err != nil {
		t.Fatalf("CreateUserTestResult(first) error = %v", err)
	}
	if first.UserID != user.ID || first.MBTIType != "INTJ" || !first.IsPrimary || !first.CreatedAt.Equal(firstTime) {
		t.Fatalf("unexpected first result: %+v", first)
	}

	store.now = func() time.Time { return secondTime }
	second, err := store.CreateUserTestResult(ctx, CreateUserTestResultParams{
		UserID:          user.ID,
		MBTIType:        "ENFP",
		ScoresJSON:      `{"E":5,"N":8,"F":7,"P":6}`,
		AnswersJSON:     `["E","N","F","P"]`,
		DurationSeconds: 90,
	})
	if err != nil {
		t.Fatalf("CreateUserTestResult(second) error = %v", err)
	}

	results, err := store.ListUserTestResults(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListUserTestResults() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != second.ID || results[1].ID != first.ID {
		t.Fatalf("expected newest result first, got %+v", results)
	}
}

func TestUserStoreSetPrimaryAndDeleteUserTestResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	user, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	otherUser, err := store.CreateUser(ctx, CreateUserParams{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("CreateUser(other) error = %v", err)
	}

	first, err := store.CreateUserTestResult(ctx, CreateUserTestResultParams{UserID: user.ID, MBTIType: "INTJ"})
	if err != nil {
		t.Fatalf("CreateUserTestResult(first) error = %v", err)
	}
	second, err := store.CreateUserTestResult(ctx, CreateUserTestResultParams{UserID: user.ID, MBTIType: "ENFP"})
	if err != nil {
		t.Fatalf("CreateUserTestResult(second) error = %v", err)
	}
	other, err := store.CreateUserTestResult(ctx, CreateUserTestResultParams{UserID: otherUser.ID, MBTIType: "ISFJ"})
	if err != nil {
		t.Fatalf("CreateUserTestResult(other) error = %v", err)
	}

	if _, err := store.SetPrimaryUserTestResult(ctx, user.ID, other.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected setting another user's result as primary to fail with sql.ErrNoRows, got %v", err)
	}
	if _, err := store.SetPrimaryUserTestResult(ctx, user.ID, first.ID); err != nil {
		t.Fatalf("SetPrimaryUserTestResult(first) error = %v", err)
	}
	if _, err := store.SetPrimaryUserTestResult(ctx, user.ID, second.ID); err != nil {
		t.Fatalf("SetPrimaryUserTestResult(second) error = %v", err)
	}

	firstAfter, err := store.GetUserTestResult(ctx, user.ID, first.ID)
	if err != nil {
		t.Fatalf("GetUserTestResult(first) error = %v", err)
	}
	secondAfter, err := store.GetUserTestResult(ctx, user.ID, second.ID)
	if err != nil {
		t.Fatalf("GetUserTestResult(second) error = %v", err)
	}
	if firstAfter.IsPrimary || !secondAfter.IsPrimary {
		t.Fatalf("expected only second result to be primary, first=%+v second=%+v", firstAfter, secondAfter)
	}

	if err := store.DeleteUserTestResult(ctx, user.ID, other.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleting another user's result to fail with sql.ErrNoRows, got %v", err)
	}
	if err := store.DeleteUserTestResult(ctx, user.ID, first.ID); err != nil {
		t.Fatalf("DeleteUserTestResult(first) error = %v", err)
	}
	if _, err := store.GetUserTestResult(ctx, user.ID, first.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected deleted result to return sql.ErrNoRows, got %v", err)
	}
}

func TestUserStoreRequiresExistingUserForResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)

	_, err := store.CreateUserTestResult(ctx, CreateUserTestResultParams{
		UserID:   999,
		MBTIType: "INTJ",
	})
	if err == nil {
		t.Fatal("expected missing user foreign key to fail")
	}
}

func TestUserStoreMissingUserReturnsNoRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)

	if _, err := store.GetUserByID(ctx, 404); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for missing user, got %v", err)
	}
}

func newTestUserStore(t *testing.T) (*sql.DB, *UserStore) {
	t.Helper()

	db, err := OpenAppDB(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("OpenAppDB() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, NewUserStore(db)
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

func sqliteColumnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()

	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("query table info for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table info for %s: %v", table, err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table info rows for %s: %v", table, err)
	}
	return false
}
