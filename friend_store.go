package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	FriendshipStatusPending  = "pending"
	FriendshipStatusAccepted = "accepted"
	FriendshipStatusRejected = "rejected"
)

var (
	ErrFriendshipSelf        = errors.New("cannot friend yourself")
	ErrFriendshipExists      = errors.New("friendship already exists")
	ErrFriendshipForbidden   = errors.New("friendship action is not allowed")
	ErrFriendshipNotPending  = errors.New("friend request is not pending")
	ErrFriendshipNotAccepted = errors.New("friendship is not accepted")
)

type Friendship struct {
	ID          int64
	RequesterID int64
	AddresseeID int64
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FriendListItem struct {
	Friendship  Friendship
	User        User
	PrimaryType string
}

type IncomingFriendRequest struct {
	Friendship  Friendship
	Requester   User
	PrimaryType string
}

func (s *UserStore) CreateFriendRequest(ctx context.Context, requesterID, addresseeID int64) (Friendship, error) {
	if requesterID == addresseeID {
		return Friendship{}, ErrFriendshipSelf
	}

	if _, err := s.GetFriendshipBetween(ctx, requesterID, addresseeID); err == nil {
		return Friendship{}, ErrFriendshipExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Friendship{}, err
	}

	now := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO friendships (requester_id, addressee_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, requesterID, addresseeID, FriendshipStatusPending, now, now)
	if err != nil {
		if isUniqueConstraintError(err) {
			return Friendship{}, ErrFriendshipExists
		}
		return Friendship{}, fmt.Errorf("create friend request: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Friendship{}, fmt.Errorf("read created friend request id: %w", err)
	}
	return s.GetFriendshipByID(ctx, id)
}

func (s *UserStore) GetFriendshipByID(ctx context.Context, id int64) (Friendship, error) {
	return s.queryFriendship(ctx, "id = ?", id)
}

func (s *UserStore) GetFriendshipBetween(ctx context.Context, userAID, userBID int64) (Friendship, error) {
	return s.queryFriendship(ctx, `
		(requester_id = ? AND addressee_id = ?)
		OR (requester_id = ? AND addressee_id = ?)
	`, userAID, userBID, userBID, userAID)
}

func (s *UserStore) ListIncomingFriendRequests(ctx context.Context, userID int64) ([]IncomingFriendRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			f.id,
			f.requester_id,
			f.addressee_id,
			f.status,
			f.created_at,
			f.updated_at,
			u.id,
			u.username,
			u.email,
			COALESCE(u.display_name, ''),
			COALESCE(u.bio, ''),
			COALESCE(u.avatar_key, ''),
			COALESCE(u.profile_visibility, 'public'),
			COALESCE(u.show_primary_result, 1),
			COALESCE(u.show_completed_count, 1),
			COALESCE(u.show_friends, 1),
			u.created_at,
			u.updated_at,
			COALESCE((
				SELECT mbti_type
				FROM user_test_results
				WHERE user_id = u.id AND is_primary = 1
				ORDER BY created_at DESC, id DESC
				LIMIT 1
			), '')
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.addressee_id = ? AND f.status = ?
		ORDER BY f.created_at DESC, f.id DESC
	`, userID, FriendshipStatusPending)
	if err != nil {
		return nil, fmt.Errorf("list incoming friend requests: %w", err)
	}
	defer rows.Close()

	var requests []IncomingFriendRequest
	for rows.Next() {
		var request IncomingFriendRequest
		friendship, user, primaryType, err := scanFriendshipUserWithPrimary(rows)
		if err != nil {
			return nil, err
		}
		request.Friendship = friendship
		request.Requester = user
		request.PrimaryType = primaryType
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list incoming friend requests rows: %w", err)
	}
	return requests, nil
}

func (s *UserStore) AcceptFriendRequest(ctx context.Context, currentUserID, friendshipID int64) (Friendship, error) {
	friendship, err := s.GetFriendshipByID(ctx, friendshipID)
	if err != nil {
		return Friendship{}, err
	}
	if friendship.AddresseeID != currentUserID {
		return Friendship{}, ErrFriendshipForbidden
	}
	if friendship.Status != FriendshipStatusPending {
		return Friendship{}, ErrFriendshipNotPending
	}

	updatedAt := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		UPDATE friendships
		SET status = ?, updated_at = ?
		WHERE id = ? AND addressee_id = ? AND status = ?
	`, FriendshipStatusAccepted, updatedAt, friendshipID, currentUserID, FriendshipStatusPending)
	if err != nil {
		return Friendship{}, fmt.Errorf("accept friend request: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return Friendship{}, fmt.Errorf("read accepted friend request rows: %w", err)
	}
	if rows == 0 {
		return Friendship{}, ErrFriendshipNotPending
	}
	return s.GetFriendshipByID(ctx, friendshipID)
}

func (s *UserStore) RemoveFriendship(ctx context.Context, currentUserID, friendshipID int64) error {
	friendship, err := s.GetFriendshipByID(ctx, friendshipID)
	if err != nil {
		return err
	}
	if friendship.RequesterID != currentUserID && friendship.AddresseeID != currentUserID {
		return ErrFriendshipForbidden
	}
	if friendship.Status != FriendshipStatusAccepted {
		return ErrFriendshipNotAccepted
	}

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM friendships
		WHERE id = ? AND status = ? AND (requester_id = ? OR addressee_id = ?)
	`, friendshipID, FriendshipStatusAccepted, currentUserID, currentUserID)
	if err != nil {
		return fmt.Errorf("remove friendship: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read removed friendship rows: %w", err)
	}
	if rows == 0 {
		return ErrFriendshipForbidden
	}
	return nil
}

func (s *UserStore) ListFriends(ctx context.Context, userID int64) ([]FriendListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			f.id,
			f.requester_id,
			f.addressee_id,
			f.status,
			f.created_at,
			f.updated_at,
			u.id,
			u.username,
			u.email,
			COALESCE(u.display_name, ''),
			COALESCE(u.bio, ''),
			COALESCE(u.avatar_key, ''),
			COALESCE(u.profile_visibility, 'public'),
			COALESCE(u.show_primary_result, 1),
			COALESCE(u.show_completed_count, 1),
			COALESCE(u.show_friends, 1),
			u.created_at,
			u.updated_at,
			COALESCE((
				SELECT mbti_type
				FROM user_test_results
				WHERE user_id = u.id AND is_primary = 1
				ORDER BY created_at DESC, id DESC
				LIMIT 1
			), '')
		FROM friendships f
		JOIN users u ON u.id = CASE
			WHEN f.requester_id = ? THEN f.addressee_id
			ELSE f.requester_id
		END
		WHERE f.status = ? AND (f.requester_id = ? OR f.addressee_id = ?)
		ORDER BY LOWER(u.username), u.id
	`, userID, FriendshipStatusAccepted, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("list friends: %w", err)
	}
	defer rows.Close()

	var friends []FriendListItem
	for rows.Next() {
		var item FriendListItem
		friendship, user, primaryType, err := scanFriendshipUserWithPrimary(rows)
		if err != nil {
			return nil, err
		}
		item.Friendship = friendship
		item.User = user
		item.PrimaryType = primaryType
		friends = append(friends, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list friends rows: %w", err)
	}
	return friends, nil
}

func (s *UserStore) queryFriendship(ctx context.Context, where string, args ...any) (Friendship, error) {
	query := `
		SELECT
			id,
			requester_id,
			addressee_id,
			status,
			created_at,
			updated_at
		FROM friendships
		WHERE ` + where + `
		LIMIT 1
	`
	friendship, err := scanFriendship(s.db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return Friendship{}, fmt.Errorf("query friendship: %w", err)
	}
	return friendship, nil
}

func scanFriendship(row rowScanner) (Friendship, error) {
	var friendship Friendship
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&friendship.ID,
		&friendship.RequesterID,
		&friendship.AddresseeID,
		&friendship.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Friendship{}, err
	}

	var err error
	friendship.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return Friendship{}, err
	}
	friendship.UpdatedAt, err = parseDBTime(updatedAt)
	if err != nil {
		return Friendship{}, err
	}
	return friendship, nil
}

