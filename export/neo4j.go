// Package export provides export functionality for analysis results.
package export

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
)

// CypherExporter exports data in Cypher format for Neo4j.
type CypherExporter struct {
	BatchSize int // Number of statements per batch (default: 500)
}

// NewCypherExporter creates a new Cypher exporter.
func NewCypherExporter() *CypherExporter {
	return &CypherExporter{BatchSize: 500}
}

// ExportSchema exports the Neo4j schema (constraints and indexes).
func (e *CypherExporter) ExportSchema(w io.Writer) error {
	schema := `// CommGraph Neo4j Schema
// Run these statements first to create constraints and indexes

// Constraints (ensure uniqueness)
CREATE CONSTRAINT actor_id IF NOT EXISTS FOR (a:Actor) REQUIRE a.id IS UNIQUE;
CREATE CONSTRAINT message_id IF NOT EXISTS FOR (m:Message) REQUIRE m.id IS UNIQUE;
CREATE CONSTRAINT thread_id IF NOT EXISTS FOR (t:Thread) REQUIRE t.id IS UNIQUE;
CREATE CONSTRAINT community_id IF NOT EXISTS FOR (c:Community) REQUIRE c.id IS UNIQUE;

// Indexes (improve query performance)
CREATE INDEX actor_email IF NOT EXISTS FOR (a:Actor) ON (a.primary_email);
CREATE INDEX actor_internal IF NOT EXISTS FOR (a:Actor) ON (a.internal);
CREATE INDEX actor_department IF NOT EXISTS FOR (a:Actor) ON (a.department);
CREATE INDEX message_date IF NOT EXISTS FOR (m:Message) ON (m.date);
CREATE INDEX interaction_type IF NOT EXISTS FOR ()-[r:SENT_TO]-() ON (r.edge_type);
`
	_, err := fmt.Fprint(w, schema)
	return err
}

// ExportActors exports actors as Cypher CREATE statements.
func (e *CypherExporter) ExportActors(w io.Writer, actors []*entity.Actor) error {
	if len(actors) == 0 {
		return nil
	}

	fmt.Fprintf(w, "// Actors (%d total)\n", len(actors))
	fmt.Fprintf(w, "// Generated at %s\n\n", time.Now().Format(time.RFC3339))

	batchSize := e.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(actors); i += batchSize {
		end := i + batchSize
		if end > len(actors) {
			end = len(actors)
		}

		fmt.Fprintf(w, "// Batch %d-%d\n", i+1, end)

		for _, actor := range actors[i:end] {
			var label string
			if actor.Internal {
				label = "Actor:Internal"
			} else {
				label = "Actor:External"
			}

			fmt.Fprintf(w, "MERGE (a:%s {id: %s})\n", label, cypherString(string(actor.ID)))
			fmt.Fprintf(w, "SET a.display_name = %s,\n", cypherString(actor.DisplayName))
			fmt.Fprintf(w, "    a.primary_email = %s,\n", cypherString(actor.PrimaryEmail))
			fmt.Fprintf(w, "    a.emails = %s,\n", cypherStringArray(actor.Emails))
			fmt.Fprintf(w, "    a.internal = %t,\n", actor.Internal)
			fmt.Fprintf(w, "    a.department = %s,\n", cypherString(actor.Department))
			fmt.Fprintf(w, "    a.title = %s;\n\n", cypherString(actor.Title))
		}
	}

	return nil
}

// ExportInteractions exports interactions as Cypher relationship statements.
func (e *CypherExporter) ExportInteractions(w io.Writer, interactions []*entity.Interaction) error {
	if len(interactions) == 0 {
		return nil
	}

	fmt.Fprintf(w, "// Interactions (%d total)\n", len(interactions))
	fmt.Fprintf(w, "// Generated at %s\n\n", time.Now().Format(time.RFC3339))

	batchSize := e.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(interactions); i += batchSize {
		end := i + batchSize
		if end > len(interactions) {
			end = len(interactions)
		}

		fmt.Fprintf(w, "// Batch %d-%d\n", i+1, end)

		for _, interaction := range interactions[i:end] {
			relType := "SENT_TO"
			switch interaction.EdgeType {
			case entity.EdgeTypeCC:
				relType = "CC_TO"
			case entity.EdgeTypeBCC:
				relType = "BCC_TO"
			case entity.EdgeTypeReply:
				relType = "REPLIED_TO"
			case entity.EdgeTypeMention:
				relType = "MENTIONED"
			}

			fmt.Fprintf(w, "MATCH (from:Actor {id: %s})\n", cypherString(string(interaction.From)))
			fmt.Fprintf(w, "MATCH (to:Actor {id: %s})\n", cypherString(string(interaction.To)))
			fmt.Fprintf(w, "MERGE (from)-[r:%s {message_id: %s}]->(to)\n", relType, cypherString(interaction.MessageID))
			fmt.Fprintf(w, "SET r.timestamp = datetime(%s),\n", cypherString(interaction.Timestamp.Format(time.RFC3339)))
			fmt.Fprintf(w, "    r.edge_type = %s,\n", cypherString(string(interaction.EdgeType)))
			fmt.Fprintf(w, "    r.platform = %s;\n\n", cypherString(interaction.Platform))
		}
	}

	return nil
}

