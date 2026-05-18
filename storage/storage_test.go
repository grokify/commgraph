package storage

import (
	"context"
	"testing"
	"time"

	"github.com/grokify/commgraph/entity"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}

	ctx := context.Background()
	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}
	if stats.MessageCount != 0 || stats.InteractionCount != 0 {
		t.Error("New store should be empty")
	}
}

func TestMemoryStoreMessage(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	msg := &entity.Message{
		ID:       "msg-001",
		Platform: "email",
		From:     "alice@test.com",
		To:       []string{"bob@test.com"},
		Subject:  "Test",
		Date:     time.Now(),
	}

	// Store message
	err := store.StoreMessage(ctx, msg)
	if err != nil {
		t.Fatalf("StoreMessage failed: %v", err)
	}

	// Get message
	retrieved, err := store.GetMessage(ctx, "msg-001")
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}
	if retrieved.ID != "msg-001" {
		t.Errorf("ID mismatch: got %s", retrieved.ID)
	}
	if retrieved.From != "alice@test.com" {
		t.Errorf("From mismatch: got %s", retrieved.From)
	}

	// Get non-existent message
	_, err = store.GetMessage(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreListMessages(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	// Store multiple messages
	msgs := []*entity.Message{
		{ID: "msg-1", Platform: "email", Date: now.Add(-2 * time.Hour)},
		{ID: "msg-2", Platform: "email", Date: now.Add(-1 * time.Hour)},
		{ID: "msg-3", Platform: "slack", Date: now},
	}
	for _, msg := range msgs {
		_ = store.StoreMessage(ctx, msg)
	}

	// List all
	result, err := store.ListMessages(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	// Filter by platform
	result, err = store.ListMessages(ctx, ListOptions{Platform: "email"})
	if err != nil {
		t.Fatalf("ListMessages with platform filter failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 email messages, got %d", len(result))
	}

	// Filter by time
	result, err = store.ListMessages(ctx, ListOptions{After: now.Add(-90 * time.Minute)})
	if err != nil {
		t.Fatalf("ListMessages with time filter failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 recent messages, got %d", len(result))
	}

	// Test limit
	result, err = store.ListMessages(ctx, ListOptions{Limit: 1})
	if err != nil {
		t.Fatalf("ListMessages with limit failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 message with limit, got %d", len(result))
	}
}

func TestMemoryStoreInteraction(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	interaction := &entity.Interaction{
		ID:        "int-001",
		MessageID: "msg-001",
		From:      entity.ActorID("alice"),
		To:        entity.ActorID("bob"),
		EdgeType:  entity.EdgeTypeTo,
		Timestamp: time.Now(),
	}

	// Store interaction
	err := store.StoreInteraction(ctx, interaction)
	if err != nil {
		t.Fatalf("StoreInteraction failed: %v", err)
	}

	// Get interactions
	result, err := store.GetInteractions(ctx, InteractionQuery{})
	if err != nil {
		t.Fatalf("GetInteractions failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 interaction, got %d", len(result))
	}
}

func TestMemoryStoreGetInteractionsQuery(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	now := time.Now()

	// Store multiple interactions
	interactions := []*entity.Interaction{
		{ID: "int-1", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo, Timestamp: now},
		{ID: "int-2", From: "alice", To: "carol", EdgeType: entity.EdgeTypeCC, Timestamp: now},
		{ID: "int-3", From: "bob", To: "alice", EdgeType: entity.EdgeTypeTo, Timestamp: now},
		{ID: "int-4", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo, Timestamp: now.Add(-2 * time.Hour)},
	}
	for _, i := range interactions {
		_ = store.StoreInteraction(ctx, i)
	}

	// Query by FromActor
	result, _ := store.GetInteractions(ctx, InteractionQuery{FromActor: "alice"})
	if len(result) != 3 {
		t.Errorf("FromActor query: expected 3, got %d", len(result))
	}

	// Query by ToActor
	result, _ = store.GetInteractions(ctx, InteractionQuery{ToActor: "bob"})
	if len(result) != 2 {
		t.Errorf("ToActor query: expected 2, got %d", len(result))
	}

	// Query by EdgeType
	result, _ = store.GetInteractions(ctx, InteractionQuery{EdgeTypes: []entity.EdgeType{entity.EdgeTypeCC}})
	if len(result) != 1 {
		t.Errorf("EdgeType query: expected 1, got %d", len(result))
	}

	// Query by time
	result, _ = store.GetInteractions(ctx, InteractionQuery{After: now.Add(-1 * time.Hour)})
	if len(result) != 3 {
		t.Errorf("Time query: expected 3, got %d", len(result))
	}

	// Query with limit
	result, _ = store.GetInteractions(ctx, InteractionQuery{Limit: 2})
	if len(result) != 2 {
		t.Errorf("Limit query: expected 2, got %d", len(result))
	}
}

func TestMemoryStoreActor(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	actor := &entity.Actor{
		ID:          entity.ActorID("alice"),
		DisplayName: "Alice Smith",
		Internal:    true,
	}

	// Store actor
	err := store.StoreActor(ctx, actor)
	if err != nil {
		t.Fatalf("StoreActor failed: %v", err)
	}

	// Get actor
	retrieved, err := store.GetActor(ctx, "alice")
	if err != nil {
		t.Fatalf("GetActor failed: %v", err)
	}
	if retrieved.DisplayName != "Alice Smith" {
		t.Errorf("DisplayName mismatch: got %s", retrieved.DisplayName)
	}

	// Get non-existent actor
	_, err = store.GetActor(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreListActors(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Store multiple actors
	actors := []*entity.Actor{
		{ID: "alice", DisplayName: "Alice"},
		{ID: "bob", DisplayName: "Bob"},
		{ID: "carol", DisplayName: "Carol"},
	}
	for _, a := range actors {
		_ = store.StoreActor(ctx, a)
	}

	// List all
	result, err := store.ListActors(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListActors failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 actors, got %d", len(result))
	}

	// Test limit
	result, _ = store.ListActors(ctx, ListOptions{Limit: 2})
	if len(result) != 2 {
		t.Errorf("Expected 2 actors with limit, got %d", len(result))
	}

	// Test offset
	result, _ = store.ListActors(ctx, ListOptions{Offset: 2})
	if len(result) != 1 {
		t.Errorf("Expected 1 actor with offset, got %d", len(result))
	}
}

func TestMemoryStoreThread(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	thread := &entity.Thread{
		ID:         "thread-001",
		Subject:    "Test Thread",
		MessageIDs: []string{"msg-1", "msg-2"},
		Size:       2,
		StartDate:  time.Now().Add(-1 * time.Hour),
		EndDate:    time.Now(),
	}

	// Store thread
	err := store.StoreThread(ctx, thread)
	if err != nil {
		t.Fatalf("StoreThread failed: %v", err)
	}

	// Get thread
	retrieved, err := store.GetThread(ctx, "thread-001")
	if err != nil {
		t.Fatalf("GetThread failed: %v", err)
	}
	if retrieved.Subject != "Test Thread" {
		t.Errorf("Subject mismatch: got %s", retrieved.Subject)
	}
	if retrieved.Size != 2 {
		t.Errorf("Size mismatch: got %d", retrieved.Size)
	}

	// Get non-existent thread
	_, err = store.GetThread(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStoreStats(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Add some data
	_ = store.StoreMessage(ctx, &entity.Message{ID: "msg-1"})
	_ = store.StoreMessage(ctx, &entity.Message{ID: "msg-2"})
	_ = store.StoreInteraction(ctx, &entity.Interaction{ID: "int-1"})
	_ = store.StoreActor(ctx, &entity.Actor{ID: "actor-1"})
	_ = store.StoreThread(ctx, &entity.Thread{ID: "thread-1"})

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if stats.MessageCount != 2 {
		t.Errorf("MessageCount: got %d, want 2", stats.MessageCount)
	}
	if stats.InteractionCount != 1 {
		t.Errorf("InteractionCount: got %d, want 1", stats.InteractionCount)
	}
	if stats.ActorCount != 1 {
		t.Errorf("ActorCount: got %d, want 1", stats.ActorCount)
	}
	if stats.ThreadCount != 1 {
		t.Errorf("ThreadCount: got %d, want 1", stats.ThreadCount)
	}
}

func TestMemoryStoreClose(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Store should work before close
	err := store.StoreMessage(ctx, &entity.Message{ID: "msg-1"})
	if err != nil {
		t.Fatalf("StoreMessage before close failed: %v", err)
	}

	// Close store
	err = store.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations should fail after close
	err = store.StoreMessage(ctx, &entity.Message{ID: "msg-2"})
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed, got %v", err)
	}

	_, err = store.GetMessage(ctx, "msg-1")
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed on GetMessage, got %v", err)
	}

	_, err = store.Stats(ctx)
	if err != ErrStoreClosed {
		t.Errorf("Expected ErrStoreClosed on Stats, got %v", err)
	}
}

func TestMemoryStoreBatchOperations(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Batch store messages
	msgs := []*entity.Message{
		{ID: "msg-1"},
		{ID: "msg-2"},
		{ID: "msg-3"},
	}
	err := store.StoreMessages(ctx, msgs)
	if err != nil {
		t.Fatalf("StoreMessages failed: %v", err)
	}

	// Verify all stored
	result, _ := store.ListMessages(ctx, ListOptions{})
	if len(result) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(result))
	}

	// Batch store interactions
	interactions := []*entity.Interaction{
		{ID: "int-1"},
		{ID: "int-2"},
	}
	err = store.StoreInteractions(ctx, interactions)
	if err != nil {
		t.Fatalf("StoreInteractions failed: %v", err)
	}

	// Verify all stored
	intResult, _ := store.GetInteractions(ctx, InteractionQuery{})
	if len(intResult) != 2 {
		t.Errorf("Expected 2 interactions, got %d", len(intResult))
	}

	// Batch store actors
	actors := []*entity.Actor{
		{ID: "actor-1"},
		{ID: "actor-2"},
	}
	err = store.StoreActors(ctx, actors)
	if err != nil {
		t.Fatalf("StoreActors failed: %v", err)
	}

	// Verify all stored
	actorResult, _ := store.ListActors(ctx, ListOptions{})
	if len(actorResult) != 2 {
		t.Errorf("Expected 2 actors, got %d", len(actorResult))
	}
}

func TestMemoryStoreAllMethods(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Store some data
	_ = store.StoreMessage(ctx, &entity.Message{ID: "msg-1"})
	_ = store.StoreInteraction(ctx, &entity.Interaction{ID: "int-1"})
	_ = store.StoreActor(ctx, &entity.Actor{ID: "actor-1"})

	// Test AllMessages
	msgs := store.AllMessages()
	if len(msgs) != 1 {
		t.Errorf("AllMessages: expected 1, got %d", len(msgs))
	}

	// Test AllInteractions
	interactions := store.AllInteractions()
	if len(interactions) != 1 {
		t.Errorf("AllInteractions: expected 1, got %d", len(interactions))
	}

	// Test AllActors
	actors := store.AllActors()
	if len(actors) != 1 {
		t.Errorf("AllActors: expected 1, got %d", len(actors))
	}
}

func TestContainsEdgeType(t *testing.T) {
	types := []entity.EdgeType{entity.EdgeTypeTo, entity.EdgeTypeCC}

	if !containsEdgeType(types, entity.EdgeTypeTo) {
		t.Error("Should contain EdgeTypeTo")
	}
	if !containsEdgeType(types, entity.EdgeTypeCC) {
		t.Error("Should contain EdgeTypeCC")
	}
	if containsEdgeType(types, entity.EdgeTypeBCC) {
		t.Error("Should not contain EdgeTypeBCC")
	}
	if containsEdgeType(nil, entity.EdgeTypeTo) {
		t.Error("Empty slice should not contain any type")
	}
}
