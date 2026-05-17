// Package entity defines core domain types for communication graph analysis.
package entity

// ActorID is a unique identifier for an actor (person or role).
type ActorID string

// Actor represents a participant in communication.
type Actor struct {
	// ID is the unique identifier for this actor.
	ID ActorID `json:"id"`

	// DisplayName is the human-readable name.
	DisplayName string `json:"display_name"`

	// Emails contains all known email addresses for this actor.
	Emails []string `json:"emails"`

	// PrimaryEmail is the canonical email address.
	PrimaryEmail string `json:"primary_email"`

	// ExternalID is an identifier from an external system (e.g., x500 DN).
	ExternalID string `json:"external_id,omitempty"`

	// Internal indicates whether this actor belongs to the organization.
	Internal bool `json:"internal"`

	// Department is the organizational unit.
	Department string `json:"department,omitempty"`

	// Title is the job title.
	Title string `json:"title,omitempty"`

	// Timezone is the actor's timezone.
	Timezone string `json:"timezone,omitempty"`

	// Metadata contains additional actor attributes.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ActorType constants for graph node types.
const (
	ActorTypeIndividual = "individual"
	ActorTypeRole       = "role"
	ActorTypeGroup      = "group"
	ActorTypeExternal   = "external"
)