// ExportCommunities exports community detection results.
func (e *CypherExporter) ExportCommunities(w io.Writer, results *analysis.CommunityResults) error {
	if results == nil || len(results.Communities) == 0 {
		return nil
	}

	fmt.Fprintf(w, "// Communities (%d total, modularity: %.4f)\n", len(results.Communities), results.Modularity)
	fmt.Fprintf(w, "// Generated at %s\n\n", time.Now().Format(time.RFC3339))

	// Create community nodes
	for _, comm := range results.Communities {
		fmt.Fprintf(w, "MERGE (c:Community {id: %d})\n", comm.ID)
		fmt.Fprintf(w, "SET c.size = %d,\n", comm.Size)
		fmt.Fprintf(w, "    c.density = %.4f,\n", comm.Density)
		fmt.Fprintf(w, "    c.conductance = %.4f;\n\n", comm.Conductance)
	}

	// Create actor-community relationships
	fmt.Fprintln(w, "// Actor community memberships")
	for actorID, commID := range results.Membership {
		fmt.Fprintf(w, "MATCH (a:Actor {id: %s})\n", cypherString(string(actorID)))
		fmt.Fprintf(w, "MATCH (c:Community {id: %d})\n", commID)
		fmt.Fprintln(w, "MERGE (a)-[:BELONGS_TO]->(c);")
		fmt.Fprintln(w)
	}

	return nil
}

// ExportThreads exports thread data.
func (e *CypherExporter) ExportThreads(w io.Writer, threads []*entity.Thread) error {
	if len(threads) == 0 {
		return nil
	}

	fmt.Fprintf(w, "// Threads (%d total)\n", len(threads))
	fmt.Fprintf(w, "// Generated at %s\n\n", time.Now().Format(time.RFC3339))

	for _, thread := range threads {
		fmt.Fprintf(w, "MERGE (t:Thread {id: %s})\n", cypherString(thread.ID))
		fmt.Fprintf(w, "SET t.subject = %s,\n", cypherString(thread.Subject))
		fmt.Fprintf(w, "    t.message_count = %d,\n", thread.Size)
		fmt.Fprintf(w, "    t.participant_count = %d;\n\n", len(thread.Participants))

		// Link messages to thread
		for _, msgID := range thread.MessageIDs {
			fmt.Fprintf(w, "MATCH (m:Message {id: %s})\n", cypherString(msgID))
			fmt.Fprintf(w, "MATCH (t:Thread {id: %s})\n", cypherString(thread.ID))
			fmt.Fprintln(w, "MERGE (m)-[:PART_OF]->(t);")
			fmt.Fprintln(w)
		}
	}

	return nil
}

// ExportFull exports a complete graph with all components.
func (e *CypherExporter) ExportFull(w io.Writer, actors []*entity.Actor, interactions []*entity.Interaction, communities *analysis.CommunityResults) error {
	fmt.Fprintln(w, "// CommGraph Full Export")
	fmt.Fprintf(w, "// Generated at %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(w)

	if err := e.ExportSchema(w); err != nil {
		return err
	}
	fmt.Fprintln(w)

	if err := e.ExportActors(w, actors); err != nil {
		return err
	}

	if err := e.ExportInteractions(w, interactions); err != nil {
		return err
	}

	if communities != nil {
		if err := e.ExportCommunities(w, communities); err != nil {
			return err
		}
	}

	return nil
}

// ExportToFile exports to a Cypher file.
func (e *CypherExporter) ExportToFile(path string, actors []*entity.Actor, interactions []*entity.Interaction, communities *analysis.CommunityResults) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return e.ExportFull(f, actors, interactions, communities)
}

// cypherString escapes a string for Cypher.
func cypherString(s string) string {
	if s == "" {
		return "''"
	}
	// Escape single quotes and backslashes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return "'" + s + "'"
}

// cypherStringArray formats a string array for Cypher.
func cypherStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	parts := make([]string, len(arr))
	for i, s := range arr {
		parts[i] = cypherString(s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ImportCommands returns helper commands for importing Cypher into Neo4j.
func ImportCommands() string {
	return `# Neo4j Import Commands

# Option 1: Using cypher-shell (recommended for large files)
cat commgraph.cypher | cypher-shell -u neo4j -p password

# Option 2: Using Neo4j Browser
# Copy and paste sections into the Neo4j Browser query editor

# Option 3: Using neo4j-admin import (for very large datasets)
# First convert to CSV format, then use:
# neo4j-admin database import full --nodes=actors.csv --relationships=interactions.csv

# After import, useful queries:

# Find most connected actors
MATCH (a:Actor)-[r]-()
RETURN a.display_name, a.id, count(r) as connections
ORDER BY connections DESC
LIMIT 20;

# Find communication between communities
MATCH (a1:Actor)-[:BELONGS_TO]->(c1:Community),
      (a2:Actor)-[:BELONGS_TO]->(c2:Community),
      (a1)-[r]->(a2)
WHERE c1.id <> c2.id
RETURN c1.id, c2.id, count(r) as cross_community_edges
ORDER BY cross_community_edges DESC;

# Find bridge actors (connect multiple communities)
MATCH (a:Actor)-[:BELONGS_TO]->(c:Community)
WITH a, collect(DISTINCT c.id) as communities
WHERE size(communities) > 1
RETURN a.display_name, communities
ORDER BY size(communities) DESC;
`
}
