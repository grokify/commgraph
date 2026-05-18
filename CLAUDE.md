# CLAUDE.md

Project-specific instructions for Claude Code.

## Project Overview

CommGraph is a communication graph analysis framework for organizational communication patterns. It analyzes email corpora to understand influence, information flow, and coordination patterns.

## Current State

**Version**: v0.1.0 (ready to tag)

**Completed Phases**:

- Phase 0: Foundation (core types, interfaces, weight profiles)
- Phase 1: Enron MVP (email adapters, identity resolution, threading, centrality analysis)
- Phase 2: Analysis Depth (community detection, temporal analysis, path analysis, bridge detection, external analysis, Gephi/Neo4j export)
- Phase 3 partial: Configuration file support with viper

**Test Coverage**: 191 tests across all packages

**Lint Status**: All `golangci-lint` checks pass

## Key Commands

```bash
# Run tests
go test -v ./...

# Run linting
golangci-lint run

# Build CLI
go build -o commgraph ./cmd/commgraph

# Run pipeline analysis (example with Enron data)
./commgraph pipeline \
    --source=maildir/skilling-j \
    --format=maildir \
    --internal-domains=enron.com \
    --enron \
    --profile=influence \
    --top=20

# Generate documentation site
mkdocs serve  # local preview
mkdocs build  # build to site/
```

## Architecture

```
cmd/commgraph/     CLI application (Cobra-based)
adapter/email/     Mbox and Maildir email parsing
entity/            Core types: Actor, Interaction, Message, Thread
identity/          SCIM-based identity resolution with alias merging
analysis/          Graph algorithms (centrality, community, temporal, paths, bridges)
storage/           In-memory store with JSON serialization
threading/         Thread reconstruction using mogo/mailutil
export/            GEXF (Gephi), Cypher (Neo4j), JSON, CSV export
session/           CLI state persistence across invocations
config/            YAML configuration with viper
weight/            Edge weight profiles (influence, information_flow, coordination)
```

## Key Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/enrondata/enron-people` | Enron employee identity data |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration management |
| `github.com/plexusone/omniretrieve` | Graph abstraction |
| `github.com/jhillyerd/enmime` | MIME email parsing |
| `github.com/grokify/mogo` | Thread reconstruction |

## Remaining Work (from ROADMAP.md)

**Phase 3 (Production Hardening)**:

- P3-1: BadgerDB persistent storage
- P3-2: Audit logging with hash chaining
- P3-3: Incremental ingestion with checkpoints
- P3-4: Graph caching
- P3-6: EDRM export for e-discovery
- P3-7: Performance optimization
- P3-8: CLI polish (progress bars, shell completion)

**Phase 4**: Multi-platform (Slack, Teams adapters)

**Phase 5**: Advanced (anomaly detection, embeddings, REST API, LLM integration)

## Known Issues

1. **Threading**: Many single-message threads (may need more aggressive subject-based fallback)
2. **Identity resolution**: Works well with enron-people data; custom corpora need manual identity mapping

## Documentation

- MkDocs site in `docs/` (Material theme)
- Release notes in `docs/releases/`
- PRD/TRD in `docs/specs/`
- CHANGELOG.json uses structured-changelog format

## Changelog Workflow

```bash
# Validate changelog
schangelog validate CHANGELOG.json

# Generate markdown
schangelog generate CHANGELOG.json -o CHANGELOG.md
```
