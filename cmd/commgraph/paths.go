package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var analyzePathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Analyze paths in the communication graph",
	Long:  `Find shortest paths, calculate average path length, and analyze network connectivity.`,
	RunE:  runAnalyzePaths,
}

var (
	pathsFrom   string
	pathsTo     string
	pathsDepth  int
	pathsSample int
	pathsOutput string
	pathsFormat string
)

func init() {
	analyzeCmd.AddCommand(analyzePathsCmd)

	analyzePathsCmd.Flags().StringVar(&pathsFrom, "from", "", "source actor ID for path finding")
	analyzePathsCmd.Flags().StringVar(&pathsTo, "to", "", "target actor ID for path finding")
	analyzePathsCmd.Flags().IntVarP(&pathsDepth, "depth", "d", 5, "maximum path depth")
	analyzePathsCmd.Flags().IntVarP(&pathsSample, "sample", "s", 100, "sample size for average path length")
	analyzePathsCmd.Flags().StringVarP(&pathsOutput, "output", "o", "", "output file (default: stdout)")
	analyzePathsCmd.Flags().StringVarP(&pathsFormat, "format", "f", "table", "output format (table, json)")
}

func runAnalyzePaths(cmd *cobra.Command, args []string) error {
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
	if pathsOutput != "" {
		f, err := os.Create(pathsOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	// Run appropriate analysis
	if pathsFrom != "" && pathsTo != "" {
		// Find specific path
		return findPath(ctx, analyzer, out)
	}

	// General path statistics
	return pathStatistics(ctx, analyzer, out)
}

func findPath(ctx context.Context, analyzer *analysis.Analyzer, out *os.File) error {
	from := entity.ActorID(pathsFrom)
	to := entity.ActorID(pathsTo)

	// Find shortest path
	result, err := analyzer.ShortestPath(ctx, from, to)
	if err != nil {
		return fmt.Errorf("shortest path: %w", err)
	}

	switch pathsFormat {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)

	case "table":
		if result == nil {
			fmt.Fprintf(out, "No path found between %s and %s\n", from, to)
			return nil
		}

		fmt.Fprintf(out, "\nShortest Path: %s -> %s\n", from, to)
		fmt.Fprintf(out, "Distance: %d hops\n\n", result.Distance)
		fmt.Fprintf(out, "Path:\n")
		for i, actor := range result.Path {
			if i > 0 {
				fmt.Fprintf(out, "  |\n  v\n")
			}
			fmt.Fprintf(out, "  [%d] %s\n", i, actor)
		}

		// Also find all paths up to depth
		if pathsDepth > 0 {
			allPaths, err := analyzer.AllPathsUpToDepth(ctx, from, to, pathsDepth)
			if err == nil && len(allPaths) > 1 {
				fmt.Fprintf(out, "\nAlternative paths (up to depth %d): %d found\n", pathsDepth, len(allPaths))
			}
		}

	default:
		return fmt.Errorf("unknown format: %s", pathsFormat)
	}

	return nil
}

func pathStatistics(ctx context.Context, analyzer *analysis.Analyzer, out *os.File) error {
	// Calculate various path statistics
	avgLength, err := analyzer.AveragePathLength(ctx, pathsSample)
	if err != nil {
		return fmt.Errorf("average path length: %w", err)
	}

	diameter, err := analyzer.GraphDiameter(ctx, pathsSample)
	if err != nil {
		return fmt.Errorf("graph diameter: %w", err)
	}

	components, err := analyzer.ConnectedComponents(ctx)
	if err != nil {
		return fmt.Errorf("connected components: %w", err)
	}

	stats := struct {
		AveragePathLength float64 `json:"average_path_length"`
		Diameter          int     `json:"diameter"`
		NumComponents     int     `json:"num_components"`
		LargestComponent  int     `json:"largest_component"`
		Components        []int   `json:"component_sizes"`
	}{
		AveragePathLength: avgLength,
		Diameter:          diameter,
		NumComponents:     len(components),
	}

	// Get component sizes
	for _, comp := range components {
		stats.Components = append(stats.Components, len(comp))
		if len(comp) > stats.LargestComponent {
			stats.LargestComponent = len(comp)
		}
	}

	switch pathsFormat {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(stats)

	case "table":
		fmt.Fprintf(out, "\nPath Analysis (sample size: %d)\n", pathsSample)
		fmt.Fprintf(out, "%-25s %v\n", "Average Path Length:", fmt.Sprintf("%.2f hops", avgLength))
		fmt.Fprintf(out, "%-25s %d hops\n", "Graph Diameter:", diameter)
		fmt.Fprintf(out, "%-25s %d\n", "Connected Components:", len(components))
		fmt.Fprintf(out, "%-25s %d actors\n", "Largest Component:", stats.LargestComponent)

		if len(components) <= 10 {
			fmt.Fprintf(out, "\nComponent sizes: ")
			for i, size := range stats.Components {
				if i > 0 {
					fmt.Fprintf(out, ", ")
				}
				fmt.Fprintf(out, "%d", size)
			}
			fmt.Fprintf(out, "\n")
		}

	default:
		return fmt.Errorf("unknown format: %s", pathsFormat)
	}

	return nil
}
