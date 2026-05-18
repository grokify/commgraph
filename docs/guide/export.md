# Export Formats

CommGraph can export graph data to various formats for visualization and further analysis.

## Gephi (GEXF)

Export to GEXF format for visualization in [Gephi](https://gephi.org/).

```bash
commgraph export gephi --output=graph.gexf
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | (required) | Output file path |
| `--with-centrality` | `false` | Include centrality scores as node size |
| `--with-communities` | `false` | Include community assignments as node colors |

### Example with All Options

```bash
commgraph export gephi \
    --output=enron.gexf \
    --with-centrality \
    --with-communities
```

### Using in Gephi

1. Open Gephi and import the GEXF file
2. In the Overview panel, run a layout algorithm:
   - **ForceAtlas2** for large graphs (recommended)
   - **Fruchterman-Reingold** for smaller graphs
3. In the Appearance panel:
   - Size nodes by centrality (if included)
   - Color nodes by community (if included) or internal/external status
4. Use the Preview panel to fine-tune visualization for export

### GEXF Structure

The exported GEXF includes:

**Node attributes:**

- `id`: Actor identifier
- `label`: Display name
- `internal`: Boolean indicating internal/external status
- `email_count`: Number of emails associated with this actor
- `centrality`: PageRank score (if `--with-centrality`)
- `community`: Community ID (if `--with-communities`)

**Edge attributes:**

- `source`: Sender actor ID
- `target`: Recipient actor ID
- `weight`: Interaction strength
- `type`: Interaction type (to, cc, bcc)

## Neo4j (Cypher)

Generate Cypher statements for importing into [Neo4j](https://neo4j.com/).

```bash
commgraph export neo4j --output=graph.cypher
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | (required) | Output file path |
| `--batch-size` | `500` | Number of statements per batch |
| `--include-schema` | `true` | Include constraint and index creation |

### Importing into Neo4j

```bash
# Using cypher-shell
cat graph.cypher | cypher-shell -u neo4j -p password

# Or load in Neo4j Browser
# Copy contents of graph.cypher and paste into query window
```

### Schema

The export creates the following schema:

**Nodes:**

```cypher
(:Actor {
    id: string,
    display_name: string,
    primary_email: string,
    internal: boolean,
    title: string,
    department: string
})

(:Community {
    id: integer,
    member_count: integer,
    density: float
})

(:Message {
    id: string,
    subject: string,
    timestamp: datetime,
    thread_id: string
})
```

**Relationships:**

```cypher
(:Actor)-[:SENT {weight: float, type: string}]->(:Actor)
(:Actor)-[:BELONGS_TO]->(:Community)
(:Actor)-[:AUTHORED]->(:Message)
(:Message)-[:IN_THREAD]->(:Thread)
```

### Example Queries

After importing, run queries like:

```cypher
// Find most connected actors
MATCH (a:Actor)-[r]-()
RETURN a.display_name, count(r) as connections
ORDER BY connections DESC
LIMIT 20;

// Find communication between departments
MATCH (a1:Actor)-[r:SENT]->(a2:Actor)
WHERE a1.department <> a2.department
RETURN a1.department, a2.department, count(r) as messages
ORDER BY messages DESC;

// Find shortest path between two people
MATCH path = shortestPath(
    (a:Actor {primary_email: 'alice@example.com'})-[*]-(b:Actor {primary_email: 'bob@example.com'})
)
RETURN path;

// Community analysis
MATCH (a:Actor)-[:BELONGS_TO]->(c:Community)
RETURN c.id, count(a) as members
ORDER BY members DESC;
```

## JSON Export

Export raw data as JSON for custom processing:

```bash
# Export centrality results
commgraph analyze centrality --format=json --output=centrality.json

# Export community data
commgraph analyze community --format=json --output=communities.json
```

### JSON Structure

Centrality output:

```json
{
  "algorithm": "pagerank",
  "profile": "influence",
  "results": [
    {
      "rank": 1,
      "actor_id": "jeff.skilling",
      "display_name": "Jeff Skilling",
      "email": "jeff.skilling@enron.com",
      "score": 0.089234,
      "internal": true
    }
  ]
}
```

Community output:

```json
{
  "algorithm": "louvain",
  "resolution": 1.0,
  "modularity": 0.4521,
  "communities": [
    {
      "id": 0,
      "member_count": 234,
      "density": 0.089,
      "members": ["actor1", "actor2", "..."]
    }
  ]
}
```

## CSV Export

Export tabular data as CSV:

```bash
commgraph analyze centrality --format=csv --output=centrality.csv
```

CSV output is useful for importing into spreadsheets or data analysis tools.
