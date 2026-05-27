package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type profileCommentRequest struct {
	Body string `json:"body"`
}

type profileCommentAuthorResponse struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarKey   string `json:"avatarKey"`
}

type profileCommentResponse struct {
	ID        int64                        `json:"id"`
	Author    profileCommentAuthorResponse `json:"author"`
	Body      string                       `json:"body"`
	CreatedAt string                       `json:"createdAt"`
}

func (a *App) handlePublicProfileComments(w http.ResponseWriter, r *http.Request, username string) {
	switch r.Method {
	case http.MethodGet:
		a.handleListPublicProfileComments(w, r, username)
	case http.MethodPost:
		a.handleCreatePublicProfileComment(w, r, username)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *App) handleListPublicProfileComments(w http.ResponseWriter, r *http.Request, username string) {
	profileUser, ok := a.publicCommentProfileUser(w, r, username)
	if !ok {
		return
	}

	comments, err := a.userStore.ListProfileComments(r.Context(), profileUser.ID, defaultProfileCommentLimit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load comments")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comments": newProfileCommentResponses(comments)})
}

func (a *App) handleCreatePublicProfileComment(w http.ResponseWriter, r *http.Request, username string) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	profileUser, ok := a.publicCommentProfileUser(w, r, username)
	if !ok {
		return
	}

	var req profileCommentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	comment, err := a.userStore.CreateProfileComment(r.Context(), profileUser.ID, currentUserID, req.Body)
	if err != nil {
		writeProfileCommentValidationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, newProfileCommentResponse(comment))
}

func (a *App) handleProfileCommentByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}

	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	commentID, ok := profileCommentIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "comment not found")
		return
	}

	if err := a.userStore.DeleteProfileComment(r.Context(), commentID, currentUserID); err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeJSONError(w, http.StatusNotFound, "comment not found")
		case errors.Is(err, ErrProfileCommentForbidden):
			writeJSONError(w, http.StatusForbidden, "comment action is not allowed")
		default:
			writeJSONError(w, http.StatusInternalServerError, "could not delete comment")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) publicCommentProfileUser(w http.ResponseWriter, r *http.Request, username string) (User, bool) {
	user, err := a.userStore.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "profile not found")
			return User{}, false
		}
		writeJSONError(w, http.StatusInternalServerError, "could not load profile")
		return User{}, false
	}
	if normalizedProfileVisibilityOrDefault(user.ProfileVisibility) == profileVisibilityPrivate {
		writeJSONError(w, http.StatusForbidden, "comments are hidden for private profiles")
		return User{}, false
	}
	return user, true
}

func publicProfileCommentsUsernameFromPath(requestPath string) (string, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/users/")
	if suffix == requestPath {
		return "", false
	}
	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "comments" {
		return "", false
	}
	username, err := normalizeUsername(parts[0])
	if err != nil {
		return "", false
	}
	return username, true
}

func profileCommentIDFromPath(requestPath string) (int64, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/profile-comments/")
	if suffix == requestPath || suffix == "" || strings.Contains(strings.Trim(suffix, "/"), "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.Trim(suffix, "/"), 10, 64)
	return id, err == nil && id > 0
}

func newProfileCommentResponses(comments []ProfileComment) []profileCommentResponse {
	out := make([]profileCommentResponse, 0, len(comments))
	for _, comment := range comments {
		out = append(out, newProfileCommentResponse(comment))
	}
	return out
}

func newProfileCommentResponse(comment ProfileComment) profileCommentResponse {
	return profileCommentResponse{
		ID: comment.ID,
		Author: profileCommentAuthorResponse{
			Username:    comment.Author.Username,
			DisplayName: comment.Author.DisplayName,
			AvatarKey:   comment.Author.AvatarKey,
		},
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt.Format(time.RFC3339Nano),
	}
}

func writeProfileCommentValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrProfileCommentBodyRequired):
		writeJSONError(w, http.StatusBadRequest, "comment cannot be empty")
	case errors.Is(err, ErrProfileCommentBodyTooLong):
		writeJSONError(w, http.StatusBadRequest, "comment is too long")
	case errors.Is(err, ErrProfileCommentBodyInvalid):
		writeJSONError(w, http.StatusBadRequest, "comment contains invalid characters")
	case errors.Is(err, ErrBlockedInteraction):
		writeJSONError(w, http.StatusForbidden, "cannot interact with blocked user")
	default:
		writeJSONError(w, http.StatusInternalServerError, "could not save comment")
	}
}
