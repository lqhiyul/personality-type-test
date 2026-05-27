package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestPublicProfileWithoutPrimaryReturnsSafeData(t *testing.T) {
	app := newTestApp(t)
	user, _ := registerUserAndCookie(t, app, "public_user", "public@example.com")
	if _, err := app.userStore.UpdateUserProfile(context.Background(), user.ID, UpdateUserProfileParams{
		DisplayName: "Public User",
		Bio:         "A short public bio.",
		AvatarKey:   "gradient-blue",
	}); err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}

	rec := performJSON(app, http.MethodGet, "/api/users/public_user", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", rec.Code, rec.Body.String())
	}

	body := rec.Body.String()
	for _, private := range []string{"public@example.com", "password_hash", "answers", "answers_json"} {
		if strings.Contains(body, private) {
			t.Fatalf("public profile leaked private field/value %q in %s", private, body)
		}
	}

	var profile publicProfileResponse
	if err := json.Unmarshal([]byte(body), &profile); err != nil {
		t.Fatalf("decode public profile: %v", err)
	}
	if profile.Username != "public_user" || profile.DisplayName != "Public User" || profile.Bio != "A short public bio." {
		t.Fatalf("unexpected public profile: %+v", profile)
	}
	if profile.AvatarKey != "gradient-blue" {
		t.Fatalf("expected avatar gradient-blue, got %q", profile.AvatarKey)
	}
	if profile.PrimaryType != "" || profile.PrimaryResultDate != "" {
		t.Fatalf("expected no primary result, got %+v", profile)
	}
	if profile.CompletedTestsCount == nil || *profile.CompletedTestsCount != 0 {
		t.Fatalf("expected completed count 0, got %+v", profile.CompletedTestsCount)
	}
}

func TestPublicProfileWithPrimaryResultAndCount(t *testing.T) {
	app := newTestApp(t)
	user, _ := registerUserAndCookie(t, app, "primary_public", "primary-public@example.com")
	first := createUserResultForTest(t, app, user.ID, "INTJ", 60)
	second := createUserResultForTest(t, app, user.ID, "ENFP", 90)
	if _, err := app.userStore.SetPrimaryUserTestResult(context.Background(), user.ID, second.ID); err != nil {
		t.Fatalf("SetPrimaryUserTestResult() error = %v", err)
	}

	rec := performJSON(app, http.MethodGet, "/api/users/primary_public", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var profile publicProfileResponse
	if err := json.NewDecoder(rec.Body).Decode(&profile); err != nil {
		t.Fatalf("decode public profile: %v", err)
	}
	if profile.PrimaryType != "ENFP" || profile.PrimaryResultDate == "" {
		t.Fatalf("expected ENFP primary result, got %+v", profile)
	}
	if profile.CompletedTestsCount == nil || *profile.CompletedTestsCount != 2 {
		t.Fatalf("expected completed count 2, got %+v", profile.CompletedTestsCount)
	}
	if profile.PrimaryType == first.MBTIType {
		t.Fatalf("expected second result to be primary, got first result: %+v", profile)
	}
}

func TestPublicProfileRespectsPrivacySettings(t *testing.T) {
	app := newTestApp(t)
	user, cookie := registerUserAndCookie(t, app, "private_public", "private-public@example.com")
	result := createUserResultForTest(t, app, user.ID, "INTJ", 60)
	if _, err := app.userStore.SetPrimaryUserTestResult(context.Background(), user.ID, result.ID); err != nil {
		t.Fatalf("SetPrimaryUserTestResult() error = %v", err)
	}

	rec := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]any{
		"displayName":        "Private Public",
		"bio":                "Private bio should stay hidden.",
		"avatarKey":          "gradient-gold",
		"profileVisibility":  "private",
		"showPrimaryResult":  true,
		"showCompletedCount": true,
		"showFriends":        true,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected privacy update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	public := performJSON(app, http.MethodGet, "/api/users/private_public", nil)
	if public.Code != http.StatusOK {
		t.Fatalf("expected private profile response 200, got %d: %s", public.Code, public.Body.String())
	}
	body := public.Body.String()
	for _, hidden := range []string{"Private bio should stay hidden.", "INTJ", "completedTestsCount", "primaryType", "friends", "private-public@example.com", "answers_json"} {
		if strings.Contains(body, hidden) {
			t.Fatalf("private profile leaked %q in %s", hidden, body)
		}
	}
	var profile publicProfileResponse
	if err := json.Unmarshal([]byte(body), &profile); err != nil {
		t.Fatalf("decode private profile: %v", err)
	}
	if !profile.IsPrivate || profile.ProfileVisibility != profileVisibilityPrivate {
		t.Fatalf("expected private profile marker, got %+v", profile)
	}
	if profile.DisplayName != "Private Public" || profile.AvatarKey != "gradient-gold" {
		t.Fatalf("expected safe public identity on private profile, got %+v", profile)
	}
}

func TestPublicProfileCanHidePrimaryResultAndCompletedCount(t *testing.T) {
	app := newTestApp(t)
	user, cookie := registerUserAndCookie(t, app, "hidden_bits", "hidden-bits@example.com")
	result := createUserResultForTest(t, app, user.ID, "ENFP", 60)
	if _, err := app.userStore.SetPrimaryUserTestResult(context.Background(), user.ID, result.ID); err != nil {
		t.Fatalf("SetPrimaryUserTestResult() error = %v", err)
	}

	rec := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]any{
		"profileVisibility":  "public",
		"showPrimaryResult":  false,
		"showCompletedCount": false,
		"showFriends":        true,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected privacy update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	public := performJSON(app, http.MethodGet, "/api/users/hidden_bits", nil)
	if public.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", public.Code, public.Body.String())
	}
	body := public.Body.String()
	for _, hidden := range []string{"ENFP", "primaryType", "primaryResultDate", "completedTestsCount", "hidden-bits@example.com", "answers_json"} {
		if strings.Contains(body, hidden) {
			t.Fatalf("public privacy setting leaked %q in %s", hidden, body)
		}
	}
	var profile publicProfileResponse
	if err := json.Unmarshal([]byte(body), &profile); err != nil {
		t.Fatalf("decode public profile: %v", err)
	}
	if profile.ShowPrimaryResult || profile.ShowCompletedCount {
		t.Fatalf("expected hidden primary and completed flags, got %+v", profile)
	}
}

