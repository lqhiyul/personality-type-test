package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

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

func (s *UserStore) UpdateUserProfile(ctx context.Context, id int64, params UpdateUserProfileParams) (User, error) {
	current, err := s.GetUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}

	profileVisibility := strings.TrimSpace(params.ProfileVisibility)
	if profileVisibility == "" {
		profileVisibility = current.ProfileVisibility
	}
	if profileVisibility == "" {
		profileVisibility = profileVisibilityPublic
	}
	showPrimaryResult := current.ShowPrimaryResult
	if params.ShowPrimaryResult != nil {
		showPrimaryResult = *params.ShowPrimaryResult
	}
	showCompletedCount := current.ShowCompletedCount
	if params.ShowCompletedCount != nil {
		showCompletedCount = *params.ShowCompletedCount
	}
	showFriends := current.ShowFriends
	if params.ShowFriends != nil {
		showFriends = *params.ShowFriends
	}

	updatedAt := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET
			display_name = ?,
			bio = ?,
			avatar_key = ?,
			profile_visibility = ?,
			show_primary_result = ?,
			show_completed_count = ?,
			show_friends = ?,
			updated_at = ?
		WHERE id = ?
	`, params.DisplayName, params.Bio, params.AvatarKey, profileVisibility, boolToInt(showPrimaryResult), boolToInt(showCompletedCount), boolToInt(showFriends), updatedAt, id)
	if err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("read updated user profile rows: %w", err)
	}
	if rows == 0 {
		return User{}, sql.ErrNoRows
	}
	return s.GetUserByID(ctx, id)
}

func (s *UserStore) getUserAuthByUsername(ctx context.Context, username string) (userAuthRecord, error) {
	return s.queryUserAuth(ctx, "username = ?", username)
}

func (s *UserStore) getUserAuthByEmail(ctx context.Context, email string) (userAuthRecord, error) {
	return s.queryUserAuth(ctx, "email = ?", email)
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
			COALESCE(profile_visibility, 'public'),
			COALESCE(show_primary_result, 1),
			COALESCE(show_completed_count, 1),
			COALESCE(show_friends, 1),
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
			COALESCE(profile_visibility, 'public'),
			COALESCE(show_primary_result, 1),
			COALESCE(show_completed_count, 1),
			COALESCE(show_friends, 1),
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
