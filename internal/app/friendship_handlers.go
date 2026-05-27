package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type createFriendRequestRequest struct {
	Username string `json:"username"`
}

type friendshipResponse struct {
	ID          int64  `json:"id"`
	RequesterID int64  `json:"requesterId"`
	AddresseeID int64  `json:"addresseeId"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type friendUserResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarKey   string `json:"avatarKey"`
	PrimaryType string `json:"primaryType"`
}

type friendListItemResponse struct {
	FriendshipID  int64                       `json:"friendshipId"`
	ID            int64                       `json:"id"`
	Username      string                      `json:"username"`
	DisplayName   string                      `json:"displayName"`
	AvatarKey     string                      `json:"avatarKey"`
	PrimaryType   string                      `json:"primaryType"`
	Compatibility friendCompatibilityResponse `json:"compatibility"`
}

type incomingFriendRequestResponse struct {
	ID        int64              `json:"id"`
	Status    string             `json:"status"`
	Requester friendUserResponse `json:"requester"`
	CreatedAt string             `json:"createdAt"`
}

func (a *App) handleCreateFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	var req createFriendRequestRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "username is required")
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

	friendship, err := a.userStore.CreateFriendRequest(r.Context(), currentUserID, target.ID)
	if err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newFriendshipResponse(friendship))
}

func (a *App) handleFriends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	currentPrimary, err := a.currentUserPrimaryType(r, currentUserID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load primary result")
		return
	}

	friends, err := a.userStore.ListFriends(r.Context(), currentUserID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load friends")
		return
	}

	responses := make([]friendListItemResponse, 0, len(friends))
	for _, friend := range friends {
		responses = append(responses, newFriendListItemResponse(friend, currentPrimary))
	}
	writeJSON(w, http.StatusOK, map[string]any{"friends": responses})
}

func (a *App) handleFriendRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	requests, err := a.userStore.ListIncomingFriendRequests(r.Context(), currentUserID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load friend requests")
		return
	}

	responses := make([]incomingFriendRequestResponse, 0, len(requests))
	for _, request := range requests {
		responses = append(responses, newIncomingFriendRequestResponse(request))
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": responses})
}

func (a *App) handleFriendRequestByID(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	friendshipID, action, ok := parseFriendRequestActionPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "friend request not found")
		return
	}

	if r.Method != http.MethodPost || action != "accept" {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	friendship, err := a.userStore.AcceptFriendRequest(r.Context(), currentUserID, friendshipID)
	if err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newFriendshipResponse(friendship))
}

func (a *App) handleFriendByID(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	friendshipID, ok := parseFriendPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "friendship not found")
		return
	}

	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}

	if err := a.userStore.RemoveFriendship(r.Context(), currentUserID, friendshipID); err != nil {
		writeFriendshipError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) currentUserPrimaryType(r *http.Request, userID int64) (string, error) {
	primary, err := a.userStore.GetPrimaryUserTestResult(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return primary.MBTIType, nil
}

func parseFriendPath(requestPath string) (int64, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/friends/")
	if suffix == requestPath || suffix == "" || strings.Contains(strings.Trim(suffix, "/"), "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.Trim(suffix, "/"), 10, 64)
	return id, err == nil && id > 0
}

func parseFriendRequestActionPath(requestPath string) (int64, string, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/friends/requests/")
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

func newFriendshipResponse(friendship Friendship) friendshipResponse {
	return friendshipResponse{
		ID:          friendship.ID,
		RequesterID: friendship.RequesterID,
		AddresseeID: friendship.AddresseeID,
		Status:      friendship.Status,
		CreatedAt:   friendship.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   friendship.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func newFriendUserResponse(user User, primaryType string) friendUserResponse {
	return friendUserResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: publicDisplayName(user),
		AvatarKey:   publicAvatarKey(user.AvatarKey),
		PrimaryType: primaryType,
	}
}

func newFriendListItemResponse(item FriendListItem, currentPrimary string) friendListItemResponse {
	user := newFriendUserResponse(item.User, item.PrimaryType)
	return friendListItemResponse{
		FriendshipID:  item.Friendship.ID,
		ID:            user.ID,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		AvatarKey:     user.AvatarKey,
		PrimaryType:   user.PrimaryType,
		Compatibility: buildFriendCompatibility(currentPrimary, user.PrimaryType),
	}
}

func newIncomingFriendRequestResponse(request IncomingFriendRequest) incomingFriendRequestResponse {
	return incomingFriendRequestResponse{
		ID:        request.Friendship.ID,
		Status:    request.Friendship.Status,
		Requester: newFriendUserResponse(request.Requester, request.PrimaryType),
		CreatedAt: request.Friendship.CreatedAt.Format(time.RFC3339Nano),
	}
}

func writeFriendshipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrFriendshipSelf):
		writeJSONError(w, http.StatusBadRequest, "cannot send a friend request to yourself")
	case errors.Is(err, ErrFriendshipExists):
		writeJSONError(w, http.StatusConflict, "friend request or friendship already exists")
	case errors.Is(err, ErrFriendshipForbidden):
		writeJSONError(w, http.StatusForbidden, "friendship action is not allowed")
	case errors.Is(err, ErrBlockedInteraction):
		writeJSONError(w, http.StatusForbidden, "cannot interact with blocked user")
	case errors.Is(err, ErrFriendshipNotPending):
		writeJSONError(w, http.StatusConflict, "friend request is not pending")
	case errors.Is(err, ErrFriendshipNotAccepted):
		writeJSONError(w, http.StatusConflict, "friendship is not accepted")
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "friendship not found")
	default:
		writeJSONError(w, http.StatusInternalServerError, "could not update friendship")
	}
}