func TestPublicProfileFriendsRespectShowFriends(t *testing.T) {
	app := newTestApp(t)
	owner, ownerCookie := registerUserAndCookie(t, app, "friend_owner", "friend-owner@example.com")
	_, friendCookie := registerUserAndCookie(t, app, "friend_public", "friend-public@example.com")

	request := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": owner.Username}, friendCookie)
	if request.Code != http.StatusCreated {
		t.Fatalf("expected friend request 201, got %d: %s", request.Code, request.Body.String())
	}
	var friendship friendshipResponse
	if err := json.NewDecoder(request.Body).Decode(&friendship); err != nil {
		t.Fatalf("decode friendship: %v", err)
	}
	accept := performJSON(app, http.MethodPost, "/api/friends/requests/"+strconvID(friendship.ID)+"/accept", nil, ownerCookie)
	if accept.Code != http.StatusOK {
		t.Fatalf("expected accept 200, got %d: %s", accept.Code, accept.Body.String())
	}

	public := performJSON(app, http.MethodGet, "/api/users/friend_owner", nil)
	if public.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", public.Code, public.Body.String())
	}
	if !strings.Contains(public.Body.String(), "friend_public") {
		t.Fatalf("expected public friends list to include accepted public friend, got %s", public.Body.String())
	}

	rec := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]any{
		"profileVisibility":  "public",
		"showPrimaryResult":  true,
		"showCompletedCount": true,
		"showFriends":        false,
	}, ownerCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected showFriends update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	hidden := performJSON(app, http.MethodGet, "/api/users/friend_owner", nil)
	if hidden.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", hidden.Code, hidden.Body.String())
	}
	if strings.Contains(hidden.Body.String(), "friend_public") || strings.Contains(hidden.Body.String(), `"friends"`) {
		t.Fatalf("expected hidden public friends list, got %s", hidden.Body.String())
	}
}

