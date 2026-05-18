# Analysis Types

CommGraph provides several analysis algorithms for understanding communication patterns.

## Centrality Analysis

Centrality measures identify the most important actors in a communication network.

### PageRank

Google's PageRank algorithm adapted for email networks. Actors who receive messages from other important actors score higher.

```bash
commgraph analyze centrality --algorithm=pagerank --top=20
```

**Best for:** Identifying influential individuals who receive attention from other influential people.

### Degree Centrality

Simple count of connections. Variants:

- **degree**: Total connections (in + out)
- **in_degree**: Incoming messages only
- **out_degree**: Outgoing messages only

```bash
commgraph analyze centrality --algorithm=degree --top=20
commgraph analyze centrality --algorithm=in_degree --top=20
commgraph analyze centrality --algorithm=out_degree --top=20
```

**Best for:** Finding the most active communicators (degree) or most sought-after individuals (in_degree).

### Betweenness Centrality

Measures how often an actor lies on the shortest path between other actors.

```bash
commgraph analyze centrality --algorithm=betweenness --top=20
```

**Best for:** Identifying information brokers and gatekeepers who control information flow.

## Community Detection

Community detection identifies groups of actors who communicate more frequently with each other than with outsiders.

### Louvain Algorithm

Fast, hierarchical community detection that optimizes modularity.

```bash
commgraph analyze community --algorithm=louvain
```

The `--resolution` parameter controls granularity:

- Higher values (>1.0) produce more, smaller communities
- Lower values (<1.0) produce fewer, larger communities

```bash
commgraph analyze community --algorithm=louvain --resolution=1.5
```

### Label Propagation

Fast algorithm where nodes adopt the most common label among their neighbors.

```bash
commgraph analyze community --algorithm=label_propagation
```

**Best for:** Quick community detection on large graphs.

## Bridge Detection

Bridge actors connect different communities and often serve as information gatekeepers.

```bash
commgraph analyze bridges --top=10
```

Output includes:

- Actor identification
- Communities they connect
- Cross-community edge count
- Betweenness score

## Path Analysis

Analyze network paths between actors.

```bash
# Shortest path between two actors
commgraph analyze paths --from=alice@example.com --to=bob@example.com

# Network diameter and average path length
commgraph analyze paths --samples=100
```

## Temporal Analysis

Detect patterns over time.

```bash
commgraph analyze temporal --window=24h
```

Identifies:

- **Communication bursts**: Sudden spikes in activity
- **Trends**: Increasing or decreasing communication over time
- **Patterns**: Regular activity cycles

The `--threshold` parameter sets the z-score threshold for burst detection:

```bash
commgraph analyze temporal --window=24h --threshold=2.0
```

## External Communication Analysis

Analyze communication patterns with external parties.

```bash
commgraph analyze external --top-domains=10
```

Output includes:

- Top external domains by message count
- Boundary spanners (internal actors with high external communication)
- Inbound vs outbound ratios

## Weight Profiles

All centrality analyses use weight profiles to adjust edge weights based on communication context:

| Profile | Description | Use Case |
|---------|-------------|----------|
| `influence` | Higher weight for direct TO recipients | Who has organizational influence? |
| `information_flow` | Equal weight for all recipients | How does information spread? |
| `coordination` | Higher weight for CC/BCC recipients | Who coordinates activities? |

```bash
commgraph analyze centrality --profile=influence
commgraph analyze centrality --profile=information_flow
commgraph analyze centrality --profile=coordination
```

See [Weight Profiles](../reference/weight-profiles.md) for detailed configuration.

## Output Formats

All analysis commands support multiple output formats:

```bash
# Table format (default)
commgraph analyze centrality --format=table

# JSON format
commgraph analyze centrality --format=json --output=results.json

# CSV format (centrality only)
commgraph analyze centrality --format=csv --output=results.csv
```
