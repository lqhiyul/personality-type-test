package app

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type startConversationRequest struct {
	Username string `json:"username"`
}

type createMessageRequest struct {
	Body string `json:"body"`
}

type messageParticipantResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarKey   string `json:"avatarKey"`
}

type conversationPreviewResponse struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"senderId"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type conversationResponse struct {
	ID           int64                        `json:"id"`
	Participants []messageParticipantResponse `json:"participants"`
	LastMessage  *conversationPreviewResponse `json:"lastMessage,omitempty"`
	Blocked      bool                         `json:"blocked"`
	CreatedAt    string                       `json:"createdAt"`
	UpdatedAt    string                       `json:"updatedAt"`
}

type messageResponse struct {
	ID             int64  `json:"id"`
	ConversationID int64  `json:"conversationId"`
	SenderID       int64  `json:"senderId"`
	Body           string `json:"body"`
	CreatedAt      string `json:"createdAt"`
}

type conversationDetailResponse struct {
	Conversation conversationResponse `json:"conversation"`
	Messages     []messageResponse    `json:"messages"`
}

func (a *App) handleStartMessageConversation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return
	}
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	var req startConversationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}
	username, err := normalizeUsername(req.Username)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "username is required")
		return
	}
	target, err := a.userStore.GetUserByUsername(r.Context(), username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "target user not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "could not load target user")
		return
	}
	conversation, err := a.userStore.GetOrCreateConversation(r.Context(), currentUserID, target.ID)
	if err != nil {
		writeMessageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.newConversationResponseForUser(r.Context(), conversation, currentUserID))
}

func (a *App) handleMessageConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, http.MethodGet)
		return
	}
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	conversations, err := a.userStore.ListConversationsForUser(r.Context(), currentUserID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load conversations")
		return
	}
	responses := make([]conversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		responses = append(responses, a.newConversationResponseForUser(r.Context(), conversation, currentUserID))
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": responses})
}

func (a *App) handleMessageConversationByID(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	conversationID, ok := messageConversationIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "conversation not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		conversation, err := a.userStore.GetConversationForUser(r.Context(), conversationID, currentUserID)
		if err != nil {
			writeMessageError(w, err)
			return
		}
		messages, err := a.userStore.ListMessages(r.Context(), conversationID, currentUserID, defaultMessageLimit)
		if err != nil {
			writeMessageError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, conversationDetailResponse{
			Conversation: a.newConversationResponseForUser(r.Context(), conversation, currentUserID),
			Messages:     newMessageResponses(messages),
		})
	case http.MethodPost:
		var req createMessageRequest
		if err := decodeJSON(r, &req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
			return
		}
		message, err := a.userStore.CreateMessage(r.Context(), conversationID, currentUserID, req.Body)
		if err != nil {
			writeMessageError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, newMessageResponse(message))
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *App) handleMessageByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		methodNotAllowed(w, http.MethodDelete)
		return
	}
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}
	messageID, ok := messageIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "message not found")
		return
	}
	if err := a.userStore.DeleteMessage(r.Context(), messageID, currentUserID); err != nil {
		writeMessageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func messageConversationIDFromPath(requestPath string) (int64, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/messages/conversations/")
	if suffix == requestPath || suffix == "" || strings.Contains(strings.Trim(suffix, "/"), "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.Trim(suffix, "/"), 10, 64)
	return id, err == nil && id > 0
}

func messageIDFromPath(requestPath string) (int64, bool) {
	suffix := strings.TrimPrefix(requestPath, "/api/messages/")
	if suffix == requestPath || suffix == "" || strings.Contains(strings.Trim(suffix, "/"), "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.Trim(suffix, "/"), 10, 64)
	return id, err == nil && id > 0
}

func newConversationResponse(conversation Conversation) conversationResponse {
	participants := make([]messageParticipantResponse, 0, len(conversation.Participants))
	for _, participant := range conversation.Participants {
		participants = append(participants, newMessageParticipantResponse(participant))
	}
	var lastMessage *conversationPreviewResponse
	if conversation.LastMessage != nil {
		lastMessage = &conversationPreviewResponse{
			ID:        conversation.LastMessage.ID,
			SenderID:  conversation.LastMessage.SenderID,
			Body:      conversation.LastMessage.Body,
			CreatedAt: conversation.LastMessage.CreatedAt.Format(time.RFC3339Nano),
		}
	}
	return conversationResponse{
		ID:           conversation.ID,
		Participants: participants,
		LastMessage:  lastMessage,
		CreatedAt:    conversation.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    conversation.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func (a *App) newConversationResponseForUser(ctx context.Context, conversation Conversation, currentUserID int64) conversationResponse {
	response := newConversationResponse(conversation)
	for _, participant := range conversation.Participants {
		if participant.ID == currentUserID {
			continue
		}
		blocked, err := a.userStore.IsBlockedBetween(ctx, currentUserID, participant.ID)
		if err == nil && blocked {
			response.Blocked = true
			return response
		}
	}
	return response
}

func newMessageParticipantResponse(participant MessageParticipant) messageParticipantResponse {
	return messageParticipantResponse(participant)
}

func newMessageResponses(messages []Message) []messageResponse {
	out := make([]messageResponse, 0, len(messages))
	for _, message := range messages {
		out = append(out, newMessageResponse(message))
	}
	return out
}

func newMessageResponse(message Message) messageResponse {
	body := message.Body
	if message.DeletedAt != nil {
		body = ""
	}
	return messageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		Body:           body,
		CreatedAt:      message.CreatedAt.Format(time.RFC3339Nano),
	}
}

func writeMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMessageSelf):
		writeJSONError(w, http.StatusBadRequest, "cannot start a conversation with yourself")
	case errors.Is(err, ErrMessageBodyRequired):
		writeJSONError(w, http.StatusBadRequest, "message cannot be empty")
	case errors.Is(err, ErrMessageBodyTooLong):
		writeJSONError(w, http.StatusBadRequest, "message is too long")
	case errors.Is(err, ErrMessageBodyInvalid):
		writeJSONError(w, http.StatusBadRequest, "message contains invalid characters")
	case errors.Is(err, ErrBlockedInteraction):
		writeJSONError(w, http.StatusForbidden, "cannot interact with blocked user")
	case errors.Is(err, ErrMessageForbidden):
		writeJSONError(w, http.StatusForbidden, "message action is not allowed")
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "message not found")
	default:
		writeJSONError(w, http.StatusInternalServerError, "could not update messages")
	}
}
