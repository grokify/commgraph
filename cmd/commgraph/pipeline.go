package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grokify/commgraph/adapter"
	"github.com/grokify/commgraph/adapter/email"
	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/identity"
	"github.com/grokify/commgraph/storage"
	"github.com/grokify/commgraph/threading"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Run full ingestion and analysis pipeline",
	Long:  `Ingest messages, reconstruct threads, and run analysis in a single command.`,
	RunE:  runPipeline,
}

var (
	pipelineSource    string
	pipelineFormat    string
	pipelineDomains   []string
	pipelineProfile   string
	pipelineAlgorithm string
	pipelineTopN      int
	pipelineLoadEnron bool
)

func init() {
	rootCmd.AddCommand(pipelineCmd)

	pipelineCmd.Flags().StringVarP(&pipelineSource, "source", "s", "", "source file or directory (required)")
	pipelineCmd.Flags().StringVarP(&pipelineFormat, "format", "f", "maildir", "source format (mbox, maildir)")
	pipelineCmd.Flags().StringSliceVarP(&pipelineDomains, "internal-domains", "d", []string{}, "internal domains")
	pipelineCmd.Flags().StringVarP(&pipelineProfile, "profile", "p", "influence", "weight profile")
	pipelineCmd.Flags().StringVarP(&pipelineAlgorithm, "algorithm", "a", "pagerank", "analysis algorithm")
	pipelineCmd.Flags().IntVarP(&pipelineTopN, "top", "n", 20, "show top N results")
	pipelineCmd.Flags().BoolVar(&pipelineLoadEnron, "enron", false, "load Enron employee identities for alias merging")

	_ = pipelineCmd.MarkFlagRequired("source")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate source
	if _, err := os.Stat(pipelineSource); err != nil {
		return fmt.Errorf("source not found: %s", pipelineSource)
	}

	// Create adapter
	var adp adapter.Adapter
	switch pipelineFormat {
	case "mbox":
		adp = email.NewMboxAdapter()
	case "maildir":
		adp = email.NewMaildirAdapter()
	default:
		return fmt.Errorf("unsupported format: %s", pipelineFormat)
	}

	// Create store
	store := storage.NewMemoryStore()

	// Create identity resolver
	resolverConfig := identity.DefaultConfig()
	resolverConfig.InternalDomains = pipelineDomains
	resolverConfig.AutoCreate = true
	resolver := identity.NewSCIMResolver(resolverConfig)

	// Load Enron people if requested
	if pipelineLoadEnron {
		fmt.Println("Loading Enron employee identities...")
		count := resolver.LoadEnronPeople()
		fmt.Printf("  Loaded %d employees with aliases\n", count)
	}

	// Ingest messages
	source := adapter.FileSource{
		SourceType: pipelineFormat,
		Path:       pipelineSource,
	}

	fmt.Printf("Ingesting from %s...\n", pipelineSource)
	start := time.Now()

	msgCh, errCh := adp.Ingest(ctx, source)

	var messages []*entity.Message
	var msgCount int

	for msg := range msgCh {
		messages = append(messages, msg)
		msgCount++

		// Store message
		_ = store.StoreMessage(ctx, msg)

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

	// Store threads
	for _, thread := range threads {
		_ = store.StoreThread(ctx, thread)
	}

	threadDuration := time.Since(threadStart)

	// Get stats
	stats, _ := store.Stats(ctx)
	resolverStats := resolver.Stats()
	threadStats := threading.ComputeStats(threads)

	// Print ingestion summary
	fmt.Printf("\nIngestion complete:\n")
	fmt.Printf("  Messages:     %d\n", stats.MessageCount)
	fmt.Printf("  Interactions: %d\n", stats.InteractionCount)
	fmt.Printf("  Actors:       %d (%d internal, %d external)\n",
		resolverStats.TotalActors, resolverStats.InternalActors, resolverStats.ExternalActors)
	fmt.Printf("  Threads:      %d (%d single, %d multi-message)\n",
		threadStats.TotalThreads, threadStats.SingleMessageThreads, threadStats.MultiMessageThreads)
	fmt.Printf("  Duration:     %v (ingest) + %v (threading)\n", ingestDuration, threadDuration)

	// Run analysis
	fmt.Printf("\nRunning %s analysis with %s profile...\n", pipelineAlgorithm, pipelineProfile)
	analysisStart := time.Now()

	// Get profile
	registry := weight.NewRegistry()
	profile, err := registry.Get(pipelineProfile)
	if err != nil {
		return fmt.Errorf("unknown profile: %s", pipelineProfile)
	}

	// Create analyzer
	analyzer := analysis.NewAnalyzer(store, resolver, profile)

	// Run analysis
	var results analysis.CentralityResults
	switch pipelineAlgorithm {
	case "pagerank":
		results, err = analyzer.PageRank(ctx, 0.85, 100)
	case "degree":
		results, err = analyzer.Degree(ctx)
	case "in_degree":
		results, err = analyzer.InDegree(ctx)
	case "out_degree":
		results, err = analyzer.OutDegree(ctx)
	default:
		return fmt.Errorf("unknown algorithm: %s", pipelineAlgorithm)
	}

	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	analysisDuration := time.Since(analysisStart)

	// Apply top N
	if pipelineTopN > 0 {
		results = results.Top(pipelineTopN)
	}

	// Print results
	fmt.Printf("\n%s Results (top %d):\n", pipelineAlgorithm, pipelineTopN)
	fmt.Printf("%-5s %-50s %s\n", "Rank", "Actor", "Score")
	fmt.Printf("%-5s %-50s %s\n", "----", "-----", "-----")
	for _, r := range results {
		name := r.DisplayName
		if name == "" {
			name = string(r.ActorID)
		}
		if len(name) > 48 {
			name = name[:45] + "..."
		}
		fmt.Printf("%-5d %-50s %.6f\n", r.Rank, name, r.Score)
	}

	fmt.Printf("\nAnalysis completed in %v\n", analysisDuration)

	// Run community detection
	fmt.Println("\nDetecting communities...")
	communityStart := time.Now()
	communities, err := analyzer.Louvain(ctx, 1.0)
	if err != nil {
		fmt.Printf("  Community detection failed: %v\n", err)
	} else {
		fmt.Printf("  Found %d communities (modularity: %.4f)\n", len(communities.Communities), communities.Modularity)
		fmt.Printf("  Top 5 communities:\n")
		for _, c := range communities.Top(5) {
			fmt.Printf("    Community %d: %d members (density: %.3f)\n", c.ID, c.Size, c.Density)
		}
		fmt.Printf("  Completed in %v\n", time.Since(communityStart))
	}

	// Run burst detection
	fmt.Println("\nDetecting activity bursts...")
	burstStart := time.Now()
	bursts, err := analyzer.BurstDetection(ctx, 2.0, 24*time.Hour)
	if err != nil {
		fmt.Printf("  Burst detection failed: %v\n", err)
	} else {
		fmt.Printf("  Found %d bursts (>2 std devs above mean)\n", len(bursts))
		if len(bursts) > 0 {
			fmt.Printf("  Top bursts:\n")
			for i, b := range bursts {
				if i >= 5 {
					break
				}
				fmt.Printf("    %s: z-score=%.2f, peak=%d messages\n",
					b.Peak.Format("2006-01-02"), b.ZScore, b.PeakCount)
			}
		}
		fmt.Printf("  Completed in %v\n", time.Since(burstStart))
	}

	// Run path analysis
	fmt.Println("\nAnalyzing network paths...")
	pathStart := time.Now()
	avgPath, err := analyzer.AveragePathLength(ctx, 100)
	if err != nil {
		fmt.Printf("  Path analysis failed: %v\n", err)
	} else {
		diameter, _ := analyzer.GraphDiameter(ctx, 100)
		components, _ := analyzer.ConnectedComponents(ctx)
		fmt.Printf("  Average path length: %.2f hops\n", avgPath)
		fmt.Printf("  Graph diameter: %d hops\n", diameter)
		fmt.Printf("  Connected components: %d\n", len(components))
		fmt.Printf("  Completed in %v\n", time.Since(pathStart))
	}

	// Run bridge detection
	if communities != nil {
		fmt.Println("\nDetecting bridge actors...")
		bridgeStart := time.Now()
		bridges, err := analyzer.DetectBridges(ctx, communities)
		if err != nil {
			fmt.Printf("  Bridge detection failed: %v\n", err)
		} else {
			fmt.Printf("  Top 5 bridges (cross-community connectors):\n")
			for _, b := range bridges.Top(5) {
				name := b.DisplayName
				if name == "" {
					name = string(b.ActorID)
				}
				if len(name) > 30 {
					name = name[:27] + "..."
				}
				fmt.Printf("    %s: score=%.2f, %d communities, %d external edges\n",
					name, b.BridgeScore, b.CommunitiesConnected, b.ExternalEdges)
			}
			fmt.Printf("  Completed in %v\n", time.Since(bridgeStart))
		}
	}

	// Run external entity analysis
	fmt.Println("\nAnalyzing external communication...")
	externalStart := time.Now()
	externalResults, err := analyzer.ExternalAnalysis(ctx)
	if err != nil {
		fmt.Printf("  External analysis failed: %v\n", err)
	} else {
		fmt.Printf("  External actors: %d across %d domains\n",
			externalResults.Summary.TotalExternalActors, externalResults.Summary.TotalExternalDomains)
		fmt.Printf("  External interactions: %d (%.1f%% of total)\n",
			externalResults.Summary.TotalExternalInteractions, externalResults.Summary.ExternalRatio*100)
		fmt.Printf("  Inbound: %d, Outbound: %d\n",
			externalResults.Summary.InboundCount, externalResults.Summary.OutboundCount)
		if len(externalResults.BoundarySpanners) > 0 {
			fmt.Printf("  Boundary spanners: %d (high external communication)\n", len(externalResults.BoundarySpanners))
		}
		fmt.Printf("  Completed in %v\n", time.Since(externalStart))
	}

	return nil
}
