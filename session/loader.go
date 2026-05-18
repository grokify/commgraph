package session

import (
	"context"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/identity"
	"github.com/grokify/commgraph/storage"
)

// ToMemoryStore creates a MemoryStore populated with session data.
func (s *Session) ToMemoryStore(ctx context.Context) (*storage.MemoryStore, error) {
	store := storage.NewMemoryStore()

	// Store messages
	for _, msg := range s.Messages {
		if err := store.StoreMessage(ctx, msg); err != nil {
			return nil, err
		}
	}

	// Store interactions
	for _, interaction := range s.Interactions {
		if err := store.StoreInteraction(ctx, interaction); err != nil {
			return nil, err
		}
	}

	// Store actors
	for _, actor := range s.Actors {
		if err := store.StoreActor(ctx, actor); err != nil {
			return nil, err
		}
	}

	// Store threads
	for _, thread := range s.Threads {
		if err := store.StoreThread(ctx, thread); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// ToResolver creates a SCIMResolver populated with session actors.
func (s *Session) ToResolver() *identity.SCIMResolver {
	config := identity.DefaultConfig()
	config.InternalDomains = s.Config.InternalDomains
	config.AutoCreate = s.Config.AutoCreate

	resolver := identity.NewSCIMResolver(config)

	// Load actors into resolver
	for _, actor := range s.Actors {
		resolver.LoadActor(actor)
	}

	return resolver
}

// FromMemoryStore populates a session from a MemoryStore.
func (s *Session) FromMemoryStore(store *storage.MemoryStore) {
	s.Messages = store.AllMessages()
	s.Interactions = store.AllInteractions()
	s.Actors = store.AllActors()
	s.Threads = store.AllThreads()
}

// FromResolver extracts actors from a SCIMResolver and adds them to the session.
func (s *Session) FromResolver(resolver *identity.SCIMResolver) {
	actors := resolver.AllActors()
	for _, actor := range actors {
		// Check if actor already exists in session
		exists := false
		for _, existing := range s.Actors {
			if existing.ID == actor.ID {
				exists = true
				break
			}
		}
		if !exists {
			s.Actors = append(s.Actors, actor)
		}
	}
}

// LoadIntoStore loads a session file into a memory store and returns both.
func LoadIntoStore(ctx context.Context, path string) (*storage.MemoryStore, *identity.SCIMResolver, error) {
	sess, err := Load(path)
	if err != nil {
		return nil, nil, err
	}

	store, err := sess.ToMemoryStore(ctx)
	if err != nil {
		return nil, nil, err
	}

	resolver := sess.ToResolver()

	return store, resolver, nil
}

// SaveFromStore creates a session from a store and resolver, and saves it.
func SaveFromStore(store *storage.MemoryStore, resolver *identity.SCIMResolver, source string, config SessionConfig, path string) error {
	sess := New()
	sess.SetSource(source)
	sess.SetConfig(config)
	sess.FromMemoryStore(store)
	sess.FromResolver(resolver)

	return sess.Save(path)
}

// AllThreads returns all threads from the MemoryStore.
// This is a helper that calls the store's AllThreads method.
func getAllThreads(store *storage.MemoryStore) []*entity.Thread {
	return store.AllThreads()
}
