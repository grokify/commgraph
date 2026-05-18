# CommGraph

Communication graph analysis framework for organizational communication patterns.

## Overview

CommGraph analyzes communication patterns across email (with future support for Slack and Teams) to understand organizational dynamics:

- **Influence**: Who has power and authority in communication networks
- **Information Flow**: How information propagates through the organization
- **Coordination**: Collaborative activity and project coordination patterns
- **Community Structure**: Informal groups and cross-team connectors

## Key Features

- **Multi-format Email Support**: Parse mbox and Maildir formats
- **Identity Resolution**: Merge multiple email aliases into unified actors
- **Thread Reconstruction**: Rebuild conversation threads from fragmented messages
- **Graph Analysis**: PageRank, community detection, path analysis, bridge detection
- **Multiple Export Formats**: JSON, CSV, GEXF (Gephi), Cypher (Neo4j)
- **Session Persistence**: Save and resume analysis sessions
- **Configurable Weight Profiles**: Customize edge weights for different analysis types

## Quick Example

```bash
# Install
go install github.com/grokify/commgraph/cmd/commgraph@latest

# Analyze an email corpus
commgraph pipeline --source=/path/to/maildir --format=maildir \
    --internal-domains=example.com --profile=influence --top=20

# Or use separate commands with session persistence
commgraph ingest --source=/path/to/emails.mbox --format=mbox
commgraph analyze centrality --profile=influence
commgraph export gephi --output=graph.gexf
```

## Use Cases

### Organizational Network Analysis

Understand informal communication structures, identify key connectors, and discover communication silos.

### E-Discovery

Process email archives for legal review with identity resolution and thread reconstruction.

### Research

Analyze the Enron email corpus or other communication datasets for academic research.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI / API                            │
├─────────────────────────────────────────────────────────────┤
│                    Analysis Layer                           │
│  (Centrality, Community Detection, Temporal, Bridges)       │
├─────────────────────────────────────────────────────────────┤
│                     Graph Layer                             │
│  (Weighted edges, Weight Profiles)                          │
├─────────────────────────────────────────────────────────────┤
│                   Processing Layer                          │
│  (Identity Resolution, Thread Reconstruction)               │
├─────────────────────────────────────────────────────────────┤
│                    Ingestion Layer                          │
│  (Adapters: Mbox, Maildir)                                  │
├─────────────────────────────────────────────────────────────┤
│                    Storage Layer                            │
│  (Memory, Session Files)                                    │
└─────────────────────────────────────────────────────────────┘
```

## Status

CommGraph is currently in **Beta** (v0.3.0). Phase 2 (Analysis Depth) is complete with full graph analytics. See the [Roadmap](reference/roadmap.md) for future plans.

## License

MIT License - see [LICENSE](https://github.com/grokify/commgraph/blob/main/LICENSE) for details.
