package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/export"
	"github.com/grokify/commgraph/storage"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data and results",
	Long:  `Export graph data, analysis results, or visualization files.`,
}

var exportActorsCmd = &cobra.Command{
	Use:   "actors",
	Short: "Export actor data",
	RunE:  runExportActors,
}

var exportInteractionsCmd = &cobra.Command{
	Use:   "interactions",
	Short: "Export interaction data",
	RunE:  runExportInteractions,
}

var exportThreadsCmd = &cobra.Command{
	Use:   "threads",
	Short: "Export thread data",
	RunE:  runExportThreads,
}

var exportGephiCmd = &cobra.Command{
	Use:   "gephi",
	Short: "Export to Gephi (GEXF format)",
	Long:  `Export graph to GEXF format for Gephi visualization.`,
	RunE:  runExportGephi,
}

var exportNeo4jCmd = &cobra.Command{
	Use:   "neo4j",
	Short: "Export to Neo4j (Cypher format)",
	Long:  `Export graph to Cypher format for Neo4j import.`,
	RunE:  runExportNeo4j,
}

var (
	exportOutput             string
	exportFormat             string
	exportIncludeCommunities bool
	exportBatchSize          int
)

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportActorsCmd)
	exportCmd.AddCommand(exportInteractionsCmd)
	exportCmd.AddCommand(exportThreadsCmd)
	exportCmd.AddCommand(exportGephiCmd)
	exportCmd.AddCommand(exportNeo4jCmd)

	// Common flags for basic exports
	for _, cmd := range []*cobra.Command{exportActorsCmd, exportInteractionsCmd, exportThreadsCmd} {
		cmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (default: stdout)")
		cmd.Flags().StringVarP(&exportFormat, "format", "f", "csv", "output format (csv, json)")
	}

	// Gephi flags
	exportGephiCmd.Flags().StringVarP(&exportOutput, "output", "o", "commgraph.gexf", "output file")
	exportGephiCmd.Flags().BoolVar(&exportIncludeCommunities, "communities", true, "include community detection")

	// Neo4j flags
	exportNeo4jCmd.Flags().StringVarP(&exportOutput, "output", "o", "commgraph.cypher", "output file")
	exportNeo4jCmd.Flags().BoolVar(&exportIncludeCommunities, "communities", true, "include community detection")
	exportNeo4jCmd.Flags().IntVar(&exportBatchSize, "batch-size", 500, "statements per batch")
}

