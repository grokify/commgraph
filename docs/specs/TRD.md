# CommGraph Technical Requirements Document

## Architecture Overview

CommGraph follows a layered architecture separating concerns for auditability, extensibility, and reproducibility.

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI / API                            │
├─────────────────────────────────────────────────────────────┤
│                    Analysis Layer                           │
│  (Centrality, Community Detection, Temporal, Anomaly)       │
├─────────────────────────────────────────────────────────────┤
│                     Graph Layer                             │
│  (Weighted Edges, Materialized Views, Traversal)            │
├─────────────────────────────────────────────────────────────┤
│                   Processing Layer                          │
│  (Identity Resolution, Thread Reconstruction)               │
├─────────────────────────────────────────────────────────────┤
│                    Ingestion Layer                          │
│  (Adapters: Email, Slack, Teams)                            │
├─────────────────────────────────────────────────────────────┤
│                    Storage Layer                            │
│  (Immutable Raw Store, Index Store, Graph Store)            │
└─────────────────────────────────────────────────────────────┘
```

## Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go | Performance, concurrency, single binary deployment |
| Graph Analytics | Gonum | Native Go, no external dependencies for core analysis |
| Graph Database | Neo4j (optional) | Persistent storage, Cypher queries, GDS algorithms |
| Email Parsing | enmime, net/mail | Robust MIME handling |
| Identity Format | SCIM | Standard format, existing tooling |
| Configuration | YAML/JSON | Human-readable, versionable |

## Package Structure

```
github.com/grokify/commgraph/
├── cmd/
│   └── commgraph/          # CLI application
│       └── main.go
├── adapter/                # Platform adapters
│   ├── adapter.go          # Common adapter interface
│   ├── email/              # Email adapter
│   │   ├── mbox.go
│   │   ├── eml.go
│   │   └── pst.go
│   ├── slack/              # Slack adapter (future)
│   └── teams/              # Teams adapter (future)
├── entity/                 # Core entity definitions
│   ├── actor.go
│   ├── message.go
│   ├── interaction.go
│   ├── thread.go
│   └── channel.go
├── identity/               # Identity resolution
│   ├── resolver.go
│   ├── scim.go
│   └── alias.go
├── threading/              # Thread reconstruction
│   └── reconstructor.go    # Wraps mogo/net/mailutil/threading
├── graph/                  # Graph construction and operations
│   ├── graph.go
│   ├── edge.go
│   ├── node.go
│   └── weighted.go
├── weight/                 # Weight profiles
│   ├── profile.go
│   └── profiles.go         # Built-in profiles
├── analysis/               # Graph analysis algorithms
│   ├── centrality.go
│   ├── community.go
│   ├── temporal.go
│   └── anomaly.go
├── storage/                # Storage backends
│   ├── storage.go          # Interface
│   ├── memory.go           # In-memory store
│   ├── badger.go           # BadgerDB store
│   └── neo4j/              # Neo4j integration
├── export/                 # Export formats
│   ├── gephi.go
│   ├── cypher.go
│   ├── edrm.go
│   └── csv.go
├── audit/                  # Audit trail
│   ├── log.go
│   └── hash.go
└── docs/
    └── specs/
```

## Core Interfaces

### Adapter Interface

```go
package adapter

import (
    "context"
    "github.com/grokify/commgraph/entity"
)

// Adapter ingests messages from a specific platform.
type Adapter interface {
    // Name returns the adapter identifier (e.g., "email", "slack").
    Name() string

    // Ingest reads from source and emits messages to the channel.
    // The channel is closed when ingestion completes.
    Ingest(ctx context.Context, source Source) (<-chan entity.Message, error)

    // IngestIncremental ingests only messages newer than checkpoint.
    IngestIncremental(ctx context.Context, source Source, checkpoint Checkpoint) (<-chan entity.Message, error)
}

// Source represents an ingestion source (file path, API config, etc.).
type Source interface {
    Type() string
    Location() string
}

// Checkpoint represents incremental ingestion state.
type Checkpoint struct {
    AdapterName   string
    LastMessageID string
    LastTimestamp time.Time
    Metadata      map[string]string
}
```

### Identity Resolver Interface

```go
package identity

