package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type User struct {
	ID          int64
	Username    string
	Email       string
	DisplayName string
	Bio         string
	AvatarKey   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type userAuthRecord struct {
	User
	PasswordHash string
}

type CreateUserParams struct {
	Username     string
	Email        string
	PasswordHash string
	DisplayName  string
	Bio          string
	AvatarKey    string
}

type UserTestResult struct {
	ID              int64
	UserID          int64
	MBTIType        string
	ScoresJSON      string
	AnswersJSON     string
	DurationSeconds int
	IsPrimary       bool
	CreatedAt       time.Time
}

type CreateUserTestResultParams struct {
	UserID          int64
	MBTIType        string
	ScoresJSON      string
	AnswersJSON     string
	DurationSeconds int
	IsPrimary       bool
}

type UserStore struct {
	db  *sql.DB
	now func() time.Time
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{
		db:  db,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *UserStore) CreateUser(ctx context.Context, params CreateUserParams) (User, error) {
	now := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, display_name, bio, avatar_key, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, params.Username, params.Email, params.PasswordHash, params.DisplayName, params.Bio, params.AvatarKey, now, now)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("read created user id: %w", err)
	}
	return s.GetUserByID(ctx, id)
}

func (s *UserStore) GetUserByID(ctx context.Context, id int64) (User, error) {
	return s.queryUser(ctx, "id = ?", id)
}

