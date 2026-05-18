package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/export"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var analyzeCommunityCmd = &cobra.Command{
	Use:   "community",
	Short: "Detect communities in the graph",
	Long:  `Detect communities using Louvain or label propagation algorithms.`,
	RunE:  runAnalyzeCommunity,
}

var (
	communityAlgorithm  string
	communityResolution float64
	communityTopN       int
	communityOutput     string
	communityFormat     string
)

func init() {
	analyzeCmd.AddCommand(analyzeCommunityCmd)

	analyzeCommunityCmd.Flags().StringVarP(&communityAlgorithm, "algorithm", "a", "louvain", "algorithm (louvain, label_propagation)")
	analyzeCommunityCmd.Flags().Float64VarP(&communityResolution, "resolution", "r", 1.0, "resolution parameter for Louvain (higher = more communities)")
	analyzeCommunityCmd.Flags().IntVarP(&communityTopN, "top", "n", 10, "show top N communities")
	analyzeCommunityCmd.Flags().StringVarP(&communityOutput, "output", "o", "", "output file (default: stdout)")
	analyzeCommunityCmd.Flags().StringVarP(&communityFormat, "format", "f", "table", "output format (table, json)")
}

func runAnalyzeCommunity(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if globalStore == nil {
		return fmt.Errorf("no data loaded. Run 'commgraph ingest' first")
	}

	// Get profile
	registry := weight.NewRegistry()
	profile, err := registry.Get(analyzeProfile)
	if err != nil {
		return fmt.Errorf("unknown profile: %s", analyzeProfile)
	}

	// Create analyzer
	analyzer := analysis.NewAnalyzer(globalStore, globalResolver, profile)

	// Run community detection
	var results *analysis.CommunityResults
	switch communityAlgorithm {
	case "louvain":
		results, err = analyzer.Louvain(ctx, communityResolution)
	case "label_propagation":
		results, err = analyzer.LabelPropagation(ctx, 100)
	default:
		return fmt.Errorf("unknown algorithm: %s", communityAlgorithm)
	}

	if err != nil {
		return fmt.Errorf("community detection failed: %w", err)
	}

	// Output results
	var out = os.Stdout
	if communityOutput != "" {
		f, err := os.Create(communityOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	topCommunities := results.Top(communityTopN)

	switch communityFormat {
	case "json":
		exporter := export.NewJSONExporter(true)
		meta := export.Metadata{
			Profile:   analyzeProfile,
			Algorithm: communityAlgorithm,
			Parameters: map[string]any{
				"resolution": communityResolution,
				"modularity": results.Modularity,
			},
		}
		return exporter.ExportCommunities(out, topCommunities, meta)

	case "table":
		fmt.Fprintf(out, "\nCommunity Detection: %s (modularity: %.4f)\n", communityAlgorithm, results.Modularity)
		fmt.Fprintf(out, "%-5s %-10s %-10s %-10s\n", "ID", "Size", "Density", "Conductance")
		fmt.Fprintf(out, "%-5s %-10s %-10s %-10s\n", "--", "----", "-------", "-----------")
		for _, c := range topCommunities {
			fmt.Fprintf(out, "%-5d %-10d %-10.4f %-10.4f\n", c.ID, c.Size, c.Density, c.Conductance)
		}
		fmt.Fprintf(out, "\nTotal communities: %d\n", len(results.Communities))

	default:
		return fmt.Errorf("unknown format: %s", communityFormat)
	}

	return nil
}