import "github.com/grokify/commgraph/entity"

// Resolver resolves addresses to canonical actor identities.
type Resolver interface {
    // Resolve returns the canonical ActorID for an address.
    // Returns ErrUnknownActor if address cannot be resolved.
    Resolve(addr string) (entity.ActorID, error)

    // ResolveOrCreate resolves or creates a new actor for unknown addresses.
    ResolveOrCreate(addr string) entity.ActorID

    // IsInternal returns true if the address belongs to the organization.
    IsInternal(addr string) bool

    // GetActor returns full actor details by ID.
    GetActor(id entity.ActorID) (*entity.Actor, error)

    // Aliases returns all known addresses for an actor.
    Aliases(id entity.ActorID) []string

    // Stats returns resolution statistics.
    Stats() ResolverStats
}

type ResolverStats struct {
    TotalActors      int
    InternalActors   int
    ExternalActors   int
    TotalAliases     int
    ResolutionHits   int64
    ResolutionMisses int64
}
```

### Storage Interface

```go
package storage

import (
    "context"
    "github.com/grokify/commgraph/entity"
)

// Store provides persistence for messages and graph data.
type Store interface {
    // Message operations (immutable)
    StoreMessage(ctx context.Context, msg *entity.Message) error
    GetMessage(ctx context.Context, id string) (*entity.Message, error)
    ListMessages(ctx context.Context, opts ListOptions) ([]*entity.Message, error)

    // Interaction operations
    StoreInteraction(ctx context.Context, interaction *entity.Interaction) error
    GetInteractions(ctx context.Context, opts InteractionQuery) ([]*entity.Interaction, error)

    // Actor operations
    StoreActor(ctx context.Context, actor *entity.Actor) error
    GetActor(ctx context.Context, id entity.ActorID) (*entity.Actor, error)

    // Thread operations
    StoreThread(ctx context.Context, thread *entity.Thread) error
    GetThread(ctx context.Context, id string) (*entity.Thread, error)

    // Graph operations
    GetGraph(ctx context.Context, opts GraphOptions) (*graph.Graph, error)

    // Audit
    AuditLog() AuditLog

    // Lifecycle
    Close() error
}

type ListOptions struct {
    After     time.Time
    Before    time.Time
    Limit     int
    Offset    int
    ActorID   entity.ActorID
    ThreadID  string
}

type GraphOptions struct {
    Profile     string              // Weight profile name
    After       time.Time           // Temporal filter
    Before      time.Time
    ActorFilter func(entity.ActorID) bool
    EdgeTypes   []entity.EdgeType   // Filter by edge type
}
```

## Entity Definitions

### Actor

```go
package entity

type ActorID string

type Actor struct {
    ID           ActorID           `json:"id"`
    DisplayName  string            `json:"display_name"`
    Emails       []string          `json:"emails"`
    PrimaryEmail string            `json:"primary_email"`
    ExternalID   string            `json:"external_id,omitempty"`
    Internal     bool              `json:"internal"`
    Department   string            `json:"department,omitempty"`
    Title        string            `json:"title,omitempty"`
    Timezone     string            `json:"timezone,omitempty"`
    Metadata     map[string]string `json:"metadata,omitempty"`
}
```

### Message

```go
package entity

type Message struct {
    ID          string            `json:"id"`
    Platform    string            `json:"platform"`
    RawHash     string            `json:"raw_hash"`

    // Headers
    MessageID   string            `json:"message_id"`
    InReplyTo   string            `json:"in_reply_to,omitempty"`
    References  []string          `json:"references,omitempty"`

    // Participants
    From        string            `json:"from"`
    To          []string          `json:"to"`
    CC          []string          `json:"cc,omitempty"`
    BCC         []string          `json:"bcc,omitempty"`

    // Content
    Subject     string            `json:"subject"`
    Date        time.Time         `json:"date"`
    BodyPreview string            `json:"body_preview,omitempty"`

    // Extracted
    Mentions    []string          `json:"mentions,omitempty"`
    Domains     []string          `json:"domains,omitempty"`

    // Threading (populated after reconstruction)
    ThreadID    string            `json:"thread_id,omitempty"`
    ParentID    string            `json:"parent_id,omitempty"`
    ThreadDepth int               `json:"thread_depth,omitempty"`

    // Audit
    IngestedAt  time.Time         `json:"ingested_at"`
    SourcePath  string            `json:"source_path,omitempty"`
}
```

### Interaction

```go
package entity

