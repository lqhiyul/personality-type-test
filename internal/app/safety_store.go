package app

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
	ReportTargetProfile = "profile"
	ReportTargetComment = "comment"
	ReportTargetMessage = "message"

	ReportStatusOpen      = "open"
	ReportStatusReviewed  = "reviewed"
	ReportStatusDismissed = "dismissed"

	maxReportReasonLength   = 80
	maxReportDetailsLength  = 500
	defaultAdminReportLimit = 100
	maxAdminReportLimit     = 200
)

var (
	ErrBlockSelf             = errors.New("cannot block yourself")
	ErrReportTargetInvalid   = errors.New("report target type is invalid")
	ErrReportReasonRequired  = errors.New("report reason is required")
	ErrReportReasonTooLong   = errors.New("report reason is too long")
	ErrReportDetailsTooLong  = errors.New("report details are too long")
	ErrReportTextInvalid     = errors.New("report text contains invalid characters")
	ErrReportStatusInvalid   = errors.New("report status is invalid")
	ErrReportTargetForbidden = errors.New("report target is not available")
	ErrBlockedInteraction    = errors.New("cannot interact with blocked user")
)

type UserBlock struct {
	ID            int64
	BlockerUserID int64
	BlockedUserID int64
	CreatedAt     time.Time
	BlockedUser   User
}

type ReportUser struct {
	ID          int64
	Username    string
	DisplayName string
	AvatarKey   string
}

type UserReport struct {
	ID             int64
	ReporterUserID int64
	TargetUserID   sql.NullInt64
	TargetType     string
	TargetID       sql.NullInt64
	Reason         string
	Details        string
	Status         string
	CreatedAt      time.Time
	ReviewedAt     *time.Time
	Reporter       ReportUser
	TargetUser     *ReportUser
}

func (s *UserStore) BlockUser(ctx context.Context, blockerID, blockedID int64) (UserBlock, error) {
	if blockerID == blockedID {
		return UserBlock{}, ErrBlockSelf
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return UserBlock{}, fmt.Errorf("begin block user: %w", err)
	}
	defer tx.Rollback()

	now := formatDBTime(s.now())
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO user_blocks (blocker_user_id, blocked_user_id, created_at)
		VALUES (?, ?, ?)
	`, blockerID, blockedID, now); err != nil {
		return UserBlock{}, fmt.Errorf("block user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM friendships
		WHERE (requester_id = ? AND addressee_id = ?)
		   OR (requester_id = ? AND addressee_id = ?)
	`, blockerID, blockedID, blockedID, blockerID); err != nil {
		return UserBlock{}, fmt.Errorf("remove friendship after block: %w", err)
	}

	block, err := queryUserBlock(ctx, tx, "ub.blocker_user_id = ? AND ub.blocked_user_id = ?", blockerID, blockedID)
	if err != nil {
		return UserBlock{}, err
	}
	if err := tx.Commit(); err != nil {
		return UserBlock{}, fmt.Errorf("commit block user: %w", err)
	}
	return block, nil
}

func (s *UserStore) UnblockUser(ctx context.Context, blockerID, blockedID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM user_blocks
		WHERE blocker_user_id = ? AND blocked_user_id = ?
	`, blockerID, blockedID)
	if err != nil {
		return fmt.Errorf("unblock user: %w", err)
	}
	return nil
}

func (s *UserStore) IsUserBlockedBy(ctx context.Context, blockerID, blockedID int64) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE blocker_user_id = ? AND blocked_user_id = ?
		)
	`, blockerID, blockedID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user block: %w", err)
	}
	return exists != 0, nil
}

func (s *UserStore) IsBlockedBetween(ctx context.Context, userAID, userBID int64) (bool, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_blocks
			WHERE (blocker_user_id = ? AND blocked_user_id = ?)
			   OR (blocker_user_id = ? AND blocked_user_id = ?)
		)
	`, userAID, userBID, userBID, userAID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check users block: %w", err)
	}
	return exists != 0, nil
}

func (s *UserStore) ListBlockedUsers(ctx context.Context, blockerID int64) ([]UserBlock, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			ub.id,
			ub.blocker_user_id,
			ub.blocked_user_id,
			ub.created_at,
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
			u.updated_at
		FROM user_blocks ub
		JOIN users u ON u.id = ub.blocked_user_id
		WHERE ub.blocker_user_id = ?
		ORDER BY ub.created_at DESC, ub.id DESC
	`, blockerID)
	if err != nil {
		return nil, fmt.Errorf("list blocked users: %w", err)
	}
	defer rows.Close()

	blocks := make([]UserBlock, 0)
	for rows.Next() {
		block, err := scanUserBlock(rows)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list blocked users rows: %w", err)
	}
	return blocks, nil
}

