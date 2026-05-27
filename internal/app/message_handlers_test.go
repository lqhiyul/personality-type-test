package app

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestMessagesAPIAuthValidationScopeAndDelete(t *testing.T) {
	app := newTestApp(t)
	alice, aliceCookie := registerUserAndCookie(t, app, "api_msg_alice", "api-msg-alice@example.com")
	bob, bobCookie := registerUserAndCookie(t, app, "api_msg_bob", "api-msg-bob@example.com")
	_, carolCookie := registerUserAndCookie(t, app, "api_msg_carol", "api-msg-carol@example.com")

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
		t.Fatalf("expected self start 400, got %d: %s", self.Code, self.Body.String())
	}

	start := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": bob.Username}, aliceCookie)
	if start.Code != http.StatusOK {
		t.Fatalf("expected start 200, got %d: %s", start.Code, start.Body.String())
	}
	assertNoPrivateMessageData(t, start.Body.String())
	var conversation conversationResponse
	if err := json.NewDecoder(start.Body).Decode(&conversation); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	if conversation.ID == 0 || len(conversation.Participants) != 2 {
		t.Fatalf("unexpected conversation: %+v", conversation)
	}
	startAgain := performJSON(app, http.MethodPost, "/api/messages/start", map[string]string{"username": bob.Username}, aliceCookie)
	var existing conversationResponse
	if err := json.NewDecoder(startAgain.Body).Decode(&existing); err != nil {
		t.Fatalf("decode existing conversation: %v", err)
	}
	if existing.ID != conversation.ID {
		t.Fatalf("expected existing conversation %d, got %d", conversation.ID, existing.ID)
	}

	listAlice := performJSON(app, http.MethodGet, "/api/messages/conversations", nil, aliceCookie)
	if listAlice.Code != http.StatusOK {
		t.Fatalf("expected Alice conversations 200, got %d: %s", listAlice.Code, listAlice.Body.String())
	}
	assertNoPrivateMessageData(t, listAlice.Body.String())
	var listResponse struct {
		Conversations []conversationResponse `json:"conversations"`
	}
	if err := json.NewDecoder(listAlice.Body).Decode(&listResponse); err != nil {
		t.Fatalf("decode conversations: %v", err)
	}
	if len(listResponse.Conversations) != 1 || listResponse.Conversations[0].ID != conversation.ID {
		t.Fatalf("unexpected conversations: %+v", listResponse.Conversations)
	}
	listCarol := performJSON(app, http.MethodGet, "/api/messages/conversations", nil, carolCookie)
	var carolList struct {
		Conversations []conversationResponse `json:"conversations"`
	}
	if err := json.NewDecoder(listCarol.Body).Decode(&carolList); err != nil {
		t.Fatalf("decode Carol conversations: %v", err)
	}
	if len(carolList.Conversations) != 0 {
		t.Fatalf("Carol should not see Alice/Bob conversation, got %+v", carolList.Conversations)
	}

	readCarol := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, carolCookie)
	if readCarol.Code != http.StatusForbidden {
		t.Fatalf("expected unrelated read 403, got %d: %s", readCarol.Code, readCarol.Body.String())
	}
	empty := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "   "}, aliceCookie)
	if empty.Code != http.StatusBadRequest {
		t.Fatalf("expected empty message 400, got %d: %s", empty.Code, empty.Body.String())
	}
	tooLong := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": strings.Repeat("x", maxMessageBodyLength+1)}, aliceCookie)
	if tooLong.Code != http.StatusBadRequest {
		t.Fatalf("expected too long message 400, got %d: %s", tooLong.Code, tooLong.Body.String())
	}

	sent := performJSON(app, http.MethodPost, "/api/messages/conversations/"+strconvID(conversation.ID), map[string]string{"body": "  <i>Hello</i>  "}, aliceCookie)
	if sent.Code != http.StatusCreated {
		t.Fatalf("expected send 201, got %d: %s", sent.Code, sent.Body.String())
	}
	assertNoPrivateMessageData(t, sent.Body.String())
	var message messageResponse
	if err := json.NewDecoder(sent.Body).Decode(&message); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if message.Body != "<i>Hello</i>" || message.SenderID != alice.ID {
		t.Fatalf("unexpected sent message: %+v", message)
	}
	readBob := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, bobCookie)
	if readBob.Code != http.StatusOK {
		t.Fatalf("expected Bob read 200, got %d: %s", readBob.Code, readBob.Body.String())
	}
	assertNoPrivateMessageData(t, readBob.Body.String())
	var detail conversationDetailResponse
	if err := json.NewDecoder(readBob.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Messages) != 1 || detail.Messages[0].ID != message.ID {
		t.Fatalf("unexpected detail messages: %+v", detail.Messages)
	}

	deleteOther := performJSON(app, http.MethodDelete, "/api/messages/"+strconvID(message.ID), nil, bobCookie)
	if deleteOther.Code != http.StatusForbidden {
		t.Fatalf("expected delete other message 403, got %d: %s", deleteOther.Code, deleteOther.Body.String())
	}
	deleteOwn := performJSON(app, http.MethodDelete, "/api/messages/"+strconvID(message.ID), nil, aliceCookie)
	if deleteOwn.Code != http.StatusOK {
		t.Fatalf("expected delete own message 200, got %d: %s", deleteOwn.Code, deleteOwn.Body.String())
	}
	readAfterDelete := performJSON(app, http.MethodGet, "/api/messages/conversations/"+strconvID(conversation.ID), nil, bobCookie)
	if strings.Contains(readAfterDelete.Body.String(), "<i>Hello</i>") {
		t.Fatalf("deleted message body should not be exposed: %s", readAfterDelete.Body.String())
	}
}

func assertNoPrivateMessageData(t *testing.T, body string) {
	t.Helper()
	for _, private := range []string{"@example.com", "password", "password_hash", "answers", "answers_json", "scores_json", "private result"} {
		if strings.Contains(body, private) {
			t.Fatalf("message API leaked private field/value %q in %s", private, body)
		}
	}
}
