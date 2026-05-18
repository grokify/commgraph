// Command commgraph is a CLI for communication graph analysis.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "commgraph",
	Short: "Communication graph analysis framework",
	Long: `CommGraph analyzes communication patterns across email, chat, and other platforms.

It builds a graph of actors and their interactions, enabling analysis of:
  - Influence and centrality
  - Information flow
  - Organizational structure
  - Coordination patterns

Supported platforms:
  - Email (mbox, EML, PST)
  - Slack (export format) [planned]
  - Microsoft Teams (export format) [planned]`,
	Version: version,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("commgraph version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default: commgraph.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}
