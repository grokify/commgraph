package session

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/identity"
	"github.com/grokify/commgraph/storage"
)

func TestToMemoryStore(t *testing.T) {
	ctx := context.Background()
	s := New()

	// Add test data
	s.AddMessage(&entity.Message{
		ID:       "msg-1",
		Platform: "email",
		From:     "alice@test.com",
		Subject:  "Test",
		Date:     time.Now(),
	})
	s.AddInteraction(&entity.Interaction{
		ID:       "int-1",
		From:     "alice",
		To:       "bob",
		EdgeType: entity.EdgeTypeTo,
	})
	s.AddActor(&entity.Actor{
		ID:          "alice",
		DisplayName: "Alice",
		Internal:    true,
	})
	s.AddThread(&entity.Thread{
		ID:      "thread-1",
		Subject: "Test Thread",
		Size:    1,
	})

	// Convert to store
	store, err := s.ToMemoryStore(ctx)
	if err != nil {
		t.Fatalf("ToMemoryStore failed: %v", err)
	}

	// Verify data
	msg, err := store.GetMessage(ctx, "msg-1")
	if err != nil {
		t.Errorf("GetMessage failed: %v", err)
	}
	if msg.Subject != "Test" {
		t.Errorf("Message subject = %q, want %q", msg.Subject, "Test")
	}

	actor, err := store.GetActor(ctx, "alice")
	if err != nil {
		t.Errorf("GetActor failed: %v", err)
	}
	if actor.DisplayName != "Alice" {
		t.Errorf("Actor name = %q, want %q", actor.DisplayName, "Alice")
	}

	thread, err := store.GetThread(ctx, "thread-1")
	if err != nil {
		t.Errorf("GetThread failed: %v", err)
	}
	if thread.Subject != "Test Thread" {
		t.Errorf("Thread subject = %q, want %q", thread.Subject, "Test Thread")
	}

	interactions, err := store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		t.Errorf("GetInteractions failed: %v", err)
	}
	if len(interactions) != 1 {
		t.Errorf("Interactions count = %d, want 1", len(interactions))
	}
}

func TestToResolver(t *testing.T) {
	s := New()
	s.SetConfig(SessionConfig{
		InternalDomains: []string{"test.com"},
		AutoCreate:      true,
	})

	// Add actors
	s.AddActor(&entity.Actor{
		ID:           "alice",
		DisplayName:  "Alice Smith",
		Emails:       []string{"alice@test.com"},
		PrimaryEmail: "alice@test.com",
		Internal:     true,
	})
	s.AddActor(&entity.Actor{
		ID:           "bob",
		DisplayName:  "Bob Jones",
		Emails:       []string{"bob@external.com"},
		PrimaryEmail: "bob@external.com",
		Internal:     false,
	})

	// Convert to resolver
	resolver := s.ToResolver()

	// Verify actors are loaded
	id, err := resolver.Resolve("alice@test.com")
	if err != nil {
		t.Errorf("Resolve alice failed: %v", err)
	}
	if id != "alice" {
		t.Errorf("Resolved ID = %q, want %q", id, "alice")
	}

	id, err = resolver.Resolve("bob@external.com")
	if err != nil {
		t.Errorf("Resolve bob failed: %v", err)
	}
	if id != "bob" {
		t.Errorf("Resolved ID = %q, want %q", id, "bob")
	}

	// Verify internal domains
	if !resolver.IsInternal("newuser@test.com") {
		t.Error("test.com should be internal")
	}
	if resolver.IsInternal("newuser@external.com") {
		t.Error("external.com should not be internal")
	}
}

func TestFromMemoryStore(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()

	// Add data to store
	_ = store.StoreMessage(ctx, &entity.Message{ID: "msg-1"})
	_ = store.StoreInteraction(ctx, &entity.Interaction{ID: "int-1"})
	_ = store.StoreActor(ctx, &entity.Actor{ID: "alice"})
	_ = store.StoreThread(ctx, &entity.Thread{ID: "thread-1"})

	// Create session and load from store
	s := New()
	s.FromMemoryStore(store)

	if len(s.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(s.Messages))
	}
	if len(s.Interactions) != 1 {
		t.Errorf("Interactions count = %d, want 1", len(s.Interactions))
	}
	if len(s.Actors) != 1 {
		t.Errorf("Actors count = %d, want 1", len(s.Actors))
	}
	if len(s.Threads) != 1 {
		t.Errorf("Threads count = %d, want 1", len(s.Threads))
	}
}

