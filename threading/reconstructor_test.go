package threading

import (
	"testing"
	"time"

	"github.com/grokify/commgraph/entity"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxParentAge != 7*24*time.Hour {
		t.Errorf("MaxParentAge = %v, want %v", config.MaxParentAge, 7*24*time.Hour)
	}
	if !config.RequireParticipantOverlap {
		t.Error("RequireParticipantOverlap should be true by default")
	}
}

func TestNewReconstructor(t *testing.T) {
	config := DefaultConfig()
	r := NewReconstructor(config)

	if r == nil {
		t.Fatal("NewReconstructor returned nil")
	}
	if r.config.MaxParentAge != config.MaxParentAge {
		t.Errorf("config.MaxParentAge = %v, want %v", r.config.MaxParentAge, config.MaxParentAge)
	}
}

func TestReconstructEmptySlice(t *testing.T) {
	r := NewReconstructor(DefaultConfig())
	threads, err := r.Reconstruct(nil)

	if err != nil {
		t.Errorf("Reconstruct(nil) error = %v, want nil", err)
	}
	if threads != nil {
		t.Errorf("Reconstruct(nil) threads = %v, want nil", threads)
	}
}

func TestReconstructSingleMessage(t *testing.T) {
	r := NewReconstructor(DefaultConfig())

	messages := []*entity.Message{
		{
			ID:        "msg-1",
			MessageID: "msg-1@example.com",
			From:      "alice@example.com",
			To:        []string{"bob@example.com"},
			Subject:   "Hello",
			Date:      time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		},
	}

	threads, err := r.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Reconstruct error: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("threads count = %d, want 1", len(threads))
	}

	thread := threads[0]
	if thread.Size != 1 {
		t.Errorf("thread.Size = %d, want 1", thread.Size)
	}
	if thread.Subject != "Hello" {
		t.Errorf("thread.Subject = %q, want %q", thread.Subject, "Hello")
	}

	// Verify message was updated with threading info
	if messages[0].ThreadID == "" {
		t.Error("message ThreadID not set")
	}
}

func TestReconstructThreadWithReply(t *testing.T) {
	r := NewReconstructor(DefaultConfig())

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	messages := []*entity.Message{
		{
			ID:        "msg-1",
			MessageID: "msg-1@example.com",
			From:      "alice@example.com",
			To:        []string{"bob@example.com"},
			Subject:   "Hello",
			Date:      baseTime,
		},
		{
			ID:         "msg-2",
			MessageID:  "msg-2@example.com",
			InReplyTo:  "msg-1@example.com",
			References: []string{"msg-1@example.com"},
			From:       "bob@example.com",
			To:         []string{"alice@example.com"},
			Subject:    "Re: Hello",
			Date:       baseTime.Add(time.Hour),
		},
	}

	threads, err := r.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Reconstruct error: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("threads count = %d, want 1 (both messages in same thread)", len(threads))
	}

	thread := threads[0]
	if thread.Size != 2 {
		t.Errorf("thread.Size = %d, want 2", thread.Size)
	}

	// Verify messages share the same ThreadID
	if messages[0].ThreadID == "" || messages[1].ThreadID == "" {
		t.Error("Messages should have ThreadID set")
	}
	if messages[0].ThreadID != messages[1].ThreadID {
		t.Errorf("Messages should have same ThreadID: %q != %q", messages[0].ThreadID, messages[1].ThreadID)
	}

	// Reply should have parent
	if messages[1].ParentID == "" {
		t.Error("Reply message should have ParentID set")
	}

	// Reply should have greater depth
	if messages[1].ThreadDepth <= messages[0].ThreadDepth {
		t.Errorf("Reply depth %d should be > root depth %d", messages[1].ThreadDepth, messages[0].ThreadDepth)
	}
}