func (s *UserStore) GetUserByUsername(ctx context.Context, username string) (User, error) {
	return s.queryUser(ctx, "username = ?", username)
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (User, error) {
	return s.queryUser(ctx, "email = ?", email)
}

func (s *UserStore) getUserAuthByUsername(ctx context.Context, username string) (userAuthRecord, error) {
	return s.queryUserAuth(ctx, "username = ?", username)
}

func (s *UserStore) getUserAuthByEmail(ctx context.Context, email string) (userAuthRecord, error) {
	return s.queryUserAuth(ctx, "email = ?", email)
}

func (s *UserStore) CreateUserTestResult(ctx context.Context, params CreateUserTestResultParams) (UserTestResult, error) {
	createdAt := formatDBTime(s.now())
	if !params.IsPrimary {
		res, err := s.db.ExecContext(ctx, `
			INSERT INTO user_test_results (
				user_id,
				mbti_type,
				scores_json,
				answers_json,
				duration_seconds,
				is_primary,
				created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, params.UserID, params.MBTIType, params.ScoresJSON, params.AnswersJSON, params.DurationSeconds, boolToInt(params.IsPrimary), createdAt)
		if err != nil {
			return UserTestResult{}, fmt.Errorf("create user test result: %w", err)
		}

		id, err := res.LastInsertId()
		if err != nil {
			return UserTestResult{}, fmt.Errorf("read created user test result id: %w", err)
		}
		return s.queryUserTestResult(ctx, "id = ?", id)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return UserTestResult{}, fmt.Errorf("begin create primary user test result: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE user_test_results SET is_primary = 0 WHERE user_id = ?`, params.UserID); err != nil {
		return UserTestResult{}, fmt.Errorf("unset existing primary user test results: %w", err)
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO user_test_results (
			user_id,
			mbti_type,
			scores_json,
			answers_json,
			duration_seconds,
			is_primary,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, params.UserID, params.MBTIType, params.ScoresJSON, params.AnswersJSON, params.DurationSeconds, boolToInt(params.IsPrimary), createdAt)
	if err != nil {
		return UserTestResult{}, fmt.Errorf("create user test result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return UserTestResult{}, fmt.Errorf("read created user test result id: %w", err)
	}
	result, err := queryUserTestResult(ctx, tx, "id = ?", id)
	if err != nil {
		return UserTestResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return UserTestResult{}, fmt.Errorf("commit create primary user test result: %w", err)
	}
	return result, nil
}

func (s *UserStore) ListUserTestResults(ctx context.Context, userID int64) ([]UserTestResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id,
			user_id,
			mbti_type,
			COALESCE(scores_json, ''),
			COALESCE(answers_json, ''),
			COALESCE(duration_seconds, 0),
			is_primary,
			created_at
		FROM user_test_results
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user test results: %w", err)
	}
	defer rows.Close()

	var results []UserTestResult
	for rows.Next() {
		result, err := scanUserTestResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user test results rows: %w", err)
	}
	return results, nil
}

func (s *UserStore) GetUserTestResult(ctx context.Context, userID, resultID int64) (UserTestResult, error) {
	return s.queryUserTestResult(ctx, "user_id = ? AND id = ?", userID, resultID)
}

func (s *UserStore) SetPrimaryUserTestResult(ctx context.Context, userID, resultID int64) (UserTestResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return UserTestResult{}, fmt.Errorf("begin set primary user test result: %w", err)
	}
	defer tx.Rollback()

	if _, err := queryUserTestResult(ctx, tx, "user_id = ? AND id = ?", userID, resultID); err != nil {
		return UserTestResult{}, fmt.Errorf("query user test result before primary update: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_test_results SET is_primary = 0 WHERE user_id = ?`, userID); err != nil {
		return UserTestResult{}, fmt.Errorf("unset primary user test results: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_test_results SET is_primary = 1 WHERE user_id = ? AND id = ?`, userID, resultID); err != nil {
		return UserTestResult{}, fmt.Errorf("set primary user test result: %w", err)
	}

	result, err := queryUserTestResult(ctx, tx, "user_id = ? AND id = ?", userID, resultID)
	if err != nil {
		return UserTestResult{}, fmt.Errorf("query primary user test result: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return UserTestResult{}, fmt.Errorf("commit primary user test result: %w", err)
	}
	return result, nil
}

func (s *UserStore) DeleteUserTestResult(ctx context.Context, userID, resultID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM user_test_results WHERE user_id = ? AND id = ?`, userID, resultID)
	if err != nil {
		return fmt.Errorf("delete user test result: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted user test result rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *UserStore) queryUser(ctx context.Context, where string, args ...any) (User, error) {
	query := `
		SELECT
			id,
			username,
			email,
			COALESCE(display_name, ''),
			COALESCE(bio, ''),
			COALESCE(avatar_key, ''),
			created_at,
			updated_at
		FROM users
		WHERE ` + where + `
		LIMIT 1
	`
	user, err := scanUser(s.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

func (s *UserStore) queryUserAuth(ctx context.Context, where string, args ...any) (userAuthRecord, error) {
	query := `
		SELECT
			id,
			username,
			email,
			password_hash,
			COALESCE(display_name, ''),
			COALESCE(bio, ''),
			COALESCE(avatar_key, ''),
			created_at,
			updated_at
		FROM users
		WHERE ` + where + `
		LIMIT 1
	`
	user, err := scanUserAuth(s.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return userAuthRecord{}, fmt.Errorf("query user auth record: %w", err)
	}
	return user, nil
}

func (s *UserStore) queryUserTestResult(ctx context.Context, where string, args ...any) (UserTestResult, error) {
	return queryUserTestResult(ctx, s.db, where, args...)
}

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func queryUserTestResult(ctx context.Context, db queryer, where string, args ...any) (UserTestResult, error) {
	query := `
		SELECT
			id,
			user_id,
			mbti_type,
			COALESCE(scores_json, ''),
			COALESCE(answers_json, ''),
			COALESCE(duration_seconds, 0),
			is_primary,
			created_at
		FROM user_test_results
		WHERE ` + where + `
		LIMIT 1
	`
	result, err := scanUserTestResult(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return UserTestResult{}, fmt.Errorf("query user test result: %w", err)
	}
	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarKey,
		&createdAt,
		&updatedAt,
	); err != nil {
		return User{}, err
	}

	var err error
	user.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return User{}, err
	}
	user.UpdatedAt, err = parseDBTime(updatedAt)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func scanUserAuth(row rowScanner) (userAuthRecord, error) {
	var record userAuthRecord
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&record.ID,
		&record.Username,
		&record.Email,
		&record.PasswordHash,
		&record.DisplayName,
		&record.Bio,
		&record.AvatarKey,
		&createdAt,
		&updatedAt,
	); err != nil {
		return userAuthRecord{}, err
	}

	var err error
	record.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return userAuthRecord{}, err
	}
	record.UpdatedAt, err = parseDBTime(updatedAt)
	if err != nil {
		return userAuthRecord{}, err
	}
	return record, nil
}

func scanUserTestResult(row rowScanner) (UserTestResult, error) {
	var result UserTestResult
	var createdAt string
	var isPrimary int
	if err := row.Scan(
		&result.ID,
		&result.UserID,
		&result.MBTIType,
		&result.ScoresJSON,
		&result.AnswersJSON,
		&result.DurationSeconds,
		&isPrimary,
		&createdAt,
	); err != nil {
		return UserTestResult{}, err
	}

	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return UserTestResult{}, err
	}
	result.CreatedAt = parsedCreatedAt
	result.IsPrimary = isPrimary != 0
	return result, nil
}

func formatDBTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseDBTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse database timestamp: %w", err)
	}
	return parsed.UTC(), nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
