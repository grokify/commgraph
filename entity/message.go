package entity

import "time"

// Message represents a communication message (email, chat, etc.).
type Message struct {
	// ID is the unique identifier for this message.
	ID string `json:"id"`

	// Platform identifies the source platform (e.g., "email", "slack").
	Platform string `json:"platform"`

	// RawHash is a hash of the raw message for integrity verification.
	RawHash string `json:"raw_hash"`

	// MessageID is the platform-specific message identifier (e.g., Message-ID header).
	MessageID string `json:"message_id"`

	// InReplyTo is the MessageID of the parent message.
	InReplyTo string `json:"in_reply_to,omitempty"`

	// References contains the chain of MessageIDs in the thread.
	References []string `json:"references,omitempty"`

	// From is the sender's email address.
	From string `json:"from"`

	// To contains direct recipient addresses.
	To []string `json:"to"`

	// CC contains carbon copy recipient addresses.
	CC []string `json:"cc,omitempty"`

	// BCC contains blind carbon copy recipient addresses.
	BCC []string `json:"bcc,omitempty"`

	// Subject is the message subject.
	Subject string `json:"subject"`

	// Date is when the message was sent.
	Date time.Time `json:"date"`

	// BodyPreview is a truncated preview of the message body.
	BodyPreview string `json:"body_preview,omitempty"`

	// Mentions contains addresses mentioned in the body.
	Mentions []string `json:"mentions,omitempty"`

	// Domains contains external domains referenced in the message.
	Domains []string `json:"domains,omitempty"`

	// ThreadID is the computed thread identifier (populated after reconstruction).
	ThreadID string `json:"thread_id,omitempty"`

	// ParentID is the ID of the parent message in the thread.
	ParentID string `json:"parent_id,omitempty"`

	// ThreadDepth is the nesting depth in the thread (0 for root).
	ThreadDepth int `json:"thread_depth,omitempty"`

	// IngestedAt is when the message was ingested into the system.
	IngestedAt time.Time `json:"ingested_at"`

	// SourcePath is the original file path or source location.
	SourcePath string `json:"source_path,omitempty"`

	// Folder is the mail folder name (e.g., "inbox", "sent").
	Folder string `json:"folder,omitempty"`
}

// AllRecipients returns all recipient addresses (To + CC + BCC).
func (m *Message) AllRecipients() []string {
	recipients := make([]string, 0, len(m.To)+len(m.CC)+len(m.BCC))
	recipients = append(recipients, m.To...)
	recipients = append(recipients, m.CC...)
	recipients = append(recipients, m.BCC...)
	return recipients
}

// AllParticipants returns all addresses involved (From + all recipients).
func (m *Message) AllParticipants() []string {
	participants := make([]string, 0, 1+len(m.To)+len(m.CC)+len(m.BCC))
	participants = append(participants, m.From)
	participants = append(participants, m.To...)
	participants = append(participants, m.CC...)
	participants = append(participants, m.BCC...)
	return participants
}
