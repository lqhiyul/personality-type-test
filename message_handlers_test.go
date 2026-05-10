package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestMessagesAPIAuthValidationAccessAndSafety(t *testing.T) {
	app := newTestApp(t)
	alice, aliceCookie := registerUserAndCookie(t, app, "api_message_alice", "api-message-alice@example.com")
	bob, bobCookie := registerUserAndCookie(t, app, "api_message_bob", "api-message-bob@example.com")
	_, carolCookie := registerUserAndCookie(t, app, "api_message_carol", "api-message-carol@example.com")

	loggedOut := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": bob.Username})
	if loggedOut.Code != http.StatusUnauthorized {
		t.Fatalf("expected logged-out start 401, got %d", loggedOut.Code)
	}

	missingUsername := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": ""}, aliceCookie)
	if missingUsername.Code != http.StatusBadRequest {
		t.Fatalf("expected missing username 400, got %d: %s", missingUsername.Code, missingUsername.Body.String())
	}

	unknown := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": "missing_user"}, aliceCookie)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("expected unknown target 404, got %d: %s", unknown.Code, unknown.Body.String())
	}

	self := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": alice.Username}, aliceCookie)
	if self.Code != http.StatusBadRequest {
		t.Fatalf("expected self conversation 400, got %d: %s", self.Code, self.Body.String())
	}

	start := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": bob.Username}, aliceCookie)
	if start.Code != http.StatusOK {
		t.Fatalf("expected start 200, got %d: %s", start.Code, start.Body.String())
	}
	assertNoPrivateMessageData(t, start.Body.String())
	var conversation conversationResponse
	if err := json.NewDecoder(start.Body).Decode(&conversation); err != nil {
		t.Fatalf("decode started conversation: %v", err)
	}
	if conversation.ID == 0 || !conversationHasParticipant(conversation, alice.Username) || !conversationHasParticipant(conversation, bob.Username) {
		t.Fatalf("unexpected started conversation: %+v", conversation)
	}

	again := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": bob.Username}, aliceCookie)
	var existing conversationResponse
	if err := json.NewDecoder(again.Body).Decode(&existing); err != nil {
		t.Fatalf("decode existing conversation: %v", err)
	}
	if existing.ID != conversation.ID {
		t.Fatalf("expected existing conversation %d, got %d", conversation.ID, existing.ID)
	}

	// Create an unrelated Bob-Carol conversation to prove Alice's list is scoped.
	bobCarol := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": "api_message_carol"}, bobCookie)
	if bobCarol.Code != http.StatusOK {
		t.Fatalf("expected Bob-Carol start 200, got %d: %s", bobCarol.Code, bobCarol.Body.String())
	}

	aliceList := performJSON(app, http.MethodGet, "/api/messages/conversations", nil, aliceCookie)
	if aliceList.Code != http.StatusOK {
		t.Fatalf("expected Alice conversations 200, got %d: %s", aliceList.Code, aliceList.Body.String())
	}
	assertNoPrivateMessageData(t, aliceList.Body.String())
	var listResponse struct {
		Conversations []conversationResponse `json:"conversations"`
	}
	if err := json.NewDecoder(aliceList.Body).Decode(&listResponse); err != nil {
		t.Fatalf("decode Alice conversations: %v", err)
	}
	if len(listResponse.Conversations) != 1 || listResponse.Conversations[0].ID != conversation.ID {
		t.Fatalf("expected Alice to see only Alice-Bob conversation, got %+v", listResponse.Conversations)
	}
	if strings.Contains(aliceList.Body.String(), "api_message_carol") {
		t.Fatalf("Alice conversation list leaked unrelated conversation: %s", aliceList.Body.String())
	}

	readEmpty := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, aliceCookie)
	if readEmpty.Code != http.StatusOK {
		t.Fatalf("expected read empty conversation 200, got %d: %s", readEmpty.Code, readEmpty.Body.String())
	}
	var emptyDetail conversationDetailResponse
	if err := json.NewDecoder(readEmpty.Body).Decode(&emptyDetail); err != nil {
		t.Fatalf("decode empty conversation: %v", err)
	}
	if len(emptyDetail.Messages) != 0 {
		t.Fatalf("expected no messages yet, got %+v", emptyDetail.Messages)
	}

	unrelatedRead := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, carolCookie)
	if unrelatedRead.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated read 403, got %d: %s", unrelatedRead.Code, unrelatedRead.Body.String())
	}

	emptyMessage := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "   "}, aliceCookie)
	if emptyMessage.Code != http.StatusBadRequest {
		t.Fatalf("expected empty message 400, got %d: %s", emptyMessage.Code, emptyMessage.Body.String())
	}
	longMessage := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": strings.Repeat("a", maxMessageBodyLength+1)}, aliceCookie)
	if longMessage.Code != http.StatusBadRequest {
		t.Fatalf("expected long message 400, got %d: %s", longMessage.Code, longMessage.Body.String())
	}
	unrelatedSend := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "blocked"}, carolCookie)
	if unrelatedSend.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated send 403, got %d: %s", unrelatedSend.Code, unrelatedSend.Body.String())
	}

	sent := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "  <em>Hello Bob</em>  "}, aliceCookie)
	if sent.Code != http.StatusCreated {
		t.Fatalf("expected send 201, got %d: %s", sent.Code, sent.Body.String())
	}
	assertNoPrivateMessageData(t, sent.Body.String())
	var message messageResponse
	if err := json.NewDecoder(sent.Body).Decode(&message); err != nil {
		t.Fatalf("decode sent message: %v", err)
	}
	if message.Body != "<em>Hello Bob</em>" || message.Sender.Username != alice.Username {
		t.Fatalf("unexpected sent message: %+v", message)
	}

	bobRead := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, bobCookie)
	if bobRead.Code != http.StatusOK {
		t.Fatalf("expected Bob read 200, got %d: %s", bobRead.Code, bobRead.Body.String())
	}
	assertNoPrivateMessageData(t, bobRead.Body.String())
	var bobDetail conversationDetailResponse
	if err := json.NewDecoder(bobRead.Body).Decode(&bobDetail); err != nil {
		t.Fatalf("decode Bob conversation: %v", err)
	}
	if len(bobDetail.Messages) != 1 || bobDetail.Messages[0].Body != "<em>Hello Bob</em>" {
		t.Fatalf("expected Bob read to include plain text body, got %+v", bobDetail.Messages)
	}

	carolDelete := performJSON(app, http.MethodDelete, "/api/messages/"+strconvID(message.ID), nil, carolCookie)
	if carolDelete.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated delete 403, got %d: %s", carolDelete.Code, carolDelete.Body.String())
	}
	bobDelete := performJSON(app, http.MethodDelete, "/api/messages/"+strconvID(message.ID), nil, bobCookie)
	if bobDelete.Code != http.StatusForbidden {
		t.Fatalf("expected participant deleting another sender's message 403, got %d: %s", bobDelete.Code, bobDelete.Body.String())
	}
	aliceDelete := performJSON(app, http.MethodDelete, "/api/messages/"+strconvID(message.ID), nil, aliceCookie)
	if aliceDelete.Code != http.StatusOK {
		t.Fatalf("expected sender delete 200, got %d: %s", aliceDelete.Code, aliceDelete.Body.String())
	}

	afterDelete := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, bobCookie)
	if afterDelete.Code != http.StatusOK {
		t.Fatalf("expected read after delete 200, got %d: %s", afterDelete.Code, afterDelete.Body.String())
	}
	var afterDeleteDetail conversationDetailResponse
	if err := json.NewDecoder(afterDelete.Body).Decode(&afterDeleteDetail); err != nil {
		t.Fatalf("decode conversation after delete: %v", err)
	}
	if len(afterDeleteDetail.Messages) != 0 || afterDeleteDetail.Conversation.LastMessage != nil {
		t.Fatalf("deleted message body leaked after delete: %+v", afterDeleteDetail)
	}
}

