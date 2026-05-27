package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	KindAdmin = "admin"
	KindUser  = "user"
)

var ErrInvalidKind = errors.New("invalid session kind")

type Store struct {
	db  *sql.DB
	now func() time.Time
}

type Option func(*Store)

func NewStore(db *sql.DB, opts ...Option) *Store {
	store := &Store{
		db:  db,
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(store)
	}
	return store
}

func WithNow(now func() time.Time) Option {
	return func(store *Store) {
		if now != nil {
			store.now = now
		}
	}
}

func (s *Store) CreateAdmin(ctx context.Context, ttl time.Duration) (string, time.Time, error) {
	return s.create(ctx, KindAdmin, nil, ttl)
}

func (s *Store) CreateUser(ctx context.Context, userID int64, ttl time.Duration) (string, time.Time, error) {
	if userID <= 0 {
		return "", time.Time{}, errors.New("user session requires a valid user id")
	}
	return s.create(ctx, KindUser, &userID, ttl)
}

func (s *Store) ValidateAdmin(ctx context.Context, token string) (bool, error) {
	session, ok, err := s.lookup(ctx, KindAdmin, token)
	if err != nil || !ok {
		return false, err
	}
	return !session.ExpiresAt.Before(s.now()) && session.RevokedAt == nil, nil
}

func (s *Store) UserID(ctx context.Context, token string) (int64, bool, error) {
	session, ok, err := s.lookup(ctx, KindUser, token)
	if err != nil || !ok {
		return 0, false, err
	}
	if session.ExpiresAt.Before(s.now()) || session.RevokedAt != nil || session.UserID == nil {
		return 0, false, nil
	}
	return *session.UserID, true, nil
}

func (s *Store) Revoke(ctx context.Context, kind, token string) error {
	if err := validateKind(kind); err != nil {
		return err
	}
	hash, ok := HashToken(token)
	if !ok {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET revoked_at = COALESCE(revoked_at, ?)
		WHERE kind = ? AND token_hash = ?
	`, formatTime(s.now()), kind, hash); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Store) CleanupExpired(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE expires_at < ? OR revoked_at IS NOT NULL
	`, formatTime(s.now())); err != nil {
		return fmt.Errorf("cleanup sessions: %w", err)
	}
	return nil
}

type record struct {
	UserID    *int64
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (s *Store) create(ctx context.Context, kind string, userID *int64, ttl time.Duration) (string, time.Time, error) {
	if err := validateKind(kind); err != nil {
		return "", time.Time{}, err
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("session ttl must be positive")
	}

	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	hash, _ := HashToken(token)
	now := s.now()
	expiresAt := now.Add(ttl)

	var nullableUserID any
	if userID != nil {
		nullableUserID = *userID
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (kind, token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, kind, hash, nullableUserID, formatTime(now), formatTime(expiresAt)); err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}
	return token, expiresAt, nil
}

func (s *Store) lookup(ctx context.Context, kind, token string) (record, bool, error) {
	if err := validateKind(kind); err != nil {
		return record{}, false, err
	}
	hash, ok := HashToken(token)
	if !ok {
		return record{}, false, nil
	}

	var userID sql.NullInt64
	var expiresAtRaw string
	var revokedAtRaw sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at, revoked_at
		FROM sessions
		WHERE kind = ? AND token_hash = ?
		LIMIT 1
	`, kind, hash).Scan(&userID, &expiresAtRaw, &revokedAtRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return record{}, false, nil
	}
	if err != nil {
		return record{}, false, fmt.Errorf("lookup session: %w", err)
	}

	expiresAt, err := parseTime(expiresAtRaw)
	if err != nil {
		return record{}, false, err
	}
	var revokedAt *time.Time
	if revokedAtRaw.Valid {
		parsed, err := parseTime(revokedAtRaw.String)
		if err != nil {
			return record{}, false, err
		}
		revokedAt = &parsed
	}
	var id *int64
	if userID.Valid {
		value := userID.Int64
		id = &value
	}
	return record{UserID: id, ExpiresAt: expiresAt, RevokedAt: revokedAt}, true, nil
}

func validateKind(kind string) error {
	switch kind {
	case KindAdmin, KindUser:
		return nil
	default:
		return ErrInvalidKind
	}
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func HashToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]), true
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse session timestamp: %w", err)
	}
	return parsed.UTC(), nil
}
