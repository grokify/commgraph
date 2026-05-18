package email

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/commgraph/adapter"
)

func TestNewMboxAdapter(t *testing.T) {
	a := NewMboxAdapter()
	if a == nil {
		t.Fatal("NewMboxAdapter returned nil")
	}
	if a.platform != "email" {
		t.Errorf("platform = %q, want %q", a.platform, "email")
	}
}

func TestMboxAdapterName(t *testing.T) {
	a := NewMboxAdapter()
	if got := a.Name(); got != "email-mbox" {
		t.Errorf("Name() = %q, want %q", got, "email-mbox")
	}
}

func TestMboxAdapterIngest(t *testing.T) {
	// Create a temporary mbox file
	tmpDir := t.TempDir()
	mboxPath := filepath.Join(tmpDir, "test.mbox")

	mboxContent := `From sender@example.com Mon Jan 1 00:00:00 2024
From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Hello World
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

This is the body of the first email.

From sender@example.com Mon Jan 2 00:00:00 2024
From: Bob <bob@example.com>
To: Alice <alice@example.com>
Cc: Carol <carol@example.com>
Subject: Re: Hello World
Date: Tue, 02 Jan 2024 11:00:00 -0500
Message-ID: <msg002@example.com>
In-Reply-To: <msg001@example.com>
References: <msg001@example.com>

This is a reply.
`

	if err := os.WriteFile(mboxPath, []byte(mboxContent), 0600); err != nil {
		t.Fatalf("Failed to create test mbox: %v", err)
	}

	ctx := context.Background()
	a := NewMboxAdapter()
	source := &testSource{location: mboxPath}

	msgCh, errCh := a.Ingest(ctx, source)

	var messages []map[string]any
	for msg := range msgCh {
		messages = append(messages, map[string]any{
			"from":       msg.From,
			"to":         msg.To,
			"cc":         msg.CC,
			"subject":    msg.Subject,
			"message_id": msg.MessageID,
			"in_reply":   msg.InReplyTo,
			"references": msg.References,
		})
	}

	// Check for errors
	for err := range errCh {
		t.Fatalf("Ingest error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Got %d messages, want 2", len(messages))
	}

	// Verify first message
	if messages[0]["from"] != "alice@example.com" {
		t.Errorf("msg[0] from = %q, want %q", messages[0]["from"], "alice@example.com")
	}
	if messages[0]["subject"] != "Hello World" {
		t.Errorf("msg[0] subject = %q, want %q", messages[0]["subject"], "Hello World")
	}
	if messages[0]["message_id"] != "msg001@example.com" {
		t.Errorf("msg[0] message_id = %q, want %q", messages[0]["message_id"], "msg001@example.com")
	}

	// Verify second message
	if messages[1]["from"] != "bob@example.com" {
		t.Errorf("msg[1] from = %q, want %q", messages[1]["from"], "bob@example.com")
	}
	if messages[1]["in_reply"] != "msg001@example.com" {
		t.Errorf("msg[1] in_reply = %q, want %q", messages[1]["in_reply"], "msg001@example.com")
	}
	cc := messages[1]["cc"].([]string)
	if len(cc) != 1 || cc[0] != "carol@example.com" {
		t.Errorf("msg[1] cc = %v, want [carol@example.com]", cc)
	}
}

