package main

import (
	"context"
	"fmt"
	"os"

	"github.com/grokify/commgraph/export"
	"github.com/grokify/commgraph/storage"
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

var (
	exportOutput string
	exportFormat string
)

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.AddCommand(exportActorsCmd)
	exportCmd.AddCommand(exportInteractionsCmd)
	exportCmd.AddCommand(exportThreadsCmd)

	// Common flags
	for _, cmd := range []*cobra.Command{exportActorsCmd, exportInteractionsCmd, exportThreadsCmd} {
		cmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (default: stdout)")
		cmd.Flags().StringVarP(&exportFormat, "format", "f", "csv", "output format (csv, json)")
	}
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
