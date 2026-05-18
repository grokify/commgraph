// Package integration contains integration tests for the commgraph pipeline.
package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grokify/commgraph/adapter"
	"github.com/grokify/commgraph/adapter/email"
	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/export"
	"github.com/grokify/commgraph/identity"
	"github.com/grokify/commgraph/session"
	"github.com/grokify/commgraph/storage"
	"github.com/grokify/commgraph/threading"
	"github.com/grokify/commgraph/weight"
)

// TestFullPipeline tests the complete ingest -> resolve -> thread -> analyze -> export pipeline.
func TestFullPipeline(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create test mbox file with known email structure
	mboxPath := filepath.Join(tmpDir, "test.mbox")
	if err := createTestMbox(mboxPath); err != nil {
		t.Fatalf("Failed to create test mbox: %v", err)
	}

	// --- Step 1: Ingest ---
	t.Log("Step 1: Ingesting messages...")

	adp := email.NewMboxAdapter()
	source := adapter.FileSource{
		SourceType: "mbox",
		Path:       mboxPath,
	}

	msgCh, errCh := adp.Ingest(ctx, source)

	var messages []*entity.Message
	for msg := range msgCh {
		messages = append(messages, msg)
	}
	for err := range errCh {
		if err != nil {
			t.Fatalf("Ingest error: %v", err)
		}
	}

	if len(messages) != 8 {
		t.Errorf("Ingested %d messages, want 8", len(messages))
	}

	// --- Step 2: Identity Resolution ---
	t.Log("Step 2: Resolving identities...")

	resolverConfig := identity.DefaultConfig()
	resolverConfig.InternalDomains = []string{"example.com"}
	resolverConfig.AutoCreate = true
	resolver := identity.NewSCIMResolver(resolverConfig)

	store := storage.NewMemoryStore()

	for _, msg := range messages {
		_ = store.StoreMessage(ctx, msg)

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
	}

	stats, _ := store.Stats(ctx)
	if stats.InteractionCount == 0 {
		t.Error("No interactions created")
	}

	resolverStats := resolver.Stats()
	if resolverStats.TotalActors == 0 {
		t.Error("No actors resolved")
	}

	// Verify internal/external classification
	if resolverStats.InternalActors == 0 {
		t.Error("Should have internal actors for example.com domain")
	}
	if resolverStats.ExternalActors == 0 {
		t.Error("Should have external actors for external.org domain")
	}

	// --- Step 3: Thread Reconstruction ---
	t.Log("Step 3: Reconstructing threads...")

	reconstructor := threading.NewReconstructor(threading.DefaultConfig())
	threads, err := reconstructor.Reconstruct(messages)
	if err != nil {
		t.Fatalf("Thread reconstruction failed: %v", err)
	}

	for _, thread := range threads {
		_ = store.StoreThread(ctx, thread)
	}

	threadStats := threading.ComputeStats(threads)
	if threadStats.TotalThreads == 0 {
		t.Error("No threads reconstructed")
	}

	// Should have at least one multi-message thread
	if threadStats.MultiMessageThreads == 0 {
		t.Error("Should have at least one multi-message thread")
	}

	// --- Step 4: Analysis ---
	t.Log("Step 4: Running analysis...")

	registry := weight.NewRegistry()
	profile, err := registry.Get("influence")
	if err != nil {
		t.Fatalf("Failed to get weight profile: %v", err)
	}

	analyzer := analysis.NewAnalyzer(store, resolver, profile)

	// Test PageRank
	pagerank, err := analyzer.PageRank(ctx, 0.85, 100)
	if err != nil {
		t.Errorf("PageRank failed: %v", err)
	}
	if len(pagerank) == 0 {
		t.Error("PageRank returned no results")
	}

	// Test Degree
	degree, err := analyzer.Degree(ctx)
	if err != nil {
		t.Errorf("Degree failed: %v", err)
	}
	if len(degree) == 0 {
		t.Error("Degree returned no results")
	}

	// Test Community Detection
	communities, err := analyzer.Louvain(ctx, 1.0)
	if err != nil {
		t.Errorf("Louvain failed: %v", err)
	}
	if communities == nil || len(communities.Communities) == 0 {
		t.Error("Louvain returned no communities")
	}

	// Test Temporal Analysis
	bursts, err := analyzer.BurstDetection(ctx, 2.0, 24*time.Hour)
	if err != nil {
		t.Errorf("BurstDetection failed: %v", err)
	}
	// Bursts may or may not be detected depending on data distribution
	_ = bursts

	// Test Path Analysis
	avgPath, err := analyzer.AveragePathLength(ctx, 100)
	if err != nil {
		t.Errorf("AveragePathLength failed: %v", err)
	}
	if avgPath <= 0 {
		t.Error("Average path length should be positive")
	}

	components, err := analyzer.ConnectedComponents(ctx)
	if err != nil {
		t.Errorf("ConnectedComponents failed: %v", err)
	}
	if len(components) == 0 {
		t.Error("Should have at least one connected component")
	}

	// Test Bridge Detection
	if communities != nil {
		bridges, err := analyzer.DetectBridges(ctx, communities)
		if err != nil {
			t.Errorf("DetectBridges failed: %v", err)
		}
		// May or may not have bridges depending on structure
		_ = bridges
	}

	// Test External Analysis
	externalResults, err := analyzer.ExternalAnalysis(ctx)
	if err != nil {
		t.Errorf("ExternalAnalysis failed: %v", err)
	}
	if externalResults.Summary.TotalExternalActors == 0 {
		t.Error("Should have external actors")
	}

	// --- Step 5: Export ---
	t.Log("Step 5: Testing exports...")

	// Test JSON export
	jsonPath := filepath.Join(tmpDir, "results.json")
	jsonFile, err := os.Create(jsonPath)
	if err != nil {
		t.Fatalf("Failed to create JSON file: %v", err)
	}

	jsonExporter := export.NewJSONExporter(true)
	meta := export.Metadata{
		Profile:   "influence",
		Algorithm: "pagerank",
	}
	if err := jsonExporter.ExportCentrality(jsonFile, pagerank, meta); err != nil {
		t.Errorf("JSON export failed: %v", err)
	}
	jsonFile.Close()

	// Verify JSON file was created
	if fi, err := os.Stat(jsonPath); err != nil || fi.Size() == 0 {
		t.Error("JSON export file is empty or missing")
	}

	// Test CSV export
	csvPath := filepath.Join(tmpDir, "results.csv")
	csvFile, err := os.Create(csvPath)
	if err != nil {
		t.Fatalf("Failed to create CSV file: %v", err)
	}

	csvExporter := export.NewCSVExporter()
	if err := csvExporter.ExportCentrality(csvFile, pagerank); err != nil {
		t.Errorf("CSV export failed: %v", err)
	}
	csvFile.Close()

	// Verify CSV file was created
	if fi, err := os.Stat(csvPath); err != nil || fi.Size() == 0 {
		t.Error("CSV export file is empty or missing")
	}

	// Get actors and interactions for graph exports
	allActors := store.AllActors()
	allInteractions := store.AllInteractions()

	// Test Gephi (GEXF) export
	gexfPath := filepath.Join(tmpDir, "graph.gexf")
	gexfFile, err := os.Create(gexfPath)
	if err != nil {
		t.Fatalf("Failed to create GEXF file: %v", err)
	}

	gexfExporter := export.NewGEXFExporter()
	gexfMeta := export.Metadata{Profile: "influence"}
	if err := gexfExporter.ExportGraph(gexfFile, allActors, allInteractions, gexfMeta); err != nil {
		t.Errorf("Gephi export failed: %v", err)
	}
	gexfFile.Close()

	// Verify GEXF file was created
	if fi, err := os.Stat(gexfPath); err != nil || fi.Size() == 0 {
		t.Error("GEXF export file is empty or missing")
	}

	// Test Neo4j (Cypher) export
	cypherPath := filepath.Join(tmpDir, "graph.cypher")
	cypherFile, err := os.Create(cypherPath)
	if err != nil {
		t.Fatalf("Failed to create Cypher file: %v", err)
	}

	cypherExporter := export.NewCypherExporter()
	if err := cypherExporter.ExportFull(cypherFile, allActors, allInteractions, communities); err != nil {
		t.Errorf("Neo4j export failed: %v", err)
	}
	cypherFile.Close()

	// Verify Cypher file was created
	if fi, err := os.Stat(cypherPath); err != nil || fi.Size() == 0 {
		t.Error("Cypher export file is empty or missing")
	}

	// --- Step 6: Session Persistence ---
	t.Log("Step 6: Testing session persistence...")

	sessionPath := filepath.Join(tmpDir, "session.json")
	sessionConfig := session.SessionConfig{
		InternalDomains: []string{"example.com"},
		AutoCreate:      true,
		Format:          "mbox",
	}

	if err := session.SaveFromStore(store, resolver, mboxPath, sessionConfig, sessionPath); err != nil {
		t.Errorf("Session save failed: %v", err)
	}

	// Load session back
	loadedStore, loadedResolver, err := session.LoadIntoStore(ctx, sessionPath)
	if err != nil {
		t.Errorf("Session load failed: %v", err)
	}

	// Verify loaded data
	loadedStats, _ := loadedStore.Stats(ctx)
	if loadedStats.MessageCount != stats.MessageCount {
		t.Errorf("Loaded message count = %d, want %d", loadedStats.MessageCount, stats.MessageCount)
	}

	loadedResolverStats := loadedResolver.Stats()
	if loadedResolverStats.TotalActors != resolverStats.TotalActors {
		t.Errorf("Loaded actor count = %d, want %d", loadedResolverStats.TotalActors, resolverStats.TotalActors)
	}

	t.Log("All integration tests passed!")
}

