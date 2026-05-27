package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestBlocksAPIAndBlockedInteractions(t *testing.T) {
	app := newTestApp(t)
	alice, aliceCookie := registerUserAndCookie(t, app, "safe_alice", "safe-alice@example.com")
	bob, bobCookie := registerUserAndCookie(t, app, "safe_bob", "safe-bob@example.com")

	loggedOut := performJSON(app, http.MethodPost, "/api/blocks", map[string]string{"username": bob.Username})
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out block 401, got %d", loggedOut.Code)
	}
	self := performJSON(app, http.MethodPost, "/api/blocks", map[string]string{"username": alice.Username}, aliceCookie)
	if self.Code != http.StatusBadRequest {
		t.Fatalf("expected self block 400, got %d: %s", self.Code, self.Body.String())
	}

	friend := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": bob.Username}, aliceCookie)
	if friend.Code != http.StatusCreated {
		t.Fatalf("expected friend request before block 201, got %d: %s", friend.Code, friend.Body.String())
	}
	var friendship friendshipResponse
	if err := json.NewDecoder(friend.Body).Decode(&friendship); err != nil {
		t.Fatalf("decode friendship: %v", err)
	}
	if accepted := performJSON(app, http.MethodPost, "/api/friends/requests/"+strconvID(friendship.ID)+"/accept", nil, bobCookie); accepted.Code != http.StatusOK {
		t.Fatalf("expected accept before block 200, got %d: %s", accepted.Code, accepted.Body.String())
	}

	start := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": bob.Username}, aliceCookie)
	if start.Code != http.StatusOK {
		t.Fatalf("expected message start before block 200, got %d: %s", start.Code, start.Body.String())
	}
	var conversation conversationResponse
	if err := json.NewDecoder(start.Body).Decode(&conversation); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	sendBeforeBlock := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "hello before block"}, aliceCookie)
	if sendBeforeBlock.Code != http.StatusCreated {
		t.Fatalf("expected send before block 201, got %d: %s", sendBeforeBlock.Code, sendBeforeBlock.Body.String())
	}

	block := performJSON(app, http.MethodPost, "/api/blocks", map[string]string{"username": bob.Username}, aliceCookie)
	if block.Code != http.StatusOK {
		t.Fatalf("expected block 200, got %d: %s", block.Code, block.Body.String())
	}
	assertNoPrivateSafetyData(t, block.Body.String())

	blocks := performJSON(app, http.MethodGet, "/api/blocks", nil, aliceCookie)
	if blocks.Code != http.StatusOK {
		t.Fatalf("expected blocks list 200, got %d: %s", blocks.Code, blocks.Body.String())
	}
	assertNoPrivateSafetyData(t, blocks.Body.String())
	var blocksResponse struct {
		Blocks []blockedUserResponse `json:"blocks"`
	}
	if err := json.NewDecoder(blocks.Body).Decode(&blocksResponse); err != nil {
		t.Fatalf("decode blocks: %v", err)
	}
	if len(blocksResponse.Blocks) != 1 || blocksResponse.Blocks[0].Username != bob.Username {
		t.Fatalf("unexpected blocks list: %+v", blocksResponse.Blocks)
	}

	bobFriendBlocked := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": alice.Username}, bobCookie)
	if bobFriendBlocked.Code != http.StatusForbidden {
		t.Fatalf("expected blocked friend request 403, got %d: %s", bobFriendBlocked.Code, bobFriendBlocked.Body.String())
	}
	bobCommentBlocked := performJSON(app, http.MethodPost, "/api/users/"+alice.Username+"/comments", map[string]string{"body": "blocked comment"}, bobCookie)
	if bobCommentBlocked.Code != http.StatusForbidden {
		t.Fatalf("expected blocked comment 403, got %d: %s", bobCommentBlocked.Code, bobCommentBlocked.Body.String())
	}
	bobStartBlocked := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": alice.Username}, bobCookie)
	if bobStartBlocked.Code != http.StatusForbidden {
		t.Fatalf("expected blocked message start 403, got %d: %s", bobStartBlocked.Code, bobStartBlocked.Body.String())
	}
	bobSendBlocked := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "blocked send"}, bobCookie)
	if bobSendBlocked.Code != http.StatusForbidden {
		t.Fatalf("expected blocked message send 403, got %d: %s", bobSendBlocked.Code, bobSendBlocked.Body.String())
	}

	aliceViewsBob := performJSON(app, http.MethodGet, "/api/users/"+bob.Username, nil, aliceCookie)
	var bobProfile publicProfileResponse
	if err := json.NewDecoder(aliceViewsBob.Body).Decode(&bobProfile); err != nil {
		t.Fatalf("decode Bob profile: %v", err)
	}
	if bobProfile.ViewerBlock == nil || !bobProfile.ViewerBlock.ViewerBlockedTarget || bobProfile.ViewerBlock.TargetBlockedViewer {
		t.Fatalf("expected Alice profile block state against Bob, got %+v", bobProfile.ViewerBlock)
	}
	bobViewsAlice := performJSON(app, http.MethodGet, "/api/users/"+alice.Username, nil, bobCookie)
	var aliceProfile publicProfileResponse
	if err := json.NewDecoder(bobViewsAlice.Body).Decode(&aliceProfile); err != nil {
		t.Fatalf("decode Alice profile: %v", err)
	}
	if aliceProfile.ViewerBlock == nil || !aliceProfile.ViewerBlock.TargetBlockedViewer || aliceProfile.ViewerBlock.ViewerBlockedTarget {
		t.Fatalf("expected Bob blocked-by-Alice state, got %+v", aliceProfile.ViewerBlock)
	}

	unblock := performJSON(app, http.MethodDelete, "/api/blocks/"+bob.Username, nil, aliceCookie)
	if unblock.Code != http.StatusOK {
		t.Fatalf("expected unblock 200, got %d: %s", unblock.Code, unblock.Body.String())
	}
	bobFriendAfterUnblock := performJSON(app, http.MethodPost, "/api/friends/request", map[string]string{"username": alice.Username}, bobCookie)
	if bobFriendAfterUnblock.Code != http.StatusCreated {
		t.Fatalf("expected friend request after unblock 201, got %d: %s", bobFriendAfterUnblock.Code, bobFriendAfterUnblock.Body.String())
	}
}

