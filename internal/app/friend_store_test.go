package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestUserStoreFriendshipsLifecycleAndGuards(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	alice := createStoreUserForFriendTest(t, store, "alice")
	bob := createStoreUserForFriendTest(t, store, "bob")
	carol := createStoreUserForFriendTest(t, store, "carol")

	if _, err := store.CreateFriendRequest(ctx, alice.ID, alice.ID); !errors.Is(err, ErrFriendshipSelf) {
		t.Fatalf("expected self friend request to fail with ErrFriendshipSelf, got %v", err)
	}

	request, err := store.CreateFriendRequest(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("CreateFriendRequest() error = %v", err)
	}
	if request.RequesterID != alice.ID || request.AddresseeID != bob.ID || request.Status != FriendshipStatusPending {
		t.Fatalf("unexpected created request: %+v", request)
	}

	if _, err := store.CreateFriendRequest(ctx, alice.ID, bob.ID); !errors.Is(err, ErrFriendshipExists) {
		t.Fatalf("expected duplicate request to fail with ErrFriendshipExists, got %v", err)
	}
	if _, err := store.CreateFriendRequest(ctx, bob.ID, alice.ID); !errors.Is(err, ErrFriendshipExists) {
		t.Fatalf("expected reverse duplicate request to fail with ErrFriendshipExists, got %v", err)
	}

	incoming, err := store.ListIncomingFriendRequests(ctx, bob.ID)
	if err != nil {
		t.Fatalf("ListIncomingFriendRequests() error = %v", err)
	}
	if len(incoming) != 1 || incoming[0].Friendship.ID != request.ID || incoming[0].Requester.ID != alice.ID {
		t.Fatalf("unexpected incoming requests: %+v", incoming)
	}

	if _, err := store.AcceptFriendRequest(ctx, carol.ID, request.ID); !errors.Is(err, ErrFriendshipForbidden) {
		t.Fatalf("expected unrelated user accepting request to fail with ErrFriendshipForbidden, got %v", err)
	}

	accepted, err := store.AcceptFriendRequest(ctx, bob.ID, request.ID)
	if err != nil {
		t.Fatalf("AcceptFriendRequest() error = %v", err)
	}
	if accepted.Status != FriendshipStatusAccepted {
		t.Fatalf("expected accepted friendship, got %+v", accepted)
	}
	if _, err := store.AcceptFriendRequest(ctx, bob.ID, request.ID); !errors.Is(err, ErrFriendshipNotPending) {
		t.Fatalf("expected accepting non-pending request to fail with ErrFriendshipNotPending, got %v", err)
	}

	aliceFriends, err := store.ListFriends(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListFriends(alice) error = %v", err)
	}
	if len(aliceFriends) != 1 || aliceFriends[0].User.ID != bob.ID || aliceFriends[0].Friendship.ID != request.ID {
		t.Fatalf("expected Bob in Alice's friends list, got %+v", aliceFriends)
	}
	bobFriends, err := store.ListFriends(ctx, bob.ID)
	if err != nil {
		t.Fatalf("ListFriends(bob) error = %v", err)
	}
	if len(bobFriends) != 1 || bobFriends[0].User.ID != alice.ID || bobFriends[0].Friendship.ID != request.ID {
		t.Fatalf("expected Alice in Bob's friends list, got %+v", bobFriends)
	}

	if err := store.RemoveFriendship(ctx, carol.ID, request.ID); !errors.Is(err, ErrFriendshipForbidden) {
		t.Fatalf("expected unrelated user removing friendship to fail with ErrFriendshipForbidden, got %v", err)
	}
	if err := store.RemoveFriendship(ctx, alice.ID, request.ID); err != nil {
		t.Fatalf("RemoveFriendship() error = %v", err)
	}
	if _, err := store.GetFriendshipByID(ctx, request.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected removed friendship to return sql.ErrNoRows, got %v", err)
	}
}

func createStoreUserForFriendTest(t *testing.T, store *UserStore, username string) User {
	t.Helper()
	user, err := store.CreateUser(context.Background(), CreateUserParams{
		Username:     username,
		Email:        username + "@example.com",
		PasswordHash: "hash",
		DisplayName:  username,
	})
	if err != nil {
		t.Fatalf("CreateUser(%s) error = %v", username, err)
	}
	return user
}