// createTestMbox creates a test mbox file with a known structure.
// Structure:
// - Thread 1: 3 messages (alice -> bob -> alice)
// - Thread 2: 2 messages (carol -> dave -> carol)
// - Thread 3: 1 message (alice -> external)
// - Thread 4: 2 messages (external -> alice -> bob)
func createTestMbox(path string) error {
	baseTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

	mbox := fmt.Sprintf(`From alice@example.com Mon Jan 1 09:00:00 2024
From: Alice Smith <alice@example.com>
To: Bob Jones <bob@example.com>
Subject: Project Update
Date: %s
Message-ID: <msg001@example.com>

Hi Bob,

Here's the project update for this week.

Best,
Alice

From bob@example.com Mon Jan 1 10:00:00 2024
From: Bob Jones <bob@example.com>
To: Alice Smith <alice@example.com>
Subject: Re: Project Update
Date: %s
Message-ID: <msg002@example.com>
In-Reply-To: <msg001@example.com>
References: <msg001@example.com>

Thanks Alice!

I have some questions about the timeline.

Bob

From alice@example.com Mon Jan 1 11:00:00 2024
From: Alice Smith <alice@example.com>
To: Bob Jones <bob@example.com>
Subject: Re: Project Update
Date: %s
Message-ID: <msg003@example.com>
In-Reply-To: <msg002@example.com>
References: <msg001@example.com> <msg002@example.com>

Bob,

The timeline is flexible. Let's discuss tomorrow.

Alice

From carol@example.com Mon Jan 1 12:00:00 2024
From: Carol White <carol@example.com>
To: Dave Green <dave@example.com>
Subject: Budget Review
Date: %s
Message-ID: <msg004@example.com>

Dave,

Can you review the Q1 budget?

Carol

From dave@example.com Mon Jan 1 13:00:00 2024
From: Dave Green <dave@example.com>
To: Carol White <carol@example.com>
Subject: Re: Budget Review
Date: %s
Message-ID: <msg005@example.com>
In-Reply-To: <msg004@example.com>
References: <msg004@example.com>

Carol,

I'll have comments by EOD.

Dave

From alice@example.com Mon Jan 1 14:00:00 2024
From: Alice Smith <alice@example.com>
To: External Contact <external@external.org>
Subject: Partnership Inquiry
Date: %s
Message-ID: <msg006@example.com>

Hello,

I'm reaching out about a potential partnership.

Alice Smith
Example Inc.

From external@external.org Mon Jan 1 15:00:00 2024
From: External Contact <external@external.org>
To: Alice Smith <alice@example.com>
Subject: Vendor Proposal
Date: %s
Message-ID: <msg007@example.com>

Hi Alice,

Please see attached proposal.

Best,
External Contact

From alice@example.com Mon Jan 1 16:00:00 2024
From: Alice Smith <alice@example.com>
To: Bob Jones <bob@example.com>
Cc: Carol White <carol@example.com>
Subject: Fwd: Vendor Proposal
Date: %s
Message-ID: <msg008@example.com>
In-Reply-To: <msg007@example.com>
References: <msg007@example.com>

Bob, Carol,

FYI - vendor proposal attached.

Alice

`,
		baseTime.Format(time.RFC1123Z),
		baseTime.Add(1*time.Hour).Format(time.RFC1123Z),
		baseTime.Add(2*time.Hour).Format(time.RFC1123Z),
		baseTime.Add(3*time.Hour).Format(time.RFC1123Z),
		baseTime.Add(4*time.Hour).Format(time.RFC1123Z),
		baseTime.Add(5*time.Hour).Format(time.RFC1123Z),
		baseTime.Add(6*time.Hour).Format(time.RFC1123Z),
		baseTime.Add(7*time.Hour).Format(time.RFC1123Z),
	)

	return os.WriteFile(path, []byte(mbox), 0644)
}

