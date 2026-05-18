package email

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/commgraph/adapter"
)

func TestNewMaildirAdapter(t *testing.T) {
	a := NewMaildirAdapter()
	if a == nil {
		t.Fatal("NewMaildirAdapter returned nil")
	}
	if a.platform != "email" {
		t.Errorf("platform = %q, want %q", a.platform, "email")
	}
}

func TestMaildirAdapterName(t *testing.T) {
	a := NewMaildirAdapter()
	if got := a.Name(); got != "email-maildir" {
		t.Errorf("Name() = %q, want %q", got, "email-maildir")
	}
}

func TestMaildirAdapterIngest(t *testing.T) {
	// Create a temporary maildir structure
	tmpDir := t.TempDir()
	inboxDir := filepath.Join(tmpDir, "inbox")
	sentDir := filepath.Join(tmpDir, "sent")

	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatalf("Failed to create inbox dir: %v", err)
	}
	if err := os.MkdirAll(sentDir, 0755); err != nil {
		t.Fatalf("Failed to create sent dir: %v", err)
	}

	// Create message files
	msg1 := `From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Hello from Inbox
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

This is inbox message.
`
	msg2 := `From: Bob <bob@example.com>
To: Alice <alice@example.com>
Subject: Hello from Sent
Date: Tue, 02 Jan 2024 11:00:00 -0500
Message-ID: <msg002@example.com>

This is sent message.
`

	if err := os.WriteFile(filepath.Join(inboxDir, "msg001.eml"), []byte(msg1), 0600); err != nil {
		t.Fatalf("Failed to write inbox message: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sentDir, "msg002.eml"), []byte(msg2), 0600); err != nil {
		t.Fatalf("Failed to write sent message: %v", err)
	}

	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

	msgCh, errCh := a.Ingest(ctx, source)

	var messages []map[string]any
	for msg := range msgCh {
		messages = append(messages, map[string]any{
			"from":       msg.From,
			"to":         msg.To,
			"subject":    msg.Subject,
			"folder":     msg.Folder,
			"message_id": msg.MessageID,
		})
	}

	for err := range errCh {
		t.Fatalf("Ingest error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Got %d messages, want 2", len(messages))
	}

	// Verify folder assignment
	foundInbox := false
	foundSent := false
	for _, msg := range messages {
		folder := msg["folder"].(string)
		if folder == "inbox" {
			foundInbox = true
			if msg["subject"] != "Hello from Inbox" {
				t.Errorf("Inbox message subject = %q, want %q", msg["subject"], "Hello from Inbox")
			}
		} else if folder == "sent" {
			foundSent = true
			if msg["subject"] != "Hello from Sent" {
				t.Errorf("Sent message subject = %q, want %q", msg["subject"], "Hello from Sent")
			}
		}
	}

	if !foundInbox {
		t.Error("Did not find inbox message")
	}
	if !foundSent {
		t.Error("Did not find sent message")
	}
}

func TestMaildirAdapterIngestIncremental(t *testing.T) {
	tmpDir := t.TempDir()

	msg1 := `From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Old Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

Old content.
`
	msg2 := `From: Bob <bob@example.com>
To: Alice <alice@example.com>
Subject: New Message
Date: Mon, 15 Jan 2024 10:00:00 -0500
Message-ID: <msg002@example.com>

New content.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "old.eml"), []byte(msg1), 0600); err != nil {
		t.Fatalf("Failed to write old message: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "new.eml"), []byte(msg2), 0600); err != nil {
		t.Fatalf("Failed to write new message: %v", err)
	}

	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

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

func TestMaildirAdapterIngestNestedFolders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested folder structure
	deepDir := filepath.Join(tmpDir, "INBOX", "Work", "Projects")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("Failed to create deep dir: %v", err)
	}

	msg := `From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Deep Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

Content.
`

	if err := os.WriteFile(filepath.Join(deepDir, "msg.eml"), []byte(msg), 0600); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

	msgCh, errCh := a.Ingest(ctx, source)

	var folder string
	for msg := range msgCh {
		folder = msg.Folder
	}

	for err := range errCh {
		t.Fatalf("Ingest error: %v", err)
	}

	expectedFolder := filepath.Join("INBOX", "Work", "Projects")
	if folder != expectedFolder {
		t.Errorf("folder = %q, want %q", folder, expectedFolder)
	}
}

func TestMaildirAdapterSkipsHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	msg := `From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Visible Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

Content.
`

	hiddenMsg := `From: Hidden <hidden@example.com>
To: Bob <bob@example.com>
Subject: Hidden Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <hidden@example.com>

Content.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "visible.eml"), []byte(msg), 0600); err != nil {
		t.Fatalf("Failed to write visible message: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden.eml"), []byte(hiddenMsg), 0600); err != nil {
		t.Fatalf("Failed to write hidden message: %v", err)
	}

	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

	msgCh, errCh := a.Ingest(ctx, source)

	var count int
	for msg := range msgCh {
		if msg.Subject == "Hidden Message" {
			t.Error("Should not have read hidden file")
		}
		count++
	}

	for err := range errCh {
		t.Fatalf("Ingest error: %v", err)
	}

	if count != 1 {
		t.Errorf("Got %d messages, want 1 (visible only)", count)
	}
}

func TestMaildirAdapterCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create many message files
	for i := 0; i < 50; i++ {
		msg := `From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg@example.com>

Body.
`
		filename := filepath.Join(tmpDir, "msg"+string(rune('0'+i%10))+".eml")
		if err := os.WriteFile(filename, []byte(msg), 0600); err != nil {
			t.Fatalf("Failed to write message: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

	msgCh, errCh := a.Ingest(ctx, source)

	// Cancel after receiving some messages
	msgCount := 0
	for range msgCh {
		msgCount++
		if msgCount == 3 {
			cancel()
		}
	}

	// Drain errors
	for range errCh {
	}

	// Should have stopped early (may have processed a few more due to buffering)
	if msgCount >= 50 {
		t.Error("Expected ingestion to stop on cancellation")
	}
}

func TestMaildirAdapterEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

	msgCh, errCh := a.Ingest(ctx, source)

	var count int
	for range msgCh {
		count++
	}

	for err := range errCh {
		t.Fatalf("Unexpected error: %v", err)
	}

	if count != 0 {
		t.Errorf("Got %d messages from empty dir, want 0", count)
	}
}

func TestMaildirAdapterNonexistentDirectory(t *testing.T) {
	// The maildir adapter gracefully handles missing directories by
	// producing no messages and no errors (filepath.Walk silently
	// handles the error via the walk function which returns nil).
	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: "/nonexistent/path"}

	msgCh, errCh := a.Ingest(ctx, source)

	var msgCount int
	for range msgCh {
		msgCount++
	}

	// Drain errors - may or may not have error depending on OS behavior
	for range errCh {
	}

	if msgCount != 0 {
		t.Errorf("Got %d messages for nonexistent dir, want 0", msgCount)
	}
}

func TestMaildirAdapterMalformedMessage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a malformed message (not valid email)
	malformed := `This is not a valid email message
just random text
`
	if err := os.WriteFile(filepath.Join(tmpDir, "malformed.eml"), []byte(malformed), 0600); err != nil {
		t.Fatalf("Failed to write malformed message: %v", err)
	}

	// Also create a valid message
	valid := `From: Alice <alice@example.com>
To: Bob <bob@example.com>
Subject: Valid Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>

Content.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "valid.eml"), []byte(valid), 0600); err != nil {
		t.Fatalf("Failed to write valid message: %v", err)
	}

	ctx := context.Background()
	a := NewMaildirAdapter()
	source := &testSource{location: tmpDir}

	msgCh, errCh := a.Ingest(ctx, source)

	var count int
	for range msgCh {
		count++
	}

	for err := range errCh {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have processed only the valid message (malformed skipped)
	if count != 1 {
		t.Errorf("Got %d messages, want 1 (valid only)", count)
	}
}

func TestMaildirParseMessageFile(t *testing.T) {
	tmpDir := t.TempDir()

	msg := `From: Alice Smith <alice@example.com>
To: Bob Jones <bob@example.com>, Carol White <carol@example.com>
Cc: Dave <dave@test.org>
Subject: Test Message
Date: Mon, 01 Jan 2024 10:00:00 -0500
Message-ID: <msg001@example.com>
In-Reply-To: <parent@example.com>
References: <grandparent@example.com> <parent@example.com>

Hello,

This is the body of the email.
Contact support@help.org for assistance.

Thanks,
Alice
`

	msgPath := filepath.Join(tmpDir, "test.eml")
	if err := os.WriteFile(msgPath, []byte(msg), 0600); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	a := NewMaildirAdapter()
	parsed, err := a.parseMessageFile(msgPath)
	if err != nil {
		t.Fatalf("parseMessageFile failed: %v", err)
	}

	// Verify parsed fields
	if parsed.From != "alice@example.com" {
		t.Errorf("From = %q, want %q", parsed.From, "alice@example.com")
	}

	if len(parsed.To) != 2 {
		t.Errorf("To len = %d, want 2", len(parsed.To))
	} else {
		if parsed.To[0] != "bob@example.com" {
			t.Errorf("To[0] = %q, want %q", parsed.To[0], "bob@example.com")
		}
	}

	if len(parsed.CC) != 1 || parsed.CC[0] != "dave@test.org" {
		t.Errorf("CC = %v, want [dave@test.org]", parsed.CC)
	}

	if parsed.Subject != "Test Message" {
		t.Errorf("Subject = %q, want %q", parsed.Subject, "Test Message")
	}

	if parsed.MessageID != "msg001@example.com" {
		t.Errorf("MessageID = %q, want %q", parsed.MessageID, "msg001@example.com")
	}

	if parsed.InReplyTo != "parent@example.com" {
		t.Errorf("InReplyTo = %q, want %q", parsed.InReplyTo, "parent@example.com")
	}

	if len(parsed.References) != 2 {
		t.Errorf("References len = %d, want 2", len(parsed.References))
	}

	// Check mentions extracted from body
	foundMention := false
	for _, mention := range parsed.Mentions {
		if mention == "support@help.org" {
			foundMention = true
			break
		}
	}
	if !foundMention {
		t.Errorf("Mentions = %v, should contain support@help.org", parsed.Mentions)
	}

	// Check domains
	if len(parsed.Domains) != 2 {
		t.Errorf("Domains len = %d, want 2 (example.com, test.org)", len(parsed.Domains))
	}

	if parsed.SourcePath != msgPath {
		t.Errorf("SourcePath = %q, want %q", parsed.SourcePath, msgPath)
	}

	if parsed.Platform != "email" {
		t.Errorf("Platform = %q, want %q", parsed.Platform, "email")
	}
}
