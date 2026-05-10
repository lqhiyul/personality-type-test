package main

import (
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

type sendMessageRequest struct {
	Body string `json:"body"`
}

type messageParticipantResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	AvatarKey   string `json:"avatarKey"`
}

type conversationPreviewResponse struct {
	ID             int64                      `json:"id"`
	ConversationID int64                      `json:"conversationId"`
	Sender         messageParticipantResponse `json:"sender"`
	Body           string                     `json:"body"`
	CreatedAt      string                     `json:"createdAt"`
}

type conversationResponse struct {
	ID           int64                        `json:"id"`
	Participants []messageParticipantResponse `json:"participants"`
	LastMessage  *conversationPreviewResponse `json:"lastMessage,omitempty"`
	CreatedAt    string                       `json:"createdAt"`
	UpdatedAt    string                       `json:"updatedAt"`
}

type messageResponse struct {
	ID             int64                      `json:"id"`
	ConversationID int64                      `json:"conversationId"`
	Sender         messageParticipantResponse `json:"sender"`
	Body           string                     `json:"body"`
	CreatedAt      string                     `json:"createdAt"`
}

type conversationDetailResponse struct {
	Conversation conversationResponse `json:"conversation"`
	Messages     []messageResponse    `json:"messages"`
}

func (a *App) handleStartConversation(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, newConversationResponse(conversation))
}

func (a *App) handleConversations(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, map[string]any{"conversations": newConversationResponses(conversations)})
}

func (a *App) handleConversationByID(w http.ResponseWriter, r *http.Request) {
	currentUserID, ok := a.requireCurrentUserID(w, r)
	if !ok {
		return
	}

	conversationID, ok := conversationIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "conversation not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.handleReadConversation(w, r, currentUserID, conversationID)
	case http.MethodPost:
		a.handleSendConversationMessage(w, r, currentUserID, conversationID)
	default:
		methodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (a *App) handleReadConversation(w http.ResponseWriter, r *http.Request, currentUserID, conversationID int64) {
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
		Conversation: newConversationResponse(conversation),
		Messages:     newMessageResponses(messages),
	})
}

func (a *App) handleSendConversationMessage(w http.ResponseWriter, r *http.Request, currentUserID, conversationID int64) {
	var req sendMessageRequest
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

func conversationIDFromPath(requestPath string) (int64, bool) {
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

func newConversationResponses(conversations []Conversation) []conversationResponse {
	out := make([]conversationResponse, 0, len(conversations))
	for _, conversation := range conversations {
		out = append(out, newConversationResponse(conversation))
	}
	return out
}

func newConversationResponse(conversation Conversation) conversationResponse {
	return conversationResponse{
		ID:           conversation.ID,
		Participants: newMessageParticipantResponses(conversation.Participants),
		LastMessage:  newConversationPreviewResponse(conversation.LastMessage),
		CreatedAt:    conversation.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:    conversation.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func newMessageResponses(messages []Message) []messageResponse {
	out := make([]messageResponse, 0, len(messages))
	for _, message := range messages {
		out = append(out, newMessageResponse(message))
	}
	return out
}

func newMessageResponse(message Message) messageResponse {
	return messageResponse{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		Sender:         newMessageParticipantResponse(message.Sender),
		Body:           message.Body,
		CreatedAt:      message.CreatedAt.Format(time.RFC3339Nano),
	}
}

func newConversationPreviewResponse(preview *ConversationMessagePreview) *conversationPreviewResponse {
	if preview == nil {
		return nil
	}
	return &conversationPreviewResponse{
		ID:             preview.ID,
		ConversationID: preview.ConversationID,
		Sender:         newMessageParticipantResponse(preview.Sender),
		Body:           preview.Body,
		CreatedAt:      preview.CreatedAt.Format(time.RFC3339Nano),
	}
}

func newMessageParticipantResponses(participants []MessageParticipant) []messageParticipantResponse {
	out := make([]messageParticipantResponse, 0, len(participants))
	for _, participant := range participants {
		out = append(out, newMessageParticipantResponse(participant))
	}
	return out
}

func newMessageParticipantResponse(participant MessageParticipant) messageParticipantResponse {
	return messageParticipantResponse{
		ID:          participant.ID,
		Username:    participant.Username,
		DisplayName: participant.DisplayName,
		AvatarKey:   participant.AvatarKey,
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
	case errors.Is(err, ErrMessageForbidden):
		writeJSONError(w, http.StatusForbidden, "message action is not allowed")
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "message not found")
	default:
		writeJSONError(w, http.StatusInternalServerError, "could not update messages")
	}
}
