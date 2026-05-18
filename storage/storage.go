// Package storage provides interfaces for persisting communication graph data.
package storage

import (
	"context"
	"errors"
	"time"

	"github.com/grokify/commgraph/entity"
)

// Common errors.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrStoreClosed   = errors.New("store is closed")
)

// Store provides persistence for communication graph data.
type Store interface {
	// Message operations (immutable once stored)
	StoreMessage(ctx context.Context, msg *entity.Message) error
	GetMessage(ctx context.Context, id string) (*entity.Message, error)
	ListMessages(ctx context.Context, opts ListOptions) ([]*entity.Message, error)

	// Interaction operations
	StoreInteraction(ctx context.Context, interaction *entity.Interaction) error
	GetInteractions(ctx context.Context, query InteractionQuery) ([]*entity.Interaction, error)

	// Actor operations
	StoreActor(ctx context.Context, actor *entity.Actor) error
	GetActor(ctx context.Context, id entity.ActorID) (*entity.Actor, error)
	ListActors(ctx context.Context, opts ListOptions) ([]*entity.Actor, error)

	// Thread operations
	StoreThread(ctx context.Context, thread *entity.Thread) error
	GetThread(ctx context.Context, id string) (*entity.Thread, error)
	ListThreads(ctx context.Context, opts ListOptions) ([]*entity.Thread, error)

	// Stats returns storage statistics.
	Stats(ctx context.Context) (*Stats, error)

	// Close closes the store and releases resources.
	Close() error
}

// ListOptions configures list queries.
type ListOptions struct {
	// After filters to items after this timestamp.
	After time.Time

	// Before filters to items before this timestamp.
	Before time.Time

	// Limit is the maximum number of items to return.
	Limit int

	// Offset is the number of items to skip.
	Offset int

	// Platform filters by platform.
	Platform string
}

// InteractionQuery configures interaction queries.
type InteractionQuery struct {
	// FromActor filters by source actor.
	FromActor entity.ActorID

	// ToActor filters by target actor.
	ToActor entity.ActorID

	// EdgeTypes filters by edge types.
	EdgeTypes []entity.EdgeType

	// After filters to interactions after this timestamp.
	After time.Time

	// Before filters to interactions before this timestamp.
	Before time.Time

	// MessageID filters by message.
	MessageID string

	// ThreadID filters by thread.
	ThreadID string

	// Limit is the maximum number of items to return.
	Limit int
}

// Stats contains storage statistics.
type Stats struct {
	// MessageCount is the total number of messages.
	MessageCount int64 `json:"message_count"`

	// InteractionCount is the total number of interactions.
	InteractionCount int64 `json:"interaction_count"`

	// ActorCount is the total number of actors.
	ActorCount int64 `json:"actor_count"`

	// ThreadCount is the total number of threads.
	ThreadCount int64 `json:"thread_count"`

	// StorageBytes is the total storage used in bytes.
	StorageBytes int64 `json:"storage_bytes,omitempty"`
}

// BatchStore extends Store with batch operations for efficiency.
type BatchStore interface {
	Store

	// StoreMessages stores multiple messages in a batch.
	StoreMessages(ctx context.Context, msgs []*entity.Message) error

	// StoreInteractions stores multiple interactions in a batch.
	StoreInteractions(ctx context.Context, interactions []*entity.Interaction) error

	// StoreActors stores multiple actors in a batch.
	StoreActors(ctx context.Context, actors []*entity.Actor) error
}