type EdgeType string

const (
    EdgeTypeTo      EdgeType = "TO"
    EdgeTypeCC      EdgeType = "CC"
    EdgeTypeBCC     EdgeType = "BCC"
    EdgeTypeMention EdgeType = "MENTION"
    EdgeTypeReply   EdgeType = "REPLY"
)

type Interaction struct {
    ID          string    `json:"id"`
    MessageID   string    `json:"message_id"`
    ThreadID    string    `json:"thread_id,omitempty"`

    From        ActorID   `json:"from"`
    To          ActorID   `json:"to"`
    EdgeType    EdgeType  `json:"edge_type"`

    Timestamp   time.Time `json:"timestamp"`
    Platform    string    `json:"platform"`

    // Weights computed per profile
    Weights     map[string]float64 `json:"weights,omitempty"`
}
```

### Thread

```go
package entity

type Thread struct {
    ID            string    `json:"id"`
    Subject       string    `json:"subject"`
    RootMessageID string    `json:"root_message_id"`
    MessageIDs    []string  `json:"message_ids"`
    Participants  []ActorID `json:"participants"`
    StartDate     time.Time `json:"start_date"`
    EndDate       time.Time `json:"end_date"`
    Size          int       `json:"size"`
    Depth         int       `json:"depth"`
}
```

## Weight Profile System

```go
package weight

type Profile struct {
    Name        string  `json:"name"`
    Description string  `json:"description"`

    // Base weights by edge type
    To          float64 `json:"to"`
    CC          float64 `json:"cc"`
    BCC         float64 `json:"bcc"`
    Mention     float64 `json:"mention"`
    Reply       float64 `json:"reply"`

    // Modifiers
    RecencyDecay    DecayFunc `json:"-"`
    RecencyHalfLife Duration  `json:"recency_half_life,omitempty"`

    // Aggregation
    Aggregation AggregationType `json:"aggregation"`
}

type AggregationType string

const (
    AggregationSum AggregationType = "sum"
    AggregationMax AggregationType = "max"
    AggregationAvg AggregationType = "avg"
)

type DecayFunc func(age time.Duration) float64

// Built-in profiles
var (
    Influence = Profile{
        Name:        "influence",
        Description: "Measures power and authority in communication",
        To: 1.0, CC: 0.2, BCC: 0.4, Mention: 0.1, Reply: 0.3,
        Aggregation: AggregationSum,
    }

    InformationFlow = Profile{
        Name:        "information_flow",
        Description: "Measures how information propagates",
        To: 1.0, CC: 0.8, BCC: 0.9, Mention: 0.5, Reply: 0.1,
        Aggregation: AggregationSum,
    }

    Coordination = Profile{
        Name:        "coordination",
        Description: "Measures collaborative activity",
        To: 0.5, CC: 0.8, BCC: 0.3, Mention: 0.2, Reply: 1.0,
        Aggregation: AggregationSum,
    }
)

// Weight computes the weight for an edge type.
func (p Profile) Weight(edgeType entity.EdgeType) float64 {
    switch edgeType {
    case entity.EdgeTypeTo:
        return p.To
    case entity.EdgeTypeCC:
        return p.CC
    case entity.EdgeTypeBCC:
        return p.BCC
    case entity.EdgeTypeMention:
        return p.Mention
    case entity.EdgeTypeReply:
        return p.Reply
    default:
        return 0
    }
}
```

## Thread Reconstruction Integration

Wraps existing `mogo/net/mailutil/threading`:

```go
package threading

import (
    "github.com/grokify/commgraph/entity"
    mogothread "github.com/grokify/mogo/net/mailutil/threading"
)

// Reconstructor wraps mogo threading for commgraph messages.
type Reconstructor struct {
    config Config
    inner  *mogothread.Reconstructor
}

