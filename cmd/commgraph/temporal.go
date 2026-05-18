package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/export"
	"github.com/grokify/commgraph/weight"
	"github.com/spf13/cobra"
)

var analyzeTemporalCmd = &cobra.Command{
	Use:   "temporal",
	Short: "Analyze temporal patterns",
	Long:  `Analyze communication patterns over time, detect bursts of activity.`,
	RunE:  runAnalyzeTemporal,
}

var (
	temporalGranularity string
	temporalThreshold   float64
	temporalOutput      string
	temporalFormat      string
)

func init() {
	analyzeCmd.AddCommand(analyzeTemporalCmd)

	analyzeTemporalCmd.Flags().StringVarP(&temporalGranularity, "granularity", "g", "day", "time granularity (hour, day, week)")
	analyzeTemporalCmd.Flags().Float64VarP(&temporalThreshold, "threshold", "t", 2.0, "burst detection threshold (std devs above mean)")
	analyzeTemporalCmd.Flags().StringVarP(&temporalOutput, "output", "o", "", "output file (default: stdout)")
	analyzeTemporalCmd.Flags().StringVarP(&temporalFormat, "format", "f", "table", "output format (table, json)")
}

func runAnalyzeTemporal(cmd *cobra.Command, args []string) error {
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

	// Detect bursts
	bursts, err := analyzer.BurstDetection(ctx, temporalThreshold, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("burst detection failed: %w", err)
	}

	// Output results
	var out = os.Stdout
	if temporalOutput != "" {
		f, err := os.Create(temporalOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch temporalFormat {
	case "json":
		exporter := export.NewJSONExporter(true)
		meta := export.Metadata{
			Profile:   analyzeProfile,
			Algorithm: "burst_detection",
			Parameters: map[string]any{
				"threshold":   temporalThreshold,
				"granularity": temporalGranularity,
			},
		}
		return exporter.ExportBursts(out, bursts, meta)

	case "table":
		fmt.Fprintf(out, "\nBurst Detection (threshold: %.1f std devs)\n", temporalThreshold)
		fmt.Fprintf(out, "%-12s %-12s %-12s %-8s %-8s %-8s\n", "Start", "Peak", "End", "PeakCnt", "Total", "Z-Score")
		fmt.Fprintf(out, "%-12s %-12s %-12s %-8s %-8s %-8s\n", "-----", "----", "---", "-------", "-----", "-------")
		for _, b := range bursts {
			fmt.Fprintf(out, "%-12s %-12s %-12s %-8d %-8d %-8.2f\n",
				b.Start.Format("2006-01-02"),
				b.Peak.Format("2006-01-02"),
				b.End.Format("2006-01-02"),
				b.PeakCount,
				b.Total,
				b.ZScore)
		}
		fmt.Fprintf(out, "\nTotal bursts detected: %d\n", len(bursts))

	default:
		return fmt.Errorf("unknown format: %s", temporalFormat)
	}

	return nil
}
