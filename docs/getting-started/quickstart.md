# Quick Start

This guide walks you through analyzing your first email corpus with CommGraph.

## Step 1: Prepare Your Data

CommGraph supports two email formats:

- **mbox**: Single file containing multiple messages (common export format)
- **maildir**: Directory with one file per message (Thunderbird, Dovecot)

## Step 2: Run the Pipeline

The easiest way to analyze email is the `pipeline` command, which runs ingestion and analysis in a single step:

```bash
commgraph pipeline \
    --source=/path/to/emails \
    --format=maildir \
    --internal-domains=yourcompany.com \
    --profile=influence \
    --top=20
```

### Parameters

| Flag | Description |
|------|-------------|
| `--source` | Path to mbox file or maildir directory |
| `--format` | `mbox` or `maildir` |
| `--internal-domains` | Comma-separated list of internal domains |
| `--profile` | Weight profile: `influence`, `information_flow`, or `coordination` |
| `--top` | Number of top results to show |

### Example Output

```
Ingesting from /path/to/emails...

Ingestion complete:
  Messages:     4,139
  Interactions: 41,438
  Actors:       5,352 (148 internal, 5,204 external)
  Threads:      2,061 (85 single, 1,976 multi-message)
  Duration:     2.3s (ingest) + 0.5s (threading)

Running pagerank analysis with influence profile...

pagerank Results (top 20):
Rank  Actor                                              Score
----  -----                                              -----
1     jeff.skilling@enron.com                            0.089234
2     kenneth.lay@enron.com                              0.067891
3     andrew.fastow@enron.com                            0.045123
...
```

## Step 3: Export Results

Export the graph for visualization in Gephi:

```bash
commgraph export gephi --output=graph.gexf
```

Or generate Cypher statements for Neo4j:

```bash
commgraph export neo4j --output=graph.cypher
```

## Using Separate Commands

For more control, use separate commands with session persistence:

```bash
# Step 1: Ingest (saves session to .commgraph-session.json)
commgraph ingest \
    --source=/path/to/emails.mbox \
    --format=mbox \
    --internal-domains=example.com

# Step 2: Analyze (loads session automatically)
commgraph analyze centrality --profile=influence --top=20

# Step 3: Run community detection
commgraph analyze community

# Step 4: Export
commgraph export gephi --output=graph.gexf
```

## Next Steps

- [Enron Tutorial](enron-tutorial.md) - Detailed walkthrough with the Enron corpus
- [CLI Reference](../guide/cli.md) - Complete command documentation
- [Analysis Types](../guide/analysis.md) - Learn about available analysis algorithms
