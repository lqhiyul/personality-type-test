package app

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestUserStoreBlocksLifecycleAndFriendshipCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	alice := createStoreUserForFriendTest(t, store, "block_alice")
	bob := createStoreUserForFriendTest(t, store, "block_bob")
	carol := createStoreUserForFriendTest(t, store, "block_carol")

	if _, err := store.BlockUser(ctx, alice.ID, alice.ID); !errors.Is(err, ErrBlockSelf) {
		t.Fatalf("expected self block ErrBlockSelf, got %v", err)
	}

	request, err := store.CreateFriendRequest(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("CreateFriendRequest() error = %v", err)
	}
	if _, err := store.AcceptFriendRequest(ctx, bob.ID, request.ID); err != nil {
		t.Fatalf("AcceptFriendRequest() error = %v", err)
	}

	block, err := store.BlockUser(ctx, alice.ID, bob.ID)
	if err != nil {
		t.Fatalf("BlockUser() error = %v", err)
	}
	if block.BlockerUserID != alice.ID || block.BlockedUserID != bob.ID || block.BlockedUser.Username != bob.Username {
		t.Fatalf("unexpected block response: %+v", block)
	}
	if _, err := store.BlockUser(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("duplicate BlockUser should be idempotent, got %v", err)
	}
	if _, err := store.GetFriendshipBetween(ctx, alice.ID, bob.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected block to remove friendship, got %v", err)
	}
	if _, err := store.CreateFriendRequest(ctx, bob.ID, alice.ID); !errors.Is(err, ErrBlockedInteraction) {
		t.Fatalf("expected blocked friend request to fail, got %v", err)
	}

	blockedByAlice, err := store.IsUserBlockedBy(ctx, alice.ID, bob.ID)
	if err != nil || !blockedByAlice {
		t.Fatalf("expected Alice to block Bob, blocked=%v err=%v", blockedByAlice, err)
	}
	blockedBetween, err := store.IsBlockedBetween(ctx, bob.ID, alice.ID)
	if err != nil || !blockedBetween {
		t.Fatalf("expected block between Alice/Bob, blocked=%v err=%v", blockedBetween, err)
	}
	notBlocked, err := store.IsBlockedBetween(ctx, alice.ID, carol.ID)
	if err != nil || notBlocked {
		t.Fatalf("expected Alice/Carol not blocked, blocked=%v err=%v", notBlocked, err)
	}

	blocks, err := store.ListBlockedUsers(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListBlockedUsers() error = %v", err)
	}
	if len(blocks) != 1 || blocks[0].BlockedUser.ID != bob.ID {
		t.Fatalf("unexpected blocks list: %+v", blocks)
	}

	if err := store.UnblockUser(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("UnblockUser() error = %v", err)
	}
	if err := store.UnblockUser(ctx, alice.ID, bob.ID); err != nil {
		t.Fatalf("missing unblock should be clean, got %v", err)
	}
	blockedBetween, err = store.IsBlockedBetween(ctx, alice.ID, bob.ID)
	if err != nil || blockedBetween {
		t.Fatalf("expected Alice/Bob unblocked, blocked=%v err=%v", blockedBetween, err)
	}
	if _, err := store.CreateFriendRequest(ctx, bob.ID, alice.ID); err != nil {
		t.Fatalf("friend request should work after unblock, got %v", err)
	}
}

func TestUserStoreReportsValidationListAndStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, store := newTestUserStore(t)
	reporter := createStoreUserForFriendTest(t, store, "reporter")
	target := createStoreUserForFriendTest(t, store, "report_target")

	if _, err := store.CreateReport(ctx, reporter.ID, "bad", target.ID, target.ID, "spam", ""); !errors.Is(err, ErrReportTargetInvalid) {
		t.Fatalf("expected invalid target type, got %v", err)
	}
	if _, err := store.CreateReport(ctx, reporter.ID, ReportTargetProfile, target.ID, target.ID, "   ", ""); !errors.Is(err, ErrReportReasonRequired) {
		t.Fatalf("expected empty reason, got %v", err)
	}
	if _, err := store.CreateReport(ctx, reporter.ID, ReportTargetProfile, target.ID, target.ID, "spam", strings.Repeat("x", maxReportDetailsLength+1)); !errors.Is(err, ErrReportDetailsTooLong) {
		t.Fatalf("expected long details, got %v", err)
	}

	report, err := store.CreateReport(ctx, reporter.ID, ReportTargetProfile, target.ID, target.ID, "  spam  ", "  public profile looks fake  ")
	if err != nil {
		t.Fatalf("CreateReport() error = %v", err)
	}
	if report.Status != ReportStatusOpen || report.Reason != "spam" || report.Details != "public profile looks fake" {
		t.Fatalf("unexpected report: %+v", report)
	}
	if report.Reporter.Username != reporter.Username || report.TargetUser == nil || report.TargetUser.Username != target.Username {
		t.Fatalf("expected safe reporter/target users, got %+v", report)
	}

	reports, err := store.ListReportsForAdmin(ctx, 10, ReportStatusOpen)
	if err != nil {
		t.Fatalf("ListReportsForAdmin() error = %v", err)
	}
	if len(reports) != 1 || reports[0].ID != report.ID {
		t.Fatalf("unexpected report list: %+v", reports)
	}
	updated, err := store.UpdateReportStatus(ctx, report.ID, ReportStatusReviewed)
	if err != nil {
		t.Fatalf("UpdateReportStatus() error = %v", err)
	}
	if updated.Status != ReportStatusReviewed || updated.ReviewedAt == nil {
		t.Fatalf("expected reviewed report with reviewed_at, got %+v", updated)
	}
	if _, err := store.UpdateReportStatus(ctx, report.ID, "closed"); !errors.Is(err, ErrReportStatusInvalid) {
		t.Fatalf("expected invalid status, got %v", err)
	}
}
