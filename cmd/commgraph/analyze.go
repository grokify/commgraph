package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/export"
	"github.com/grokify/commgraph/session"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Run graph analysis",
	Long:  `Run various graph analysis algorithms on the communication graph.`,
}

var analyzeCentralityCmd = &cobra.Command{
	Use:   "centrality",
	Short: "Compute centrality metrics",
	Long:  `Compute centrality metrics (PageRank, degree, betweenness) for actors in the graph.`,
	RunE:  runAnalyzeCentrality,
}

var (
	analyzeProfile   string
	analyzeAlgorithm string
	analyzeTopN      int
	analyzeOutput    string
	analyzeFormat    string
	analyzeSession   string
)

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.AddCommand(analyzeCentralityCmd)

	analyzeCentralityCmd.Flags().StringVarP(&analyzeProfile, "profile", "p", "influence", "weight profile (influence, information_flow, coordination)")
	analyzeCentralityCmd.Flags().StringVarP(&analyzeAlgorithm, "algorithm", "a", "pagerank", "algorithm (pagerank, degree, in_degree, out_degree, betweenness)")
	analyzeCentralityCmd.Flags().IntVarP(&analyzeTopN, "top", "n", 20, "show top N results")
	analyzeCentralityCmd.Flags().StringVarP(&analyzeOutput, "output", "o", "", "output file (default: stdout)")
	analyzeCentralityCmd.Flags().StringVarP(&analyzeFormat, "format", "f", "table", "output format (table, json, csv)")
	analyzeCentralityCmd.Flags().StringVar(&analyzeSession, "session", "", "session file to load (default: .commgraph-session.json)")
}

func runAnalyzeCentrality(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load from session if not already loaded
	if globalStore == nil {
		sessionPath := analyzeSession
		if sessionPath == "" {
			sessionPath = session.DefaultPath()
		}

		if !session.Exists(sessionPath) {
			return fmt.Errorf("no data loaded and no session file found at %s. Run 'commgraph ingest' first", sessionPath)
		}

		store, resolver, err := session.LoadIntoStore(ctx, sessionPath)
		if err != nil {
			return fmt.Errorf("load session: %w", err)
		}
		globalStore = store
		globalResolver = resolver
		fmt.Printf("Loaded session from %s\n", sessionPath)
	}

	// Get profile
	registry := weight.NewRegistry()
	profile, err := registry.Get(analyzeProfile)
	if err != nil {
		return fmt.Errorf("unknown profile: %s (available: %v)", analyzeProfile, registry.List())
	}

	// Create analyzer
	analyzer := analysis.NewAnalyzer(globalStore, globalResolver, profile)

	// Run analysis
	var results analysis.CentralityResults
	switch analyzeAlgorithm {
	case "pagerank":
		results, err = analyzer.PageRank(ctx, 0.85, 100)
	case "degree":
		results, err = analyzer.Degree(ctx)
	case "in_degree":
		results, err = analyzer.InDegree(ctx)
	case "out_degree":
		results, err = analyzer.OutDegree(ctx)
	case "betweenness":
		results, err = analyzer.Betweenness(ctx)
	default:
		return fmt.Errorf("unknown algorithm: %s", analyzeAlgorithm)
	}

	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Apply top N
	if analyzeTopN > 0 {
		results = results.Top(analyzeTopN)
	}

	// Output results
	var out = os.Stdout
	if analyzeOutput != "" {
		f, err := os.Create(analyzeOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch analyzeFormat {
	case "json":
		exporter := export.NewJSONExporter(true)
		meta := export.Metadata{
			Profile:   analyzeProfile,
			Algorithm: analyzeAlgorithm,
			Parameters: map[string]any{
				"top_n": analyzeTopN,
			},
		}
		if err := exporter.ExportCentrality(out, results, meta); err != nil {
			return fmt.Errorf("export JSON: %w", err)
		}

	case "csv":
		exporter := export.NewCSVExporter()
		if err := exporter.ExportCentrality(out, results); err != nil {
			return fmt.Errorf("export CSV: %w", err)
		}

	case "table":
		fmt.Fprintf(out, "\nCentrality Analysis: %s (profile: %s)\n", analyzeAlgorithm, analyzeProfile)
		fmt.Fprintf(out, "%-5s %-40s %s\n", "Rank", "Actor", "Score")
		fmt.Fprintf(out, "%-5s %-40s %s\n", "----", "-----", "-----")
		for _, r := range results {
			name := r.DisplayName
			if name == "" {
				name = string(r.ActorID)
			}
			if len(name) > 38 {
				name = name[:35] + "..."
			}
			fmt.Fprintf(out, "%-5d %-40s %.6f\n", r.Rank, name, r.Score)
		}

	default:
		return fmt.Errorf("unknown format: %s", analyzeFormat)
	}

	return nil
}
