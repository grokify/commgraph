package email

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grokify/commgraph/adapter"
	"github.com/grokify/commgraph/entity"
	"github.com/jhillyerd/enmime"
)

// MaildirAdapter ingests messages from Maildir format directories.
// Maildir stores each message as a separate file, with subdirectories
// for different mail folders (inbox, sent, etc.).
type MaildirAdapter struct {
	platform string
}

// NewMaildirAdapter creates a new Maildir adapter.
func NewMaildirAdapter() *MaildirAdapter {
	return &MaildirAdapter{
		platform: "email",
	}
}

// Name returns the adapter name.
func (a *MaildirAdapter) Name() string {
	return "email-maildir"
}

// Ingest reads messages from a Maildir directory structure.
func (a *MaildirAdapter) Ingest(ctx context.Context, source adapter.Source) (<-chan *entity.Message, <-chan error) {
	msgCh := make(chan *entity.Message, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		rootPath := source.Location()

		// Walk the directory tree
		err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
			// Check for cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if err != nil {
				// Log but continue on file errors
				return nil
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Skip hidden files and non-message files
			name := info.Name()
			if strings.HasPrefix(name, ".") {
				return nil
			}

			// Read and parse the message file
			msg, parseErr := a.parseMessageFile(path)
			if parseErr != nil {
				// Skip unparseable files
				return nil
			}

			if msg != nil {
				// Add folder context from path
				relPath, _ := filepath.Rel(rootPath, path)
				if relPath != "" {
					dir := filepath.Dir(relPath)
					if dir != "." {
						msg.Folder = dir
					}
				}

				select {
				case msgCh <- msg:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		})

		if err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("walk maildir: %w", err)
		}
	}()

	return msgCh, errCh
}

// IngestIncremental ingests messages newer than checkpoint.
func (a *MaildirAdapter) IngestIncremental(ctx context.Context, source adapter.Source, checkpoint adapter.Checkpoint) (<-chan *entity.Message, <-chan error) {
	msgCh := make(chan *entity.Message, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		baseMsgCh, baseErrCh := a.Ingest(ctx, source)

		for msg := range baseMsgCh {
			// Filter by checkpoint
			if !checkpoint.LastTimestamp.IsZero() && !msg.Date.After(checkpoint.LastTimestamp) {
				continue
			}
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}

		// Forward any errors
		for err := range baseErrCh {
			errCh <- err
		}
	}()

	return msgCh, errCh
}

// parseMessageFile reads and parses a single message file.
func (a *MaildirAdapter) parseMessageFile(path string) (*entity.Message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open message file: %w", err)
	}
	defer file.Close()

	// Read entire file for hashing
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read message file: %w", err)
	}

	// Parse with enmime
	env, err := enmime.ReadEnvelope(strings.NewReader(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}

	// Extract message ID
	messageID := env.GetHeader("Message-ID")
	if messageID == "" {
		messageID = env.GetHeader("Message-Id")
	}

	// Generate stable ID from content hash
	hash := sha256.Sum256(raw)
	id := hex.EncodeToString(hash[:16])

	// Parse date
	dateStr := env.GetHeader("Date")
	date, _ := mail.ParseDate(dateStr)
	if date.IsZero() {
		// Try file modification time as fallback
		if info, err := os.Stat(path); err == nil {
			date = info.ModTime()
		} else {
			date = time.Now()
		}
	}

	// Parse addresses
	from := parseAddress(env.GetHeader("From"))
	to := parseAddressList(env.GetHeader("To"))
	cc := parseAddressList(env.GetHeader("Cc"))
	bcc := parseAddressList(env.GetHeader("Bcc"))

	// Parse references
	inReplyTo := cleanMessageID(env.GetHeader("In-Reply-To"))
	references := parseReferences(env.GetHeader("References"))

	// Extract body preview
	bodyPreview := truncate(env.Text, 500)

	// Extract mentioned emails from body
	mentions := extractEmailMentions(env.Text)

	// Extract external domains
	allAddrs := append(append(to, cc...), bcc...)
	domains := extractUniqueDomains(allAddrs)

	return &entity.Message{
		ID:          id,
		Platform:    a.platform,
		RawHash:     hex.EncodeToString(hash[:]),
		MessageID:   cleanMessageID(messageID),
		InReplyTo:   inReplyTo,
		References:  references,
		From:        from,
		To:          to,
		CC:          cc,
		BCC:         bcc,
		Subject:     env.GetHeader("Subject"),
		Date:        date,
		BodyPreview: bodyPreview,
		Mentions:    mentions,
		Domains:     domains,
		IngestedAt:  time.Now(),
		SourcePath:  path,
	}, nil
}

// Verify interface compliance.
var _ adapter.Adapter = (*MaildirAdapter)(nil)
