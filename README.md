# CommGraph

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/commgraph/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/commgraph/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/commgraph/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/commgraph/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/commgraph/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/commgraph/actions/workflows/go-sast-codeql.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/commgraph
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/commgraph
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/commgraph
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/commgraph
 [viz-svg]: https://img.shields.io/badge/visualization-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fcommgraph
 [loc-svg]: https://tokei.rs/b1/github/grokify/commgraph
 [repo-url]: https://github.com/grokify/commgraph
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/commgraph/blob/main/LICENSE

Communication graph analysis framework for organizational communication patterns.

## Overview

CommGraph analyzes communication patterns across multiple platforms (email, Slack, Teams) to understand:

- **Influence**: Who has power and authority in communication
- **Information Flow**: How information propagates through the organization
- **Coordination**: Collaborative activity and project coordination patterns
- **Organizational Structure**: Informal vs formal hierarchy

## Features

- Multi-platform support via adapter architecture
- Configurable weight profiles for different analysis types
- Thread reconstruction for fragmented email conversations
- Identity resolution across aliases and platforms
- Graph database abstraction (via [omniretrieve](https://github.com/plexusone/omniretrieve))
- E-discovery ready (immutable storage, audit trails, EDRM export)

## Installation

```bash
go install github.com/grokify/commgraph/cmd/commgraph@latest
```

## Quick Start

```bash
# Run full pipeline (ingest + analyze)
commgraph pipeline \
    --source=/path/to/emails \
    --format=maildir \
    --internal-domains=company.com \
    --profile=influence \
    --top=20

# Or use separate commands:
commgraph ingest --source=/path/to/mailbox.mbox --format=mbox
commgraph analyze centrality --profile=influence
commgraph export gephi --output=graph.gexf
```

## Documentation

Full documentation is available at [grokify.github.io/commgraph](https://grokify.github.io/commgraph).

- [Installation](docs/getting-started/installation.md)
- [Quick Start](docs/getting-started/quickstart.md)
- [Enron Tutorial](docs/getting-started/enron-tutorial.md)
- [CLI Reference](docs/guide/cli.md)
- [Configuration](docs/guide/configuration.md)

### Specifications

- [Product Requirements](docs/specs/PRD.md)
- [Technical Requirements](docs/specs/TRD.md)
- [Roadmap](docs/specs/ROADMAP.md)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI / API                            │
├─────────────────────────────────────────────────────────────┤
│                    Analysis Layer                           │
│  (Centrality, Community Detection, Temporal, Anomaly)       │
├─────────────────────────────────────────────────────────────┤
│                     Graph Layer                             │
│  (omniretrieve abstraction, Weight Profiles)                │
├─────────────────────────────────────────────────────────────┤
│                   Processing Layer                          │
│  (Identity Resolution, Thread Reconstruction)               │
├─────────────────────────────────────────────────────────────┤
│                    Ingestion Layer                          │
│  (Adapters: Email, Slack, Teams)                            │
├─────────────────────────────────────────────────────────────┤
│                    Storage Layer                            │
│  (Memory, BadgerDB, Neo4j)                                  │
└─────────────────────────────────────────────────────────────┘
```

## Weight Profiles

CommGraph supports configurable weight profiles for different analysis objectives:

| Profile | TO | CC | BCC | Mention | Reply | Use Case |
|---------|----|----|-----|---------|-------|----------|
| influence | 1.0 | 0.2 | 0.4 | 0.1 | 0.3 | Power/authority analysis |
| information_flow | 1.0 | 0.8 | 0.9 | 0.5 | 0.1 | Information propagation |
| coordination | 0.5 | 0.8 | 0.3 | 0.2 | 1.0 | Collaborative activity |

## Dependencies

- [omniretrieve](https://github.com/plexusone/omniretrieve) - Graph abstraction layer
- [mogo](https://github.com/grokify/mogo) - Go utilities, thread reconstruction
- [enron-people](https://github.com/enrondata/enron-people) - Enron corpus identity data
- [enmime](https://github.com/jhillyerd/enmime) - MIME email parsing
- [cobra](https://github.com/spf13/cobra) - CLI framework

## License

MIT
