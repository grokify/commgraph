package storage

import (
	"context"
	"sync"

	"github.com/grokify/commgraph/entity"
)

// MemoryStore is an in-memory implementation of Store.
type MemoryStore struct {
	mu           sync.RWMutex
	messages     map[string]*entity.Message
	interactions map[string]*entity.Interaction
	actors       map[entity.ActorID]*entity.Actor
	threads      map[string]*entity.Thread
	closed       bool
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		messages:     make(map[string]*entity.Message),
		interactions: make(map[string]*entity.Interaction),
		actors:       make(map[entity.ActorID]*entity.Actor),
		threads:      make(map[string]*entity.Thread),
	}
}

// StoreMessage stores a message.
func (s *MemoryStore) StoreMessage(ctx context.Context, msg *entity.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	s.messages[msg.ID] = msg
	return nil
}

// GetMessage retrieves a message by ID.
func (s *MemoryStore) GetMessage(ctx context.Context, id string) (*entity.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}
	msg, ok := s.messages[id]
	if !ok {
		return nil, ErrNotFound
	}
	return msg, nil
}

// ListMessages lists messages with optional filtering.
func (s *MemoryStore) ListMessages(ctx context.Context, opts ListOptions) ([]*entity.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}

	var result []*entity.Message
	for _, msg := range s.messages {
		if !opts.After.IsZero() && !msg.Date.After(opts.After) {
			continue
		}
		if !opts.Before.IsZero() && !msg.Date.Before(opts.Before) {
			continue
		}
		if opts.Platform != "" && msg.Platform != opts.Platform {
			continue
		}
		result = append(result, msg)
	}

	// Apply offset and limit
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		return []*entity.Message{}, nil
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// StoreInteraction stores an interaction.
func (s *MemoryStore) StoreInteraction(ctx context.Context, interaction *entity.Interaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	s.interactions[interaction.ID] = interaction
	return nil
}

// GetInteractions retrieves interactions matching the query.
func (s *MemoryStore) GetInteractions(ctx context.Context, query InteractionQuery) ([]*entity.Interaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}

	var result []*entity.Interaction
	for _, interaction := range s.interactions {
		if query.FromActor != "" && interaction.From != query.FromActor {
			continue
		}
		if query.ToActor != "" && interaction.To != query.ToActor {
			continue
		}
		if len(query.EdgeTypes) > 0 && !containsEdgeType(query.EdgeTypes, interaction.EdgeType) {
			continue
		}
		if !query.After.IsZero() && !interaction.Timestamp.After(query.After) {
			continue
		}
		if !query.Before.IsZero() && !interaction.Timestamp.Before(query.Before) {
			continue
		}
		if query.MessageID != "" && interaction.MessageID != query.MessageID {
			continue
		}
		if query.ThreadID != "" && interaction.ThreadID != query.ThreadID {
			continue
		}
		result = append(result, interaction)
	}

	if query.Limit > 0 && query.Limit < len(result) {
		result = result[:query.Limit]
	}

	return result, nil
}

// StoreActor stores an actor.
func (s *MemoryStore) StoreActor(ctx context.Context, actor *entity.Actor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	s.actors[actor.ID] = actor
	return nil
}

// GetActor retrieves an actor by ID.
func (s *MemoryStore) GetActor(ctx context.Context, id entity.ActorID) (*entity.Actor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}
	actor, ok := s.actors[id]
	if !ok {
		return nil, ErrNotFound
	}
	return actor, nil
}

// ListActors lists actors with optional filtering.
func (s *MemoryStore) ListActors(ctx context.Context, opts ListOptions) ([]*entity.Actor, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}

	result := make([]*entity.Actor, 0, len(s.actors))
	for _, actor := range s.actors {
		result = append(result, actor)
	}

	// Apply offset and limit
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		return []*entity.Actor{}, nil
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// StoreThread stores a thread.
func (s *MemoryStore) StoreThread(ctx context.Context, thread *entity.Thread) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	s.threads[thread.ID] = thread
	return nil
}

// GetThread retrieves a thread by ID.
func (s *MemoryStore) GetThread(ctx context.Context, id string) (*entity.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}
	thread, ok := s.threads[id]
	if !ok {
		return nil, ErrNotFound
	}
	return thread, nil
}

// ListThreads lists threads with optional filtering.
func (s *MemoryStore) ListThreads(ctx context.Context, opts ListOptions) ([]*entity.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}

	var result []*entity.Thread
	for _, thread := range s.threads {
		if !opts.After.IsZero() && !thread.StartDate.After(opts.After) {
			continue
		}
		if !opts.Before.IsZero() && !thread.EndDate.Before(opts.Before) {
			continue
		}
		result = append(result, thread)
	}

	// Apply offset and limit
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		return []*entity.Thread{}, nil
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// Stats returns storage statistics.
func (s *MemoryStore) Stats(ctx context.Context) (*Stats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, ErrStoreClosed
	}
	return &Stats{
		MessageCount:     int64(len(s.messages)),
		InteractionCount: int64(len(s.interactions)),
		ActorCount:       int64(len(s.actors)),
		ThreadCount:      int64(len(s.threads)),
	}, nil
}

// Close closes the store.
func (s *MemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Batch operations

// StoreMessages stores multiple messages.
func (s *MemoryStore) StoreMessages(ctx context.Context, msgs []*entity.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	for _, msg := range msgs {
		s.messages[msg.ID] = msg
	}
	return nil
}

// StoreInteractions stores multiple interactions.
func (s *MemoryStore) StoreInteractions(ctx context.Context, interactions []*entity.Interaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	for _, interaction := range interactions {
		s.interactions[interaction.ID] = interaction
	}
	return nil
}

// StoreActors stores multiple actors.
func (s *MemoryStore) StoreActors(ctx context.Context, actors []*entity.Actor) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrStoreClosed
	}
	for _, actor := range actors {
		s.actors[actor.ID] = actor
	}
	return nil
}

// AllMessages returns all messages (for graph building).
func (s *MemoryStore) AllMessages() []*entity.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*entity.Message, 0, len(s.messages))
	for _, msg := range s.messages {
		result = append(result, msg)
	}
	return result
}

// AllInteractions returns all interactions (for graph building).
func (s *MemoryStore) AllInteractions() []*entity.Interaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*entity.Interaction, 0, len(s.interactions))
	for _, interaction := range s.interactions {
		result = append(result, interaction)
	}
	return result
}

// AllActors returns all actors (for graph building).
func (s *MemoryStore) AllActors() []*entity.Actor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*entity.Actor, 0, len(s.actors))
	for _, actor := range s.actors {
		result = append(result, actor)
	}
	return result
}

// AllThreads returns all threads (for session persistence).
func (s *MemoryStore) AllThreads() []*entity.Thread {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*entity.Thread, 0, len(s.threads))
	for _, thread := range s.threads {
		result = append(result, thread)
	}
	return result
}

func containsEdgeType(types []entity.EdgeType, t entity.EdgeType) bool {
	for _, et := range types {
		if et == t {
			return true
		}
	}
	return false
}

// Verify interface compliance.
var (
	_ Store      = (*MemoryStore)(nil)
	_ BatchStore = (*MemoryStore)(nil)
)