type Config struct {
    MaxParentAge              time.Duration
    RequireParticipantOverlap bool
}

func DefaultConfig() Config {
    return Config{
        MaxParentAge:              7 * 24 * time.Hour,
        RequireParticipantOverlap: true,
    }
}

// messageAdapter adapts entity.Message to mogothread.ThreadableMessage
type messageAdapter struct {
    msg  *entity.Message
    info mogothread.ThreadingInfo
}

func (m *messageAdapter) GetMessageID() string      { return m.msg.MessageID }
func (m *messageAdapter) GetDate() time.Time        { return m.msg.Date }
func (m *messageAdapter) GetSubject() string        { return m.msg.Subject }
func (m *messageAdapter) GetInReplyTo() string      { return m.msg.InReplyTo }
func (m *messageAdapter) GetReferences() []string   { return m.msg.References }
func (m *messageAdapter) GetParticipants() []string {
    participants := []string{m.msg.From}
    participants = append(participants, m.msg.To...)
    participants = append(participants, m.msg.CC...)
    return participants
}
func (m *messageAdapter) GetEmbeddedMessageHints() []mogothread.EmbeddedHint {
    return nil // TODO: implement body parsing for hints
}
func (m *messageAdapter) SetThreadingInfo(info mogothread.ThreadingInfo) {
    m.info = info
}

// Reconstruct processes messages and populates threading info.
func (r *Reconstructor) Reconstruct(messages []*entity.Message) ([]*entity.Thread, error) {
    // Adapt messages
    adapters := make([]*messageAdapter, len(messages))
    threadable := make([]mogothread.ThreadableMessage, len(messages))
    for i, msg := range messages {
        adapters[i] = &messageAdapter{msg: msg}
        threadable[i] = adapters[i]
    }

    // Run reconstruction
    r.inner.AddMessages(threadable)
    r.inner.Reconstruct()

    // Extract results
    for i, adapter := range adapters {
        messages[i].ThreadID = adapter.info.ThreadID
        messages[i].ParentID = adapter.info.ParentID
        messages[i].ThreadDepth = adapter.info.Depth
    }

    // Convert threads
    mogoThreads := r.inner.GetThreads()
    threads := make([]*entity.Thread, len(mogoThreads))
    for i, mt := range mogoThreads {
        threads[i] = &entity.Thread{
            ID:            mt.ID,
            Subject:       mt.Subject,
            RootMessageID: mt.RootMessageID,
            MessageIDs:    mt.MessageIDs,
            StartDate:     mt.StartDate,
            EndDate:       mt.EndDate,
            Size:          mt.Size,
        }
    }

    return threads, nil
}
```

## Incremental Update Model

CommGraph uses an append-only event log for incremental updates:

```go
package storage

type Event struct {
    ID        string          `json:"id"`
    Type      EventType       `json:"type"`
    Timestamp time.Time       `json:"timestamp"`
    Payload   json.RawMessage `json:"payload"`
    Hash      string          `json:"hash"`
    PrevHash  string          `json:"prev_hash"`
}

type EventType string

const (
    EventMessageIngested    EventType = "MESSAGE_INGESTED"
    EventThreadReconstructed EventType = "THREAD_RECONSTRUCTED"
    EventActorResolved      EventType = "ACTOR_RESOLVED"
    EventAnalysisComputed   EventType = "ANALYSIS_COMPUTED"
)

// IncrementalProcessor handles incremental updates.
type IncrementalProcessor struct {
    store    Store
    resolver identity.Resolver

    // Indexes that need updating
    actorIndex   *ActorIndex
    threadIndex  *ThreadIndex
    graphCache   *GraphCache
}

