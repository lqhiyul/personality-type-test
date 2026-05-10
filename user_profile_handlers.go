package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultAvatarKey     = "gradient-violet"
	maxDisplayNameLength = 64
	maxProfileBioLength  = 280
)

var allowedAvatarKeys = map[string]struct{}{
	"gradient-violet": {},
	"gradient-blue":   {},
	"gradient-gold":   {},
	"gradient-green":  {},
	"gradient-red":    {},
	"symbol-analyst":  {},
	"symbol-explorer": {},
	"symbol-guardian": {},
	"symbol-creator":  {},
}

type profileUpdateRequest struct {
	DisplayName string `json:"displayName"`
	Bio         string `json:"bio"`
	AvatarKey   string `json:"avatarKey"`
}

type publicProfileResponse struct {
	Username            string                           `json:"username"`
	DisplayName         string                           `json:"displayName"`
	Bio                 string                           `json:"bio"`
	AvatarKey           string                           `json:"avatarKey"`
	PrimaryType         string                           `json:"primaryType"`
	PrimaryResultDate   string                           `json:"primaryResultDate,omitempty"`
	CompletedTestsCount int                              `json:"completedTestsCount"`
	ViewerFriendship    *publicProfileFriendshipResponse `json:"viewerFriendship,omitempty"`
}

type publicProfileFriendshipResponse struct {
	Status       string `json:"status"`
	FriendshipID int64  `json:"friendshipId,omitempty"`
}

func (a *App) handlePublicUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	username, ok := publicProfileUsernameFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "profile not found")
		return
	}

	user, err := a.userStore.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "profile not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "could not load profile")
		return
	}

	response, err := a.newPublicProfileResponse(r, user)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load profile")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *App) handleMyProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		methodNotAllowed(w, http.MethodPatch)
		return
	}

	userID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	var req profileUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	params, err := validateProfileUpdate(req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := a.userStore.UpdateUserProfile(r.Context(), userID, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "could not update profile")
		return
	}
	writeJSON(w, http.StatusOK, newCurrentUserResponse(user))
}

func (a *App) newPublicProfileResponse(r *http.Request, user User) (publicProfileResponse, error) {
	count, err := a.userStore.CountUserTestResults(r.Context(), user.ID)
	if err != nil {
		return publicProfileResponse{}, err
	}

	response := publicProfileResponse{
		Username:            user.Username,
		DisplayName:         publicDisplayName(user),
		Bio:                 user.Bio,
		AvatarKey:           publicAvatarKey(user.AvatarKey),
		CompletedTestsCount: count,
	}

	viewerFriendship, err := a.newPublicProfileFriendshipResponse(r, user)
	if err != nil {
		return publicProfileResponse{}, err
	}
	response.ViewerFriendship = viewerFriendship

	primary, err := a.userStore.GetPrimaryUserTestResult(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return response, nil
		}
		return publicProfileResponse{}, err
	}
	response.PrimaryType = primary.MBTIType
	response.PrimaryResultDate = primary.CreatedAt.Format(time.RFC3339Nano)
	return response, nil
}

func (a *App) newPublicProfileFriendshipResponse(r *http.Request, user User) (*publicProfileFriendshipResponse, error) {
	viewerID, ok := a.currentUserID(r)
	if !ok || viewerID == user.ID {
		return nil, nil
	}

	friendship, err := a.userStore.GetFriendshipBetween(r.Context(), viewerID, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &publicProfileFriendshipResponse{Status: "none"}, nil
		}
		return nil, err
	}

	response := &publicProfileFriendshipResponse{FriendshipID: friendship.ID}
	switch friendship.Status {
	case FriendshipStatusAccepted:
		response.Status = "friends"
	case FriendshipStatusPending:
		if friendship.RequesterID == viewerID {
			response.Status = "request_sent"
		} else {
			response.Status = "request_received"
		}
	default:
		response.Status = "none"
	}
	return response, nil
}

func publicProfileUsernameFromPath(requestPath string) (string, bool) {
	raw := strings.Trim(strings.TrimPrefix(requestPath, "/api/users/"), "/")
	if raw == "" || raw == requestPath || strings.Contains(raw, "/") {
		return "", false
	}
	username, err := normalizeUsername(raw)
	if err != nil {
		return "", false
	}
	return username, true
}

func validateProfileUpdate(req profileUpdateRequest) (UpdateUserProfileParams, error) {
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return UpdateUserProfileParams{}, errors.New("display name is required")
	}
	if utf8.RuneCountInString(displayName) > maxDisplayNameLength || containsProfileControl(displayName) {
		return UpdateUserProfileParams{}, errors.New("display name must be 1 to 64 characters")
	}

	bio := strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(bio) > maxProfileBioLength || containsProfileControl(bio) {
		return UpdateUserProfileParams{}, errors.New("bio must be 280 characters or fewer")
	}

	avatarKey := strings.TrimSpace(req.AvatarKey)
	if avatarKey == "" {
		avatarKey = defaultAvatarKey
	}
	if !isAllowedAvatarKey(avatarKey) {
		return UpdateUserProfileParams{}, errors.New("avatar preset is not valid")
	}

	return UpdateUserProfileParams{
		DisplayName: displayName,
		Bio:         bio,
		AvatarKey:   avatarKey,
	}, nil
}

func containsProfileControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func publicDisplayName(user User) string {
	if strings.TrimSpace(user.DisplayName) != "" {
		return user.DisplayName
	}
	return user.Username
}

func publicAvatarKey(value string) string {
	key := strings.TrimSpace(value)
	if isAllowedAvatarKey(key) {
		return key
	}
	return defaultAvatarKey
}

func isAllowedAvatarKey(key string) bool {
	_, ok := allowedAvatarKeys[key]
	return ok
}