// TestSessionRoundTrip tests that session save/load preserves data correctly.
func TestSessionRoundTrip(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create original session data
	originalSession := session.New()
	originalSession.SetConfig(session.SessionConfig{
		InternalDomains: []string{"test.com"},
		AutoCreate:      true,
		Format:          "mbox",
	})

	// Add test data
	originalSession.AddMessage(&entity.Message{
		ID:      "msg-1",
		From:    "alice@test.com",
		To:      []string{"bob@test.com"},
		Subject: "Test",
		Date:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
	})

	originalSession.AddInteraction(&entity.Interaction{
		ID:       "int-1",
		From:     "alice",
		To:       "bob",
		EdgeType: entity.EdgeTypeTo,
	})

	originalSession.AddActor(&entity.Actor{
		ID:           "alice",
		DisplayName:  "Alice",
		Emails:       []string{"alice@test.com"},
		PrimaryEmail: "alice@test.com",
		Internal:     true,
	})

	originalSession.AddActor(&entity.Actor{
		ID:           "bob",
		DisplayName:  "Bob",
		Emails:       []string{"bob@test.com"},
		PrimaryEmail: "bob@test.com",
		Internal:     true,
	})

	originalSession.AddThread(&entity.Thread{
		ID:      "thread-1",
		Subject: "Test",
		Size:    1,
	})

	// Save session
	sessionPath := filepath.Join(tmpDir, "session.json")
	if err := originalSession.Save(sessionPath); err != nil {
		t.Fatalf("Session save failed: %v", err)
	}

	// Load session
	loadedSession, err := session.Load(sessionPath)
	if err != nil {
		t.Fatalf("Session load failed: %v", err)
	}

	// Verify counts
	if len(loadedSession.Messages) != len(originalSession.Messages) {
		t.Errorf("Messages: got %d, want %d", len(loadedSession.Messages), len(originalSession.Messages))
	}
	if len(loadedSession.Interactions) != len(originalSession.Interactions) {
		t.Errorf("Interactions: got %d, want %d", len(loadedSession.Interactions), len(originalSession.Interactions))
	}
	if len(loadedSession.Actors) != len(originalSession.Actors) {
		t.Errorf("Actors: got %d, want %d", len(loadedSession.Actors), len(originalSession.Actors))
	}
	if len(loadedSession.Threads) != len(originalSession.Threads) {
		t.Errorf("Threads: got %d, want %d", len(loadedSession.Threads), len(originalSession.Threads))
	}

	// Verify config
	if loadedSession.Config.Format != originalSession.Config.Format {
		t.Errorf("Config.Format: got %q, want %q", loadedSession.Config.Format, originalSession.Config.Format)
	}

	// Convert to store and resolver
	store, err := loadedSession.ToMemoryStore(ctx)
	if err != nil {
		t.Fatalf("ToMemoryStore failed: %v", err)
	}

	msg, err := store.GetMessage(ctx, "msg-1")
	if err != nil || msg == nil {
		t.Error("Failed to get message from loaded store")
	}

	resolver := loadedSession.ToResolver()
	actorID, err := resolver.Resolve("alice@test.com")
	if err != nil || actorID != "alice" {
		t.Errorf("Failed to resolve alice: got %q, err=%v", actorID, err)
	}
}

