package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestFriendRequestAPIRequiresAuthAndValidatesCreate(t *testing.T) {
	app := newTestApp(t)
	alice, aliceCookie := registerUserAndCookie(t, app, "friend_alice", "friend-alice@example.com")
	_, bobCookie := registerUserAndCookie(t, app, "friend_bob", "friend-bob@example.com")

	loggedOut := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": "friend_bob"})
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out friend request 401, got %d", loggedOut.Code)
	}

	missingUsername := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": ""}, aliceCookie)
	if missingUsername.Code != http.StatusBadRequest {
		t.Fatalf("expected missing username 400, got %d: %s", missingUsername.Code, missingUsername.Body.String())
	}

	unknown := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": "missing_user"}, aliceCookie)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("expected unknown target 404, got %d: %s", unknown.Code, unknown.Body.String())
	}

	self := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": alice.Username}, aliceCookie)
	if self.Code != http.StatusBadRequest {
		t.Fatalf("expected self request 400, got %d: %s", self.Code, self.Body.String())
	}

	created := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": "friend_bob"}, aliceCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected friend request 201, got %d: %s", created.Code, created.Body.String())
	}

	duplicate := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": "friend_bob"}, aliceCookie)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("expected duplicate friend request 409, got %d: %s", duplicate.Code, duplicate.Body.String())
	}

	reverseDuplicate := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": "friend_alice"}, bobCookie)
	if reverseDuplicate.Code != http.StatusConflict {
		t.Fatalf("expected reverse duplicate friend request 409, got %d: %s", reverseDuplicate.Code, reverseDuplicate.Body.String())
	}
}

