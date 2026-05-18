package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/session"
	"github.com/spf13/cobra"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identity resolution",
	Long:  `Commands for managing and inspecting identity resolution.`,
}

var identityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all resolved actors",
	Long:  `List all actors known to the identity resolver.`,
	RunE:  runIdentityList,
}

var identityAliasesCmd = &cobra.Command{
	Use:   "aliases [actor-id]",
	Short: "Show aliases for an actor",
	Long:  `Show all email addresses associated with an actor.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runIdentityAliases,
}

var identityStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show resolution statistics",
	Long:  `Show identity resolution statistics.`,
	RunE:  runIdentityStats,
}

var (
	identitySession  string
	identityFormat   string
	identityInternal bool
	identityExternal bool
	identityLimit    int
)

func init() {
	rootCmd.AddCommand(identityCmd)
	identityCmd.AddCommand(identityListCmd)
	identityCmd.AddCommand(identityAliasesCmd)
	identityCmd.AddCommand(identityStatsCmd)

	// Common flags
	identityCmd.PersistentFlags().StringVar(&identitySession, "session", "", "session file to load (default: .commgraph-session.json)")

	// List flags
	identityListCmd.Flags().StringVarP(&identityFormat, "format", "f", "table", "output format (table, json)")
	identityListCmd.Flags().BoolVar(&identityInternal, "internal", false, "show only internal actors")
	identityListCmd.Flags().BoolVar(&identityExternal, "external", false, "show only external actors")
	identityListCmd.Flags().IntVarP(&identityLimit, "limit", "n", 0, "limit number of results (0 = no limit)")

	// Aliases flags
	identityAliasesCmd.Flags().StringVarP(&identityFormat, "format", "f", "table", "output format (table, json)")

	// Stats flags
	identityStatsCmd.Flags().StringVarP(&identityFormat, "format", "f", "table", "output format (table, json)")
}

func loadIdentityData(ctx context.Context) error {
	if globalResolver != nil {
		return nil // Already loaded
	}

	sessionPath := identitySession
	if sessionPath == "" {
		sessionPath = session.DefaultPath()
	}

	if !session.Exists(sessionPath) {
		return fmt.Errorf("no session file found at %s. Run 'commgraph ingest' first", sessionPath)
	}

	store, resolver, err := session.LoadIntoStore(ctx, sessionPath)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	globalStore = store
	globalResolver = resolver
	return nil
}

func runIdentityList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := loadIdentityData(ctx); err != nil {
		return err
	}

	actors := globalResolver.AllActors()

	// Filter by internal/external
	if identityInternal || identityExternal {
		var filtered []*entity.Actor
		for _, actor := range actors {
			if identityInternal && actor.Internal {
				filtered = append(filtered, actor)
			} else if identityExternal && !actor.Internal {
				filtered = append(filtered, actor)
			}
		}
		actors = filtered
	}

	// Sort by ID
	sort.Slice(actors, func(i, j int) bool {
		return actors[i].ID < actors[j].ID
	})

	// Apply limit
	if identityLimit > 0 && identityLimit < len(actors) {
		actors = actors[:identityLimit]
	}

	switch identityFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(actors)

	case "table":
		fmt.Printf("\nActors (%d total):\n", len(actors))
		fmt.Printf("%-30s %-40s %-10s %s\n", "ID", "Display Name", "Type", "Emails")
		fmt.Printf("%-30s %-40s %-10s %s\n", "---", "------------", "----", "------")
		for _, actor := range actors {
			actorType := "external"
			if actor.Internal {
				actorType = "internal"
			}
			name := actor.DisplayName
			if len(name) > 38 {
				name = name[:35] + "..."
			}
			id := string(actor.ID)
			if len(id) > 28 {
				id = id[:25] + "..."
			}
			emailCount := len(actor.Emails)
			fmt.Printf("%-30s %-40s %-10s %d\n", id, name, actorType, emailCount)
		}

	default:
		return fmt.Errorf("unknown format: %s", identityFormat)
	}

	return nil
}

func runIdentityAliases(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := loadIdentityData(ctx); err != nil {
		return err
	}

	actorID := entity.ActorID(args[0])
	actor, err := globalResolver.GetActor(actorID)
	if err != nil {
		return fmt.Errorf("actor not found: %s", actorID)
	}

	aliases := globalResolver.Aliases(actorID)

	switch identityFormat {
	case "json":
		result := struct {
			ActorID     entity.ActorID `json:"actor_id"`
			DisplayName string         `json:"display_name"`
			Primary     string         `json:"primary_email"`
			Aliases     []string       `json:"aliases"`
		}{
			ActorID:     actorID,
			DisplayName: actor.DisplayName,
			Primary:     actor.PrimaryEmail,
			Aliases:     aliases,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)

	case "table":
		fmt.Printf("\nActor: %s\n", actorID)
		fmt.Printf("Display Name: %s\n", actor.DisplayName)
		fmt.Printf("Primary Email: %s\n", actor.PrimaryEmail)
		fmt.Printf("Internal: %t\n", actor.Internal)
		if actor.Department != "" {
			fmt.Printf("Department: %s\n", actor.Department)
		}
		if actor.Title != "" {
			fmt.Printf("Title: %s\n", actor.Title)
		}
		fmt.Printf("\nAliases (%d):\n", len(aliases))
		for _, alias := range aliases {
			marker := ""
			if alias == actor.PrimaryEmail {
				marker = " (primary)"
			}
			fmt.Printf("  %s%s\n", alias, marker)
		}

	default:
		return fmt.Errorf("unknown format: %s", identityFormat)
	}

	return nil
}

func runIdentityStats(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := loadIdentityData(ctx); err != nil {
		return err
	}

	stats := globalResolver.Stats()

	switch identityFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(stats)

	case "table":
		fmt.Printf("\nIdentity Resolution Statistics:\n")
		fmt.Printf("  Total Actors:      %d\n", stats.TotalActors)
		fmt.Printf("  Internal Actors:   %d\n", stats.InternalActors)
		fmt.Printf("  External Actors:   %d\n", stats.ExternalActors)
		fmt.Printf("  Total Aliases:     %d\n", stats.TotalAliases)
		fmt.Printf("  Resolution Hits:   %d\n", stats.ResolutionHits)
		fmt.Printf("  Resolution Misses: %d\n", stats.ResolutionMisses)
		fmt.Printf("  Auto-Created:      %d\n", stats.AutoCreated)
		if stats.ResolutionHits+stats.ResolutionMisses > 0 {
			hitRate := float64(stats.ResolutionHits) / float64(stats.ResolutionHits+stats.ResolutionMisses) * 100
			fmt.Printf("  Hit Rate:          %.1f%%\n", hitRate)
		}

	default:
		return fmt.Errorf("unknown format: %s", identityFormat)
	}

	return nil
}
