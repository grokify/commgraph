package entity

import (
	"testing"
	"time"
)

func TestActorID(t *testing.T) {
	id := ActorID("test-actor")
	if string(id) != "test-actor" {
		t.Errorf("ActorID string conversion failed: got %s, want test-actor", string(id))
	}
}

func TestActor(t *testing.T) {
	actor := &Actor{
		ID:           ActorID("jeff.skilling"),
		DisplayName:  "Jeff Skilling",
		Emails:       []string{"jeff.skilling@enron.com", "jskilli@enron.com"},
		PrimaryEmail: "jeff.skilling@enron.com",
		Internal:     true,
		Department:   "Executive",
		Title:        "CEO",
	}

	if actor.ID != "jeff.skilling" {
		t.Errorf("Actor ID mismatch: got %s, want jeff.skilling", actor.ID)
	}
	if actor.DisplayName != "Jeff Skilling" {
		t.Errorf("Actor DisplayName mismatch: got %s", actor.DisplayName)
	}
	if len(actor.Emails) != 2 {
		t.Errorf("Actor Emails count mismatch: got %d, want 2", len(actor.Emails))
	}
	if !actor.Internal {
		t.Error("Actor should be internal")
	}
}

func TestMessage(t *testing.T) {
	now := time.Now()
	msg := &Message{
		ID:        "msg-001",
		Platform:  "email",
		MessageID: "<abc123@enron.com>",
		From:      "alice@enron.com",
		To:        []string{"bob@enron.com", "carol@enron.com"},
		CC:        []string{"dave@enron.com"},
		BCC:       []string{"eve@enron.com"},
		Subject:   "Test Subject",
		Date:      now,
	}

	if msg.ID != "msg-001" {
		t.Errorf("Message ID mismatch: got %s", msg.ID)
	}
	if msg.Platform != "email" {
		t.Errorf("Message Platform mismatch: got %s", msg.Platform)
	}
}

func TestMessageAllRecipients(t *testing.T) {
	msg := &Message{
		To:  []string{"a@test.com", "b@test.com"},
		CC:  []string{"c@test.com"},
		BCC: []string{"d@test.com"},
	}

	recipients := msg.AllRecipients()
	if len(recipients) != 4 {
		t.Errorf("AllRecipients count mismatch: got %d, want 4", len(recipients))
	}

	expected := map[string]bool{
		"a@test.com": true,
		"b@test.com": true,
		"c@test.com": true,
		"d@test.com": true,
	}
	for _, r := range recipients {
		if !expected[r] {
			t.Errorf("Unexpected recipient: %s", r)
		}
	}
}

func TestMessageAllRecipientsEmpty(t *testing.T) {
	msg := &Message{}
	recipients := msg.AllRecipients()
	if len(recipients) != 0 {
		t.Errorf("AllRecipients should be empty: got %d", len(recipients))
	}
}

func TestMessageAllParticipants(t *testing.T) {
	msg := &Message{
		From: "sender@test.com",
		To:   []string{"a@test.com"},
		CC:   []string{"b@test.com"},
		BCC:  []string{"c@test.com"},
	}

	participants := msg.AllParticipants()
	if len(participants) != 4 {
		t.Errorf("AllParticipants count mismatch: got %d, want 4", len(participants))
	}

	// First participant should be the sender
	if participants[0] != "sender@test.com" {
		t.Errorf("First participant should be sender: got %s", participants[0])
	}
}

func TestEdgeType(t *testing.T) {
	tests := []struct {
		edgeType EdgeType
		want     string
	}{
		{EdgeTypeTo, "TO"},
		{EdgeTypeCC, "CC"},
		{EdgeTypeBCC, "BCC"},
		{EdgeTypeMention, "MENTION"},
		{EdgeTypeReply, "REPLY"},
	}

	for _, tt := range tests {
		if tt.edgeType.String() != tt.want {
			t.Errorf("EdgeType.String() = %s, want %s", tt.edgeType.String(), tt.want)
		}
	}
}

func TestAllEdgeTypes(t *testing.T) {
	types := AllEdgeTypes()
	if len(types) != 5 {
		t.Errorf("AllEdgeTypes count mismatch: got %d, want 5", len(types))
	}

	expected := map[EdgeType]bool{
		EdgeTypeTo:      true,
		EdgeTypeCC:      true,
		EdgeTypeBCC:     true,
		EdgeTypeMention: true,
		EdgeTypeReply:   true,
	}
	for _, et := range types {
		if !expected[et] {
			t.Errorf("Unexpected edge type: %s", et)
		}
	}
}

func TestInteraction(t *testing.T) {
	now := time.Now()
	interaction := &Interaction{
		ID:        "int-001",
		MessageID: "msg-001",
		ThreadID:  "thread-001",
		From:      ActorID("alice"),
		To:        ActorID("bob"),
		EdgeType:  EdgeTypeTo,
		Timestamp: now,
		Platform:  "email",
	}

	if interaction.ID != "int-001" {
		t.Errorf("Interaction ID mismatch: got %s", interaction.ID)
	}
	if interaction.From != "alice" {
		t.Errorf("Interaction From mismatch: got %s", interaction.From)
	}
	if interaction.To != "bob" {
		t.Errorf("Interaction To mismatch: got %s", interaction.To)
	}
	if interaction.EdgeType != EdgeTypeTo {
		t.Errorf("Interaction EdgeType mismatch: got %s", interaction.EdgeType)
	}
}

func TestThread(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	thread := &Thread{
		ID:            "thread-001",
		Subject:       "Test Thread",
		RootMessageID: "msg-001",
		MessageIDs:    []string{"msg-001", "msg-002", "msg-003"},
		Participants:  []ActorID{"alice", "bob", "carol"},
		StartDate:     start,
		EndDate:       end,
		Size:          3,
		Depth:         2,
	}

	if thread.ID != "thread-001" {
		t.Errorf("Thread ID mismatch: got %s", thread.ID)
	}
	if thread.Size != 3 {
		t.Errorf("Thread Size mismatch: got %d, want 3", thread.Size)
	}
}

func TestThreadDuration(t *testing.T) {
	start := time.Now().Add(-24 * time.Hour)
	end := time.Now()

	thread := &Thread{
		StartDate: start,
		EndDate:   end,
	}

	duration := thread.Duration()
	// Allow 1 second tolerance for test execution time
	if duration < 23*time.Hour || duration > 25*time.Hour {
		t.Errorf("Thread Duration mismatch: got %v, want ~24h", duration)
	}
}

func TestThreadIsSingleMessage(t *testing.T) {
	singleThread := &Thread{Size: 1}
	multiThread := &Thread{Size: 5}

	if !singleThread.IsSingleMessage() {
		t.Error("Single message thread should return true")
	}
	if multiThread.IsSingleMessage() {
		t.Error("Multi message thread should return false")
	}
}

func TestActorTypeConstants(t *testing.T) {
	if ActorTypeIndividual != "individual" {
		t.Errorf("ActorTypeIndividual mismatch: got %s", ActorTypeIndividual)
	}
	if ActorTypeRole != "role" {
		t.Errorf("ActorTypeRole mismatch: got %s", ActorTypeRole)
	}
	if ActorTypeGroup != "group" {
		t.Errorf("ActorTypeGroup mismatch: got %s", ActorTypeGroup)
	}
	if ActorTypeExternal != "external" {
		t.Errorf("ActorTypeExternal mismatch: got %s", ActorTypeExternal)
	}
}