func (p *IncrementalProcessor) ProcessMessage(ctx context.Context, msg *entity.Message) error {
    // 1. Store raw message (immutable)
    if err := p.store.StoreMessage(ctx, msg); err != nil {
        return err
    }

    // 2. Resolve identities
    fromActor := p.resolver.ResolveOrCreate(msg.From)

    // 3. Create interactions
    for _, to := range msg.To {
        toActor := p.resolver.ResolveOrCreate(to)
        interaction := &entity.Interaction{
            MessageID: msg.ID,
            From:      fromActor,
            To:        toActor,
            EdgeType:  entity.EdgeTypeTo,
            Timestamp: msg.Date,
            Platform:  msg.Platform,
        }
        if err := p.store.StoreInteraction(ctx, interaction); err != nil {
            return err
        }
    }
    // Similar for CC, BCC, mentions...

    // 4. Mark thread index as stale for affected subjects
    p.threadIndex.MarkStale(msg.Subject)

    // 5. Mark graph cache as stale
    p.graphCache.Invalidate()

    return nil
}
```

## Analysis Algorithms

### Centrality

```go
package analysis

import (
    "gonum.org/v1/gonum/graph"
    "gonum.org/v1/gonum/graph/network"
)

type CentralityResult struct {
    ActorID entity.ActorID
    Score   float64
}

type CentralityAnalyzer struct {
    g *commgraph.Graph
}

// PageRank computes PageRank centrality.
func (a *CentralityAnalyzer) PageRank(damping float64, tol float64) []CentralityResult {
    pr := network.PageRank(a.g, damping, tol)

    results := make([]CentralityResult, 0, len(pr))
    for nodeID, score := range pr {
        actorID := a.g.NodeToActor(nodeID)
        results = append(results, CentralityResult{
            ActorID: actorID,
            Score:   score,
        })
    }

    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })

    return results
}

// Betweenness computes betweenness centrality.
func (a *CentralityAnalyzer) Betweenness() []CentralityResult {
    bc := network.Betweenness(a.g)
    // ... similar conversion
}
```

## CLI Interface

```
commgraph - Communication graph analysis framework

Usage:
  commgraph [command]

Commands:
  ingest      Ingest messages from a source
  resolve     Run identity resolution
  thread      Reconstruct threads
  analyze     Run graph analysis
  export      Export graph or results
  serve       Start API server (future)

Flags:
  -c, --config string    Config file (default "commgraph.yaml")
  -v, --verbose          Verbose output
      --audit-log        Enable audit logging

Examples:
  # Ingest Enron corpus
  commgraph ingest email --format=mbox --source=/path/to/enron

  # Run analysis with influence profile
  commgraph analyze centrality --profile=influence --output=results.json

  # Export to Gephi
  commgraph export gephi --output=graph.gexf
```

## Configuration

```yaml
# commgraph.yaml
storage:
  type: badger  # memory, badger, neo4j
  path: ./data/commgraph.db

identity:
  source: file
  path: ./identity/people.json
  format: scim
  internal_domains:
    - enron.com

threading:
  max_parent_age: 168h  # 7 days
  require_participant_overlap: true

profiles:
  default: influence
  custom:
    - name: my_profile
      to: 1.0
      cc: 0.5
      bcc: 0.7
      mention: 0.2
      reply: 0.4

analysis:
  pagerank:
    damping: 0.85
    tolerance: 0.0001
  community:
    algorithm: louvain
    resolution: 1.0

audit:
  enabled: true
  path: ./audit/commgraph.audit.log
```

## Error Handling

All errors are typed for programmatic handling:

```go
package commgraph

type ErrorCode string

const (
    ErrUnknownActor     ErrorCode = "UNKNOWN_ACTOR"
    ErrInvalidMessage   ErrorCode = "INVALID_MESSAGE"
    ErrStorageFailure   ErrorCode = "STORAGE_FAILURE"
    ErrThreadingFailure ErrorCode = "THREADING_FAILURE"
    ErrAnalysisFailure  ErrorCode = "ANALYSIS_FAILURE"
)

type Error struct {
    Code    ErrorCode
    Message string
    Cause   error
}
```

## Testing Strategy

| Layer | Test Type | Coverage Target |
|-------|-----------|-----------------|
| Adapters | Unit + Integration | 90% |
| Identity | Unit | 95% |
| Threading | Unit (via mogo tests) | 90% |
| Graph | Unit | 90% |
| Analysis | Unit + Property-based | 85% |
| Storage | Integration | 80% |
| CLI | E2E | 70% |

### Test Datasets

- **Enron subset**: 1,000 messages for unit tests
- **Enron full**: 500,000+ messages for integration tests
- **Synthetic**: Generated graphs with known properties for algorithm verification

## Dependencies

```go
// go.mod
module github.com/grokify/commgraph