func TestReconstructMultipleThreads(t *testing.T) {
	r := NewReconstructor(DefaultConfig())

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	messages := []*entity.Message{
		{
			ID:        "msg-1",
			MessageID: "msg-1@example.com",
			From:      "alice@example.com",
			To:        []string{"bob@example.com"},
			Subject:   "Thread 1",
			Date:      baseTime,
		},
		{
			ID:        "msg-2",
			MessageID: "msg-2@example.com",
			From:      "carol@example.com",
			To:        []string{"dave@example.com"},
			Subject:   "Thread 2",
			Date:      baseTime.Add(time.Hour),
		},
	}

	threads, err := r.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Reconstruct error: %v", err)
	}

	if len(threads) != 2 {
		t.Errorf("threads count = %d, want 2 (separate threads)", len(threads))
	}

	// Messages should have different ThreadIDs
	if messages[0].ThreadID == messages[1].ThreadID {
		t.Error("Unrelated messages should have different ThreadIDs")
	}
}

func TestReconstructDeepThread(t *testing.T) {
	r := NewReconstructor(DefaultConfig())

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// Create a chain of 4 messages
	messages := []*entity.Message{
		{
			ID:        "msg-1",
			MessageID: "msg-1@example.com",
			From:      "alice@example.com",
			To:        []string{"bob@example.com"},
			Subject:   "Start",
			Date:      baseTime,
		},
		{
			ID:         "msg-2",
			MessageID:  "msg-2@example.com",
			InReplyTo:  "msg-1@example.com",
			References: []string{"msg-1@example.com"},
			From:       "bob@example.com",
			To:         []string{"alice@example.com"},
			Subject:    "Re: Start",
			Date:       baseTime.Add(time.Hour),
		},
		{
			ID:         "msg-3",
			MessageID:  "msg-3@example.com",
			InReplyTo:  "msg-2@example.com",
			References: []string{"msg-1@example.com", "msg-2@example.com"},
			From:       "alice@example.com",
			To:         []string{"bob@example.com"},
			Subject:    "Re: Re: Start",
			Date:       baseTime.Add(2 * time.Hour),
		},
		{
			ID:         "msg-4",
			MessageID:  "msg-4@example.com",
			InReplyTo:  "msg-3@example.com",
			References: []string{"msg-1@example.com", "msg-2@example.com", "msg-3@example.com"},
			From:       "bob@example.com",
			To:         []string{"alice@example.com"},
			Subject:    "Re: Re: Re: Start",
			Date:       baseTime.Add(3 * time.Hour),
		},
	}

	threads, err := r.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Reconstruct error: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("threads count = %d, want 1", len(threads))
	}

	thread := threads[0]
	if thread.Size != 4 {
		t.Errorf("thread.Size = %d, want 4", thread.Size)
	}
	if thread.Depth < 3 {
		t.Errorf("thread.Depth = %d, should be at least 3", thread.Depth)
	}

	// Verify increasing depth
	for i := 1; i < len(messages); i++ {
		if messages[i].ThreadDepth <= messages[i-1].ThreadDepth {
			t.Errorf("message[%d].ThreadDepth %d should be > message[%d].ThreadDepth %d",
				i, messages[i].ThreadDepth, i-1, messages[i-1].ThreadDepth)
		}
	}
}

func TestReconstructWithParticipants(t *testing.T) {
	r := NewReconstructor(DefaultConfig())

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	messages := []*entity.Message{
		{
			ID:        "msg-1",
			MessageID: "msg-1@example.com",
			From:      "alice@example.com",
			To:        []string{"bob@example.com"},
			CC:        []string{"carol@example.com"},
			Subject:   "Hello",
			Date:      baseTime,
		},
		{
			ID:         "msg-2",
			MessageID:  "msg-2@example.com",
			InReplyTo:  "msg-1@example.com",
			From:       "bob@example.com",
			To:         []string{"alice@example.com", "carol@example.com"},
			Subject:    "Re: Hello",
			Date:       baseTime.Add(time.Hour),
		},
	}

	threads, err := r.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Reconstruct error: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("threads count = %d, want 1", len(threads))
	}

	thread := threads[0]

	// Should have all participants: alice, bob, carol
	if len(thread.Participants) < 3 {
		t.Errorf("Participants count = %d, want at least 3 (alice, bob, carol)", len(thread.Participants))
	}

	// Check that key participants are included
	participantMap := make(map[entity.ActorID]bool)
	for _, p := range thread.Participants {
		participantMap[p] = true
	}

	expectedParticipants := []string{"alice@example.com", "bob@example.com", "carol@example.com"}
	for _, email := range expectedParticipants {
		if !participantMap[entity.ActorID(email)] {
			t.Errorf("Missing participant: %s", email)
		}
	}
}

