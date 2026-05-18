package session

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/grokify/commgraph/entity"
)

func TestNew(t *testing.T) {
	s := New()

	if s == nil {
		t.Fatal("New() returned nil")
	}
	if s.Version != CurrentVersion {
		t.Errorf("Version = %q, want %q", s.Version, CurrentVersion)
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if s.Messages == nil {
		t.Error("Messages should be initialized")
	}
	if s.Interactions == nil {
		t.Error("Interactions should be initialized")
	}
	if s.Actors == nil {
		t.Error("Actors should be initialized")
	}
	if s.Threads == nil {
		t.Error("Threads should be initialized")
	}
}

func TestSessionAddMessage(t *testing.T) {
	s := New()

	msg := &entity.Message{
		ID:       "msg-001",
		Platform: "email",
		From:     "alice@test.com",
		Subject:  "Test",
	}
	s.AddMessage(msg)

	if len(s.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(s.Messages))
	}
	if s.Messages[0].ID != "msg-001" {
		t.Errorf("Message ID = %q, want %q", s.Messages[0].ID, "msg-001")
	}
}

func TestSessionAddInteraction(t *testing.T) {
	s := New()

	interaction := &entity.Interaction{
		ID:       "int-001",
		From:     "alice",
		To:       "bob",
		EdgeType: entity.EdgeTypeTo,
	}
	s.AddInteraction(interaction)

	if len(s.Interactions) != 1 {
		t.Errorf("Interactions count = %d, want 1", len(s.Interactions))
	}
}

func TestSessionAddActor(t *testing.T) {
	s := New()

	actor := &entity.Actor{
		ID:          "alice",
		DisplayName: "Alice Smith",
		Internal:    true,
	}
	s.AddActor(actor)

	if len(s.Actors) != 1 {
		t.Errorf("Actors count = %d, want 1", len(s.Actors))
	}
	if s.Actors[0].DisplayName != "Alice Smith" {
		t.Errorf("Actor name = %q, want %q", s.Actors[0].DisplayName, "Alice Smith")
	}
}

func TestSessionAddThread(t *testing.T) {
	s := New()

	thread := &entity.Thread{
		ID:      "thread-001",
		Subject: "Test Thread",
		Size:    3,
	}
	s.AddThread(thread)

	if len(s.Threads) != 1 {
		t.Errorf("Threads count = %d, want 1", len(s.Threads))
	}
}

func TestSessionSetConfig(t *testing.T) {
	s := New()

	config := SessionConfig{
		InternalDomains: []string{"enron.com"},
		AutoCreate:      true,
		Format:          "mbox",
	}
	s.SetConfig(config)

	if len(s.Config.InternalDomains) != 1 {
		t.Errorf("InternalDomains count = %d, want 1", len(s.Config.InternalDomains))
	}
	if s.Config.InternalDomains[0] != "enron.com" {
		t.Errorf("InternalDomains[0] = %q, want %q", s.Config.InternalDomains[0], "enron.com")
	}
	if !s.Config.AutoCreate {
		t.Error("AutoCreate should be true")
	}
}

func TestSessionSetSource(t *testing.T) {
	s := New()
	s.SetSource("/path/to/data.mbox")

	if s.Source != "/path/to/data.mbox" {
		t.Errorf("Source = %q, want %q", s.Source, "/path/to/data.mbox")
	}
}

func TestSessionWriteToReadFrom(t *testing.T) {
	// Create a session with data
	s := New()
	s.SetSource("/test/source.mbox")
	s.SetConfig(SessionConfig{
		InternalDomains: []string{"test.com"},
		AutoCreate:      true,
	})
	s.AddMessage(&entity.Message{
		ID:       "msg-1",
		Platform: "email",
		From:     "alice@test.com",
		To:       []string{"bob@test.com"},
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

	// Write to buffer
	var buf bytes.Buffer
	err := s.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	// Read back
	loaded, err := ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}

	// Verify
	if loaded.Version != s.Version {
		t.Errorf("Version = %q, want %q", loaded.Version, s.Version)
	}
	if loaded.Source != s.Source {
		t.Errorf("Source = %q, want %q", loaded.Source, s.Source)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(loaded.Messages))
	}
	if len(loaded.Interactions) != 1 {
		t.Errorf("Interactions count = %d, want 1", len(loaded.Interactions))
	}
	if len(loaded.Actors) != 1 {
		t.Errorf("Actors count = %d, want 1", len(loaded.Actors))
	}
	if len(loaded.Threads) != 1 {
		t.Errorf("Threads count = %d, want 1", len(loaded.Threads))
	}
}

