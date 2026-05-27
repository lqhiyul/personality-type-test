package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestUserStoreProfileCommentsLifecycleAndCascades(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, store := newTestUserStore(t)
	owner := createStoreUserForFriendTest(t, store, "comment_owner")
	author := createStoreUserForFriendTest(t, store, "comment_author")
	stranger := createStoreUserForFriendTest(t, store, "comment_stranger")

	fixedNow := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return fixedNow }
	comment, err := store.CreateProfileComment(ctx, owner.ID, author.ID, "  Hello profile  ")
	if err != nil {
		t.Fatalf("CreateProfileComment() error = %v", err)
	}
	if comment.Body != "Hello profile" || comment.ProfileUserID != owner.ID || comment.AuthorUserID != author.ID {
		t.Fatalf("unexpected created comment: %+v", comment)
	}
	if comment.Author.Username != author.Username || comment.Author.DisplayName != author.DisplayName || comment.Author.AvatarKey != defaultAvatarKey {
		t.Fatalf("unexpected safe author info: %+v", comment.Author)
	}
	if !comment.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected fixed created_at, got %s", comment.CreatedAt)
	}

	list, err := store.ListProfileComments(ctx, owner.ID, 50)
	if err != nil {
		t.Fatalf("ListProfileComments() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != comment.ID {
		t.Fatalf("expected one listed comment, got %+v", list)
	}

	if _, err := store.GetProfileCommentByID(ctx, comment.ID); err != nil {
		t.Fatalf("GetProfileCommentByID() error = %v", err)
	}
	if err := store.DeleteProfileComment(ctx, comment.ID, stranger.ID); !errors.Is(err, ErrProfileCommentForbidden) {
		t.Fatalf("expected unrelated delete to fail with ErrProfileCommentForbidden, got %v", err)
	}
	if _, err := store.GetProfileCommentByID(ctx, comment.ID); err != nil {
		t.Fatalf("comment should remain after forbidden delete: %v", err)
	}
	if err := store.DeleteProfileComment(ctx, comment.ID, author.ID); err != nil {
		t.Fatalf("author DeleteProfileComment() error = %v", err)
	}
	if _, err := store.GetProfileCommentByID(ctx, comment.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected author-deleted comment to be gone, got %v", err)
	}

	second, err := store.CreateProfileComment(ctx, owner.ID, author.ID, "Owner can moderate this")
	if err != nil {
		t.Fatalf("CreateProfileComment(second) error = %v", err)
	}
	if err := store.DeleteProfileComment(ctx, second.ID, owner.ID); err != nil {
		t.Fatalf("profile owner DeleteProfileComment() error = %v", err)
	}

	authorCascade, err := store.CreateProfileComment(ctx, owner.ID, author.ID, "Cascade by author")
	if err != nil {
		t.Fatalf("CreateProfileComment(author cascade) error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, author.ID); err != nil {
		t.Fatalf("delete author user: %v", err)
	}
	if _, err := store.GetProfileCommentByID(ctx, authorCascade.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected author cascade to remove comment, got %v", err)
	}

	ownerCascadeAuthor := createStoreUserForFriendTest(t, store, "comment_author_two")
	ownerCascade, err := store.CreateProfileComment(ctx, owner.ID, ownerCascadeAuthor.ID, "Cascade by owner")
	if err != nil {
		t.Fatalf("CreateProfileComment(owner cascade) error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, owner.ID); err != nil {
		t.Fatalf("delete profile owner user: %v", err)
	}
	if _, err := store.GetProfileCommentByID(ctx, ownerCascade.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected profile owner cascade to remove comment, got %v", err)
	}
}

func TestProfileCommentsAPIAuthValidationPrivacyAndSafety(t *testing.T) {
	app := newTestApp(t)
	owner, ownerCookie := registerUserAndCookie(t, app, "profile_comment_owner", "profile-comment-owner@example.com")
	author, authorCookie := registerUserAndCookie(t, app, "profile_comment_author", "profile-comment-author@example.com")

	if _, err := app.userStore.UpdateUserProfile(context.Background(), author.ID, UpdateUserProfileParams{
		DisplayName: "Comment Author",
		AvatarKey:   "symbol-analyst",
	}); err != nil {
		t.Fatalf("UpdateUserProfile(author) error = %v", err)
	}

	emptyList := performJSON(app, http.MethodGet, "/api/users/"+owner.Username+"/comments", nil)
	if emptyList.Code != http.StatusOK {
		t.Fatalf("expected empty comments 200, got %d: %s", emptyList.Code, emptyList.Body.String())
	}
	var emptyResponse struct {
		Comments []profileCommentResponse `json:"comments"`
	}
	if err := json.NewDecoder(emptyList.Body).Decode(&emptyResponse); err != nil {
		t.Fatalf("decode empty comments: %v", err)
	}
	if len(emptyResponse.Comments) != 0 {
		t.Fatalf("expected clean empty comments list, got %+v", emptyResponse.Comments)
	}

	loggedOutPost := performJSON(app, http.MethodPost, "/api/users/"+owner.Username+"/comments", map[string]string{"body": "hello"})
	if loggedOutPost.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out post 401, got %d", loggedOutPost.Code)
	}

	emptyPost := performJSON(app, http.MethodPost, "/api/users/"+owner.Username+"/comments", map[string]string{"body": "   "}, authorCookie)
	if emptyPost.Code != http.StatusBadRequest {
		t.Fatalf("expected empty comment 400, got %d: %s", emptyPost.Code, emptyPost.Body.String())
	}
	longPost := performJSON(app, http.MethodPost, "/api/users/"+owner.Username+"/comments", map[string]string{"body": strings.Repeat("a", maxProfileCommentBodyLength+1)}, authorCookie)
	if longPost.Code != http.StatusBadRequest {
		t.Fatalf("expected long comment 400, got %d: %s", longPost.Code, longPost.Body.String())
	}

	created := performJSON(app, http.MethodPost, "/api/users/"+owner.Username+"/comments", map[string]string{
		"body": "  <strong>Hello</strong>\nplain text only  ",
	}, authorCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected create comment 201, got %d: %s", created.Code, created.Body.String())
	}
	assertNoPrivateCommentData(t, created.Body.String())
	var createdComment profileCommentResponse
	if err := json.NewDecoder(created.Body).Decode(&createdComment); err != nil {
		t.Fatalf("decode created comment: %v", err)
	}
	if createdComment.Body != "<strong>Hello</strong>\nplain text only" {
		t.Fatalf("expected trimmed plain text body, got %q", createdComment.Body)
	}
	if createdComment.Author.Username != author.Username || createdComment.Author.DisplayName != "Comment Author" || createdComment.Author.AvatarKey != "symbol-analyst" {
		t.Fatalf("unexpected safe author response: %+v", createdComment.Author)
	}

	list := performJSON(app, http.MethodGet, "/api/users/"+owner.Username+"/comments", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("expected comments list 200, got %d: %s", list.Code, list.Body.String())
	}
	assertNoPrivateCommentData(t, list.Body.String())
	var listResponse struct {
		Comments []profileCommentResponse `json:"comments"`
	}
	if err := json.NewDecoder(list.Body).Decode(&listResponse); err != nil {
		t.Fatalf("decode comments list: %v", err)
	}
	if len(listResponse.Comments) != 1 || listResponse.Comments[0].ID != createdComment.ID || listResponse.Comments[0].Body != createdComment.Body {
		t.Fatalf("unexpected comments list: %+v", listResponse.Comments)
	}

	privateUpdate := performJSON(app, http.MethodPatch, "/api/me/profile", map[string]any{
		"profileVisibility":  "private",
		"showPrimaryResult":  true,
		"showCompletedCount": true,
		"showFriends":        true,
	}, ownerCookie)
	if privateUpdate.Code != http.StatusOK {
		t.Fatalf("expected private update 200, got %d: %s", privateUpdate.Code, privateUpdate.Body.String())
	}
	privateList := performJSON(app, http.MethodGet, "/api/users/"+owner.Username+"/comments", nil)
	if privateList.Code != http.StatusForbidden {
		t.Fatalf("expected private comments list 403, got %d: %s", privateList.Code, privateList.Body.String())
	}
	privatePost := performJSON(app, http.MethodPost, "/api/users/"+owner.Username+"/comments", map[string]string{"body": "blocked"}, authorCookie)
	if privatePost.Code != http.StatusForbidden {
		t.Fatalf("expected private comments post 403, got %d: %s", privatePost.Code, privatePost.Body.String())
	}
}

func TestProfileCommentsAPIDeletePermissions(t *testing.T) {
	app := newTestApp(t)
	owner, ownerCookie := registerUserAndCookie(t, app, "comment_delete_owner", "comment-delete-owner@example.com")
	_, authorCookie := registerUserAndCookie(t, app, "comment_delete_author", "comment-delete-author@example.com")
	_, strangerCookie := registerUserAndCookie(t, app, "comment_delete_stranger", "comment-delete-stranger@example.com")

	first := createProfileCommentForAPITest(t, app, owner.Username, authorCookie, "Delete by author")

	loggedOut := performJSON(app, http.MethodDelete, "/api/profile-comments/"+strconvID(first.ID), nil)
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out delete 401, got %d", loggedOut.Code)
	}
	strangerDelete := performJSON(app, http.MethodDelete, "/api/profile-comments/"+strconvID(first.ID), nil, strangerCookie)
	if strangerDelete.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated delete 403, got %d: %s", strangerDelete.Code, strangerDelete.Body.String())
	}
	authorDelete := performJSON(app, http.MethodDelete, "/api/profile-comments/"+strconvID(first.ID), nil, authorCookie)
	if authorDelete.Code != http.StatusOK {
		t.Fatalf("expected author delete 200, got %d: %s", authorDelete.Code, authorDelete.Body.String())
	}
	missingDelete := performJSON(app, http.MethodDelete, "/api/profile-comments/"+strconvID(first.ID), nil, authorCookie)
	if missingDelete.Code != http.StatusNotFound {
		t.Fatalf("expected missing delete 404, got %d: %s", missingDelete.Code, missingDelete.Body.String())
	}

	second := createProfileCommentForAPITest(t, app, owner.Username, authorCookie, "Delete by owner")
	ownerDelete := performJSON(app, http.MethodDelete, "/api/profile-comments/"+strconvID(second.ID), nil, ownerCookie)
	if ownerDelete.Code != http.StatusOK {
		t.Fatalf("expected profile owner delete 200, got %d: %s", ownerDelete.Code, ownerDelete.Body.String())
	}

	list := performJSON(app, http.MethodGet, "/api/users/"+owner.Username+"/comments", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("expected comments list 200, got %d: %s", list.Code, list.Body.String())
	}
	var response struct {
		Comments []profileCommentResponse `json:"comments"`
	}
	if err := json.NewDecoder(list.Body).Decode(&response); err != nil {
		t.Fatalf("decode comments after delete: %v", err)
	}
	if len(response.Comments) != 0 {
		t.Fatalf("expected all comments deleted, got %+v", response.Comments)
	}
}

func createProfileCommentForAPITest(t *testing.T, app *App, username string, cookie *http.Cookie, body string) profileCommentResponse {
	t.Helper()
	rec := performJSON(app, http.MethodPost, "/api/users/"+username+"/comments", map[string]string{"body": body}, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create profile comment: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var comment profileCommentResponse
	if err := json.NewDecoder(rec.Body).Decode(&comment); err != nil {
		t.Fatalf("decode created profile comment: %v", err)
	}
	return comment
}

func assertNoPrivateCommentData(t *testing.T, body string) {
	t.Helper()
	for _, private := range []string{"@example.com", "password", "password_hash", "answers", "answers_json", "scores_json", "private result"} {
		if strings.Contains(body, private) {
			t.Fatalf("profile comments API leaked private field/value %q in %s", private, body)
		}
	}
}
