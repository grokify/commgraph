package entity

import "time"

// EdgeType represents the type of communication edge.
type EdgeType string

// Edge type constants.
const (
	EdgeTypeTo      EdgeType = "TO"
	EdgeTypeCC      EdgeType = "CC"
	EdgeTypeBCC     EdgeType = "BCC"
	EdgeTypeMention EdgeType = "MENTION"
	EdgeTypeReply   EdgeType = "REPLY"
)

// Interaction represents a directed communication edge between actors.
type Interaction struct {
	// ID is the unique identifier for this interaction.
	ID string `json:"id"`

	// MessageID is the source message that created this interaction.
	MessageID string `json:"message_id"`

	// ThreadID is the thread this interaction belongs to.
	ThreadID string `json:"thread_id,omitempty"`

	// From is the source actor.
	From ActorID `json:"from"`

	// To is the target actor.
	To ActorID `json:"to"`

	// EdgeType is the type of interaction.
	EdgeType EdgeType `json:"edge_type"`

	// Timestamp is when the interaction occurred.
	Timestamp time.Time `json:"timestamp"`

	// Platform is the source platform.
	Platform string `json:"platform"`
}

// AllEdgeTypes returns all defined edge types.
func AllEdgeTypes() []EdgeType {
	return []EdgeType{
		EdgeTypeTo,
		EdgeTypeCC,
		EdgeTypeBCC,
		EdgeTypeMention,
		EdgeTypeReply,
	}
}

// String returns the string representation of the edge type.
func (e EdgeType) String() string {
	return string(e)
}