func TestPublicProfileMissingUserReturnsNotFound(t *testing.T) {
	app := newTestApp(t)
	rec := performJSON(app, http.MethodGet, "/api/users/missing_user", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing public profile 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProfileUpdateRequiresLoginAndValidatesInput(t *testing.T) {
	app := newTestApp(t)
	_, cookie := registerUserAndCookie(t, app, "editable_user", "editable@example.com")

	loggedOut := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]string{
		"displayName": "Editable",
		"bio":         "Public bio",
		"avatarKey":   "gradient-green",
	})
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out profile update 401, got %d", loggedOut.Code)
	}

	cases := []struct {
		name string
		body any
	}{
		{
			name: "empty display name",
			body: map[string]string{"displayName": " ", "bio": "", "avatarKey": "gradient-green"},
		},
		{
			name: "too long display name",
			body: map[string]string{"displayName": strings.Repeat("a", maxDisplayNameLength+1), "bio": "", "avatarKey": "gradient-green"},
		},
		{
			name: "too long bio",
			body: map[string]string{"displayName": "Editable", "bio": strings.Repeat("b", maxProfileBioLength+1), "avatarKey": "gradient-green"},
		},
		{
			name: "invalid avatar",
			body: map[string]string{"displayName": "Editable", "bio": "", "avatarKey": "custom-upload"},
		},
		{
			name: "invalid profile visibility",
			body: map[string]any{"profileVisibility": "friends-only"},
		},
		{
			name: "unknown identity fields",
			body: map[string]string{"displayName": "Editable", "bio": "", "avatarKey": "gradient-green", "email": "takeover@example.com"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := performJSON(app, http.MethodPatch, "/api/me/profile", tc.body, cookie)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected bad request, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestLoggedInUserCanUpdateOwnPublicProfile(t *testing.T) {
	app := newTestApp(t)
	user, cookie := registerUserAndCookie(t, app, "own_profile", "own-profile@example.com")

	rec := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]string{
		"displayName": "Own Profile",
		"bio":         "This is safe public text.",
		"avatarKey":   "symbol-creator",
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected profile update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var current currentUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&current); err != nil {
		t.Fatalf("decode current user: %v", err)
	}
	if current.Username != user.Username || current.Email != user.Email {
		t.Fatalf("profile update changed identity fields: %+v", current)
	}
	if current.DisplayName != "Own Profile" || current.Bio != "This is safe public text." || current.AvatarKey != "symbol-creator" {
		t.Fatalf("unexpected updated current user: %+v", current)
	}

	public := performJSON(app, http.MethodGet, "/api/users/own_profile", nil)
	if public.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", public.Code, public.Body.String())
	}
	if strings.Contains(public.Body.String(), user.Email) {
		t.Fatalf("public profile leaked email: %s", public.Body.String())
	}
	var profile publicProfileResponse
	if err := json.NewDecoder(public.Body).Decode(&profile); err != nil {
		t.Fatalf("decode public profile: %v", err)
	}
	if profile.DisplayName != "Own Profile" || profile.Bio != "This is safe public text." || profile.AvatarKey != "symbol-creator" {
		t.Fatalf("unexpected public profile after update: %+v", profile)
	}
}

func TestLoggedInUserCanUpdateProfilePrivacySettings(t *testing.T) {
	app := newTestApp(t)
	user, cookie := registerUserAndCookie(t, app, "privacy_owner", "privacy-owner@example.com")

	rec := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]any{
		"profileVisibility":  "private",
		"showPrimaryResult":  false,
		"showCompletedCount": false,
		"showFriends":        false,
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected profile privacy update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var current currentUserResponse
	if err := json.NewDecoder(rec.Body).Decode(&current); err != nil {
		t.Fatalf("decode current user: %v", err)
	}
	if current.Username != user.Username || current.Email != user.Email {
		t.Fatalf("privacy update changed identity fields: %+v", current)
	}
	if current.ProfileVisibility != profileVisibilityPrivate || current.ShowPrimaryResult || current.ShowCompletedCount || current.ShowFriends {
		t.Fatalf("unexpected privacy settings in current user: %+v", current)
	}

	after, err := app.userStore.GetUserByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if after.Username != user.Username || after.Email != user.Email {
		t.Fatalf("privacy update changed stored identity fields: %+v", after)
	}
}

func TestProfileUpdateDoesNotAffectAnotherUser(t *testing.T) {
	app := newTestApp(t)
	first, _ := registerUserAndCookie(t, app, "first_profile", "first-profile@example.com")
	_, secondCookie := registerUserAndCookie(t, app, "second_profile", "second-profile@example.com")

	rec := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]string{
		"displayName": "Second Profile",
		"bio":         "Only second user changes.",
		"avatarKey":   "gradient-red",
	}, secondCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected second profile update 200, got %d: %s", rec.Code, rec.Body.String())
	}

	firstAfter, err := app.userStore.GetUserByID(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("GetUserByID(first) error = %v", err)
	}
	if firstAfter.DisplayName != first.DisplayName || firstAfter.Email != first.Email || firstAfter.Username != first.Username {
		t.Fatalf("second user's profile update affected first user: before=%+v after=%+v", first, firstAfter)
	}
}
