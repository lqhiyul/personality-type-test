package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestUserStoreConversationAndMessagesLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, store := newTestUserStore(t)
	alice := createStoreUserForFriendTest(t, store, "message_alice")
	bob := createStoreUserForFriendTest(t, store, "message_bob")
	carol := createStoreUserForFriendTest(t, store, "message_carol")

	fixedNow := time.Date(2026, 5, 10, 15, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return fixedNow }
	conversation, err := store.CreateConversation(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if conversation.ID == 0 || !conversation.CreatedAt.Equal(fixedNow) || !conversation.UpdatedAt.Equal(fixedNow) {
		t.Fatalf("unexpected created conversation: %+v", conversation)
	}
	assertConversationParticipants(t, conversation, alice.Username, bob.Username)

	existing, err := store.GetOrCreateConversation(ctx, bob.ID, alice.ID)
	if err != nil {
		t.Fatalf("GetOrCreateConversation() error = %v", err)
	}
	if existing.ID != conversation.ID {
		t.Fatalf("expected existing conversation %d, got %d", conversation.ID, existing.ID)
	}
	if _, err := store.CreateConversation(ctx, alice.ID, alice.ID); !errors.Is(err, ErrMessageSelf) {
		t.Fatalf("expected self conversation to fail with ErrMessageSelf, got %v", err)
	}

	aliceConversations, err := store.ListConversationsForUser(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListConversationsForUser(alice) error = %v", err)
	}
	if len(aliceConversations) != 1 || aliceConversations[0].ID != conversation.ID {
		t.Fatalf("expected Alice to see one conversation, got %+v", aliceConversations)
	}
	carolConversations, err := store.ListConversationsForUser(ctx, carol.ID)
	if err != nil {
		t.Fatalf("ListConversationsForUser(carol) error = %v", err)
	}
	if len(carolConversations) != 0 {
		t.Fatalf("expected Carol to see no conversations, got %+v", carolConversations)
	}
	if _, err := store.GetConversationForUser(ctx, conversation.ID, carol.ID); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected unrelated conversation read to fail with ErrMessageForbidden, got %v", err)
	}

	store.now = func() time.Time { return fixedNow.Add(time.Minute) }
	first, err := store.CreateMessage(ctx, conversation.ID, alice.ID, "  <strong>Hello</strong> Bob  ")
	if err != nil {
		t.Fatalf("CreateMessage(alice) error = %v", err)
	}
	if first.Body != "<strong>Hello</strong> Bob" || first.Sender.Username != alice.Username {
		t.Fatalf("unexpected first message: %+v", first)
	}
	if _, err := store.CreateMessage(ctx, conversation.ID, carol.ID, "not a participant"); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected non-participant send to fail with ErrMessageForbidden, got %v", err)
	}
	if _, err := store.CreateMessage(ctx, conversation.ID, alice.ID, "   "); !errors.Is(err, ErrMessageBodyRequired) {
		t.Fatalf("expected empty message to fail with ErrMessageBodyRequired, got %v", err)
	}
	if _, err := store.CreateMessage(ctx, conversation.ID, alice.ID, strings.Repeat("a", maxMessageBodyLength+1)); !errors.Is(err, ErrMessageBodyTooLong) {
		t.Fatalf("expected long message to fail with ErrMessageBodyTooLong, got %v", err)
	}

	store.now = func() time.Time { return fixedNow.Add(2 * time.Minute) }
	second, err := store.CreateMessage(ctx, conversation.ID, bob.ID, "Reply")
	if err != nil {
		t.Fatalf("CreateMessage(bob) error = %v", err)
	}

	messages, err := store.ListMessages(ctx, conversation.ID, bob.ID, 50)
	if err != nil {
		t.Fatalf("ListMessages(bob) error = %v", err)
	}
	if len(messages) != 2 || messages[0].ID != first.ID || messages[1].ID != second.ID {
		t.Fatalf("expected messages oldest first, got %+v", messages)
	}
	if _, err := store.ListMessages(ctx, conversation.ID, carol.ID, 50); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected unrelated message list to fail with ErrMessageForbidden, got %v", err)
	}

	updatedConversations, err := store.ListConversationsForUser(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListConversationsForUser(after messages) error = %v", err)
	}
	if updatedConversations[0].LastMessage == nil || updatedConversations[0].LastMessage.ID != second.ID || updatedConversations[0].LastMessage.Body != "Reply" {
		t.Fatalf("expected safe last message preview, got %+v", updatedConversations[0].LastMessage)
	}

	if err := store.DeleteMessage(ctx, first.ID, bob.ID); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected deleting another user's message to fail with ErrMessageForbidden, got %v", err)
	}
	if err := store.DeleteMessage(ctx, first.ID, alice.ID); err != nil {
		t.Fatalf("DeleteMessage(alice) error = %v", err)
	}

	afterDelete, err := store.ListMessages(ctx, conversation.ID, alice.ID, 50)
	if err != nil {
		t.Fatalf("ListMessages(after delete) error = %v", err)
	}
	if len(afterDelete) != 1 || afterDelete[0].ID != second.ID {
		t.Fatalf("expected deleted message to be hidden, got %+v", afterDelete)
	}
	var storedBody string
	var deletedAt sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT body, deleted_at FROM messages WHERE id = ?`, first.ID).Scan(&storedBody, &deletedAt); err != nil {
		t.Fatalf("read deleted message row: %v", err)
	}
	if storedBody != "" || !deletedAt.Valid {
		t.Fatalf("expected deleted message body to be cleared and timestamped, body=%q deleted_at=%+v", storedBody, deletedAt)
	}
}

func assertConversationParticipants(t *testing.T, conversation Conversation, usernames ...string) {
	t.Helper()
	got := map[string]bool{}
	for _, participant := range conversation.Participants {
		got[participant.Username] = true
		if participant.DisplayName == "" || participant.AvatarKey == "" {
			t.Fatalf("expected safe participant fields, got %+v", participant)
		}
	}
	for _, username := range usernames {
		if !got[username] {
			t.Fatalf("expected participant %q in %+v", username, conversation.Participants)
		}
	}
}