func TestReconstructThreadDates(t *testing.T) {
	r := NewReconstructor(DefaultConfig())

	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC)

	messages := []*entity.Message{
		{
			ID:        "msg-1",
			MessageID: "msg-1@example.com",
			From:      "alice@example.com",
			To:        []string{"bob@example.com"},
			Subject:   "Hello",
			Date:      startTime,
		},
		{
			ID:         "msg-2",
			MessageID:  "msg-2@example.com",
			InReplyTo:  "msg-1@example.com",
			From:       "bob@example.com",
			To:         []string{"alice@example.com"},
			Subject:    "Re: Hello",
			Date:       endTime,
		},
	}

	threads, err := r.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Reconstruct error: %v", err)
	}

	if len(threads) != 1 {
		t.Fatalf("threads count = %d, want 1", len(threads))
	}

	thread := threads[0]

	if !thread.StartDate.Equal(startTime) {
		t.Errorf("thread.StartDate = %v, want %v", thread.StartDate, startTime)
	}
	if !thread.EndDate.Equal(endTime) {
		t.Errorf("thread.EndDate = %v, want %v", thread.EndDate, endTime)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	stats := ComputeStats(nil)

	if stats.TotalThreads != 0 {
		t.Errorf("TotalThreads = %d, want 0", stats.TotalThreads)
	}
	if stats.TotalMessages != 0 {
		t.Errorf("TotalMessages = %d, want 0", stats.TotalMessages)
	}
}

func TestComputeStats(t *testing.T) {
	threads := []*entity.Thread{
		{
			ID:    "thread-1",
			Size:  1,
			Depth: 0,
		},
		{
			ID:    "thread-2",
			Size:  5,
			Depth: 3,
		},
		{
			ID:    "thread-3",
			Size:  3,
			Depth: 2,
		},
		{
			ID:    "thread-4",
			Size:  1,
			Depth: 0,
		},
	}

	stats := ComputeStats(threads)

	if stats.TotalThreads != 4 {
		t.Errorf("TotalThreads = %d, want 4", stats.TotalThreads)
	}
	if stats.TotalMessages != 10 { // 1 + 5 + 3 + 1
		t.Errorf("TotalMessages = %d, want 10", stats.TotalMessages)
	}
	if stats.SingleMessageThreads != 2 { // threads 1 and 4
		t.Errorf("SingleMessageThreads = %d, want 2", stats.SingleMessageThreads)
	}
	if stats.MultiMessageThreads != 2 { // threads 2 and 3
		t.Errorf("MultiMessageThreads = %d, want 2", stats.MultiMessageThreads)
	}
	if stats.MaxThreadSize != 5 {
		t.Errorf("MaxThreadSize = %d, want 5", stats.MaxThreadSize)
	}
	if stats.MaxThreadDepth != 3 {
		t.Errorf("MaxThreadDepth = %d, want 3", stats.MaxThreadDepth)
	}
	if stats.MessagesWithParent != 6 { // (5-1) + (3-1) = 4 + 2 = 6
		t.Errorf("MessagesWithParent = %d, want 6", stats.MessagesWithParent)
	}
}

