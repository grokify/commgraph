// Package threading provides thread reconstruction for messages.
package threading

import (
	"time"

	mogothread "github.com/grokify/mogo/net/mailutil/threading"

	"github.com/grokify/commgraph/entity"
)

// Config configures thread reconstruction.
type Config struct {
	// MaxParentAge is the maximum age difference for subject-based matching.
	MaxParentAge time.Duration

	// RequireParticipantOverlap requires messages to share at least one
	// participant for subject-based matching.
	RequireParticipantOverlap bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		MaxParentAge:              7 * 24 * time.Hour,
		RequireParticipantOverlap: true,
	}
}

// Reconstructor reconstructs threads from messages.
type Reconstructor struct {
	config Config
}

// NewReconstructor creates a new thread reconstructor.
func NewReconstructor(config Config) *Reconstructor {
	return &Reconstructor{config: config}
}

// Reconstruct processes messages and returns threads.
// It modifies the messages in-place to set ThreadID, ParentID, and ThreadDepth.
func (r *Reconstructor) Reconstruct(messages []*entity.Message) ([]*entity.Thread, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	// Create adapters for mogo threading
	adapters := make([]*messageAdapter, len(messages))
	threadable := make([]mogothread.ThreadableMessage, len(messages))
	for i, msg := range messages {
		adapters[i] = &messageAdapter{msg: msg}
		threadable[i] = adapters[i]
	}

	// Create mogo reconstructor
	mogoConfig := mogothread.Config{
		MaxParentAge:              r.config.MaxParentAge,
		RequireParticipantOverlap: r.config.RequireParticipantOverlap,
	}
	inner := mogothread.NewReconstructorWithConfig(mogoConfig)

	// Run reconstruction
	inner.AddMessages(threadable)
	inner.Reconstruct()

	// Extract results back to messages
	for i, adapter := range adapters {
		messages[i].ThreadID = adapter.info.ThreadID
		messages[i].ParentID = adapter.info.ParentID
		messages[i].ThreadDepth = adapter.info.Depth
	}

	// Convert threads
	mogoThreads := inner.GetThreads()
	threads := make([]*entity.Thread, len(mogoThreads))
	for i, mt := range mogoThreads {
		// Collect participants
		participantSet := make(map[entity.ActorID]bool)
		for _, msgID := range mt.MessageIDs {
			for _, adapter := range adapters {
				if adapter.msg.ID == msgID || adapter.msg.MessageID == msgID {
					// Add From as participant
					if adapter.msg.From != "" {
						participantSet[entity.ActorID(adapter.msg.From)] = true
					}
					// Add all recipients
					for _, addr := range adapter.msg.AllRecipients() {
						participantSet[entity.ActorID(addr)] = true
					}
					break
				}
			}
		}
		participants := make([]entity.ActorID, 0, len(participantSet))
		for p := range participantSet {
			participants = append(participants, p)
		}

		// Calculate depth
		maxDepth := 0
		for _, adapter := range adapters {
			if adapter.info.ThreadID == mt.ID && adapter.info.Depth > maxDepth {
				maxDepth = adapter.info.Depth
			}
		}

		threads[i] = &entity.Thread{
			ID:            mt.ID,
			Subject:       mt.Subject,
			RootMessageID: mt.RootMessageID,
			MessageIDs:    mt.MessageIDs,
			Participants:  participants,
			StartDate:     mt.StartDate,
			EndDate:       mt.EndDate,
			Size:          mt.Size,
			Depth:         maxDepth,
		}
	}

	return threads, nil
}

// messageAdapter adapts entity.Message to mogothread.ThreadableMessage.
type messageAdapter struct {
	msg  *entity.Message
	info mogothread.ThreadingInfo
}

func (m *messageAdapter) GetMessageID() string {
	if m.msg.MessageID != "" {
		return m.msg.MessageID
	}
	return m.msg.ID
}

func (m *messageAdapter) GetDate() time.Time {
	return m.msg.Date
}

func (m *messageAdapter) GetSubject() string {
	return m.msg.Subject
}

func (m *messageAdapter) GetInReplyTo() string {
	return m.msg.InReplyTo
}

func (m *messageAdapter) GetReferences() []string {
	return m.msg.References
}

func (m *messageAdapter) GetParticipants() []string {
	return m.msg.AllParticipants()
}

func (m *messageAdapter) GetEmbeddedMessageHints() []mogothread.EmbeddedHint {
	// TODO: implement body parsing for embedded message hints
	return nil
}

func (m *messageAdapter) SetThreadingInfo(info mogothread.ThreadingInfo) {
	m.info = info
}

// Stats contains threading statistics.
type Stats struct {
	TotalMessages        int `json:"total_messages"`
	TotalThreads         int `json:"total_threads"`
	MessagesWithParent   int `json:"messages_with_parent"`
	SingleMessageThreads int `json:"single_message_threads"`
	MultiMessageThreads  int `json:"multi_message_threads"`
	MaxThreadSize        int `json:"max_thread_size"`
	MaxThreadDepth       int `json:"max_thread_depth"`
}

// ComputeStats computes statistics from threads.
func ComputeStats(threads []*entity.Thread) Stats {
	stats := Stats{
		TotalThreads: len(threads),
	}

	for _, thread := range threads {
		stats.TotalMessages += thread.Size

		if thread.Size == 1 {
			stats.SingleMessageThreads++
		} else {
			stats.MultiMessageThreads++
			stats.MessagesWithParent += thread.Size - 1 // All but root have parent
		}

		if thread.Size > stats.MaxThreadSize {
			stats.MaxThreadSize = thread.Size
		}
		if thread.Depth > stats.MaxThreadDepth {
			stats.MaxThreadDepth = thread.Depth
		}
	}

	return stats
}