func (s *UserStore) CreateReport(ctx context.Context, reporterID int64, targetType string, targetID, targetUserID int64, reason, details string) (UserReport, error) {
	normalizedType, err := normalizeReportTargetType(targetType)
	if err != nil {
		return UserReport{}, err
	}
	normalizedReason, normalizedDetails, err := normalizeReportText(reason, details)
	if err != nil {
		return UserReport{}, err
	}

	var targetIDArg any
	if targetID > 0 {
		targetIDArg = targetID
	}
	var targetUserIDArg any
	if targetUserID > 0 {
		targetUserIDArg = targetUserID
	}

	createdAt := formatDBTime(s.now())
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO user_reports (
			reporter_user_id,
			target_user_id,
			target_type,
			target_id,
			reason,
			details,
			status,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, reporterID, targetUserIDArg, normalizedType, targetIDArg, normalizedReason, normalizedDetails, ReportStatusOpen, createdAt)
	if err != nil {
		return UserReport{}, fmt.Errorf("create report: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return UserReport{}, fmt.Errorf("read created report id: %w", err)
	}
	return s.GetReportByID(ctx, id)
}

func (s *UserStore) GetReportByID(ctx context.Context, reportID int64) (UserReport, error) {
	return queryUserReport(ctx, s.db, "ur.id = ?", reportID)
}

func (s *UserStore) ListReportsForAdmin(ctx context.Context, limit int, status string) ([]UserReport, error) {
	limit = normalizeAdminReportLimit(limit)
	status = strings.TrimSpace(status)
	args := []any{}
	where := "1 = 1"
	if status != "" {
		normalizedStatus, err := normalizeReportStatus(status)
		if err != nil {
			return nil, err
		}
		where = "ur.status = ?"
		args = append(args, normalizedStatus)
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, reportSelectQuery(where)+`
		ORDER BY ur.created_at DESC, ur.id DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	reports := make([]UserReport, 0)
	for rows.Next() {
		report, err := scanUserReport(rows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list reports rows: %w", err)
	}
	return reports, nil
}

func (s *UserStore) UpdateReportStatus(ctx context.Context, reportID int64, status string) (UserReport, error) {
	normalizedStatus, err := normalizeReportStatus(status)
	if err != nil {
		return UserReport{}, err
	}
	var reviewedAt any
	if normalizedStatus == ReportStatusOpen {
		reviewedAt = nil
	} else {
		reviewedAt = formatDBTime(s.now())
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE user_reports
		SET status = ?, reviewed_at = ?
		WHERE id = ?
	`, normalizedStatus, reviewedAt, reportID)
	if err != nil {
		return UserReport{}, fmt.Errorf("update report status: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return UserReport{}, fmt.Errorf("read report status rows: %w", err)
	}
	if rows == 0 {
		return UserReport{}, sql.ErrNoRows
	}
	return s.GetReportByID(ctx, reportID)
}

func queryUserBlock(ctx context.Context, db queryer, where string, args ...any) (UserBlock, error) {
	query := `
		SELECT
			ub.id,
			ub.blocker_user_id,
			ub.blocked_user_id,
			ub.created_at,
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
			u.updated_at
		FROM user_blocks ub
		JOIN users u ON u.id = ub.blocked_user_id
		WHERE ` + where + `
		LIMIT 1
	`
	block, err := scanUserBlock(db.QueryRowContext(ctx, query, args...))
	if err != nil {
		return UserBlock{}, fmt.Errorf("query user block: %w", err)
	}
	return block, nil
}

func scanUserBlock(row rowScanner) (UserBlock, error) {
	var block UserBlock
	var blockCreatedAt string
	var userCreatedAt string
	var userUpdatedAt string
	var showPrimaryResult int
	var showCompletedCount int
	var showFriends int
	if err := row.Scan(
		&block.ID,
		&block.BlockerUserID,
		&block.BlockedUserID,
		&blockCreatedAt,
		&block.BlockedUser.ID,
		&block.BlockedUser.Username,
		&block.BlockedUser.Email,
		&block.BlockedUser.DisplayName,
		&block.BlockedUser.Bio,
		&block.BlockedUser.AvatarKey,
		&block.BlockedUser.ProfileVisibility,
		&showPrimaryResult,
		&showCompletedCount,
		&showFriends,
		&userCreatedAt,
		&userUpdatedAt,
	); err != nil {
		return UserBlock{}, err
	}

	var err error
	block.CreatedAt, err = parseDBTime(blockCreatedAt)
	if err != nil {
		return UserBlock{}, err
	}
	block.BlockedUser.CreatedAt, err = parseDBTime(userCreatedAt)
	if err != nil {
		return UserBlock{}, err
	}
	block.BlockedUser.UpdatedAt, err = parseDBTime(userUpdatedAt)
	if err != nil {
		return UserBlock{}, err
	}
	block.BlockedUser.ProfileVisibility = normalizedProfileVisibilityOrDefault(block.BlockedUser.ProfileVisibility)
	block.BlockedUser.ShowPrimaryResult = showPrimaryResult != 0
	block.BlockedUser.ShowCompletedCount = showCompletedCount != 0
	block.BlockedUser.ShowFriends = showFriends != 0
	return block, nil
}

func queryUserReport(ctx context.Context, db queryer, where string, args ...any) (UserReport, error) {
	report, err := scanUserReport(db.QueryRowContext(ctx, reportSelectQuery(where)+` LIMIT 1`, args...))
	if err != nil {
		return UserReport{}, fmt.Errorf("query report: %w", err)
	}
	return report, nil
}

func reportSelectQuery(where string) string {
	return `
		SELECT
			ur.id,
			ur.reporter_user_id,
			ur.target_user_id,
			ur.target_type,
			ur.target_id,
			ur.reason,
			COALESCE(ur.details, ''),
			ur.status,
			ur.created_at,
			ur.reviewed_at,
			reporter.id,
			reporter.username,
			COALESCE(reporter.display_name, ''),
			COALESCE(reporter.avatar_key, ''),
			target.id,
			target.username,
			COALESCE(target.display_name, ''),
			COALESCE(target.avatar_key, '')
		FROM user_reports ur
		JOIN users reporter ON reporter.id = ur.reporter_user_id
		LEFT JOIN users target ON target.id = ur.target_user_id
		WHERE ` + where + `
	`
}

func scanUserReport(row rowScanner) (UserReport, error) {
	var report UserReport
	var createdAt string
	var reviewedAt sql.NullString
	var targetID sql.NullInt64
	var targetUsername sql.NullString
	var targetDisplayName sql.NullString
	var targetAvatarKey sql.NullString
	if err := row.Scan(
		&report.ID,
		&report.ReporterUserID,
		&report.TargetUserID,
		&report.TargetType,
		&report.TargetID,
		&report.Reason,
		&report.Details,
		&report.Status,
		&createdAt,
		&reviewedAt,
		&report.Reporter.ID,
		&report.Reporter.Username,
		&report.Reporter.DisplayName,
		&report.Reporter.AvatarKey,
		&targetID,
		&targetUsername,
		&targetDisplayName,
		&targetAvatarKey,
	); err != nil {
		return UserReport{}, err
	}

	var err error
	report.CreatedAt, err = parseDBTime(createdAt)
	if err != nil {
		return UserReport{}, err
	}
	if reviewedAt.Valid && strings.TrimSpace(reviewedAt.String) != "" {
		parsed, err := parseDBTime(reviewedAt.String)
		if err != nil {
			return UserReport{}, err
		}
		report.ReviewedAt = &parsed
	}
	report.Reporter.DisplayName = strings.TrimSpace(report.Reporter.DisplayName)
	if report.Reporter.DisplayName == "" {
		report.Reporter.DisplayName = report.Reporter.Username
	}
	report.Reporter.AvatarKey = publicAvatarKey(report.Reporter.AvatarKey)
	if targetID.Valid {
		target := ReportUser{
			ID:          targetID.Int64,
			Username:    targetUsername.String,
			DisplayName: strings.TrimSpace(targetDisplayName.String),
			AvatarKey:   publicAvatarKey(targetAvatarKey.String),
		}
		if target.DisplayName == "" {
			target.DisplayName = target.Username
		}
		report.TargetUser = &target
	}
	return report, nil
}

func normalizeReportTargetType(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case ReportTargetProfile:
		return ReportTargetProfile, nil
	case ReportTargetComment:
		return ReportTargetComment, nil
	case ReportTargetMessage:
		return ReportTargetMessage, nil
	default:
		return "", ErrReportTargetInvalid
	}
}

func normalizeReportStatus(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case ReportStatusOpen:
		return ReportStatusOpen, nil
	case ReportStatusReviewed:
		return ReportStatusReviewed, nil
	case ReportStatusDismissed:
		return ReportStatusDismissed, nil
	default:
		return "", ErrReportStatusInvalid
	}
}

func normalizeReportText(reason, details string) (string, string, error) {
	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		return "", "", ErrReportReasonRequired
	}
	if utf8.RuneCountInString(normalizedReason) > maxReportReasonLength {
		return "", "", ErrReportReasonTooLong
	}
	normalizedDetails := strings.TrimSpace(details)
	if utf8.RuneCountInString(normalizedDetails) > maxReportDetailsLength {
		return "", "", ErrReportDetailsTooLong
	}
	if containsSafetyDisallowedControl(normalizedReason) || containsSafetyDisallowedControl(normalizedDetails) {
		return "", "", ErrReportTextInvalid
	}
	return normalizedReason, normalizedDetails, nil
}

func containsSafetyDisallowedControl(value string) bool {
	for _, r := range value {
		if !unicode.IsControl(r) || r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		return true
	}
	return false
}

func normalizeAdminReportLimit(limit int) int {
	if limit <= 0 {
		return defaultAdminReportLimit
	}
	if limit > maxAdminReportLimit {
		return maxAdminReportLimit
	}
	return limit
}