func TestMessagesRegressionCoreFlowsStillWork(t *testing.T) {
	app := newTestApp(t)
	_, cookie := registerUserAndCookie(t, app, "message_regression", "message-regression@example.com")

	submit := performJSON(app, http.MethodPost, "/api/submit", map[string]any{
		"name":     "Message Regression",
		"answers":  answersForType("INTJ"),
		"duration": 25,
	}, cookie)
	if submit.Code != http.StatusOK {
		t.Fatalf("expected logged-in submit 200, got %d: %s", submit.Code, submit.Body.String())
	}

	publicProfile := performJSON(app, http.MethodGet, "/api/users/message_regression", nil)
	if publicProfile.Code != http.StatusOK {
		t.Fatalf("expected public profile 200, got %d: %s", publicProfile.Code, publicProfile.Body.String())
	}

	comment := performJSON(app, http.MethodPost, "/api/users/message_regression/comments", map[string]string{"body": "Messaging did not break comments"}, cookie)
	if comment.Code != http.StatusCreated {
		t.Fatalf("expected profile comment 201, got %d: %s", comment.Code, comment.Body.String())
	}

	friends := performJSON(app, http.MethodGet, "/api/friends", nil, cookie)
	if friends.Code != http.StatusOK {
		t.Fatalf("expected friends list 200, got %d: %s", friends.Code, friends.Body.String())
	}

	adminResults := performJSON(app, http.MethodGet, "/api/results", nil, login(t, app)...)
	if adminResults.Code != http.StatusOK {
		t.Fatalf("expected admin results 200, got %d: %s", adminResults.Code, adminResults.Body.String())
	}
}

func conversationHasParticipant(conversation conversationResponse, username string) bool {
	for _, participant := range conversation.Participants {
		if participant.Username == username {
			return true
		}
	}
	return false
}

func assertNoPrivateMessageData(t *testing.T, body string) {
	t.Helper()
	for _, private := range []string{"@example.com", "password", "password_hash", "answers", "answers_json", "scores_json", "private result"} {
		if strings.Contains(body, private) {
			t.Fatalf("message API leaked private field/value %q in %s", private, body)
		}
	}
}
