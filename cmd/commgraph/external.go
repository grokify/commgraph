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

var analyzeExternalCmd = &cobra.Command{
	Use:   "external",
	Short: "Analyze external entity communication",
	Long:  `Analyze communication patterns with external entities, domains, and boundary spanners.`,
	RunE:  runAnalyzeExternal,
}

var (
	externalTopN   int
	externalOutput string
	externalFormat string
	externalMode   string
)

func init() {
	analyzeCmd.AddCommand(analyzeExternalCmd)

	analyzeExternalCmd.Flags().IntVarP(&externalTopN, "top", "n", 20, "show top N results")
	analyzeExternalCmd.Flags().StringVarP(&externalOutput, "output", "o", "", "output file (default: stdout)")
	analyzeExternalCmd.Flags().StringVarP(&externalFormat, "format", "f", "table", "output format (table, json)")
	analyzeExternalCmd.Flags().StringVarP(&externalMode, "mode", "m", "summary", "analysis mode (summary, contacts, domains, ratios, spanners)")
}

func runAnalyzeExternal(cmd *cobra.Command, args []string) error {
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

	// Run analysis
	results, err := analyzer.ExternalAnalysis(ctx)
	if err != nil {
		return fmt.Errorf("external analysis failed: %w", err)
	}

	// Determine output
	var out = os.Stdout
	if externalOutput != "" {
		f, err := os.Create(externalOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch externalFormat {
	case "json":
		return outputExternalJSON(out, results)
	case "table":
		return outputExternalTable(out, results)
	default:
		return fmt.Errorf("unknown format: %s", externalFormat)
	}
}

func outputExternalJSON(out *os.File, results *analysis.ExternalAnalysisResults) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")

	switch externalMode {
	case "summary":
		return encoder.Encode(results.Summary)
	case "contacts":
		return encoder.Encode(results.TopExternalActors.Top(externalTopN))
	case "domains":
		top := results.TopDomains
		if externalTopN < len(top) {
			top = top[:externalTopN]
		}
		return encoder.Encode(top)
	case "ratios":
		top := results.InternalRatios
		if externalTopN < len(top) {
			top = top[:externalTopN]
		}
		return encoder.Encode(top)
	case "spanners":
		return encoder.Encode(results.BoundarySpanners)
	default:
		return encoder.Encode(results)
	}
}

func outputExternalTable(out *os.File, results *analysis.ExternalAnalysisResults) error {
	switch externalMode {
	case "summary":
		fmt.Fprintf(out, "\nExternal Communication Summary\n")
		fmt.Fprintf(out, "%-30s %v\n", "External Actors:", results.Summary.TotalExternalActors)
		fmt.Fprintf(out, "%-30s %v\n", "External Domains:", results.Summary.TotalExternalDomains)
		fmt.Fprintf(out, "%-30s %v\n", "External Interactions:", results.Summary.TotalExternalInteractions)
		fmt.Fprintf(out, "%-30s %v\n", "Inbound (from external):", results.Summary.InboundCount)
		fmt.Fprintf(out, "%-30s %v\n", "Outbound (to external):", results.Summary.OutboundCount)
		fmt.Fprintf(out, "%-30s %.2f%%\n", "External Ratio:", results.Summary.ExternalRatio*100)

	case "contacts":
		fmt.Fprintf(out, "\nTop External Contacts (top %d)\n", externalTopN)
		fmt.Fprintf(out, "%-5s %-35s %-20s %-8s %-8s %-8s\n", "Rank", "Actor", "Domain", "In", "Out", "Total")
		fmt.Fprintf(out, "%-5s %-35s %-20s %-8s %-8s %-8s\n", "----", "-----", "------", "--", "---", "-----")
		for i, c := range results.TopExternalActors.Top(externalTopN) {
			name := c.DisplayName
			if name == "" {
				name = string(c.ActorID)
			}
			if len(name) > 33 {
				name = name[:30] + "..."
			}
			domain := c.Domain
			if len(domain) > 18 {
				domain = domain[:15] + "..."
			}
			fmt.Fprintf(out, "%-5d %-35s %-20s %-8d %-8d %-8d\n",
				i+1, name, domain, c.InboundCount, c.OutboundCount, c.TotalCount)
		}

	case "domains":
		fmt.Fprintf(out, "\nTop External Domains (top %d)\n", externalTopN)
		fmt.Fprintf(out, "%-5s %-30s %-10s %-8s %-8s %-8s %-10s\n", "Rank", "Domain", "Contacts", "In", "Out", "Total", "Internal")
		fmt.Fprintf(out, "%-5s %-30s %-10s %-8s %-8s %-8s %-10s\n", "----", "------", "--------", "--", "---", "-----", "--------")
		top := results.TopDomains
		if externalTopN < len(top) {
			top = top[:externalTopN]
		}
		for i, d := range top {
			domain := d.Domain
			if len(domain) > 28 {
				domain = domain[:25] + "..."
			}
			fmt.Fprintf(out, "%-5d %-30s %-10d %-8d %-8d %-8d %-10d\n",
				i+1, domain, d.TotalContacts, d.InboundCount, d.OutboundCount, d.TotalCount, d.UniqueInternal)
		}

	case "ratios":
		fmt.Fprintf(out, "\nInternal Actors by External Ratio (top %d)\n", externalTopN)
		fmt.Fprintf(out, "%-5s %-35s %-8s %-8s %-8s %-10s\n", "Rank", "Actor", "Ext", "Int", "Total", "ExtRatio")
		fmt.Fprintf(out, "%-5s %-35s %-8s %-8s %-8s %-10s\n", "----", "-----", "---", "---", "-----", "--------")
		top := results.InternalRatios
		if externalTopN < len(top) {
			top = top[:externalTopN]
		}
		for i, r := range top {
			name := r.DisplayName
			if name == "" {
				name = string(r.ActorID)
			}
			if len(name) > 33 {
				name = name[:30] + "..."
			}
			fmt.Fprintf(out, "%-5d %-35s %-8d %-8d %-8d %-10.2f%%\n",
				i+1, name, r.ExternalCount, r.InternalCount, r.TotalCount, r.ExternalRatio*100)
		}

	case "spanners":
		fmt.Fprintf(out, "\nBoundary Spanners (high external communication)\n")
		fmt.Fprintf(out, "%-5s %-35s %-10s %-8s %-10s\n", "Rank", "Actor", "ExtRatio", "Domains", "TopDomains")
		fmt.Fprintf(out, "%-5s %-35s %-10s %-8s %-10s\n", "----", "-----", "--------", "-------", "----------")
		for i, r := range results.BoundarySpanners {
			name := r.DisplayName
			if name == "" {
				name = string(r.ActorID)
			}
			if len(name) > 33 {
				name = name[:30] + "..."
			}
			domains := ""
			if len(r.TopDomains) > 0 {
				for j, d := range r.TopDomains {
					if j > 0 {
						domains += ", "
					}
					if len(d) > 10 {
						d = d[:7] + "..."
					}
					domains += d
				}
			}
			fmt.Fprintf(out, "%-5d %-35s %-10.2f%% %-8d %s\n",
				i+1, name, r.ExternalRatio*100, r.UniqueDomains, domains)
		}
		fmt.Fprintf(out, "\nTotal boundary spanners: %d\n", len(results.BoundarySpanners))

	default:
		// Show summary by default
		return outputExternalTable(out, results)
	}

	return nil
}