func scanFriendshipUserWithPrimary(row rowScanner) (Friendship, User, string, error) {
	var friendship Friendship
	var user User
	var primaryType string
	var friendshipCreatedAt string
	var friendshipUpdatedAt string
	var createdAt string
	var updatedAt string
	var showPrimaryResult int
	var showCompletedCount int
	var showFriends int
	if err := row.Scan(
		&friendship.ID,
		&friendship.RequesterID,
		&friendship.AddresseeID,
		&friendship.Status,
		&friendshipCreatedAt,
		&friendshipUpdatedAt,
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarKey,
		&user.ProfileVisibility,
		&showPrimaryResult,
		&showCompletedCount,
		&showFriends,
		&createdAt,
		&updatedAt,
		&primaryType,
	); err != nil {
		return Friendship{}, User{}, "", err
	}

	var err error
	friendship.CreatedAt, err = parseDBTime(friendshipCreatedAt)
	if err != nil {
		return Friendship{}, User{}, "", err
	}
	friendship.UpdatedAt, err = parseDBTime(friendshipUpdatedAt)
	if err != nil {
		return Friendship{}, User{}, "", err
	}
	user.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return Friendship{}, User{}, "", err
	}
	user.UpdatedAt, err = parseDBTime(updatedAt)
	if err != nil {
		return Friendship{}, User{}, "", err
	}
	user.ProfileVisibility = normalizedProfileVisibilityOrDefault(user.ProfileVisibility)
	user.ShowPrimaryResult = showPrimaryResult != 0
	user.ShowCompletedCount = showCompletedCount != 0
	user.ShowFriends = showFriends != 0
	return friendship, user, primaryType, nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "constraint failed")
}
