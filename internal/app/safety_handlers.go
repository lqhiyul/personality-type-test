package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type blockUserRequest struct {
	Username string `json:"username"`
}

type blockedUserResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarKey   string `json:"avatarKey"`
	CreatedAt   string `json:"createdAt"`
}

type createReportRequest struct {
	TargetType string `json:"targetType"`
	TargetID   int64  `json:"targetId"`
	Username   string `json:"username"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

type reportStatusRequest struct {
	Status string `json:"status"`
}

type reportUserResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarKey   string `json:"avatarKey"`
}

type reportResponse struct {
	ID         int64               `json:"id"`
	TargetType string              `json:"targetType"`
	TargetID   *int64              `json:"targetId,omitempty"`
	Reason     string              `json:"reason,omitempty"`
	Details    string              `json:"details,omitempty"`
	Status     string              `json:"status"`
	CreatedAt  string              `json:"createdAt"`
	ReviewedAt string              `json:"reviewedAt,omitempty"`
	Reporter   *reportUserResponse `json:"reporter,omitempty"`
	TargetUser *reportUserResponse `json:"targetUser,omitempty"`
}

func (a *App) handleBlocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListBlocks(w, r)
	case http.MethodPost:
		a.handleBlockUser(w, r)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *App) handleListBlocks(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	blocks, err := a.userStore.ListBlockedUsers(r.Context(), currentUserID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load blocked users")
		return
	}
	responses := make([]blockedUserResponse, 0, len(blocks))
	for _, block := range blocks {
		responses = append(responses, newBlockedUserResponse(block))
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": responses})
}

func (a *App) handleBlockUser(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	var req blockUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	target, ok := a.blockTargetUser(w, r, req.Username)
	if !ok {
		return
	}
	block, err := a.userStore.BlockUser(r.Context(), currentUserID, target.ID)
	if err != nil {
		writeBlockError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newBlockedUserResponse(block))
}

func (a *App) handleBlockByUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	username, ok := blockUsernameFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "blocked user not found")
		return
	}
	target, err := a.userStore.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "target user not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "could not load target user")
		return
	}
	if err := a.userStore.UnblockUser(r.Context(), currentUserID, target.ID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not unblock user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) blockTargetUser(w http.ResponseWriter, r *http.Request, rawUsername string) (User, bool) {
	username, err := normalizeUsername(rawUsername)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return User{}, false
	}
	target, err := a.userStore.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "target user not found")
			return User{}, false
		}
		writeJSONError(w, http.StatusInternalServerError, "could not load target user")
		return User{}, false
	}
	return target, true
}

func (a *App) handleCreateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	var req createReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	targetType, err := normalizeReportTargetType(req.TargetType)
	if err != nil {
		writeReportError(w, err)
		return
	}
	targetID, targetUserID, ok := a.resolveReportTarget(w, r, currentUserID, targetType, req.TargetID, req.Username)
	if !ok {
		return
	}
	report, err := a.userStore.CreateReport(r.Context(), currentUserID, targetType, targetID, targetUserID, req.Reason, req.Details)
	if err != nil {
		writeReportError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newReportResponse(report, false))
}

func (a *App) resolveReportTarget(w http.ResponseWriter, r *http.Request, currentUserID int64, targetType string, targetID int64, username string) (int64, int64, bool) {
	switch targetType {
	case ReportTargetProfile:
		target, ok := a.blockTargetUser(w, r, username)
		if !ok {
			return 0, 0, false
		}
		return target.ID, target.ID, true
	case ReportTargetComment:
		if targetID <= 0 {
			writeJSONError(w, http.StatusBadRequest, "targetId is required")
			return 0, 0, false
		}
		comment, err := a.userStore.GetProfileCommentByID(r.Context(), targetID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSONError(w, http.StatusNotFound, "comment not found")
				return 0, 0, false
			}
			writeJSONError(w, http.StatusInternalServerError, "could not load comment")
			return 0, 0, false
		}
		return comment.ID, comment.AuthorUserID, true
	case ReportTargetMessage:
		if targetID <= 0 {
			writeJSONError(w, http.StatusBadRequest, "targetId is required")
			return 0, 0, false
		}
		message, err := a.userStore.GetMessageForUser(r.Context(), targetID, currentUserID)
		if err != nil {
			if errors.Is(err, ErrMessageForbidden) {
				writeJSONError(w, http.StatusForbidden, "message is not available")
				return 0, 0, false
			}
			writeJSONError(w, http.StatusInternalServerError, "could not load message")
			return 0, 0, false
		}
		return message.ID, message.SenderID, true
	default:
		writeReportError(w, ErrReportTargetInvalid)
		return 0, 0, false
	}
}

func (a *App) handleAdminReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin authentication required")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit := defaultAdminReportLimit
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil {
			limit = parsed
		}
	}
	reports, err := a.userStore.ListReportsForAdmin(r.Context(), limit, status)
	if err != nil {
		writeReportError(w, err)
		return
	}
	responses := make([]reportResponse, 0, len(reports))
	for _, report := range reports {
		responses = append(responses, newReportResponse(report, true))
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": responses})
}

func (a *App) handleAdminReportByID(w http.ResponseWriter, r *http.Request) {
	if !a.authorized(r) {
		writeJSONError(w, http.StatusUnauthorized, "admin authentication required")
		return
	}
	reportID, action, ok := adminReportActionFromPath(r.URL.Path)
	if !ok || action != "status" {
		writeJSONError(w, http.StatusNotFound, "report not found")
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	var req reportStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	report, err := a.userStore.UpdateReportStatus(r.Context(), reportID, req.Status)
	if err != nil {
		writeReportError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newReportResponse(report, true))
}

func blockUsernameFromPath(requestPath string) (string, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/blocks/")
	if suffix == requestPath || suffix == "" || strings.Contains(strings.Trim(suffix, "/"), "/") {
		return "", false
	}
	username, err := normalizeUsername(strings.Trim(suffix, "/"))
	return username, err == nil
}

func adminReportActionFromPath(requestPath string) (int64, string, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/admin/reports/")
	if suffix == requestPath {
		return 0, "", false
	}
	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	return id, parts[1], true
}

func newBlockedUserResponse(block UserBlock) blockedUserResponse {
	return blockedUserResponse{
		ID:          block.BlockedUser.ID,
		Username:    block.BlockedUser.Username,
		DisplayName: publicDisplayName(block.BlockedUser),
		AvatarKey:   publicAvatarKey(block.BlockedUser.AvatarKey),
		CreatedAt:   block.CreatedAt.Format(time.RFC3339Nano),
	}
}

func newReportUserResponse(user ReportUser) reportUserResponse {
	return reportUserResponse(user)
}

func newReportResponse(report UserReport, includeAdminFields bool) reportResponse {
	var targetID *int64
	if report.TargetID.Valid {
		value := report.TargetID.Int64
		targetID = &value
	}
	response := reportResponse{
		ID:         report.ID,
		TargetType: report.TargetType,
		TargetID:   targetID,
		Status:     report.Status,
		CreatedAt:  report.CreatedAt.Format(time.RFC3339Nano),
	}
	if includeAdminFields {
		reporter := newReportUserResponse(report.Reporter)
		response.Reporter = &reporter
		if report.TargetUser != nil {
			target := newReportUserResponse(*report.TargetUser)
			response.TargetUser = &target
		}
		response.Reason = report.Reason
		response.Details = report.Details
		if report.ReviewedAt != nil {
			response.ReviewedAt = report.ReviewedAt.Format(time.RFC3339Nano)
		}
	}
	return response
}

func writeBlockError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrBlockSelf):
		writeJSONError(w, http.StatusBadRequest, "cannot block yourself")
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "target user not found")
	default:
		writeJSONError(w, http.StatusInternalServerError, "could not update block")
	}
}

func writeReportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrReportTargetInvalid):
		writeJSONError(w, http.StatusBadRequest, "report target type is invalid")
	case errors.Is(err, ErrReportReasonRequired):
		writeJSONError(w, http.StatusBadRequest, "report reason is required")
	case errors.Is(err, ErrReportReasonTooLong):
		writeJSONError(w, http.StatusBadRequest, "report reason is too long")
	case errors.Is(err, ErrReportDetailsTooLong):
		writeJSONError(w, http.StatusBadRequest, "report details are too long")
	case errors.Is(err, ErrReportTextInvalid):
		writeJSONError(w, http.StatusBadRequest, "report text contains invalid characters")
	case errors.Is(err, ErrReportStatusInvalid):
		writeJSONError(w, http.StatusBadRequest, "report status is invalid")
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "report not found")
	default:
		writeJSONError(w, http.StatusInternalServerError, "could not update report")
	}
}
