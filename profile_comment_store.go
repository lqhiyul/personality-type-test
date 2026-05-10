package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultProfileCommentLimit  = 50
	maxProfileCommentLimit      = 50
	maxProfileCommentBodyLength = 500
)

var (
	ErrProfileCommentBodyRequired = errors.New("comment cannot be empty")
	ErrProfileCommentBodyTooLong  = errors.New("comment is too long")
	ErrProfileCommentBodyInvalid  = errors.New("comment contains invalid characters")
	ErrProfileCommentForbidden    = errors.New("profile comment action is not allowed")
)

type ProfileCommentAuthor struct {
	Username    string
	DisplayName string
	AvatarKey   string
}

type ProfileComment struct {
	ID            int64
	ProfileUserID int64
	AuthorUserID  int64
	Body          string
	CreatedAt     time.Time
	Author        ProfileCommentAuthor
}

func (s *UserStore) CreateProfileComment(ctx context.Context, profileUserID, authorUserID int64, body string) (ProfileComment, error) {
	normalizedBody, err := normalizeProfileCommentBody(body)
	if err != nil {
		return ProfileComment{}, err
	}

	createdAt := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO profile_comments (profile_user_id, author_user_id, body, created_at)
		VALUES (?, ?, ?, ?)
	`, profileUserID, authorUserID, normalizedBody, createdAt)
	if err != nil {
		return ProfileComment{}, fmt.Errorf("create profile comment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return ProfileComment{}, fmt.Errorf("read created profile comment id: %w", err)
	}
	return s.GetProfileCommentByID(ctx, id)
}

func (s *UserStore) ListProfileComments(ctx context.Context, profileUserID int64, limit int) ([]ProfileComment, error) {
	limit = normalizedProfileCommentLimit(limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			pc.id,
			pc.profile_user_id,
			pc.author_user_id,
			pc.body,
			pc.created_at,
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_key, '')
		FROM profile_comments pc
		JOIN users u ON u.id = pc.author_user_id
		WHERE pc.profile_user_id = ?
		ORDER BY pc.created_at DESC, pc.id DESC
		LIMIT ?
	`, profileUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("list profile comments: %w", err)
	}
	defer rows.Close()

	comments := make([]ProfileComment, 0)
	for rows.Next() {
		comment, err := scanProfileComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list profile comments rows: %w", err)
	}
	return comments, nil
}

func (s *UserStore) GetProfileCommentByID(ctx context.Context, commentID int64) (ProfileComment, error) {
	return s.queryProfileComment(ctx, "pc.id = ?", commentID)
}

func (s *UserStore) DeleteProfileComment(ctx context.Context, commentID, requesterUserID int64) error {
	comment, err := s.GetProfileCommentByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.AuthorUserID != requesterUserID && comment.ProfileUserID != requesterUserID {
		return ErrProfileCommentForbidden
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM profile_comments WHERE id = ?`, commentID)
	if err != nil {
		return fmt.Errorf("delete profile comment: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted profile comment rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *UserStore) queryProfileComment(ctx context.Context, where string, args ...any) (ProfileComment, error) {
	query := `
		SELECT
			pc.id,
			pc.profile_user_id,
			pc.author_user_id,
			pc.body,
			pc.created_at,
			u.username,
			COALESCE(u.display_name, ''),
			COALESCE(u.avatar_key, '')
		FROM profile_comments pc
		JOIN users u ON u.id = pc.author_user_id
		WHERE ` + where + `
		LIMIT 1
	`
	comment, err := scanProfileComment(s.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return ProfileComment{}, fmt.Errorf("query profile comment: %w", err)
	}
	return comment, nil
}

func scanProfileComment(row rowScanner) (ProfileComment, error) {
	var comment ProfileComment
	var createdAt string
	if err := row.Scan(
		&comment.ID,
		&comment.ProfileUserID,
		&comment.AuthorUserID,
		&comment.Body,
		&createdAt,
		&comment.Author.Username,
		&comment.Author.DisplayName,
		&comment.Author.AvatarKey,
	); err != nil {
		return ProfileComment{}, err
	}

	parsedCreatedAt, err := parseDBTime(createdAt)
	if err != nil {
		return ProfileComment{}, err
	}
	comment.CreatedAt = parsedCreatedAt
	comment.Author.DisplayName = strings.TrimSpace(comment.Author.DisplayName)
	if comment.Author.DisplayName == "" {
		comment.Author.DisplayName = comment.Author.Username
	}
	comment.Author.AvatarKey = publicAvatarKey(comment.Author.AvatarKey)
	return comment, nil
}

func normalizeProfileCommentBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", ErrProfileCommentBodyRequired
	}
	if utf8.RuneCountInString(trimmed) > maxProfileCommentBodyLength {
		return "", ErrProfileCommentBodyTooLong
	}
	if containsProfileCommentDisallowedControl(trimmed) {
		return "", ErrProfileCommentBodyInvalid
	}
	return trimmed, nil
}

func containsProfileCommentDisallowedControl(value string) bool {
	for _, r := range value {
		if !unicode.IsControl(r) || r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		return true
	}
	return false
}

func normalizedProfileCommentLimit(limit int) int {
	if limit <= 0 {
		return defaultProfileCommentLimit
	}
	if limit > maxProfileCommentLimit {
		return maxProfileCommentLimit
	}
	return limit
}
