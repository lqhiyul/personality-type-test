package app

import (
	"fmt"
	"strings"
	"time"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (User, error) {
	var user User
	var createdAt string
	var updatedAt string
	var showPrimaryResult int
	var showCompletedCount int
	var showFriends int
	if err := row.Scan(
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
	user.ProfileVisibility = normalizedProfileVisibilityOrDefault(user.ProfileVisibility)
	user.ShowPrimaryResult = showPrimaryResult != 0
	user.ShowCompletedCount = showCompletedCount != 0
	user.ShowFriends = showFriends != 0
	return user, nil
}

func scanUserAuth(row rowScanner) (userAuthRecord, error) {
	var record userAuthRecord
	var createdAt string
	var updatedAt string
	var showPrimaryResult int
	var showCompletedCount int
	var showFriends int
	if err := row.Scan(
		&record.ID,
		&record.Username,
		&record.Email,
		&record.PasswordHash,
		&record.DisplayName,
		&record.Bio,
		&record.AvatarKey,
		&record.ProfileVisibility,
		&showPrimaryResult,
		&showCompletedCount,
		&showFriends,
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
	record.ProfileVisibility = normalizedProfileVisibilityOrDefault(record.ProfileVisibility)
	record.ShowPrimaryResult = showPrimaryResult != 0
	record.ShowCompletedCount = showCompletedCount != 0
	record.ShowFriends = showFriends != 0
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

func normalizedProfileVisibilityOrDefault(value string) string {
	switch strings.TrimSpace(value) {
	case profileVisibilityPrivate:
		return profileVisibilityPrivate
	default:
		return profileVisibilityPublic
	}
}
