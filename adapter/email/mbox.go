// Package email provides adapters for ingesting email messages.
package email

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/grokify/commgraph/adapter"
	"github.com/grokify/commgraph/entity"
	"github.com/jhillyerd/enmime"
)

// MboxAdapter ingests messages from mbox format files.
type MboxAdapter struct {
	platform string
}

// NewMboxAdapter creates a new mbox adapter.
func NewMboxAdapter() *MboxAdapter {
	return &MboxAdapter{
		platform: "email",
	}
}

// Name returns the adapter name.
func (a *MboxAdapter) Name() string {
	return "email-mbox"
}

// Ingest reads messages from an mbox file.
func (a *MboxAdapter) Ingest(ctx context.Context, source adapter.Source) (<-chan *entity.Message, <-chan error) {
	msgCh := make(chan *entity.Message, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(msgCh)
		defer close(errCh)

		file, err := os.Open(source.Location())
		if err != nil {
			errCh <- fmt.Errorf("open mbox: %w", err)
			return
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		var rawMessage strings.Builder
		inMessage := false
		lineNum := 0

		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Process last message
					if rawMessage.Len() > 0 {
						msg, parseErr := a.parseMessage(rawMessage.String(), source.Location())
						if parseErr == nil && msg != nil {
							msgCh <- msg
						}
					}
					return
				}
				errCh <- fmt.Errorf("read mbox line %d: %w", lineNum, err)
				return
			}
			lineNum++

			// Mbox format: messages start with "From " at beginning of line
			if strings.HasPrefix(line, "From ") && (lineNum == 1 || inMessage) {
				// Process previous message
				if rawMessage.Len() > 0 {
					msg, parseErr := a.parseMessage(rawMessage.String(), source.Location())
					if parseErr == nil && msg != nil {
						select {
						case msgCh <- msg:
						case <-ctx.Done():
							errCh <- ctx.Err()
							return
						}
					}
					rawMessage.Reset()
				}
				inMessage = true
				continue
			}

			if inMessage {
				rawMessage.WriteString(line)
			}
		}
	}()

	return msgCh, errCh
}

// IngestIncremental ingests messages newer than checkpoint.
func (a *MboxAdapter) IngestIncremental(ctx context.Context, source adapter.Source, checkpoint adapter.Checkpoint) (<-chan *entity.Message, <-chan error) {
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

// parseMessage parses a raw email message.
func (a *MboxAdapter) parseMessage(raw string, sourcePath string) (*entity.Message, error) {
	env, err := enmime.ReadEnvelope(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}

	// Extract message ID
	messageID := env.GetHeader("Message-ID")
	if messageID == "" {
		messageID = env.GetHeader("Message-Id")
	}

	// Generate stable ID from content hash
	hash := sha256.Sum256([]byte(raw))
	id := hex.EncodeToString(hash[:16])

	// Parse date
	dateStr := env.GetHeader("Date")
	date, _ := mail.ParseDate(dateStr)
	if date.IsZero() {
		date = time.Now() // Fallback
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
		SourcePath:  sourcePath,
	}, nil
}

// parseAddress extracts the email address from a header value.
func parseAddress(header string) string {
	if header == "" {
		return ""
	}
	addr, err := mail.ParseAddress(header)
	if err != nil {
		// Try to extract raw email
		if idx := strings.Index(header, "<"); idx >= 0 {
			if end := strings.Index(header[idx:], ">"); end > 0 {
				return strings.ToLower(strings.TrimSpace(header[idx+1 : idx+end]))
			}
		}
		return strings.ToLower(strings.TrimSpace(header))
	}
	return strings.ToLower(addr.Address)
}

// parseAddressList extracts email addresses from a header value.
func parseAddressList(header string) []string {
	if header == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(header)
	if err != nil {
		// Fallback: split by comma
		parts := strings.Split(header, ",")
		var result []string
		for _, p := range parts {
			addr := parseAddress(strings.TrimSpace(p))
			if addr != "" {
				result = append(result, addr)
			}
		}
		return result
	}
	result := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		result = append(result, strings.ToLower(addr.Address))
	}
	return result
}

// cleanMessageID removes angle brackets from message ID.
func cleanMessageID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "<")
	id = strings.TrimSuffix(id, ">")
	return id
}

// parseReferences parses the References header.
func parseReferences(header string) []string {
	if header == "" {
		return nil
	}
	// References are space-separated message IDs
	parts := strings.Fields(header)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		id := cleanMessageID(p)
		if id != "" {
			result = append(result, id)
		}
	}
	return result
}

// truncate truncates a string to max length.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// extractEmailMentions finds email addresses in text.
func extractEmailMentions(text string) []string {
	// Simple regex-free extraction
	var mentions []string
	seen := make(map[string]bool)

	words := strings.Fields(text)
	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,;:!?()[]<>\"'")
		if strings.Contains(word, "@") && strings.Contains(word, ".") {
			email := strings.ToLower(word)
			if !seen[email] {
				seen[email] = true
				mentions = append(mentions, email)
			}
		}
	}
	return mentions
}

// extractUniqueDomains extracts unique domains from email addresses.
func extractUniqueDomains(addrs []string) []string {
	seen := make(map[string]bool)
	var domains []string
	for _, addr := range addrs {
		parts := strings.Split(addr, "@")
		if len(parts) == 2 {
			domain := strings.ToLower(parts[1])
			if !seen[domain] {
				seen[domain] = true
				domains = append(domains, domain)
			}
		}
	}
	return domains
}

// Verify interface compliance.
var _ adapter.Adapter = (*MboxAdapter)(nil)