func TestReportsAPIAndAdminReview(t *testing.T) {
	app := newTestApp(t)
	_, reporterCookie := registerUserAndCookie(t, app, "report_api_user", "report-api-user@example.com")
	target, targetCookie := registerUserAndCookie(t, app, "report_api_target", "report-api-target@example.com")

	loggedOut := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "profile",
		"username":   target.Username,
		"reason":     "spam",
	})
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out report 401, got %d", loggedOut.Code)
	}
	invalidType := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "result",
		"reason":     "spam",
	}, reporterCookie)
	if invalidType.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid target type 400, got %d: %s", invalidType.Code, invalidType.Body.String())
	}
	emptyReason := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "profile",
		"username":   target.Username,
		"reason":     "   ",
	}, reporterCookie)
	if emptyReason.Code != http.StatusBadRequest {
		t.Fatalf("expected empty reason 400, got %d: %s", emptyReason.Code, emptyReason.Body.String())
	}
	longDetails := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "profile",
		"username":   target.Username,
		"reason":     "spam",
		"details":    strings.Repeat("x", maxReportDetailsLength+1),
	}, reporterCookie)
	if longDetails.Code != http.StatusBadRequest {
		t.Fatalf("expected long details 400, got %d: %s", longDetails.Code, longDetails.Body.String())
	}

	profileReport := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "profile",
		"username":   target.Username,
		"reason":     "spam",
		"details":    "short details",
	}, reporterCookie)
	if profileReport.Code != http.StatusCreated {
		t.Fatalf("expected profile report 201, got %d: %s", profileReport.Code, profileReport.Body.String())
	}
	assertNoPrivateSafetyData(t, profileReport.Body.String())

	comment := performJSON(app, http.MethodPost, "/api/users/"+target.Username+"/comments", map[string]string{"body": "please review"}, reporterCookie)
	if comment.Code != http.StatusCreated {
		t.Fatalf("expected comment create 201, got %d: %s", comment.Code, comment.Body.String())
	}
	var commentResponse profileCommentResponse
	if err := json.NewDecoder(comment.Body).Decode(&commentResponse); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	commentReport := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "comment",
		"targetId":   commentResponse.ID,
		"reason":     "abuse",
	}, targetCookie)
	if commentReport.Code != http.StatusCreated {
		t.Fatalf("expected comment report 201, got %d: %s", commentReport.Code, commentReport.Body.String())
	}

	start := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": target.Username}, reporterCookie)
	if start.Code != http.StatusOK {
		t.Fatalf("expected start message 200, got %d: %s", start.Code, start.Body.String())
	}
	var conversation conversationResponse
	if err := json.NewDecoder(start.Body).Decode(&conversation); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	targetMessage := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "message to report"}, targetCookie)
	if targetMessage.Code != http.StatusCreated {
		t.Fatalf("expected target message 201, got %d: %s", targetMessage.Code, targetMessage.Body.String())
	}
	var message messageResponse
	if err := json.NewDecoder(targetMessage.Body).Decode(&message); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	messageReport := performJSON(app, http.MethodPost, "/api/reports", map[string]any{
		"targetType": "message",
		"targetId":   message.ID,
		"reason":     "harassment",
	}, reporterCookie)
	if messageReport.Code != http.StatusCreated {
		t.Fatalf("expected message report 201, got %d: %s", messageReport.Code, messageReport.Body.String())
	}

	userCannotReview := performJSON(app, http.MethodGet, "/api/admin/reports", nil, reporterCookie)
	if userCannotReview.Code != http.StatusUnauthorized {
		t.Fatalf("regular user must not read admin reports, got %d", userCannotReview.Code)
	}
	adminReports := performJSON(app, http.MethodGet, "/api/admin/reports", nil, login(t, app)...)
	if adminReports.Code != http.StatusOK {
		t.Fatalf("expected admin reports 200, got %d: %s", adminReports.Code, adminReports.Body.String())
	}
	assertNoPrivateSafetyData(t, adminReports.Body.String())
	var adminResponse struct {
		Reports []reportResponse `json:"reports"`
	}
	if err := json.NewDecoder(adminReports.Body).Decode(&adminResponse); err != nil {
		t.Fatalf("decode admin reports: %v", err)
	}
	if len(adminResponse.Reports) != 3 {
		t.Fatalf("expected 3 reports, got %+v", adminResponse.Reports)
	}
	update := performJSON(app, http.MethodPost, "/api/admin/reports/"+strconvID(adminResponse.Reports[0].ID)+"/status", map[string]string{"status": "reviewed"}, login(t, app)...)
	if update.Code != http.StatusOK {
		t.Fatalf("expected admin report status update 200, got %d: %s", update.Code, update.Body.String())
	}
	if !strings.Contains(update.Body.String(), `"status":"reviewed"`) {
		t.Fatalf("expected reviewed status in update response, got %s", update.Body.String())
	}
}

func assertNoPrivateSafetyData(t *testing.T, body string) {
	t.Helper()
	for _, private := range []string{"@example.com", "password", "password_hash", "answers_json", "scores_json", "private result"} {
		if strings.Contains(body, private) {
			t.Fatalf("safety API leaked private field/value %q in %s", private, body)
		}
	}
}
