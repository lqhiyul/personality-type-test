package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreAddAndDeletePersist(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "results.json")
	store, err := NewStore(file)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	first := Result{
		ID:       "one",
		Name:     "First",
		Type:     "INTJ",
		Answers:  "EIEEIIISSNNSNSTFTTFTFPJPPJPP",
		Duration: 30,
		Created:  time.Unix(100, 0).UTC(),
	}
	second := Result{
		ID:       "two",
		Name:     "Second",
		Type:     "INFJ",
		Answers:  "IIIIIIISNNNNSSFTFTFTJJPJJPJ",
		Duration: 45,
		Created:  time.Unix(200, 0).UTC(),
	}

	if err := store.Add(first); err != nil {
		t.Fatalf("Add(first) error = %v", err)
	}
	if err := store.Add(second); err != nil {
		t.Fatalf("Add(second) error = %v", err)
	}

	reloaded, err := NewStore(file)
	if err != nil {
		t.Fatalf("NewStore(reload) error = %v", err)
	}

	results, err := reloaded.All()
	if err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 persisted results, got %d", len(results))
	}

	if err := reloaded.DeleteByID("one"); err != nil {
		t.Fatalf("DeleteByID(one) error = %v", err)
	}

	afterDelete, err := NewStore(file)
	if err != nil {
		t.Fatalf("NewStore(after delete) error = %v", err)
	}

	results, err = afterDelete.All()
	if err != nil {
		t.Fatalf("All() after delete error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 persisted result after delete, got %d", len(results))
	}
	if results[0].ID != "two" {
		t.Fatalf("expected remaining id %q, got %q", "two", results[0].ID)
	}

	if err := afterDelete.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	afterClear, err := NewStore(file)
	if err != nil {
		t.Fatalf("NewStore(after clear) error = %v", err)
	}

	results, err = afterClear.All()
	if err != nil {
		t.Fatalf("All() after clear error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results after clear, got %d", len(results))
	}

	matches, err := filepath.Glob(filepath.Join(filepath.Dir(file), filepath.Base(file)+".*.tmp"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected atomic write temp files to be cleaned up, got %v", matches)
	}
}
