// Package session provides session file persistence for the commgraph CLI.
package session

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/grokify/commgraph/entity"
)

// Session represents the state of a commgraph session that can be persisted to disk.
type Session struct {
	// Version is the session file format version.
	Version string `json:"version"`

	// CreatedAt is when the session was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the session was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// Source is the original data source (file path).
	Source string `json:"source,omitempty"`

	// Config holds resolver configuration.
	Config SessionConfig `json:"config"`

	// Messages contains all ingested messages.
	Messages []*entity.Message `json:"messages"`

	// Interactions contains all extracted interactions.
	Interactions []*entity.Interaction `json:"interactions"`

	// Actors contains all resolved actors.
	Actors []*entity.Actor `json:"actors"`

	// Threads contains reconstructed message threads.
	Threads []*entity.Thread `json:"threads"`

	// Stats contains summary statistics.
	Stats SessionStats `json:"stats"`
}

// SessionConfig holds configuration used during ingestion.
type SessionConfig struct {
	InternalDomains []string `json:"internal_domains,omitempty"`
	AutoCreate      bool     `json:"auto_create"`
	Format          string   `json:"format,omitempty"`
}

// SessionStats holds summary statistics.
type SessionStats struct {
	MessageCount     int `json:"message_count"`
	InteractionCount int `json:"interaction_count"`
	ActorCount       int `json:"actor_count"`
	ThreadCount      int `json:"thread_count"`
	InternalActors   int `json:"internal_actors"`
	ExternalActors   int `json:"external_actors"`
}

// CurrentVersion is the current session file format version.
const CurrentVersion = "1.0"

// New creates a new empty session.
func New() *Session {
	now := time.Now()
	return &Session{
		Version:      CurrentVersion,
		CreatedAt:    now,
		UpdatedAt:    now,
		Messages:     make([]*entity.Message, 0),
		Interactions: make([]*entity.Interaction, 0),
		Actors:       make([]*entity.Actor, 0),
		Threads:      make([]*entity.Thread, 0),
	}
}

// Save writes the session to a file.
func (s *Session) Save(path string) error {
	s.UpdatedAt = time.Now()
	s.updateStats()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating session file: %w", err)
	}
	defer f.Close()

	return s.WriteTo(f)
}

// WriteTo writes the session to a writer.
func (s *Session) WriteTo(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(s); err != nil {
		return fmt.Errorf("encoding session: %w", err)
	}
	return nil
}

// Load reads a session from a file.
func Load(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer f.Close()

	return ReadFrom(f)
}

// ReadFrom reads a session from a reader.
func ReadFrom(r io.Reader) (*Session, error) {
	var s Session
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding session: %w", err)
	}
	return &s, nil
}

// updateStats updates the summary statistics.
func (s *Session) updateStats() {
	s.Stats.MessageCount = len(s.Messages)
	s.Stats.InteractionCount = len(s.Interactions)
	s.Stats.ActorCount = len(s.Actors)
	s.Stats.ThreadCount = len(s.Threads)

	internal := 0
	external := 0
	for _, actor := range s.Actors {
		if actor.Internal {
			internal++
		} else {
			external++
		}
	}
	s.Stats.InternalActors = internal
	s.Stats.ExternalActors = external
}

// AddMessage adds a message to the session.
func (s *Session) AddMessage(msg *entity.Message) {
	s.Messages = append(s.Messages, msg)
}

// AddInteraction adds an interaction to the session.
func (s *Session) AddInteraction(interaction *entity.Interaction) {
	s.Interactions = append(s.Interactions, interaction)
}

// AddActor adds an actor to the session.
func (s *Session) AddActor(actor *entity.Actor) {
	s.Actors = append(s.Actors, actor)
}

// AddThread adds a thread to the session.
func (s *Session) AddThread(thread *entity.Thread) {
	s.Threads = append(s.Threads, thread)
}

// SetConfig sets the session configuration.
func (s *Session) SetConfig(config SessionConfig) {
	s.Config = config
}

// SetSource sets the original data source.
func (s *Session) SetSource(source string) {
	s.Source = source
}

// Exists checks if a session file exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DefaultPath returns the default session file path.
func DefaultPath() string {
	return ".commgraph-session.json"
}