func TestFriendAPIAcceptListCompatibilityRemoveAndPrivacy(t *testing.T) {
	app := newTestApp(t)
	alice, aliceCookie := registerUserAndCookie(t, app, "compat_alice", "compat-alice@example.com")
	bob, bobCookie := registerUserAndCookie(t, app, "compat_bob", "compat-bob@example.com")
	_, carolCookie := registerUserAndCookie(t, app, "compat_carol", "compat-carol@example.com")

	setPrimaryResultForFriendTest(t, app, alice.ID, "INTJ")
	setPrimaryResultForFriendTest(t, app, bob.ID, "ENFJ")

	created := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": bob.Username}, aliceCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected friend request 201, got %d: %s", created.Code, created.Body.String())
	}
	var createdFriendship friendshipResponse
	if err := json.NewDecoder(created.Body).Decode(&createdFriendship); err != nil {
		t.Fatalf("decode created friendship: %v", err)
	}

	incoming := performJSON(app, http.MethodGet, "/api/friends/requests", nil, bobCookie)
	if incoming.Code != http.StatusOK {
		t.Fatalf("expected incoming requests 200, got %d: %s", incoming.Code, incoming.Body.String())
	}
	assertNoPrivateFriendData(t, incoming.Body.String())
	var incomingResponse struct {
		Requests []incomingFriendRequestResponse `json:"requests"`
	}
	if err := json.NewDecoder(incoming.Body).Decode(&incomingResponse); err != nil {
		t.Fatalf("decode incoming requests: %v", err)
	}
	if len(incomingResponse.Requests) != 1 || incomingResponse.Requests[0].ID != createdFriendship.ID || incomingResponse.Requests[0].Requester.Username != alice.Username {
		t.Fatalf("unexpected incoming requests: %+v", incomingResponse.Requests)
	}

	wrongUserAccept := performJSON(app, http.MethodPost, "/api/friends/requests/"+strconvID(createdFriendship.ID)+"/accept", nil, carolCookie)
	if wrongUserAccept.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated accept 403, got %d: %s", wrongUserAccept.Code, wrongUserAccept.Body.String())
	}

	accepted := performJSON(app, http.MethodPost, "/api/friends/requests/"+strconvID(createdFriendship.ID)+"/accept", nil, bobCookie)
	if accepted.Code != http.StatusOK {
		t.Fatalf("expected accept 200, got %d: %s", accepted.Code, accepted.Body.String())
	}

	aliceFriends := performJSON(app, http.MethodGet, "/api/friends", nil, aliceCookie)
	if aliceFriends.Code != http.StatusOK {
		t.Fatalf("expected Alice friends 200, got %d: %s", aliceFriends.Code, aliceFriends.Body.String())
	}
	assertNoPrivateFriendData(t, aliceFriends.Body.String())
	var aliceFriendsResponse struct {
		Friends []friendListItemResponse `json:"friends"`
	}
	if err := json.NewDecoder(aliceFriends.Body).Decode(&aliceFriendsResponse); err != nil {
		t.Fatalf("decode Alice friends: %v", err)
	}
	if len(aliceFriendsResponse.Friends) != 1 {
		t.Fatalf("expected one Alice friend, got %+v", aliceFriendsResponse.Friends)
	}
	friend := aliceFriendsResponse.Friends[0]
	if friend.Username != bob.Username || friend.PrimaryType != "ENFJ" || friend.FriendshipID != createdFriendship.ID {
		t.Fatalf("unexpected Alice friend response: %+v", friend)
	}
	if !friend.Compatibility.Available || friend.Compatibility.Friendship == 0 || friend.Compatibility.Relationship == 0 || friend.Compatibility.Work == 0 {
		t.Fatalf("expected available compatibility scores, got %+v", friend.Compatibility)
	}

	bobFriends := performJSON(app, http.MethodGet, "/api/friends", nil, bobCookie)
	if bobFriends.Code != http.StatusOK {
		t.Fatalf("expected Bob friends 200, got %d: %s", bobFriends.Code, bobFriends.Body.String())
	}
	var bobFriendsResponse struct {
		Friends []friendListItemResponse `json:"friends"`
	}
	if err := json.NewDecoder(bobFriends.Body).Decode(&bobFriendsResponse); err != nil {
		t.Fatalf("decode Bob friends: %v", err)
	}
	if len(bobFriendsResponse.Friends) != 1 || bobFriendsResponse.Friends[0].Username != alice.Username {
		t.Fatalf("expected accepted friend in both lists, got %+v", bobFriendsResponse.Friends)
	}

	publicProfile := performJSON(app, http.MethodGet, "/api/users/"+bob.Username, nil, aliceCookie)
	if publicProfile.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", publicProfile.Code, publicProfile.Body.String())
	}
	var profile publicProfileResponse
	if err := json.NewDecoder(publicProfile.Body).Decode(&profile); err != nil {
		t.Fatalf("decode public profile: %v", err)
	}
	if profile.ViewerFriendship == nil || profile.ViewerFriendship.Status != "friends" || profile.ViewerFriendship.FriendshipID != createdFriendship.ID {
		t.Fatalf("expected friends relationship state on public profile, got %+v", profile.ViewerFriendship)
	}

	wrongUserRemove := performJSON(app, http.MethodDelete, "/api/friends/"+strconvID(createdFriendship.ID), nil, carolCookie)
	if wrongUserRemove.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated remove 403, got %d: %s", wrongUserRemove.Code, wrongUserRemove.Body.String())
	}

	removed := performJSON(app, http.MethodDelete, "/api/friends/"+strconvID(createdFriendship.ID), nil, aliceCookie)
	if removed.Code != http.StatusOK {
		t.Fatalf("expected remove 200, got %d: %s", removed.Code, removed.Body.String())
	}
	afterRemove := performJSON(app, http.MethodGet, "/api/friends", nil, bobCookie)
	var afterRemoveResponse struct {
		Friends []friendListItemResponse `json:"friends"`
	}
	if err := json.NewDecoder(afterRemove.Body).Decode(&afterRemoveResponse); err != nil {
		t.Fatalf("decode friends after remove: %v", err)
	}
	if len(afterRemoveResponse.Friends) != 0 {
		t.Fatalf("expected no friends after remove, got %+v", afterRemoveResponse.Friends)
	}
}

