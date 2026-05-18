// Package identity provides identity resolution for communication participants.
package identity

import (
	"errors"

	"github.com/grokify/commgraph/entity"
)

// Common errors.
var (
	ErrUnknownActor = errors.New("unknown actor")
	ErrInvalidEmail = errors.New("invalid email address")
)

// Resolver resolves email addresses to canonical actor identities.
type Resolver interface {
	// Resolve returns the canonical ActorID for an address.
	// Returns ErrUnknownActor if the address cannot be resolved.
	Resolve(addr string) (entity.ActorID, error)

	// ResolveOrCreate resolves an address or creates a new actor for unknown addresses.
	ResolveOrCreate(addr string) entity.ActorID

	// IsInternal returns true if the address belongs to the organization.
	IsInternal(addr string) bool

	// GetActor returns full actor details by ID.
	GetActor(id entity.ActorID) (*entity.Actor, error)

	// Aliases returns all known addresses for an actor.
	Aliases(id entity.ActorID) []string

	// Stats returns resolution statistics.
	Stats() Stats
}

// Stats contains identity resolution statistics.
type Stats struct {
	// TotalActors is the number of known actors.
	TotalActors int `json:"total_actors"`

	// InternalActors is the number of internal actors.
	InternalActors int `json:"internal_actors"`

	// ExternalActors is the number of external actors.
	ExternalActors int `json:"external_actors"`

	// TotalAliases is the total number of email aliases.
	TotalAliases int `json:"total_aliases"`

	// ResolutionHits is the number of successful resolutions.
	ResolutionHits int64 `json:"resolution_hits"`

	// ResolutionMisses is the number of failed resolutions.
	ResolutionMisses int64 `json:"resolution_misses"`

	// AutoCreated is the number of actors auto-created from unknown addresses.
	AutoCreated int64 `json:"auto_created"`
}

// Config configures identity resolution.
type Config struct {
	// InternalDomains is the list of domains considered internal.
	InternalDomains []string `json:"internal_domains"`

	// AutoCreate enables automatic actor creation for unknown addresses.
	AutoCreate bool `json:"auto_create"`

	// NormalizeAddresses enables email address normalization.
	NormalizeAddresses bool `json:"normalize_addresses"`
}

// DefaultConfig returns the default identity resolver configuration.
func DefaultConfig() Config {
	return Config{
		InternalDomains:    []string{},
		AutoCreate:         true,
		NormalizeAddresses: true,
	}
}
