package app

import (
	"context"
	"database/sql"
	"fmt"
)

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

func (s *UserStore) GetPrimaryUserTestResult(ctx context.Context, userID int64) (UserTestResult, error) {
	return s.queryUserTestResult(ctx, "user_id = ? AND is_primary = 1 ORDER BY created_at DESC, id DESC", userID)
}

func (s *UserStore) CountUserTestResults(ctx context.Context, userID int64) (int, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_test_results WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count user test results: %w", err)
	}
	return count, nil
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
