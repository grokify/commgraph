package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var analyzeBridgesCmd = &cobra.Command{
	Use:   "bridges",
	Short: "Detect bridge actors between communities",
	Long:  `Identify actors who bridge different communities or span structural holes in the network.`,
	RunE:  runAnalyzeBridges,
}

var (
	bridgesAlgorithm  string
	bridgesResolution float64
	bridgesTopN       int
	bridgesOutput     string
	bridgesFormat     string
)

func init() {
	analyzeCmd.AddCommand(analyzeBridgesCmd)

	analyzeBridgesCmd.Flags().StringVarP(&bridgesAlgorithm, "algorithm", "a", "community", "detection method (community, structural_holes, gatekeepers)")
	analyzeBridgesCmd.Flags().Float64VarP(&bridgesResolution, "resolution", "r", 1.0, "resolution for community detection")
	analyzeBridgesCmd.Flags().IntVarP(&bridgesTopN, "top", "n", 20, "show top N bridges")
	analyzeBridgesCmd.Flags().StringVarP(&bridgesOutput, "output", "o", "", "output file (default: stdout)")
	analyzeBridgesCmd.Flags().StringVarP(&bridgesFormat, "format", "f", "table", "output format (table, json)")
}

func runAnalyzeBridges(cmd *cobra.Command, args []string) error {
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

	// Determine output
	var out = os.Stdout
	if bridgesOutput != "" {
		f, err := os.Create(bridgesOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch bridgesAlgorithm {
	case "community":
		return detectCommunityBridges(ctx, analyzer, out)
	case "structural_holes":
		return detectStructuralHoles(ctx, analyzer, out)
	case "gatekeepers":
		return detectGatekeepers(ctx, analyzer, out)
	default:
		return fmt.Errorf("unknown algorithm: %s", bridgesAlgorithm)
	}
}

func detectCommunityBridges(ctx context.Context, analyzer *analysis.Analyzer, out *os.File) error {
	// First detect communities
	fmt.Fprintln(os.Stderr, "Detecting communities...")
	communities, err := analyzer.Louvain(ctx, bridgesResolution)
	if err != nil {
		return fmt.Errorf("community detection: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d communities (modularity: %.4f)\n", len(communities.Communities), communities.Modularity)

	// Then detect bridges
	fmt.Fprintln(os.Stderr, "Detecting bridges...")
	bridges, err := analyzer.DetectBridges(ctx, communities)
	if err != nil {
		return fmt.Errorf("bridge detection: %w", err)
	}

	topBridges := bridges.Top(bridgesTopN)

	return outputBridges(out, topBridges, "Community Bridges", communities)
}

func detectStructuralHoles(ctx context.Context, analyzer *analysis.Analyzer, out *os.File) error {
	fmt.Fprintln(os.Stderr, "Analyzing structural holes...")
	bridges, err := analyzer.StructuralHoles(ctx)
	if err != nil {
		return fmt.Errorf("structural holes: %w", err)
	}

	topBridges := bridges.Top(bridgesTopN)

	return outputBridges(out, topBridges, "Structural Hole Spanners", nil)
}

func detectGatekeepers(ctx context.Context, analyzer *analysis.Analyzer, out *os.File) error {
	// Detect communities first
	fmt.Fprintln(os.Stderr, "Detecting communities...")
	communities, err := analyzer.Louvain(ctx, bridgesResolution)
	if err != nil {
		return fmt.Errorf("community detection: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Detecting gatekeepers...")
	gatekeepers, err := analyzer.DetectGatekeepers(ctx, communities, bridgesTopN*2)
	if err != nil {
		return fmt.Errorf("gatekeeper detection: %w", err)
	}

	if len(gatekeepers) > bridgesTopN {
		gatekeepers = gatekeepers[:bridgesTopN]
	}

	switch bridgesFormat {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(gatekeepers)

	case "table":
		fmt.Fprintf(out, "\nGatekeeper Detection (top %d)\n", bridgesTopN)
		fmt.Fprintf(out, "%-5s %-40s %-10s %-10s %-12s\n", "Rank", "Actor", "BtwRank", "Groups", "Score")
		fmt.Fprintf(out, "%-5s %-40s %-10s %-10s %-12s\n", "----", "-----", "-------", "------", "-----")

		for i, g := range gatekeepers {
			name := g.DisplayName
			if name == "" {
				name = string(g.ActorID)
			}
			if len(name) > 38 {
				name = name[:35] + "..."
			}
			fmt.Fprintf(out, "%-5d %-40s %-10d %-10d %-12.4f\n",
				i+1, name, g.BetweennessRank, g.GroupsConnected, g.GatekeeperScore)
		}
		fmt.Fprintf(out, "\nTotal gatekeepers analyzed: %d\n", len(gatekeepers))

	default:
		return fmt.Errorf("unknown format: %s", bridgesFormat)
	}

	return nil
}

func outputBridges(out *os.File, bridges analysis.BridgeResults, title string, communities *analysis.CommunityResults) error {
	switch bridgesFormat {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(bridges)

	case "table":
		fmt.Fprintf(out, "\n%s (top %d)\n", title, bridgesTopN)
		fmt.Fprintf(out, "%-5s %-40s %-8s %-8s %-8s %-10s\n", "Rank", "Actor", "Comm", "ExtEdge", "IntEdge", "Score")
		fmt.Fprintf(out, "%-5s %-40s %-8s %-8s %-8s %-10s\n", "----", "-----", "----", "-------", "-------", "-----")

		for i, b := range bridges {
			name := b.DisplayName
			if name == "" {
				name = string(b.ActorID)
			}
			if len(name) > 38 {
				name = name[:35] + "..."
			}
			fmt.Fprintf(out, "%-5d %-40s %-8d %-8d %-8d %-10.4f\n",
				i+1, name, b.PrimaryCommunity, b.ExternalEdges, b.InternalEdges, b.BridgeScore)
		}

		if communities != nil {
			fmt.Fprintf(out, "\nCommunities: %d (modularity: %.4f)\n", len(communities.Communities), communities.Modularity)
		}

	default:
		return fmt.Errorf("unknown format: %s", bridgesFormat)
	}

	return nil
}