// TestKnownResults tests that analysis produces expected results on known data.
func TestKnownResults(t *testing.T) {
	ctx := context.Background()

	// Create a simple star graph: center -> A, B, C, D
	// Center should have highest out-degree
	store := storage.NewMemoryStore()
	resolverConfig := identity.DefaultConfig()
	resolverConfig.AutoCreate = true
	resolver := identity.NewSCIMResolver(resolverConfig)

	center := resolver.ResolveOrCreate("center@test.com")
	actors := []entity.ActorID{
		resolver.ResolveOrCreate("a@test.com"),
		resolver.ResolveOrCreate("b@test.com"),
		resolver.ResolveOrCreate("c@test.com"),
		resolver.ResolveOrCreate("d@test.com"),
	}

	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	for i, to := range actors {
		msgID := fmt.Sprintf("msg-%d", i)
		_ = store.StoreMessage(ctx, &entity.Message{
			ID:   msgID,
			From: "center@test.com",
			To:   []string{string(to) + "@test.com"},
			Date: baseTime.Add(time.Duration(i) * time.Hour),
		})
		_ = store.StoreInteraction(ctx, &entity.Interaction{
			ID:        fmt.Sprintf("int-%d", i),
			MessageID: msgID,
			From:      center,
			To:        to,
			EdgeType:  entity.EdgeTypeTo,
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
		})
	}

	registry := weight.NewRegistry()
	profile, _ := registry.Get("influence")
	analyzer := analysis.NewAnalyzer(store, resolver, profile)

	// Test out-degree
	outDegree, err := analyzer.OutDegree(ctx)
	if err != nil {
		t.Fatalf("OutDegree failed: %v", err)
	}

	if len(outDegree) == 0 {
		t.Fatal("OutDegree returned no results")
	}

	// Center should have highest out-degree (4 outgoing edges)
	topActor := outDegree[0].ActorID
	if topActor != center {
		t.Errorf("Top out-degree actor = %s, want %s (center)", topActor, center)
	}

	// Test in-degree - leaf nodes should have degree 1
	inDegree, err := analyzer.InDegree(ctx)
	if err != nil {
		t.Fatalf("InDegree failed: %v", err)
	}

	// All leaf nodes should have in-degree of 1
	for _, actor := range actors {
		found := false
		for _, result := range inDegree {
			if result.ActorID == actor {
				found = true
				if result.Score != 1.0 {
					t.Errorf("Leaf actor %s in-degree = %.1f, want 1.0", actor, result.Score)
				}
				break
			}
		}
		if !found {
			t.Errorf("Leaf actor %s not found in in-degree results", actor)
		}
	}

	// Test degree (total) - center should have 4, leaves should have 1
	degree, err := analyzer.Degree(ctx)
	if err != nil {
		t.Fatalf("Degree failed: %v", err)
	}

	for _, result := range degree {
		if result.ActorID == center {
			if result.Score != 4.0 {
				t.Errorf("Center degree = %.1f, want 4.0", result.Score)
			}
		}
	}
}
