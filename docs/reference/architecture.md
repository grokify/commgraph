# Architecture

CommGraph is built as a modular Go library with a CLI interface. This document describes the high-level architecture.

## Package Overview

```
commgraph/
├── cmd/commgraph/     # CLI application
├── adapter/           # Data source adapters
│   └── email/         # Email format adapters (mbox, maildir)
├── entity/            # Core domain entities
├── identity/          # Identity resolution
├── analysis/          # Graph analysis algorithms
├── storage/           # Data storage abstraction
├── threading/         # Email threading reconstruction
├── session/           # Session persistence
├── export/            # Export formats
└── docs/              # Documentation (this site)
```

## Core Components

### Entity Layer

The `entity` package defines the core domain model:

- **Actor**: A person or entity that participates in communication
- **Interaction**: A directed edge representing communication between actors
- **Message**: An email message with metadata
- **Thread**: A group of related messages

```go
type Actor struct {
    ID           string
    DisplayName  string
    PrimaryEmail string
    Aliases      []string
    Internal     bool
    Title        string
    Department   string
}

type Interaction struct {
    Source    string    // Actor ID
    Target    string    // Actor ID
    Weight    float64
    Type      InteractionType // To, CC, BCC
    MessageID string
    Timestamp time.Time
}
```

### Adapter Layer

Adapters convert external data formats into the internal entity model.

**Email Adapters:**

- `MboxAdapter`: Parses mbox files (single file with multiple messages)
- `MaildirAdapter`: Parses maildir directories (one file per message)

Adapters implement the `Adapter` interface:

```go
type Adapter interface {
    Ingest(source string, opts Options) (*IngestResult, error)
}
```

### Identity Resolution

The `identity` package handles actor deduplication:

1. **Resolver**: Matches email addresses to actors
2. **Store**: Persists actor data with aliases
3. **External Data**: Loads pre-defined identity mappings

### Analysis Layer

The `analysis` package provides graph algorithms:

- **Centrality**: PageRank, degree, betweenness
- **Community**: Louvain, label propagation
- **Bridge Detection**: Identifies cross-community connectors
- **Temporal**: Activity patterns over time

### Storage Layer

The `storage` package provides an abstraction over data persistence:

```go
type Store interface {
    AddActor(actor *entity.Actor) error
    GetActor(id string) (*entity.Actor, error)
    AddInteraction(interaction *entity.Interaction) error
    AllActors() ([]*entity.Actor, error)
    AllInteractions() ([]*entity.Interaction, error)
}
```

Currently implemented: in-memory store with JSON serialization.

### Session Management

The `session` package enables state persistence across CLI invocations:

```go
type Session struct {
    Store    storage.Store
    Identity identity.Store
    Stats    *IngestStats
}
```

Sessions are serialized to JSON for persistence.

### Export Layer

The `export` package generates output formats:

- **GEXF**: XML format for Gephi visualization
- **Cypher**: Neo4j query language for graph database import

## Data Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Source    │────▶│   Adapter   │────▶│  Identity   │
│ (mbox/dir)  │     │   Layer     │     │ Resolution  │
└─────────────┘     └─────────────┘     └─────────────┘
                                              │
                                              ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Export    │◀────│  Analysis   │◀────│   Storage   │
│   Layer     │     │   Layer     │     │   Layer     │
└─────────────┘     └─────────────┘     └─────────────┘
```

1. **Ingestion**: Adapter reads source data and produces messages
2. **Resolution**: Identity resolver maps email addresses to actors
3. **Storage**: Actors and interactions are stored
4. **Analysis**: Algorithms compute metrics on the stored graph
5. **Export**: Results are formatted for output

## Threading Model

CommGraph uses References and In-Reply-To headers to reconstruct email threads:

```
┌──────────────┐
│  Message A   │  (original)
└──────┬───────┘
       │
  ┌────┴────┐
  │         │
┌─▼──┐   ┌──▼─┐
│ B  │   │ C  │  (replies)
└─┬──┘   └────┘
  │
┌─▼──┐
│ D  │  (reply to reply)
└────┘
```

The threading algorithm:

1. Parses Message-ID, References, and In-Reply-To headers
2. Builds a tree structure of related messages
3. Computes thread statistics (depth, participants, duration)

## Weight Profiles

Weight profiles adjust edge weights based on communication context:

```go
type WeightProfile struct {
    Name        string
    ToWeight    float64
    CCWeight    float64
    BCCWeight   float64
}
```

| Profile | TO | CC | BCC | Use Case |
|---------|----|----|-----|----------|
| influence | 1.0 | 0.5 | 0.25 | Organizational influence |
| information_flow | 1.0 | 1.0 | 1.0 | Information spread |
| coordination | 0.5 | 1.0 | 0.75 | Activity coordination |

## Extensibility

### Adding a New Adapter

1. Implement the `Adapter` interface
2. Add format detection in the CLI
3. Register the adapter in the factory

### Adding a New Algorithm

1. Implement the algorithm in `analysis/`
2. Add a CLI subcommand
3. Document in the User Guide

### Adding an Export Format

1. Implement the `Exporter` interface
2. Add a CLI subcommand
3. Document in the Export Guide