func TestFriendAPICompatibilityUnavailableWithoutPrimary(t *testing.T) {
	app := newTestApp(t)
	alice, aliceCookie := registerUserAndCookie(t, app, "unavailable_alice", "unavailable-alice@example.com")
	bob, bobCookie := registerUserAndCookie(t, app, "unavailable_bob", "unavailable-bob@example.com")

	setPrimaryResultForFriendTest(t, app, alice.ID, "INTJ")
	created := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": bob.Username}, aliceCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected friend request 201, got %d: %s", created.Code, created.Body.String())
	}
	var createdFriendship friendshipResponse
	if err := json.NewDecoder(created.Body).Decode(&createdFriendship); err != nil {
		t.Fatalf("decode created friendship: %v", err)
	}
	accepted := performJSON(app, http.MethodPost, "/api/friends/requests/"+strconvID(createdFriendship.ID)+"/accept", nil, bobCookie)
	if accepted.Code != http.StatusOK {
		t.Fatalf("expected accept 200, got %d: %s", accepted.Code, accepted.Body.String())
	}

	friends := performJSON(app, http.MethodGet, "/api/friends", nil, aliceCookie)
	if friends.Code != http.StatusOK {
		t.Fatalf("expected friends 200, got %d: %s", friends.Code, friends.Body.String())
	}
	var response struct {
		Friends []friendListItemResponse `json:"friends"`
	}
	if err := json.NewDecoder(friends.Body).Decode(&response); err != nil {
		t.Fatalf("decode friends: %v", err)
	}
	if len(response.Friends) != 1 {
		t.Fatalf("expected one friend, got %+v", response.Friends)
	}
	if response.Friends[0].Compatibility.Available || response.Friends[0].Compatibility.Reason == "" {
		t.Fatalf("expected unavailable compatibility with reason, got %+v", response.Friends[0].Compatibility)
	}
}

func TestFriendsRegressionCoreAccountProfileResultsAndAdminSeparation(t *testing.T) {
	app := newTestApp(t)
	_, cookie := registerUserAndCookie(t, app, "regression_user", "regression-user@example.com")

	me := performJSON(app, http.MethodGet, "/api/auth/me", nil, cookie)
	if me.Code != http.StatusOK {
		t.Fatalf("expected /api/auth/me 200, got %d: %s", me.Code, me.Body.String())
	}

	submit := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Regression User",
		"answers":  answersForType("INFJ"),
		"duration": 30,
	}, cookie)
	if submit.Code != http.StatusOK {
		t.Fatalf("expected logged-in submit 200, got %d: %s", submit.Code, submit.Body.String())
	}

	myResults := performJSON(app, http.MethodGet, "/api/me/results", nil, cookie)
	if myResults.Code != http.StatusOK || !strings.Contains(myResults.Body.String(), "INFJ") {
		t.Fatalf("expected saved results to work, got %d: %s", myResults.Code, myResults.Body.String())
	}

	publicProfile := performJSON(app, http.MethodGet, "/api/users/regression_user", nil)
	if publicProfile.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", publicProfile.Code, publicProfile.Body.String())
	}

	adminWithUserCookie := performJSON(app, http.MethodGet, "/api/results", nil, cookie)
	if adminWithUserCookie.Code != http.StatusUnauthorized {
		t.Fatalf("regular user cookie should not authorize admin results, got %d", adminWithUserCookie.Code)
	}
	adminResults := performJSON(app, http.MethodGet, "/api/results", nil, login(t, app)...)
	if adminResults.Code != http.StatusOK {
		t.Fatalf("expected admin session to still load results, got %d: %s", adminResults.Code, adminResults.Body.String())
	}
}

func setPrimaryResultForFriendTest(t *testing.T, app *App, userID int64, mbtiType string) UserTestResult {
	t.Helper()
	result := createUserResultForTest(t, app, userID, mbtiType, 45)
	primary, err := app.userStore.SetPrimaryUserTestResult(context.Background(), userID, result.ID)
	if err != nil {
		t.Fatalf("SetPrimaryUserTestResult(%s) error = %v", mbtiType, err)
	}
	return primary
}

func assertNoPrivateFriendData(t *testing.T, body string) {
	t.Helper()
	for _, private := range []string{"@example.com", "password", "password_hash", "answers", "answers_json", "scores_json"} {
		if strings.Contains(body, private) {
			t.Fatalf("friend API leaked private field/value %q in %s", private, body)
		}
	}
}