func TestFromResolver(t *testing.T) {
	config := identity.DefaultConfig()
	resolver := identity.NewSCIMResolver(config)

	// Add actors to resolver
	resolver.LoadActor(&entity.Actor{
		ID:     "alice",
		Emails: []string{"alice@test.com"},
	})
	resolver.LoadActor(&entity.Actor{
		ID:     "bob",
		Emails: []string{"bob@test.com"},
	})

	// Create session and load from resolver
	s := New()
	s.FromResolver(resolver)

	if len(s.Actors) != 2 {
		t.Errorf("Actors count = %d, want 2", len(s.Actors))
	}

	// Should not duplicate if called again
	s.FromResolver(resolver)
	if len(s.Actors) != 2 {
		t.Errorf("Actors count after second load = %d, want 2", len(s.Actors))
	}
}

func TestLoadIntoStore(t *testing.T) {
	ctx := context.Background()

	// Create and save a session
	s := New()
	s.SetConfig(SessionConfig{
		InternalDomains: []string{"test.com"},
		AutoCreate:      true,
	})
	s.AddMessage(&entity.Message{ID: "msg-1"})
	s.AddInteraction(&entity.Interaction{ID: "int-1"})
	s.AddActor(&entity.Actor{
		ID:     "alice",
		Emails: []string{"alice@test.com"},
	})

	tmpFile, err := os.CreateTemp("", "session-loader-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if err := s.Save(tmpPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load into store
	store, resolver, err := LoadIntoStore(ctx, tmpPath)
	if err != nil {
		t.Fatalf("LoadIntoStore failed: %v", err)
	}

	// Verify store
	msg, err := store.GetMessage(ctx, "msg-1")
	if err != nil {
		t.Errorf("GetMessage failed: %v", err)
	}
	if msg == nil {
		t.Error("Message should not be nil")
	}

	// Verify resolver
	id, err := resolver.Resolve("alice@test.com")
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
	if id != "alice" {
		t.Errorf("Resolved ID = %q, want %q", id, "alice")
	}
}

func TestSaveFromStore(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	config := identity.DefaultConfig()
	config.InternalDomains = []string{"test.com"}
	resolver := identity.NewSCIMResolver(config)

	// Add data
	_ = store.StoreMessage(ctx, &entity.Message{ID: "msg-1"})
	_ = store.StoreActor(ctx, &entity.Actor{
		ID:     "alice",
		Emails: []string{"alice@test.com"},
	})
	resolver.LoadActor(&entity.Actor{
		ID:     "alice",
		Emails: []string{"alice@test.com"},
	})

	// Save to file
	tmpFile, err := os.CreateTemp("", "session-save-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	sessionConfig := SessionConfig{
		InternalDomains: []string{"test.com"},
		AutoCreate:      true,
		Format:          "mbox",
	}

	err = SaveFromStore(store, resolver, "/test/source.mbox", sessionConfig, tmpPath)
	if err != nil {
		t.Fatalf("SaveFromStore failed: %v", err)
	}

	// Verify by loading
	loaded, err := Load(tmpPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Source != "/test/source.mbox" {
		t.Errorf("Source = %q, want %q", loaded.Source, "/test/source.mbox")
	}
	if loaded.Config.Format != "mbox" {
		t.Errorf("Config.Format = %q, want %q", loaded.Config.Format, "mbox")
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(loaded.Messages))
	}
}

func TestLoadIntoStoreError(t *testing.T) {
	ctx := context.Background()

	_, _, err := LoadIntoStore(ctx, "/nonexistent/session.json")
	if err == nil {
		t.Error("LoadIntoStore should fail for nonexistent file")
	}
}