func TestMboxAdapterIngestIncremental(t *testing.T) {
	tmpDir := t.TempDir()
	mboxPath := filepath.Join(tmpDir, "test.mbox")

	mboxContent := `From sender@example.com Mon Jan 1 00:00:00 2024
From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Old Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

Old content.

From sender@example.com Mon Jan 15 00:00:00 2024
From: Bob <bob@example.com>
To: Alice <alice@example.com>
Subject: New Message
Date: Mon, 15 Jan 2024 10:00:00 -0500
Message-ID: <msg002@example.com>

New content.
`

	if err := os.WriteFile(mboxPath, []byte(mboxContent), 0600); err != nil {
		t.Fatalf("Failed to create test mbox: %v", err)
	}

	ctx := context.Background()
	a := NewMboxAdapter()
	source := &testSource{location: mboxPath}

	// Use checkpoint after first message
	checkpoint := adapter.Checkpoint{
		LastTimestamp: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	msgCh, errCh := a.IngestIncremental(ctx, source, checkpoint)

	var count int
	for msg := range msgCh {
		count++
		if msg.Subject != "New Message" {
			t.Errorf("Expected only new message, got %q", msg.Subject)
		}
	}

	for err := range errCh {
		t.Fatalf("Ingest error: %v", err)
	}

	if count != 1 {
		t.Errorf("Got %d messages after checkpoint, want 1", count)
	}
}

func TestMboxAdapterIngestCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	mboxPath := filepath.Join(tmpDir, "test.mbox")

	// Create a larger mbox file
	var content string
	for i := 0; i < 100; i++ {
		content += `From sender@example.com Mon Jan 1 00:00:00 2024
From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg@example.com>

Body text.

`
	}
	if err := os.WriteFile(mboxPath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test mbox: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := NewMboxAdapter()
	source := &testSource{location: mboxPath}

	msgCh, errCh := a.Ingest(ctx, source)

	// Cancel after receiving first message
	msgCount := 0
	for range msgCh {
		msgCount++
		if msgCount == 1 {
			cancel()
		}
	}

	// Should have received context error or stopped early
	for range errCh {
		// Error is expected on cancellation
	}

	// Should have stopped before processing all messages
	if msgCount >= 100 {
		t.Error("Expected ingestion to stop on cancellation")
	}
}

func TestMboxAdapterIngestError(t *testing.T) {
	ctx := context.Background()
	a := NewMboxAdapter()
	source := &testSource{location: "/nonexistent/path.mbox"}

	msgCh, errCh := a.Ingest(ctx, source)

	// Drain messages
	for range msgCh {
	}

	// Should receive an error
	var gotError bool
	for err := range errCh {
		if err != nil {
			gotError = true
		}
	}

	if !gotError {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"alice@example.com", "alice@example.com"},
		{"Alice <alice@example.com>", "alice@example.com"},
		{"<alice@example.com>", "alice@example.com"},
		{"Alice Smith <ALICE@EXAMPLE.COM>", "alice@example.com"},
		{"  alice@example.com  ", "alice@example.com"},
		{"", ""},
		{"invalid", "invalid"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseAddress(tc.input)
			if got != tc.expected {
				t.Errorf("parseAddress(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseAddressList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"alice@example.com", []string{"alice@example.com"}},
		{"Alice <alice@example.com>, Bob <bob@example.com>", []string{"alice@example.com", "bob@example.com"}},
		{"alice@example.com, bob@example.com", []string{"alice@example.com", "bob@example.com"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseAddressList(tc.input)
			if len(got) != len(tc.expected) {
				t.Errorf("parseAddressList(%q) len = %d, want %d", tc.input, len(got), len(tc.expected))
				return
			}
			for i, addr := range got {
				if addr != tc.expected[i] {
					t.Errorf("parseAddressList(%q)[%d] = %q, want %q", tc.input, i, addr, tc.expected[i])
				}
			}
		})
	}
}

func TestCleanMessageID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<msg001@example.com>", "msg001@example.com"},
		{"msg001@example.com", "msg001@example.com"},
		{"  <msg001@example.com>  ", "msg001@example.com"},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := cleanMessageID(tc.input)
			if got != tc.expected {
				t.Errorf("cleanMessageID(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestParseReferences(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"<msg001@example.com>", []string{"msg001@example.com"}},
		{"<msg001@example.com> <msg002@example.com>", []string{"msg001@example.com", "msg002@example.com"}},
		{"<msg001@example.com>  <msg002@example.com>  <msg003@example.com>", []string{"msg001@example.com", "msg002@example.com", "msg003@example.com"}},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseReferences(tc.input)
			if len(got) != len(tc.expected) {
				t.Errorf("parseReferences(%q) len = %d, want %d", tc.input, len(got), len(tc.expected))
				return
			}
			for i, ref := range got {
				if ref != tc.expected[i] {
					t.Errorf("parseReferences(%q)[%d] = %q, want %q", tc.input, i, ref, tc.expected[i])
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hello"},
		{"", 5, ""},
		{"hi", 0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := truncate(tc.input, tc.max)
			if got != tc.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.max, got, tc.expected)
			}
		})
	}
}

func TestExtractEmailMentions(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"no emails here", nil},
		{"Contact alice@example.com for more info", []string{"alice@example.com"}},
		{"Contact alice@example.com or bob@test.org for help.", []string{"alice@example.com", "bob@test.org"}},
		{"Send to <alice@example.com> please", []string{"alice@example.com"}},
		{"alice@example.com alice@example.com", []string{"alice@example.com"}}, // Deduplication
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := extractEmailMentions(tc.input)
			if len(got) != len(tc.expected) {
				t.Errorf("extractEmailMentions(%q) len = %d, want %d", tc.input, len(got), len(tc.expected))
				return
			}
			for i, mention := range got {
				if mention != tc.expected[i] {
					t.Errorf("extractEmailMentions(%q)[%d] = %q, want %q", tc.input, i, mention, tc.expected[i])
				}
			}
		})
	}
}

func TestExtractUniqueDomains(t *testing.T) {
	tests := []struct {
		input    []string
		expected int // Just check count due to map ordering
	}{
		{nil, 0},
		{[]string{}, 0},
		{[]string{"alice@example.com"}, 1},
		{[]string{"alice@example.com", "bob@example.com"}, 1},
		{[]string{"alice@example.com", "bob@test.org"}, 2},
		{[]string{"alice@example.com", "bob@example.com", "carol@test.org"}, 2},
	}

	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			got := extractUniqueDomains(tc.input)
			if len(got) != tc.expected {
				t.Errorf("extractUniqueDomains(%v) len = %d, want %d", tc.input, len(got), tc.expected)
			}
		})
	}
}

// testSource implements adapter.Source for testing.
type testSource struct {
	location string
}

func (s *testSource) Type() string     { return "mbox" }
func (s *testSource) Location() string { return s.location }
func (s *testSource) Credentials() any { return nil }
func (s *testSource) Options() any     { return nil }