func runExportActors(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if globalStore == nil {
		return fmt.Errorf("no data loaded. Run 'commgraph ingest' first")
	}

	actors, err := globalStore.ListActors(ctx, storage.ListOptions{})
	if err != nil {
		return fmt.Errorf("list actors: %w", err)
	}

	var out = os.Stdout
	if exportOutput != "" {
		f, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch exportFormat {
	case "csv":
		exporter := export.NewCSVExporter()
		return exporter.ExportActors(out, actors)
	case "json":
		exporter := export.NewJSONExporter(true)
		return exporter.ExportActors(out, actors, export.Metadata{})
	default:
		return fmt.Errorf("unknown format: %s", exportFormat)
	}
}

func runExportInteractions(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if globalStore == nil {
		return fmt.Errorf("no data loaded. Run 'commgraph ingest' first")
	}

	interactions, err := globalStore.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return fmt.Errorf("get interactions: %w", err)
	}

	var out = os.Stdout
	if exportOutput != "" {
		f, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch exportFormat {
	case "csv":
		exporter := export.NewCSVExporter()
		return exporter.ExportInteractions(out, interactions)
	case "json":
		exporter := export.NewJSONExporter(true)
		return exporter.ExportActors(out, interactions, export.Metadata{})
	default:
		return fmt.Errorf("unknown format: %s", exportFormat)
	}
}

func runExportThreads(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if globalStore == nil {
		return fmt.Errorf("no data loaded. Run 'commgraph ingest' first")
	}

	threads, err := globalStore.ListThreads(ctx, storage.ListOptions{})
	if err != nil {
		return fmt.Errorf("list threads: %w", err)
	}

	var out = os.Stdout
	if exportOutput != "" {
		f, err := os.Create(exportOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch exportFormat {
	case "csv":
		exporter := export.NewCSVExporter()
		return exporter.ExportThreads(out, threads)
	case "json":
		exporter := export.NewJSONExporter(true)
		return exporter.ExportThreads(out, threads, export.Metadata{})
	default:
		return fmt.Errorf("unknown format: %s", exportFormat)
	}
}

func runExportGephi(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if globalStore == nil {
		return fmt.Errorf("no data loaded. Run 'commgraph ingest' first")
	}

	fmt.Println("Exporting to Gephi (GEXF) format...")

	// Get actors and interactions
	actors, interactions, err := getGraphData(ctx)
	if err != nil {
		return err
	}

	// Optionally run community detection
	var communities *analysis.CommunityResults
	if exportIncludeCommunities {
		fmt.Println("Running community detection...")
		registry := weight.NewRegistry()
		profile, _ := registry.Get("influence")
		analyzer := analysis.NewAnalyzer(globalStore, globalResolver, profile)
		communities, err = analyzer.Louvain(ctx, 1.0)
		if err != nil {
			fmt.Printf("Warning: community detection failed: %v\n", err)
		} else {
			fmt.Printf("Found %d communities (modularity: %.4f)\n", len(communities.Communities), communities.Modularity)
		}
	}

	// Create exporter
	exporter := export.NewGEXFExporter()

	// Create output file
	f, err := os.Create(exportOutput)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	meta := export.Metadata{
		Algorithm: "louvain",
		Parameters: map[string]any{
			"resolution": 1.0,
		},
	}

	if communities != nil {
		err = exporter.ExportGraphWithCommunities(f, actors, interactions, communities, meta)
	} else {
		err = exporter.ExportGraph(f, actors, interactions, meta)
	}

	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Printf("Exported %d actors and %d interactions to %s\n", len(actors), len(interactions), exportOutput)
	return nil
}

func runExportNeo4j(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if globalStore == nil {
		return fmt.Errorf("no data loaded. Run 'commgraph ingest' first")
	}

	fmt.Println("Exporting to Neo4j (Cypher) format...")

	// Get actors and interactions
	actors, interactions, err := getGraphData(ctx)
	if err != nil {
		return err
	}

	// Optionally run community detection
	var communities *analysis.CommunityResults
	if exportIncludeCommunities {
		fmt.Println("Running community detection...")
		registry := weight.NewRegistry()
		profile, _ := registry.Get("influence")
		analyzer := analysis.NewAnalyzer(globalStore, globalResolver, profile)
		communities, err = analyzer.Louvain(ctx, 1.0)
		if err != nil {
			fmt.Printf("Warning: community detection failed: %v\n", err)
		} else {
			fmt.Printf("Found %d communities (modularity: %.4f)\n", len(communities.Communities), communities.Modularity)
		}
	}

	// Create exporter
	exporter := export.NewCypherExporter()
	exporter.BatchSize = exportBatchSize

	// Export to file
	err = exporter.ExportToFile(exportOutput, actors, interactions, communities)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Printf("Exported %d actors and %d interactions to %s\n", len(actors), len(interactions), exportOutput)
	fmt.Println("\nTo import into Neo4j:")
	fmt.Println("  cat", exportOutput, "| cypher-shell -u neo4j -p <password>")
	return nil
}

// getGraphData retrieves actors and interactions from the store.
func getGraphData(ctx context.Context) ([]*entity.Actor, []*entity.Interaction, error) {
	// Get all interactions
	interactions, err := globalStore.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, nil, fmt.Errorf("get interactions: %w", err)
	}

	// Build actor set from interactions
	actorIDs := make(map[entity.ActorID]bool)
	for _, interaction := range interactions {
		actorIDs[interaction.From] = true
		actorIDs[interaction.To] = true
	}

	// Get actor details
	actors := make([]*entity.Actor, 0, len(actorIDs))
	for id := range actorIDs {
		actor, err := globalResolver.GetActor(id)
		if err != nil {
			// Create minimal actor if not found
			actor = &entity.Actor{
				ID:          id,
				DisplayName: string(id),
			}
		}
		actors = append(actors, actor)
	}

	return actors, interactions, nil
}
