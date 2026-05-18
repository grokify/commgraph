# CLI Reference

Complete reference for all CommGraph commands.

## Global Flags

These flags are available for all commands:

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help for any command |
| `--session` | Path to session file (default: `.commgraph-session.json`) |

## Commands

### commgraph ingest

Ingest messages from an email source.

```bash
commgraph ingest [flags]
```

#### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--source` | `-s` | (required) | Source file or directory |
| `--format` | `-f` | `maildir` | Source format: `mbox` or `maildir` |
| `--internal-domains` | `-d` | | Comma-separated internal domains |
| `--session` | | `.commgraph-session.json` | Session file path |

#### Examples

```bash
# Ingest mbox file
commgraph ingest --source=archive.mbox --format=mbox

# Ingest maildir with internal domain
commgraph ingest --source=/mail/user --format=maildir \
    --internal-domains=company.com,company.org
```

---

### commgraph analyze

Run analysis algorithms on the communication graph.

#### commgraph analyze centrality

Compute centrality metrics.

```bash
commgraph analyze centrality [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--profile` | `-p` | `influence` | Weight profile |
| `--algorithm` | `-a` | `pagerank` | Algorithm: `pagerank`, `degree`, `in_degree`, `out_degree`, `betweenness` |
| `--top` | `-n` | `20` | Number of results |
| `--output` | `-o` | stdout | Output file |
| `--format` | `-f` | `table` | Output format: `table`, `json`, `csv` |

#### commgraph analyze community

Detect communities in the graph.

```bash
commgraph analyze community [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--algorithm` | `-a` | `louvain` | Algorithm: `louvain`, `label_propagation` |
| `--resolution` | | `1.0` | Resolution parameter for Louvain |
| `--output` | `-o` | stdout | Output file |
| `--format` | `-f` | `table` | Output format: `table`, `json` |

#### commgraph analyze bridges

Detect bridge actors connecting communities.

```bash
commgraph analyze bridges [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--top` | `-n` | `10` | Number of results |
| `--output` | `-o` | stdout | Output file |
| `--format` | `-f` | `table` | Output format: `table`, `json` |

#### commgraph analyze paths

Analyze network paths.

```bash
commgraph analyze paths [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--from` | | | Source actor ID |
| `--to` | | | Target actor ID |
| `--max-hops` | | `6` | Maximum path length |
| `--samples` | | `100` | Sample size for statistics |

#### commgraph analyze temporal

Analyze temporal patterns.

```bash
commgraph analyze temporal [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--window` | `-w` | `24h` | Time window for aggregation |
| `--threshold` | | `2.0` | Z-score threshold for burst detection |

#### commgraph analyze external

Analyze external communication patterns.

```bash
commgraph analyze external [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--top-domains` | | `10` | Number of top domains to show |
| `--output` | `-o` | stdout | Output file |
| `--format` | `-f` | `table` | Output format: `table`, `json` |

---

### commgraph export

Export graph data to various formats.

#### commgraph export gephi

Export to GEXF format for Gephi.

```bash
commgraph export gephi [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | (required) | Output file path |
| `--with-centrality` | | `false` | Include centrality as node size |
| `--with-communities` | | `false` | Include community colors |

#### commgraph export neo4j

Export Cypher statements for Neo4j.

```bash
commgraph export neo4j [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | (required) | Output file path |
| `--batch-size` | | `500` | Statements per batch |
| `--include-schema` | | `true` | Include schema creation |

---

### commgraph identity

Manage identity resolution.

#### commgraph identity list

List resolved actors.

```bash
commgraph identity list [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--internal` | | `false` | Show only internal actors |
| `--external` | | `false` | Show only external actors |
| `--limit` | `-n` | `0` | Limit results (0 = no limit) |
| `--format` | `-f` | `table` | Output format: `table`, `json` |

#### commgraph identity aliases

Show aliases for an actor.

```bash
commgraph identity aliases <actor-id> [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `table` | Output format: `table`, `json` |

#### commgraph identity stats

Show identity resolution statistics.

```bash
commgraph identity stats [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--format` | `-f` | `table` | Output format: `table`, `json` |

---

### commgraph pipeline

Run full ingestion and analysis pipeline in a single command.

```bash
commgraph pipeline [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--source` | `-s` | (required) | Source file or directory |
| `--format` | `-f` | `maildir` | Source format |
| `--internal-domains` | `-d` | | Internal domains |
| `--profile` | `-p` | `influence` | Weight profile |
| `--algorithm` | `-a` | `pagerank` | Centrality algorithm |
| `--top` | `-n` | `20` | Top results to show |
| `--enron` | | `false` | Load Enron employee identities |

#### Example

```bash
commgraph pipeline \
    --source=maildir/skilling-j \
    --format=maildir \
    --internal-domains=enron.com \
    --enron \
    --profile=influence \
    --top=20
```
