package sessions

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	storagesqlite "github.com/lqhiyul/personality-type-test/internal/storage/sqlite"
)

func TestStoreCreatesHashesValidatesAndRevokesUserSession(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestDB(t)
	userID := createSessionTestUser(t, db)
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	store := NewStore(db, WithNow(func() time.Time { return now }))

	token, expiresAt, err := store.CreateUser(ctx, userID, time.Hour)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if token == "" || !expiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("unexpected created session token=%q expiresAt=%v", token, expiresAt)
	}

	var storedHash string
	if err := db.QueryRowContext(ctx, `SELECT token_hash FROM sessions WHERE kind = ?`, KindUser).Scan(&storedHash); err != nil {
		t.Fatalf("read token_hash: %v", err)
	}
	if storedHash == token {
		t.Fatal("session token must not be stored in plaintext")
	}
	if wantHash, _ := HashToken(token); storedHash != wantHash {
		t.Fatalf("stored hash mismatch: want %q got %q", wantHash, storedHash)
	}

	gotUserID, ok, err := store.UserID(ctx, token)
	if err != nil || !ok || gotUserID != userID {
		t.Fatalf("UserID() = %d, %t, %v; want %d, true, nil", gotUserID, ok, err, userID)
	}

	if err := store.Revoke(ctx, KindUser, token); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, ok, err := store.UserID(ctx, token); err != nil || ok {
		t.Fatalf("expected revoked session to be invalid, ok=%t err=%v", ok, err)
	}
}

func TestStoreAdminSessionExpiryAndCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := newTestDB(t)
	now := time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)
	store := NewStore(db, WithNow(func() time.Time { return now }))

	token, _, err := store.CreateAdmin(ctx, 10*time.Minute)
	if err != nil {
		t.Fatalf("CreateAdmin() error = %v", err)
	}
	valid, err := store.ValidateAdmin(ctx, token)
	if err != nil || !valid {
		t.Fatalf("ValidateAdmin() before expiry = %t, %v; want true, nil", valid, err)
	}

	now = now.Add(11 * time.Minute)
	valid, err = store.ValidateAdmin(ctx, token)
	if err != nil || valid {
		t.Fatalf("ValidateAdmin() after expiry = %t, %v; want false, nil", valid, err)
	}
	if err := store.CleanupExpired(ctx); err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cleanup to remove expired session, got %d", count)
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storagesqlite.Open(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createSessionTestUser(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, display_name, created_at, updated_at)
		VALUES ('session-user', 'session@example.com', 'hash', 'Session User', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatalf("insert session test user: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId(): %v", err)
	}
	return id
}
