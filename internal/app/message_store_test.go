package app

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestUserStoreMessagesLifecycleAndAccessControl(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	alice := createStoreUserForFriendTest(t, store, "message_alice")
	bob := createStoreUserForFriendTest(t, store, "message_bob")
	carol := createStoreUserForFriendTest(t, store, "message_carol")

	if _, err := store.CreateConversation(ctx, alice.ID, alice.ID); !errors.Is(err, ErrMessageSelf) {
		t.Fatalf("expected self conversation ErrMessageSelf, got %v", err)
	}

	conversation, err := store.CreateConversation(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if conversation.ID == 0 || len(conversation.Participants) != 2 {
		t.Fatalf("unexpected conversation: %+v", conversation)
	}
	existing, err := store.GetOrCreateConversation(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("GetOrCreateConversation() error = %v", err)
	}
	if existing.ID != conversation.ID {
		t.Fatalf("expected existing conversation %d, got %d", conversation.ID, existing.ID)
	}

	aliceConversations, err := store.ListConversationsForUser(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListConversationsForUser() error = %v", err)
	}
	if len(aliceConversations) != 1 || aliceConversations[0].ID != conversation.ID {
		t.Fatalf("unexpected Alice conversations: %+v", aliceConversations)
	}
	if _, err := store.GetConversationForUser(ctx, conversation.ID, carol.ID); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected unrelated conversation read to fail, got %v", err)
	}

	if _, err := store.CreateMessage(ctx, conversation.ID, carol.ID, "hi"); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected non-participant send to fail, got %v", err)
	}
	if _, err := store.CreateMessage(ctx, conversation.ID, alice.ID, "   "); !errors.Is(err, ErrMessageBodyRequired) {
		t.Fatalf("expected empty message to fail, got %v", err)
	}
	if _, err := store.CreateMessage(ctx, conversation.ID, alice.ID, strings.Repeat("x", maxMessageBodyLength+1)); !errors.Is(err, ErrMessageBodyTooLong) {
		t.Fatalf("expected long message to fail, got %v", err)
	}

	message, err := store.CreateMessage(ctx, conversation.ID, alice.ID, "  <b>Hello</b>\nplain text  ")
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if message.Body != "<b>Hello</b>\nplain text" || message.SenderID != alice.ID {
		t.Fatalf("unexpected message: %+v", message)
	}

	messages, err := store.ListMessages(ctx, conversation.ID, bob.ID, 50)
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if len(messages) != 1 || messages[0].ID != message.ID {
		t.Fatalf("unexpected messages: %+v", messages)
	}
	if _, err := store.ListMessages(ctx, conversation.ID, carol.ID, 50); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected unrelated list to fail, got %v", err)
	}
	if err := store.DeleteMessage(ctx, message.ID, bob.ID); !errors.Is(err, ErrMessageForbidden) {
		t.Fatalf("expected deleting another user's message to fail, got %v", err)
	}
	if err := store.DeleteMessage(ctx, message.ID, alice.ID); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	afterDelete, err := store.GetMessageForUser(ctx, message.ID, alice.ID)
	if err != nil {
		t.Fatalf("GetMessageForUser() after delete error = %v", err)
	}
	if afterDelete.Body != "" || afterDelete.DeletedAt == nil {
		t.Fatalf("expected soft-deleted message body hidden, got %+v", afterDelete)
	}
	messages, err = store.ListMessages(ctx, conversation.ID, bob.ID, 50)
	if err != nil {
		t.Fatalf("ListMessages() after delete error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("deleted messages should not be listed, got %+v", messages)
	}
}

func TestUserStoreMessagesRespectBlocking(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	alice := createStoreUserForFriendTest(t, store, "blocked_msg_alice")
	bob := createStoreUserForFriendTest(t, store, "blocked_msg_bob")
	carol := createStoreUserForFriendTest(t, store, "blocked_msg_carol")

	conversation, err := store.CreateConversation(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if _, err := store.BlockUser(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("BlockUser() error = %v", err)
	}
	if _, err := store.GetOrCreateConversation(ctx, alice.ID, bob.ID); !errors.Is(err, ErrBlockedInteraction) {
		t.Fatalf("expected blocked start to fail, got %v", err)
	}
	if _, err := store.CreateMessage(ctx, conversation.ID, bob.ID, "blocked"); !errors.Is(err, ErrBlockedInteraction) {
		t.Fatalf("expected blocked send to fail, got %v", err)
	}
	if _, err := store.CreateConversation(ctx, bob.ID, carol.ID); err != nil {
		t.Fatalf("unrelated conversations should still work, got %v", err)
	}
}