func TestSessionSaveLoad(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "session-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create and save session
	s := New()
	s.AddMessage(&entity.Message{ID: "msg-1"})
	s.AddActor(&entity.Actor{ID: "alice", Internal: true})
	s.AddActor(&entity.Actor{ID: "bob", Internal: false})

	err = s.Save(tmpPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load session
	loaded, err := Load(tmpPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(loaded.Messages))
	}
	if len(loaded.Actors) != 2 {
		t.Errorf("Actors count = %d, want 2", len(loaded.Actors))
	}

	// Check stats were updated
	if loaded.Stats.MessageCount != 1 {
		t.Errorf("Stats.MessageCount = %d, want 1", loaded.Stats.MessageCount)
	}
	if loaded.Stats.ActorCount != 2 {
		t.Errorf("Stats.ActorCount = %d, want 2", loaded.Stats.ActorCount)
	}
	if loaded.Stats.InternalActors != 1 {
		t.Errorf("Stats.InternalActors = %d, want 1", loaded.Stats.InternalActors)
	}
	if loaded.Stats.ExternalActors != 1 {
		t.Errorf("Stats.ExternalActors = %d, want 1", loaded.Stats.ExternalActors)
	}
}

func TestSessionLoadError(t *testing.T) {
	_, err := Load("/nonexistent/path/session.json")
	if err == nil {
		t.Error("Load should fail for nonexistent file")
	}
}

func TestExists(t *testing.T) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "session-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	if !Exists(tmpPath) {
		t.Error("Exists should return true for existing file")
	}

	if Exists("/nonexistent/path/session.json") {
		t.Error("Exists should return false for nonexistent file")
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Error("DefaultPath should return non-empty string")
	}
	if path != ".commgraph-session.json" {
		t.Errorf("DefaultPath = %q, want %q", path, ".commgraph-session.json")
	}
}

func TestUpdateStats(t *testing.T) {
	s := New()

	// Add mixed data
	s.AddMessage(&entity.Message{ID: "1"})
	s.AddMessage(&entity.Message{ID: "2"})
	s.AddInteraction(&entity.Interaction{ID: "i1"})
	s.AddActor(&entity.Actor{ID: "a1", Internal: true})
	s.AddActor(&entity.Actor{ID: "a2", Internal: false})
	s.AddActor(&entity.Actor{ID: "a3", Internal: true})
	s.AddThread(&entity.Thread{ID: "t1"})

	// Stats should be updated on save
	var buf bytes.Buffer
	err := s.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	// The session should have updated stats
	// Note: WriteTo doesn't call updateStats, Save does
	// Let's create a temp file to test
	tmpFile, err := os.CreateTemp("", "session-test-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	err = s.Save(tmpPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if s.Stats.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", s.Stats.MessageCount)
	}
	if s.Stats.InteractionCount != 1 {
		t.Errorf("InteractionCount = %d, want 1", s.Stats.InteractionCount)
	}
	if s.Stats.ActorCount != 3 {
		t.Errorf("ActorCount = %d, want 3", s.Stats.ActorCount)
	}
	if s.Stats.ThreadCount != 1 {
		t.Errorf("ThreadCount = %d, want 1", s.Stats.ThreadCount)
	}
	if s.Stats.InternalActors != 2 {
		t.Errorf("InternalActors = %d, want 2", s.Stats.InternalActors)
	}
	if s.Stats.ExternalActors != 1 {
		t.Errorf("ExternalActors = %d, want 1", s.Stats.ExternalActors)
	}
}
