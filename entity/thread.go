package entity

import "time"

// Thread represents a collection of related messages.
type Thread struct {
	// ID is the unique identifier for this thread.
	ID string `json:"id"`

	// Subject is the normalized subject line.
	Subject string `json:"subject"`

	// RootMessageID is the ID of the first message in the thread.
	RootMessageID string `json:"root_message_id"`

	// MessageIDs contains all message IDs in the thread, sorted by date.
	MessageIDs []string `json:"message_ids"`

	// Participants contains all actor IDs involved in the thread.
	Participants []ActorID `json:"participants"`

	// StartDate is the timestamp of the first message.
	StartDate time.Time `json:"start_date"`

	// EndDate is the timestamp of the last message.
	EndDate time.Time `json:"end_date"`

	// Size is the number of messages in the thread.
	Size int `json:"size"`

	// Depth is the maximum nesting depth in the thread.
	Depth int `json:"depth"`
}

// Duration returns the time span of the thread.
func (t *Thread) Duration() time.Duration {
	return t.EndDate.Sub(t.StartDate)
}

// IsSingleMessage returns true if the thread contains only one message.
func (t *Thread) IsSingleMessage() bool {
	return t.Size == 1
}
