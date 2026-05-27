package app

import (
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/lqhiyul/personality-type-test/internal/sessions"
)

const (
	minUsernameLength = 3
	maxUsernameLength = 32
	minPasswordLength = 8
)

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userLoginRequest struct {
	EmailOrUsername string `json:"emailOrUsername"`
	Password        string `json:"password"`
}

type authUserResponse struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	DisplayName        string `json:"displayName"`
	Bio                string `json:"bio"`
	AvatarKey          string `json:"avatarKey"`
	ProfileVisibility  string `json:"profileVisibility"`
	ShowPrimaryResult  bool   `json:"showPrimaryResult"`
	ShowCompletedCount bool   `json:"showCompletedCount"`
	ShowFriends        bool   `json:"showFriends"`
}

type currentUserResponse struct {
	ID                 int64  `json:"id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	DisplayName        string `json:"displayName"`
	Bio                string `json:"bio"`
	AvatarKey          string `json:"avatarKey"`
	ProfileVisibility  string `json:"profileVisibility"`
	ShowPrimaryResult  bool   `json:"showPrimaryResult"`
	ShowCompletedCount bool   `json:"showCompletedCount"`
	ShowFriends        bool   `json:"showFriends"`
	CreatedAt          string `json:"createdAt"`
}

func (a *App) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()
	if _, err := a.userStore.GetUserByUsername(ctx, username); err == nil {
		writeJSONError(w, http.StatusConflict, "username or email is already registered")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusInternalServerError, "could not check username")
		return
	}
	if _, err := a.userStore.GetUserByEmail(ctx, email); err == nil {
		writeJSONError(w, http.StatusConflict, "username or email is already registered")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusInternalServerError, "could not check email")
		return
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "password is not valid")
		return
	}

	user, err := a.userStore.CreateUser(ctx, CreateUserParams{
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  username,
	})
	if err != nil {
		writeJSONError(w, http.StatusConflict, "user could not be created")
		return
	}

	if !a.setUserSession(w, r, user.ID) {
		return
	}
	writeJSON(w, http.StatusCreated, newAuthUserResponse(user))
}

func (a *App) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}

	var req userLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	limitKey := a.loginRateLimitKey(r)
	if retryAfter, ok := a.userLoginLimiter.allow(limitKey); !ok {
		writeRateLimitError(w, retryAfter)
		return
	}

	record, err := a.findUserForLogin(r, req.EmailOrUsername)
	if err != nil || !CheckPasswordHash(req.Password, record.PasswordHash) {
		if retryAfter, limited := a.userLoginLimiter.recordFailure(limitKey); limited {
			writeRateLimitError(w, retryAfter)
			return
		}
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	a.userLoginLimiter.recordSuccess(limitKey)
	if !a.setUserSession(w, r, record.ID) {
		return
	}
	writeJSON(w, http.StatusOK, newAuthUserResponse(record.User))
}

func (a *App) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	if cookie, err := r.Cookie(userSessionCookieName); err == nil && cookie.Value != "" {
		if err := a.sessionStore.Revoke(r.Context(), sessions.KindUser, cookie.Value); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not log out")
			return
		}
	}
	clearUserSessionCookie(w, a.cookieSecure)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}

	userID, ok := a.currentUserID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := a.userStore.GetUserByID(r.Context(), userID)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, newCurrentUserResponse(user))
}

func (a *App) findUserForLogin(r *http.Request, rawLogin string) (userAuthRecord, error) {
	login := strings.TrimSpace(rawLogin)
	if strings.Contains(login, "@") {
		email, err := normalizeEmail(login)
		if err != nil {
			return userAuthRecord{}, err
		}
		return a.userStore.getUserAuthByEmail(r.Context(), email)
	}

	username, err := normalizeUsername(login)
	if err != nil {
		return userAuthRecord{}, err
	}
	return a.userStore.getUserAuthByUsername(r.Context(), username)
}

func (a *App) setUserSession(w http.ResponseWriter, r *http.Request, userID int64) bool {
	token, expiresAt, err := a.sessionStore.CreateUser(r.Context(), userID, userSessionTTL)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create session")
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(userSessionTTL.Seconds()),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	return true
}

func (a *App) currentUserID(r *http.Request) (int64, bool) {
	cookie, err := r.Cookie(userSessionCookieName)
	if err != nil || cookie.Value == "" {
		return 0, false
	}
	userID, ok, err := a.sessionStore.UserID(r.Context(), cookie.Value)
	if err != nil {
		return 0, false
	}
	return userID, ok
}

func clearUserSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func normalizeUsername(value string) (string, error) {
	username := strings.ToLower(strings.TrimSpace(value))
	if utf8.RuneCountInString(username) < minUsernameLength || utf8.RuneCountInString(username) > maxUsernameLength {
		return "", errors.New("username must be 3 to 32 characters")
	}
	for _, r := range username {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", errors.New("username can use only letters, numbers, underscore, and hyphen")
	}
	return username, nil
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" || len(email) > 254 {
		return "", errors.New("email is required")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email, ".") {
		return "", errors.New("email is not valid")
	}
	return email, nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLength {
		return errors.New("password must be at least 8 characters")
	}
	if len([]byte(password)) > 72 {
		return errPasswordTooLong
	}
	return nil
}

func newAuthUserResponse(user User) authUserResponse {
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}
	return authUserResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Email:              user.Email,
		DisplayName:        displayName,
		Bio:                user.Bio,
		AvatarKey:          user.AvatarKey,
		ProfileVisibility:  normalizedProfileVisibilityOrDefault(user.ProfileVisibility),
		ShowPrimaryResult:  user.ShowPrimaryResult,
		ShowCompletedCount: user.ShowCompletedCount,
		ShowFriends:        user.ShowFriends,
	}
}

func newCurrentUserResponse(user User) currentUserResponse {
	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}
	return currentUserResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Email:              user.Email,
		DisplayName:        displayName,
		Bio:                user.Bio,
		AvatarKey:          user.AvatarKey,
		ProfileVisibility:  normalizedProfileVisibilityOrDefault(user.ProfileVisibility),
		ShowPrimaryResult:  user.ShowPrimaryResult,
		ShowCompletedCount: user.ShowCompletedCount,
		ShowFriends:        user.ShowFriends,
		CreatedAt:          user.CreatedAt.Format(time.RFC3339Nano),
	}
}
