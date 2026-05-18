# Configuration

CommGraph can be configured through command-line flags, configuration files, or environment variables.

## Configuration File

Create a `.commgraph.yaml` file in your working directory or home directory:

```yaml
# .commgraph.yaml

# Default source settings
source:
  format: maildir
  internal_domains:
    - company.com
    - company.org

# Analysis defaults
analysis:
  profile: influence
  top_n: 20

# Identity resolution
identity:
  auto_create: true
  load_enron: false

# Session settings
session:
  path: .commgraph-session.json
  auto_save: true

# Export defaults
export:
  format: table
  pretty_json: true
```

### Configuration Locations

CommGraph looks for configuration in this order:

1. `--config` flag (if specified)
2. `.commgraph.yaml` in current directory
3. `.commgraph.yaml` in home directory

## Configuration Reference

### source

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `format` | string | `maildir` | Default source format (`mbox`, `maildir`) |
| `internal_domains` | []string | `[]` | Domains to classify as internal |

### analysis

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `profile` | string | `influence` | Default weight profile |
| `top_n` | int | `20` | Default number of top results |
| `centrality_algorithm` | string | `pagerank` | Default centrality algorithm |
| `community_algorithm` | string | `louvain` | Default community algorithm |
| `community_resolution` | float | `1.0` | Louvain resolution parameter |

### identity

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `auto_create` | bool | `true` | Auto-create actors for unknown emails |
| `load_enron` | bool | `false` | Load Enron employee data |

### session

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `path` | string | `.commgraph-session.json` | Session file path |
| `auto_save` | bool | `true` | Auto-save session after ingest |

### export

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `format` | string | `table` | Default output format |
| `pretty_json` | bool | `true` | Pretty-print JSON output |
| `neo4j_batch_size` | int | `500` | Batch size for Cypher export |

## Environment Variables

All configuration options can be set via environment variables using the `COMMGRAPH_` prefix:

```bash
export COMMGRAPH_SOURCE_FORMAT=mbox
export COMMGRAPH_ANALYSIS_PROFILE=coordination
export COMMGRAPH_IDENTITY_LOAD_ENRON=true
```

Nested keys use underscores:

| Config Key | Environment Variable |
|------------|---------------------|
| `source.format` | `COMMGRAPH_SOURCE_FORMAT` |
| `source.internal_domains` | `COMMGRAPH_SOURCE_INTERNAL_DOMAINS` |
| `analysis.profile` | `COMMGRAPH_ANALYSIS_PROFILE` |
| `identity.auto_create` | `COMMGRAPH_IDENTITY_AUTO_CREATE` |

## Precedence

Configuration values are applied in this order (later overrides earlier):

1. Default values
2. Configuration file
3. Environment variables
4. Command-line flags

## Example Configurations

### Corporate Email Analysis

```yaml
# .commgraph.yaml
source:
  format: maildir
  internal_domains:
    - acme.com
    - acme.corp
    - acmeinc.com

analysis:
  profile: influence
  top_n: 50

identity:
  auto_create: true
```

### Research / Enron Analysis

```yaml
# .commgraph.yaml
source:
  format: maildir
  internal_domains:
    - enron.com

identity:
  load_enron: true

analysis:
  profile: influence
  top_n: 100

export:
  format: json
  pretty_json: true
```

### E-Discovery

```yaml
# .commgraph.yaml
source:
  format: mbox
  internal_domains:
    - company.com

analysis:
  profile: information_flow

session:
  auto_save: true
  path: ./discovery-session.json

export:
  format: json
```
