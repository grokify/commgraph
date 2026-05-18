package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grokify/commgraph/adapter"
	"github.com/grokify/commgraph/adapter/email"
	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/identity"
	"github.com/grokify/commgraph/session"
	"github.com/grokify/commgraph/storage"
	"github.com/grokify/commgraph/threading"
	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest messages from a source",
	Long:  `Ingest messages from various sources (email, slack, teams) into the graph.`,
}

var ingestEmailCmd = &cobra.Command{
	Use:   "email",
	Short: "Ingest email messages",
	Long:  `Ingest email messages from mbox, EML, or PST files.`,
	RunE:  runIngestEmail,
}

var (
	ingestSource  string
	ingestFormat  string
	ingestDomains []string
	ingestOutput  string
	ingestVerbose bool
	ingestSession string
)

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.AddCommand(ingestEmailCmd)

	ingestEmailCmd.Flags().StringVarP(&ingestSource, "source", "s", "", "source file or directory (required)")
	ingestEmailCmd.Flags().StringVarP(&ingestFormat, "format", "f", "mbox", "source format (mbox, eml, pst)")
	ingestEmailCmd.Flags().StringSliceVarP(&ingestDomains, "internal-domains", "d", []string{}, "internal domains for identity resolution")
	ingestEmailCmd.Flags().StringVarP(&ingestOutput, "output", "o", "", "output file for stats (JSON)")
	ingestEmailCmd.Flags().BoolVarP(&ingestVerbose, "verbose", "v", false, "verbose output")
	ingestEmailCmd.Flags().StringVar(&ingestSession, "session", "", "session file to save (default: .commgraph-session.json)")

	_ = ingestEmailCmd.MarkFlagRequired("source")
}

func runIngestEmail(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate source
	if _, err := os.Stat(ingestSource); err != nil {
		return fmt.Errorf("source not found: %s", ingestSource)
	}

	// Create adapter
	var adp adapter.Adapter
	switch ingestFormat {
	case "mbox":
		adp = email.NewMboxAdapter()
	case "maildir":
		adp = email.NewMaildirAdapter()
	default:
		return fmt.Errorf("unsupported format: %s (supported: mbox, maildir)", ingestFormat)
	}

	// Create store
	store := storage.NewMemoryStore()

	// Create identity resolver
	resolverConfig := identity.DefaultConfig()
	resolverConfig.InternalDomains = ingestDomains
	resolverConfig.AutoCreate = true
	resolver := identity.NewSCIMResolver(resolverConfig)

	// Ingest messages
	source := adapter.FileSource{
		SourceType: ingestFormat,
		Path:       ingestSource,
	}

	fmt.Printf("Ingesting from %s...\n", ingestSource)
	start := time.Now()

	msgCh, errCh := adp.Ingest(ctx, source)

	var messages []*entity.Message
	var msgCount int

	for msg := range msgCh {
		messages = append(messages, msg)
		msgCount++

		// Store message
		if err := store.StoreMessage(ctx, msg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to store message: %v\n", err)
		}

		// Resolve identities and create interactions
		fromActor := resolver.ResolveOrCreate(msg.From)

		for _, to := range msg.To {
			toActor := resolver.ResolveOrCreate(to)
			interaction := &entity.Interaction{
				ID:        fmt.Sprintf("%s-to-%s", msg.ID, toActor),
				MessageID: msg.ID,
				From:      fromActor,
				To:        toActor,
				EdgeType:  entity.EdgeTypeTo,
				Timestamp: msg.Date,
				Platform:  msg.Platform,
			}
			_ = store.StoreInteraction(ctx, interaction)
		}

		for _, cc := range msg.CC {
			ccActor := resolver.ResolveOrCreate(cc)
			interaction := &entity.Interaction{
				ID:        fmt.Sprintf("%s-cc-%s", msg.ID, ccActor),
				MessageID: msg.ID,
				From:      fromActor,
				To:        ccActor,
				EdgeType:  entity.EdgeTypeCC,
				Timestamp: msg.Date,
				Platform:  msg.Platform,
			}
			_ = store.StoreInteraction(ctx, interaction)
		}

		for _, bcc := range msg.BCC {
			bccActor := resolver.ResolveOrCreate(bcc)
			interaction := &entity.Interaction{
				ID:        fmt.Sprintf("%s-bcc-%s", msg.ID, bccActor),
				MessageID: msg.ID,
				From:      fromActor,
				To:        bccActor,
				EdgeType:  entity.EdgeTypeBCC,
				Timestamp: msg.Date,
				Platform:  msg.Platform,
			}
			_ = store.StoreInteraction(ctx, interaction)
		}

		if ingestVerbose && msgCount%100 == 0 {
			fmt.Printf("  Processed %d messages...\n", msgCount)
		}
	}

	// Check for errors
	for err := range errCh {
		if err != nil {
			return fmt.Errorf("ingestion error: %w", err)
		}
	}

	ingestDuration := time.Since(start)

	// Reconstruct threads
	fmt.Println("Reconstructing threads...")
	threadStart := time.Now()

	reconstructor := threading.NewReconstructor(threading.DefaultConfig())
	threads, err := reconstructor.Reconstruct(messages)
	if err != nil {
		return fmt.Errorf("thread reconstruction: %w", err)
	}

	// Store threads and update messages
	for _, thread := range threads {
		_ = store.StoreThread(ctx, thread)
	}

	// Update interactions with thread IDs
	for _, msg := range messages {
		if msg.ThreadID != "" {
			interactions, _ := store.GetInteractions(ctx, storage.InteractionQuery{MessageID: msg.ID})
			for _, interaction := range interactions {
				interaction.ThreadID = msg.ThreadID
			}
		}
	}

	threadDuration := time.Since(threadStart)

	// Get stats
	stats, _ := store.Stats(ctx)
	resolverStats := resolver.Stats()
	threadStats := threading.ComputeStats(threads)

	// Print summary
	fmt.Printf("\nIngestion complete:\n")
	fmt.Printf("  Messages:     %d\n", stats.MessageCount)
	fmt.Printf("  Interactions: %d\n", stats.InteractionCount)
	fmt.Printf("  Actors:       %d (%d internal, %d external)\n",
		resolverStats.TotalActors, resolverStats.InternalActors, resolverStats.ExternalActors)
	fmt.Printf("  Threads:      %d (%d single, %d multi-message)\n",
		threadStats.TotalThreads, threadStats.SingleMessageThreads, threadStats.MultiMessageThreads)
	fmt.Printf("  Duration:     %v (ingest) + %v (threading)\n", ingestDuration, threadDuration)

	// Save to global state for analyze command
	globalStore = store
	globalResolver = resolver

	// Save session if requested
	sessionPath := ingestSession
	if sessionPath == "" {
		sessionPath = session.DefaultPath()
	}

	config := session.SessionConfig{
		InternalDomains: ingestDomains,
		AutoCreate:      true,
		Format:          ingestFormat,
	}

	if err := session.SaveFromStore(store, resolver, ingestSource, config, sessionPath); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	fmt.Printf("  Session saved to %s\n", sessionPath)

	return nil
}

// Global state for CLI commands (would be replaced by proper session management)
var (
	globalStore    *storage.MemoryStore
	globalResolver *identity.SCIMResolver
)
