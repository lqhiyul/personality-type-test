package main

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
	if profile.CompletedTestsCount != 0 {
		t.Fatalf("expected completed count 0, got %d", profile.CompletedTestsCount)
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
	if profile.CompletedTestsCount != 2 {
		t.Fatalf("expected completed count 2, got %d", profile.CompletedTestsCount)
	}
	if profile.PrimaryType == first.MBTIType {
		t.Fatalf("expected second result to be primary, got first result: %+v", profile)
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
