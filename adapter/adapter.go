// Package adapter provides interfaces for ingesting messages from various platforms.
package adapter

import (
	"context"
	"time"

	"github.com/grokify/commgraph/entity"
)

// Adapter ingests messages from a specific platform.
type Adapter interface {
	// Name returns the adapter identifier (e.g., "email", "slack").
	Name() string

	// Ingest reads messages from the source and emits them to the channel.
	// The channel is closed when ingestion completes or an error occurs.
	Ingest(ctx context.Context, source Source) (<-chan *entity.Message, <-chan error)

	// IngestIncremental ingests only messages newer than the checkpoint.
	IngestIncremental(ctx context.Context, source Source, checkpoint Checkpoint) (<-chan *entity.Message, <-chan error)
}

// Source represents an ingestion source.
type Source interface {
	// Type returns the source type (e.g., "mbox", "eml", "pst", "slack_export").
	Type() string

	// Location returns the source location (file path, URL, etc.).
	Location() string
}

// FileSource represents a file-based source.
type FileSource struct {
	// SourceType is the type of file (e.g., "mbox", "eml").
	SourceType string

	// Path is the file or directory path.
	Path string
}

// Type implements Source.
func (s FileSource) Type() string {
	return s.SourceType
}

// Location implements Source.
func (s FileSource) Location() string {
	return s.Path
}

// Checkpoint represents incremental ingestion state.
type Checkpoint struct {
	// AdapterName is the adapter that created this checkpoint.
	AdapterName string `json:"adapter_name"`

	// LastMessageID is the ID of the last ingested message.
	LastMessageID string `json:"last_message_id"`

	// LastTimestamp is the timestamp of the last ingested message.
	LastTimestamp time.Time `json:"last_timestamp"`

	// MessageCount is the total number of messages ingested.
	MessageCount int64 `json:"message_count"`

	// Metadata contains adapter-specific checkpoint data.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Result contains the results of an ingestion operation.
type Result struct {
	// MessagesIngested is the number of messages successfully ingested.
	MessagesIngested int64 `json:"messages_ingested"`

	// MessagesSkipped is the number of messages skipped (duplicates, errors).
	MessagesSkipped int64 `json:"messages_skipped"`

	// Errors contains any non-fatal errors encountered.
	Errors []error `json:"errors,omitempty"`

	// Checkpoint is the checkpoint for incremental ingestion.
	Checkpoint Checkpoint `json:"checkpoint"`

	// Duration is how long ingestion took.
	Duration time.Duration `json:"duration"`
}
