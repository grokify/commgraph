# Enron Email Corpus Tutorial

This tutorial demonstrates CommGraph's capabilities using the famous Enron email corpus.

## About the Enron Corpus

The Enron email corpus contains approximately 500,000 emails from 150 Enron employees. It was released during the Federal Energy Regulatory Commission's investigation and is widely used for research.

## Getting the Data

Download the Enron corpus in maildir format:

```bash
# Download from CMU
wget https://www.cs.cmu.edu/~enron/enron_mail_20150507.tar.gz
tar -xzf enron_mail_20150507.tar.gz
```

This creates a `maildir/` directory with subdirectories for each custodian (employee).

## Analyzing a Single Mailbox

Start with a single employee's mailbox for faster iteration:

```bash
commgraph pipeline \
    --source=maildir/skilling-j \
    --format=maildir \
    --internal-domains=enron.com \
    --enron \
    --profile=influence \
    --top=20
```

The `--enron` flag loads pre-curated identity data from the [enron-people](https://github.com/enrondata/enron-people) package, which provides known aliases for key Enron employees.

### Expected Results

```
Loading Enron employee identities...
  Loaded 14 employees with aliases

Ingesting from maildir/skilling-j...

Ingestion complete:
  Messages:     4,139
  Interactions: 41,438
  Actors:       5,352 (148 internal, 5,204 external)
  Threads:      2,061

Running pagerank analysis with influence profile...

pagerank Results (top 20):
Rank  Actor                                              Score
----  -----                                              -----
1     jeff.skilling@enron.com                            0.089234
2     kenneth.lay@enron.com                              0.067891
3     sherri.sera@enron.com                              0.045123
4     rebecca.carter@enron.com                           0.038901
5     andrew.fastow@enron.com                            0.032456
...

Detecting communities...
  Found 47 communities (modularity: 0.4521)
  Top 5 communities:
    Community 0: 234 members (density: 0.089)
    Community 1: 156 members (density: 0.112)
    ...
```

## Identity Resolution

The Enron corpus has many duplicate identities due to email aliases. Use the identity commands to inspect:

```bash
# List all resolved actors
commgraph identity list --internal --limit=20

# Show aliases for Jeff Skilling
commgraph identity aliases jeff.skilling

# View resolution statistics
commgraph identity stats
```

Example alias output:

```
Actor: jeff.skilling
Display Name: Jeff Skilling
Primary Email: jeff.skilling@enron.com
Internal: true
Title: CEO

Aliases (5):
  jeff.skilling@enron.com (primary)
  jskilli@enron.com
  skilling@enron.com
  jeff_skilling@enron.com
  jeffrey.skilling@enron.com
```

## Community Detection

Identify informal groups within the organization:

```bash
commgraph analyze community --format=json --output=communities.json
```

## Bridge Detection

Find employees who connect different communities:

```bash
commgraph analyze bridges --top=10
```

Bridge actors often have significant organizational influence as information gatekeepers.

## External Communication Analysis

Analyze communication with external parties:

```bash
commgraph analyze external
```

This identifies:

- Most contacted external domains
- Boundary spanners (employees with high external communication)
- Inbound vs outbound communication patterns

## Temporal Analysis

Detect unusual activity patterns:

```bash
commgraph analyze temporal --window=24h
```

This can identify:

- Communication bursts (sudden activity spikes)
- Timeline trends
- Activity patterns over time

## Exporting for Visualization

### Gephi

Export to GEXF format for Gephi visualization:

```bash
commgraph export gephi --output=enron.gexf
```

In Gephi:

1. Open `enron.gexf`
2. Run ForceAtlas2 layout
3. Color nodes by community or internal/external
4. Size nodes by PageRank score

### Neo4j

Generate Cypher statements for Neo4j:

```bash
commgraph export neo4j --output=enron.cypher
```

Import into Neo4j:

```bash
cat enron.cypher | cypher-shell -u neo4j -p password
```

Then run queries like:

```cypher
// Find most connected actors
MATCH (a:Actor)-[r]-()
RETURN a.display_name, count(r) as connections
ORDER BY connections DESC
LIMIT 20;

// Find communication between communities
MATCH (a1:Actor)-[:BELONGS_TO]->(c1:Community),
      (a2:Actor)-[:BELONGS_TO]->(c2:Community),
      (a1)-[r]->(a2)
WHERE c1.id <> c2.id
RETURN c1.id, c2.id, count(r) as cross_edges
ORDER BY cross_edges DESC;
```

## Processing the Full Corpus

To analyze the entire Enron corpus:

```bash
commgraph pipeline \
    --source=maildir \
    --format=maildir \
    --internal-domains=enron.com \
    --enron \
    --profile=influence \
    --top=50
```

!!! warning "Memory Usage"
    The full corpus requires significant memory (8GB+ recommended).
    Consider using the session file to save progress.

## Next Steps

- [Analysis Types](../guide/analysis.md) - Learn about all available algorithms
- [Weight Profiles](../reference/weight-profiles.md) - Customize analysis for your use case
- [Export Formats](../guide/export.md) - Detailed export documentation