func TestMessageAdapterGetMessageID(t *testing.T) {
	// Test with MessageID set
	msg := &entity.Message{
		ID:        "internal-id",
		MessageID: "external@example.com",
	}
	adapter := &messageAdapter{msg: msg}

	if got := adapter.GetMessageID(); got != "external@example.com" {
		t.Errorf("GetMessageID() = %q, want %q", got, "external@example.com")
	}

	// Test with empty MessageID (falls back to ID)
	msg2 := &entity.Message{
		ID:        "internal-id",
		MessageID: "",
	}
	adapter2 := &messageAdapter{msg: msg2}

	if got := adapter2.GetMessageID(); got != "internal-id" {
		t.Errorf("GetMessageID() with empty MessageID = %q, want %q", got, "internal-id")
	}
}

func TestMessageAdapterGetDate(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	msg := &entity.Message{Date: date}
	adapter := &messageAdapter{msg: msg}

	if got := adapter.GetDate(); !got.Equal(date) {
		t.Errorf("GetDate() = %v, want %v", got, date)
	}
}

func TestMessageAdapterGetSubject(t *testing.T) {
	msg := &entity.Message{Subject: "Test Subject"}
	adapter := &messageAdapter{msg: msg}

	if got := adapter.GetSubject(); got != "Test Subject" {
		t.Errorf("GetSubject() = %q, want %q", got, "Test Subject")
	}
}

func TestMessageAdapterGetInReplyTo(t *testing.T) {
	msg := &entity.Message{InReplyTo: "parent@example.com"}
	adapter := &messageAdapter{msg: msg}

	if got := adapter.GetInReplyTo(); got != "parent@example.com" {
		t.Errorf("GetInReplyTo() = %q, want %q", got, "parent@example.com")
	}
}

func TestMessageAdapterGetReferences(t *testing.T) {
	refs := []string{"ref1@example.com", "ref2@example.com"}
	msg := &entity.Message{References: refs}
	adapter := &messageAdapter{msg: msg}

	got := adapter.GetReferences()
	if len(got) != 2 {
		t.Errorf("GetReferences() len = %d, want 2", len(got))
	}
	for i, ref := range got {
		if ref != refs[i] {
			t.Errorf("GetReferences()[%d] = %q, want %q", i, ref, refs[i])
		}
	}
}

func TestMessageAdapterGetParticipants(t *testing.T) {
	msg := &entity.Message{
		From: "alice@example.com",
		To:   []string{"bob@example.com"},
		CC:   []string{"carol@example.com"},
		BCC:  []string{"dave@example.com"},
	}
	adapter := &messageAdapter{msg: msg}

	participants := adapter.GetParticipants()

	// Should include from, to, cc, bcc
	if len(participants) != 4 {
		t.Errorf("GetParticipants() len = %d, want 4", len(participants))
	}

	expected := map[string]bool{
		"alice@example.com": true,
		"bob@example.com":   true,
		"carol@example.com": true,
		"dave@example.com":  true,
	}

	for _, p := range participants {
		if !expected[p] {
			t.Errorf("Unexpected participant: %s", p)
		}
	}
}

func TestMessageAdapterGetEmbeddedMessageHints(t *testing.T) {
	msg := &entity.Message{}
	adapter := &messageAdapter{msg: msg}

	hints := adapter.GetEmbeddedMessageHints()
	if hints != nil {
		t.Errorf("GetEmbeddedMessageHints() = %v, want nil (not implemented)", hints)
	}
}

func TestConfigCustomization(t *testing.T) {
	config := Config{
		MaxParentAge:              24 * time.Hour,
		RequireParticipantOverlap: false,
	}
	r := NewReconstructor(config)

	if r.config.MaxParentAge != 24*time.Hour {
		t.Errorf("config.MaxParentAge = %v, want %v", r.config.MaxParentAge, 24*time.Hour)
	}
	if r.config.RequireParticipantOverlap {
		t.Error("config.RequireParticipantOverlap should be false")
	}
}