go 1.22

require (
    github.com/grokify/mogo v0.x.x
    github.com/enrondata/enron-people v0.x.x
    github.com/jhillyerd/enmime v1.x.x
    github.com/spf13/cobra v1.x.x
    github.com/dgraph-io/badger/v4 v4.x.x
    gonum.org/v1/gonum v0.x.x
    gopkg.in/yaml.v3 v3.x.x
)
```

## Security Considerations

- No secrets stored in configuration (use environment variables)
- Raw message storage encrypted at rest (BadgerDB encryption)
- Audit log integrity via hash chaining
- No network access in default mode (local processing only)

## Design Decisions

This section documents key architectural decisions and their rationale.

### DD-1: Emails as First-Class Nodes

**Decision**: Model emails/messages as first-class graph nodes rather than collapsing directly into person-to-person edges.

**Rationale**:

- Preserves provenance (which specific message created an edge)
- Preserves temporal sequencing (exact timestamps, not aggregated)
- Preserves threading relationships (reply chains, conversation structure)
- Preserves attachment context
- Enables NLP enrichment on message content
- Allows materialization of person-to-person weighted edges later

**Trade-off**: Larger graph size, but information loss from collapsing is unrecoverable.

**Alternative considered**: Direct person-to-person edges with aggregated weights. Rejected because it loses the ability to answer "which specific emails connected these people" and makes temporal analysis impossible.

### DD-2: Semantic Edge Types (TO/CC/BCC/MENTION)

**Decision**: Maintain distinct edge types for TO, CC, BCC, and body mentions rather than flattening to a single "EMAILED" edge.

**Rationale**:

- These relationships carry different semantic weight
- TO implies direct communication/action required
- CC implies awareness/FYI (high information flow, low influence)
- BCC implies confidential awareness (trust signal)
- Mentions in body imply reference without direct communication
- Different analysis profiles weight these differently

**Trade-off**: More complex edge model, but enables questions like:
- "Who is central only through CC?" (information brokers)
- "Who gets BCC'd frequently?" (trust/influence signals)
- "Who is mentioned but rarely directly emailed?" (referenced authorities)

### DD-3: Weight Profiles Over Fixed Weights

**Decision**: Implement configurable weight profiles rather than hardcoded edge weights.

**Rationale**:

- Different analysis goals require different weightings
- Influence analysis: TO high, CC low, BCC medium
- Information flow: TO high, CC high, BCC high
- Coordination: Reply high, CC high, TO medium
- Research may discover better weights empirically
- Users may have domain-specific weighting needs

**Profiles are configuration, not code changes**.

### DD-4: Wrap Existing Threading Library

**Decision**: Wrap `github.com/grokify/mogo/net/mailutil/threading` rather than reimplementing thread reconstruction.

**Rationale**:

- Threading reconstruction is a solved problem with known edge cases
- Existing implementation handles:
  - In-Reply-To/References headers (when present)
  - Subject+date+participant heuristics (when headers missing)
  - Embedded message hint extraction
  - Configurable parameters
- Adapter pattern allows swapping implementations if needed
- Avoids duplicating tested code

**Integration**: `threading.Reconstructor` adapts `entity.Message` to `ThreadableMessage` interface.

### DD-5: SCIM Format for Identity Data

**Decision**: Use SCIM (System for Cross-domain Identity Management) format for identity/actor data.

**Rationale**:

- Industry standard (RFC 7644)
- Supports multiple email aliases per person natively
- Supports organizational metadata (department, title, groups)
- Existing tooling ecosystem
- `enron-people` package already uses SCIM via `github.com/grokify/goauth/scim`

**Schema supports**:

- Multiple emails with primary designation
- Display name, given/family names, nicknames
- External IDs (e.g., x500 DN for Exchange)
- Groups/department membership
- Timezone, locale

### DD-6: Append-Only Storage for E-Discovery

**Decision**: Use append-only, immutable storage for raw messages with hash-chained audit log.

**Rationale**:

- E-discovery requires legal defensibility
- Chain of custody must be provable
- Reproducibility requires identical inputs
- Hash chaining detects tampering
- Append-only simplifies backup/replication

**Trade-off**: No in-place updates, requires more storage. Acceptable for the use case.

### DD-7: Platform-Agnostic Core with Adapters

**Decision**: Design graph engine to know nothing about email specifically; adapters translate platform semantics.

**Rationale**:

- Enables multi-platform support (Slack, Teams, Discord)
- Core graph algorithms work on normalized interactions
- Platform-specific edge types extend base types
- Identity resolution spans platforms
- Future-proofs for additional communication sources

**Adapter responsibility**:

- Parse platform-specific format
- Extract participants and relationships
- Map to normalized `Interaction` model
- Preserve platform-specific metadata

### DD-8: Gonum for Core Analytics, Neo4j Optional

**Decision**: Use Gonum for in-memory graph analytics; Neo4j as optional persistence/query layer.

**Rationale**:

- Gonum provides native Go graph algorithms (no CGO, no external process)
- Sufficient for graphs up to ~1M nodes in memory
- Neo4j adds value for:
  - Persistent storage across sessions
  - Cypher query language for ad-hoc exploration
  - Graph Data Science library for advanced algorithms
  - Bloom visualization

**Scaling path**: Start with Gonum in-memory, add Neo4j when persistence needed.

### DD-9: Naming Choice "commgraph"

**Decision**: Name the project "commgraph" (communication graph) rather than "email-graph".

**Rationale**:

- Short and memorable
- Implies "communication graph" without constraining to email
- Naturally extensible to Slack, Teams, Discord, etc.
- Works well as Go package name and CLI command
- Future-proof for multi-platform vision

**Rejected alternatives**:

- `email-graph`: Too narrow for multi-platform goals
- `mailgraph`: Conflicts with existing projects, sounds like SMTP monitoring
- `commsgraph`: Awkward pluralization
- `signalgraph`: Good but more enterprise/security connotation

## Identity Resolution Challenges

Identity resolution is one of the hardest problems in communication graph analysis. This section documents known challenges and mitigation strategies.

### Challenge 1: Email Address Variants

The same person may appear as:

```
[email protected]
[email protected]
Bob Smith <[email protected]>
"Smith, Bob" <[email protected]>
```

**Mitigation**: Normalize addresses, maintain alias mappings in identity data.

### Challenge 2: Display Name Variations

Display names vary by client, over time, and by formality:

```
Bob Smith
Robert Smith
Bob
R. Smith
```

**Mitigation**: Do not rely on display names for identity. Use email address as primary key, display name as hint only.

### Challenge 3: Domain Normalization

Organizations may have multiple domains:

```
enron.com
enron.net
ect.enron.com (Enron Capital & Trade)
```

**Mitigation**: Configure internal domain list. Support domain-to-organization mapping.

### Challenge 4: External Actor Disambiguation

External actors lack authoritative identity data:

```
[email protected]  # Which John Smith?
```

**Mitigation**: For external actors, treat each unique email as a distinct actor. Optional: merge based on display name similarity + communication pattern analysis.

### Challenge 5: Role-Based Addresses

Some addresses represent roles, not individuals:

```
[email protected]
[email protected]
[email protected]
```

**Mitigation**: Support actor type (individual vs. role/group). May exclude from individual-level analysis.

## Graph Algorithm Selection Guide

| Analysis Goal | Primary Algorithm | Secondary | Notes |
|---------------|-------------------|-----------|-------|
| Find influential people | PageRank | Eigenvector centrality | Use influence weight profile |
| Find information brokers | Betweenness centrality | - | High betweenness = controls information flow |
| Find department bridges | Betweenness + department filter | - | Filter to cross-department edges |
| Detect organizational silos | Connected components | Modularity | Disconnected components = silos |
| Discover informal groups | Louvain community detection | Label propagation | Compare to formal org chart |
| Find similar actors | Node embeddings (Node2Vec) | Jaccard similarity | Based on communication patterns |
| Detect coordination bursts | Temporal motif detection | Time-windowed density | Unusual spikes in cross-team communication |
| Trace information path | Shortest path | All paths (k-hops) | How did information flow from A to B |
| Identify key custodians | Degree centrality + PageRank | - | For e-discovery scoping |

## Visualization Tools

CommGraph does not include built-in visualization. Recommended external tools:

### Gephi

- **Best for**: Exploratory investigation, cluster visualization, force-directed layouts
- **Export format**: GEXF (Graph Exchange XML Format)
- **Strengths**: Excellent for management demos, supports temporal dynamic graphs
- **URL**: https://gephi.org/

### Graphistry

- **Best for**: Large-scale graph visualization, GPU-accelerated rendering
- **Export format**: CSV/JSON via API
- **Strengths**: Handles millions of edges, web-based
- **URL**: https://www.graphistry.com/

### Neo4j Bloom

- **Best for**: Interactive exploration with Cypher queries
- **Export format**: Direct Neo4j integration
- **Strengths**: Natural language search, pattern-based exploration
- **Requires**: Neo4j Enterprise or Aura

### Cytoscape

- **Best for**: Academic/research visualization, biological network heritage
- **Export format**: GraphML, XGMML
- **Strengths**: Extensive plugin ecosystem

## External Dependencies

### Required

| Package | Import Path | Purpose |
|---------|-------------|---------|
| mogo | `github.com/grokify/mogo` | Utilities, threading reconstruction |
| mogo threading | `github.com/grokify/mogo/net/mailutil/threading` | Thread reconstruction |
| enron-people | `github.com/enrondata/enron-people` | Enron corpus identity data |
| enmime | `github.com/jhillyerd/enmime` | MIME email parsing |
| Gonum | `gonum.org/v1/gonum/graph` | Graph data structures and algorithms |
| Cobra | `github.com/spf13/cobra` | CLI framework |

### Optional

| Package | Import Path | Purpose |
|---------|-------------|---------|
| BadgerDB | `github.com/dgraph-io/badger/v4` | Persistent key-value storage |
| Neo4j Driver | `github.com/neo4j/neo4j-go-driver/v5` | Neo4j graph database integration |
| goauth SCIM | `github.com/grokify/goauth/scim` | SCIM data structures |

## References

### Research Papers

- **Graph Theoretic and Spectral Analysis of Enron Email Data** (Priebe et al.)
  Analysis of Enron corpus using graph-theoretic methods. Foundational research for email graph analysis.
  https://www.researchgate.net/publication/220556827

- **Network Analysis with the Enron Email Corpus** (Diesner et al.)
  Communication network patterns in organizational email.
  https://arxiv.org/abs/1410.2759

- **Communication Networks from the Enron Email Corpus** (Computational & Mathematical Organization Theory)
  Temporal analysis of communication patterns during organizational crisis.
  https://www.ovid.com/journals/cmot/fulltext/00075118-200511030-00003

### Datasets

- **SNAP Enron Email Dataset**
  Stanford Large Network Dataset Collection. Pre-processed Enron email network.
  https://snap.stanford.edu/data/email-Enron.html

- **CMU Enron Email Dataset**
  Full Enron corpus with folder structure preserved.
  https://www.cs.cmu.edu/~enron/

### Tools and Platforms

- **Neo4j Graph Database**
  Property graph database with Cypher query language.
  https://neo4j.com/

- **Memgraph**
  Neo4j-compatible in-memory graph database. Faster for streaming workloads.
  https://memgraph.com/

- **Gephi**
  Open-source graph visualization platform.
  https://gephi.org/

- **Flowsint**
  Graph-centric investigative platform using Neo4j.
  https://github.com/reconurge/flowsint

### Standards

- **SCIM Protocol (RFC 7644)**
  System for Cross-domain Identity Management.
  https://datatracker.ietf.org/doc/html/rfc7644

- **EDRM XML Specification**
  Electronic Discovery Reference Model for legal data exchange.
  https://edrm.net/resources/frameworks-and-standards/edrm-xml/

### Tutorials

- **Import Email into Neo4j for Graph Analysis** (Nylas)
  Modern walkthrough of email-to-graph pipeline.
  https://cli.nylas.com/guides/import-email-graph-database
