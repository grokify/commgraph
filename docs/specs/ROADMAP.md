# CommGraph Roadmap

## Overview

This roadmap defines the implementation phases for CommGraph, a communication graph analysis framework. Each phase delivers incremental, usable functionality while building toward the complete vision.

## Phase Summary

| Phase | Name | Goal | Duration |
|-------|------|------|----------|
| 0 | Foundation | Project structure and core types | - |
| 1 | Enron MVP | End-to-end analysis of Enron corpus | - |
| 2 | Analysis Depth | Full graph analytics suite | - |
| 3 | Production | Persistence, incremental updates, CLI polish | - |
| 4 | Multi-Platform | Slack/Teams adapters | - |
| 5 | Advanced | Anomaly detection, embeddings, API | - |

---

## Phase 0: Foundation

**Goal**: Establish project structure, core types, and development infrastructure.

### Tasks

- [x] **P0-1**: Initialize Go module and directory structure
  - Create `cmd/commgraph/main.go` stub
  - Create package directories: `adapter/`, `entity/`, `identity/`, `threading/`, `graph/`, `weight/`, `analysis/`, `storage/`, `export/`, `audit/`
  - Add `.gitignore`, `LICENSE`, `README.md`

- [x] **P0-2**: Define core entity types
  - `entity/actor.go`: ActorID, Actor struct
  - `entity/message.go`: Message struct with all fields
  - `entity/interaction.go`: Interaction, EdgeType constants
  - `entity/thread.go`: Thread struct

- [x] **P0-3**: Define core interfaces
  - `adapter/adapter.go`: Adapter interface, Source interface, Checkpoint
  - `identity/resolver.go`: Resolver interface
  - `storage/storage.go`: Store interface

- [x] **P0-4**: Implement weight profile system
  - `weight/profile.go`: Profile struct, Weight method
  - `weight/profiles.go`: Built-in profiles (Influence, InformationFlow, Coordination)
  - Unit tests for weight calculations

- [x] **P0-5**: Set up CI/CD
  - GitHub Actions workflow for test, lint, build
  - golangci-lint configuration
  - Test coverage reporting

### Deliverables

- Compilable project with all interfaces defined
- Weight profile system with tests
- CI pipeline running

### Exit Criteria

- `go build ./...` succeeds
- `go test ./...` passes
- `golangci-lint run` passes

---

## Phase 1: Enron MVP

**Goal**: Complete end-to-end analysis pipeline for the Enron email corpus.

### Tasks

- [x] **P1-1**: Implement email adapters
  - `adapter/email/mbox.go`: Parse mbox files
  - `adapter/email/maildir.go`: Parse Maildir format (one file per message)
  - Use `enmime` for MIME parsing
  - Extract all required headers
  - Unit tests with sample data

- [x] **P1-2**: Implement identity resolver with enron-people
  - `identity/scim.go`: Load SCIM-format identity data
  - `identity/resolver.go`: Resolver implementation
  - Integrate `github.com/enrondata/enron-people`
  - Support alias resolution and merging (multiple email variations → single actor)
  - Internal/external classification
  - `identity/enron.go`: LoadEnronPeople loads curated SCIM data and custodians JSON

- [x] **P1-3**: Integrate thread reconstruction
  - `threading/reconstructor.go`: Wrap `mogo/net/mailutil/threading`
  - Implement `messageAdapter` for entity.Message
  - Populate ThreadID, ParentID, ThreadDepth on messages
  - Unit tests for threading

- [x] **P1-4**: Implement in-memory graph construction
  - `graph/commgraph.go`: Graph struct using omniretrieve abstraction
  - Support weighted edges by profile
  - Node mapping (ActorID <-> graph node ID)

- [x] **P1-5**: Implement basic centrality analysis
  - `analysis/centrality.go`: PageRank, Betweenness, Degree, InDegree, OutDegree
  - Return sorted CentralityResult slices
  - Unit tests with known graph structures

- [x] **P1-6**: Implement in-memory storage
  - `storage/memory.go`: In-memory Store implementation
  - Message, Interaction, Actor, Thread storage
  - Basic query support

- [x] **P1-7**: Implement JSON/CSV export
  - `export/json.go`: Export results to JSON
  - `export/csv.go`: Export results to CSV
  - Include analysis metadata (profile, parameters)

- [x] **P1-8**: Basic CLI commands
  - `cmd/commgraph/ingest.go`: Ingest command
  - `cmd/commgraph/analyze.go`: Analyze command
  - `cmd/commgraph/export.go`: Export command
  - `cmd/commgraph/pipeline.go`: Combined ingest+analyze command (workaround for state sharing)
  - Use Cobra for CLI framework
  - **Note**: Separate commands don't share state; use `pipeline` command or P3 persistence for multi-step workflows

- [x] **P1-9**: Enron integration test
  - Tested with Jeff Skilling's mailbox (4,139 messages)
  - End-to-end test: ingest -> resolve -> thread -> analyze -> export
  - Results: 41,438 interactions, 5,352 actors, 2,061 threads
  - PageRank correctly identifies key actors (Skilling, Kenneth Lay, Andrew Fastow)

### Deliverables

- Working CLI that can analyze Enron corpus
- JSON/CSV output with centrality rankings
- Documentation for basic usage

### Exit Criteria

- `commgraph ingest email --source=enron.mbox` succeeds
- `commgraph analyze centrality --profile=influence` produces valid output
- Integration test passes with Enron subset

---

## Phase 2: Analysis Depth

**Goal**: Comprehensive graph analysis capabilities.

### Tasks

- [x] **P2-1**: Implement community detection
  - `analysis/community.go`: Louvain algorithm
  - Label propagation as alternative
  - Community membership output
  - Modularity score calculation

- [x] **P2-2**: Implement temporal analysis
  - `analysis/temporal.go`: Time-windowed graph construction
  - Burst detection (sudden activity spikes)
  - Trend analysis (communication volume over time)
  - Response latency metrics

- [x] **P2-3**: Implement path analysis
  - `analysis/paths.go`: Shortest path between actors
  - All paths up to N hops
  - Average path length, graph diameter
  - Connected components, ego networks

- [x] **P2-4**: Implement cross-department bridge detection
  - `analysis/bridges.go`: Identify actors connecting different communities
  - Structural holes analysis (Burt's constraint)
  - Gatekeeper detection
  - Report format for bridge actors

- [x] **P2-5**: Implement external entity analysis
  - `analysis/external.go`: External contact analysis
  - Identify heavily-referenced external domains
  - External actor centrality (who talks to outsiders most)
  - Domain-level aggregation
  - Boundary spanner detection

- [x] **P2-6**: Implement Gephi export
  - `export/gephi.go`: GEXF format export
  - Node attributes (department, internal/external)
  - Edge attributes (type, weight, timestamp)
  - Community-colored node visualization

- [x] **P2-7**: Implement Neo4j export
  - `export/neo4j.go`: Generate Cypher CREATE statements
  - Schema creation (constraints, indexes)
  - Batch import format
  - Include all node and edge properties

- [x] **P2-8**: Analysis CLI enhancements
  - `commgraph analyze community` command
  - `commgraph analyze temporal` command
  - `commgraph analyze bridges` command
  - `commgraph analyze paths` command
  - `commgraph analyze external` command
  - `commgraph export gephi` command
  - `commgraph export neo4j` command
  - Profile selection for all commands

### Deliverables

- Full analysis suite (centrality, community, temporal, paths)
- Gephi and Neo4j export
- Enhanced CLI with all analysis commands

### Exit Criteria

- Community detection produces meaningful clusters on Enron data
- Temporal analysis identifies known Enron crisis periods
- Gephi export renders correctly in Gephi

---

## Phase 3: Production Hardening

**Goal**: Production-ready storage, incremental updates, and operational polish.

### Tasks

- [ ] **P3-1**: Implement BadgerDB storage
  - `storage/badger.go`: BadgerDB-backed Store
  - Efficient key design for queries
  - Transaction support
  - Encryption at rest option

- [ ] **P3-2**: Implement audit logging
  - `audit/log.go`: Append-only audit log
  - Hash chaining for integrity
  - Event types for all operations
  - Query audit log by time range

- [ ] **P3-3**: Implement incremental ingestion
  - Checkpoint-based incremental ingest
  - Detect new/modified messages
  - Update indexes incrementally
  - Mark stale analysis results

- [ ] **P3-4**: Implement graph caching
  - Cache materialized weighted graphs
  - Invalidation on new data
  - Profile-specific caches

- [ ] **P3-5**: Configuration file support
  - YAML configuration parsing
  - Environment variable overrides
  - Configuration validation

- [ ] **P3-6**: Implement EDRM export
  - `export/edrm.go`: EDRM XML format
  - Document metadata
  - Relationship mapping
  - Validation against EDRM schema

- [ ] **P3-7**: Performance optimization
  - Profile with pprof
  - Optimize hot paths
  - Parallel processing where applicable
  - Memory usage optimization for large graphs

- [ ] **P3-8**: CLI polish
  - Progress indicators
  - Verbose/quiet modes
  - Error messages with actionable guidance
  - Shell completion

- [ ] **P3-9**: Documentation
  - User guide
  - API documentation (godoc)
  - Configuration reference
  - Example workflows

### Deliverables

- Persistent storage with audit trail
- Incremental update support
- EDRM export for e-discovery
- Comprehensive documentation

### Exit Criteria

- Can process full Enron corpus with persistence
- Incremental ingest adds new messages without full reprocess
- Audit log maintains integrity chain
- EDRM export validates against schema

---

## Phase 4: Multi-Platform

**Goal**: Extend beyond email to Slack and Teams.

### Tasks

- [ ] **P4-1**: Generalize adapter interface
  - Review adapter interface for multi-platform needs
  - Add platform-specific metadata support
  - Channel/workspace concepts

- [ ] **P4-2**: Implement Slack adapter
  - `adapter/slack/export.go`: Parse Slack export format
  - Map Slack concepts to entity model
  - Handle channels, threads, reactions
  - Identity resolution for Slack users

- [ ] **P4-3**: Implement Teams adapter
  - `adapter/teams/export.go`: Parse Teams export format
  - Map Teams concepts to entity model
  - Handle channels, meetings metadata
  - Identity resolution for Teams users

- [ ] **P4-4**: Cross-platform identity linking
  - Link identities across platforms
  - Merge graphs from multiple sources
  - Platform weight adjustments

- [ ] **P4-5**: Platform-specific edge types
  - Slack: REACTION, THREAD_REPLY, CHANNEL_MESSAGE
  - Teams: MEETING_INVITE, CALL, FILE_SHARE
  - Weight profile extensions

- [ ] **P4-6**: Multi-platform analysis
  - Combined graph analysis
  - Platform comparison metrics
  - Cross-platform communication patterns

### Deliverables

- Slack and Teams adapters
- Unified multi-platform analysis
- Cross-platform identity resolution

### Exit Criteria

- Can ingest and analyze Slack export
- Can ingest and analyze Teams export
- Combined email+Slack analysis produces coherent graph

---

## Phase 5: Advanced Capabilities

**Goal**: Advanced analytics, ML integration, and API access.

### Tasks

- [ ] **P5-1**: Implement anomaly detection
  - `analysis/anomaly.go`: Baseline model
  - Deviation scoring
  - Configurable thresholds
  - Alert generation

- [ ] **P5-2**: Implement node embeddings
  - `analysis/embedding.go`: Node2Vec or similar
  - Similarity search
  - Clustering based on embeddings

- [ ] **P5-3**: Implement REST API server
  - `cmd/commgraph/serve.go`: HTTP server
  - Query endpoints
  - Analysis endpoints
  - Export endpoints
  - OpenAPI specification

- [ ] **P5-4**: Implement LLM integration for entity extraction
  - Extract entities from message bodies
  - Project/topic identification
  - Relationship extraction
  - Pluggable LLM backend (Ollama, OpenAI)

- [ ] **P5-5**: Implement streaming ingestion
  - Real-time message ingestion
  - Incremental graph updates
  - Streaming analysis triggers

- [ ] **P5-6**: Implement access control
  - Role-based access to data
  - Query-level filtering
  - Audit log for access

### Deliverables

- Anomaly detection with alerting
- Node embeddings for similarity analysis
- REST API for programmatic access
- Optional LLM-powered entity extraction

### Exit Criteria

- Anomaly detection flags known Enron irregularities
- API serves analysis requests
- Embeddings enable similarity queries

---

## Milestones

| Milestone | Phase | Description |
|-----------|-------|-------------|
| M1: Core Types | P0 | All interfaces and types defined |
| M2: Enron Analysis | P1 | End-to-end Enron analysis working |
| M3: Full Analytics | P2 | All analysis algorithms implemented |
| M4: Production Ready | P3 | Persistent storage, audit, EDRM export |
| M5: Multi-Platform | P4 | Slack and Teams support |
| M6: Advanced | P5 | Anomaly detection, API, embeddings |

---

## Dependencies

### External

| Dependency | Phase | Purpose |
|------------|-------|---------|
| github.com/grokify/mogo | P1 | Threading reconstruction |
| github.com/enrondata/enron-people | P1 | Enron identity data |
| github.com/jhillyerd/enmime | P1 | MIME parsing |
| gonum.org/v1/gonum | P1 | Graph algorithms |
| github.com/spf13/cobra | P1 | CLI framework |
| github.com/dgraph-io/badger/v4 | P3 | Persistent storage |

### Internal

```
entity (P0)
    ↓
identity (P1) ← depends on entity
    ↓
threading (P1) ← depends on entity, mogo
    ↓
graph (P1) ← depends on entity, gonum
    ↓
analysis (P1) ← depends on graph, weight
    ↓
storage (P1) ← depends on entity
    ↓
export (P1) ← depends on graph, analysis
    ↓
adapter (P1) ← depends on entity
```

---

## Risk Register

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Gonum insufficient for large graphs | High | Low | Fallback to Neo4j for persistence/queries |
| Thread reconstruction accuracy | Medium | Medium | Extensive testing with Enron ground truth |
| EDRM format complexity | Medium | Medium | Start with minimal subset, expand as needed |
| Slack/Teams export format changes | Low | Medium | Version detection, format adapters |

---

## Version Strategy

| Version | Phases | Stability |
|---------|--------|-----------|
| v0.1.0 | P0 | Alpha - types only |
| v0.2.0 | P1 | Alpha - Enron MVP |
| v0.3.0 | P2 | Beta - full analysis |
| v1.0.0 | P3 | Stable - production ready |
| v1.1.0 | P4 | Stable - multi-platform |
| v2.0.0 | P5 | Stable - advanced features |

---

## Tracking

Progress is tracked via:

- GitHub Issues for individual tasks (labeled by phase)
- GitHub Milestones for phase completion
- CHANGELOG.md for released versions

Task IDs (e.g., P1-3) map directly to GitHub issue numbers when created.

---

## Known Issues & Improvements

Issues discovered during development and testing that should be addressed.

### Identity Resolution (Priority: High)

**Issue**: Auto-created actors are not merged when the same person uses multiple email addresses or aliases.

**Example**: Jeff Skilling appears as multiple separate actors:
- "Jeff Skilling" (jeff.skilling@enron.com)
- "Jskilli" (jskilli@enron.com)
- "Tomskilljr" (tomskilljr@aol.com)
- "Tskilling" (tskilling@enron.com)

**Impact**: Centrality metrics are fragmented across aliases, reducing accuracy.

**Solution**: Load SCIM data from enron-people which contains known aliases, then merge actors on ingest.

**Tasks**:

- [x] **IMP-1**: Load enron-people SCIM data on startup
  - Parse SCIM JSON files from enron-people repository
  - Build alias map (all email variations → canonical actor ID)
  - Pre-populate resolver with known identities
  - Added `--enron` flag to pipeline command

- [x] **IMP-2**: Implement alias merging in SCIMResolver
  - `LoadActor` merges emails when actor with same ID exists
  - `ResolveOrCreate` checks alias map before creating new actor
  - Loads both curated SCIM data (14 key people) and custodians JSON (148 employees)

- [x] **IMP-3**: Add identity CLI commands
  - `commgraph identity list` to show resolved actors (with --internal, --external filters)
  - `commgraph identity aliases <actor-id>` to show all aliases for an actor
  - `commgraph identity stats` to show resolution statistics

### CLI State Persistence (Priority: Medium) - RESOLVED

**Issue**: Separate CLI commands (ingest, analyze, export) don't share state because they run in separate processes.

**Solution**: Session file persistence implemented via `session` package.

**Tasks**:

- [x] **IMP-4**: Add session file for lightweight state sharing
  - `session` package with `Session`, `Load`, `Save`, `ToMemoryStore`, `ToResolver` functions
  - Save in-memory state to JSON file after ingest
  - Load state file in analyze/export commands
  - `--session=<path>` flag to specify session file
  - Automatic session file at `.commgraph-session.json`

### Thread Reconstruction (Priority: Low)

**Issue**: Many threads are single-message (1,976 out of 2,061 in Skilling mailbox), which may indicate incomplete threading.

**Possible Causes**:

- Messages without In-Reply-To or References headers
- Partial mailbox (only one side of conversation)
- Subject-based fallback not aggressive enough

**Tasks**:

- [ ] **IMP-5**: Improve subject-based threading fallback
  - Strip common prefixes (Re:, Fwd:, etc.) more aggressively
  - Use fuzzy matching for similar subjects
  - Consider time-window constraints for subject matching

- [ ] **IMP-6**: Add threading diagnostics
  - Report on header availability (% with In-Reply-To, % with References)
  - Identify orphan messages that should be part of threads
  - `commgraph analyze threading-quality` command

### Test Coverage (Priority: Medium) - COMPLETE

**Status**: 142 tests across all packages.

**Tasks**:

- [x] **IMP-7**: Add unit tests for entity package (13 tests)
  - Test Actor, Message, Interaction, Thread types
  - Test EdgeType methods
  - Test AllRecipients, AllParticipants, Duration helpers

- [x] **IMP-8**: Add unit tests for adapter/email package (26 tests)
  - Test mbox parsing and ingestion
  - Test maildir parsing with nested folders
  - Test header extraction (parseAddress, parseAddressList, parseReferences)
  - Test helper functions (cleanMessageID, truncate, extractEmailMentions, extractUniqueDomains)
  - Test cancellation and error handling

- [x] **IMP-9**: Add unit tests for identity package (13 tests)
  - Test SCIMResolver alias merging
  - Test ResolveOrCreate behavior
  - Test internal/external classification
  - Test extractDisplayName and toTitleCase helpers

- [x] **IMP-10**: Add unit tests for storage package (13 tests)
  - Test MemoryStore operations (CRUD for all entity types)
  - Test query filtering (time, platform, edge types)
  - Test stats calculation
  - Test batch operations and close behavior

- [x] **IMP-11**: Add unit tests for analysis package (35 tests)
  - Test centrality calculations (PageRank, Degree, InDegree, OutDegree, Betweenness)
  - Test community detection (Louvain, LabelPropagation)
  - Test temporal analysis (Timeline, BurstDetection)
  - Test path analysis (ShortestPath, AveragePathLength, GraphDiameter, ConnectedComponents)
  - Test bridge detection (DetectBridges, StructuralHoles)

- [x] **IMP-12**: Add unit tests for threading package (19 tests)
  - Test DefaultConfig and NewReconstructor
  - Test thread reconstruction (single, reply chain, multiple threads, deep threads)
  - Test participant extraction and thread dates
  - Test ComputeStats function
  - Test messageAdapter interface methods

- [x] **IMP-13**: Add integration test with small sample dataset (3 tests)
  - End-to-end pipeline test (ingest -> resolve -> thread -> analyze -> export)
  - Session round-trip test (save/load with data verification)
  - Known results test (star graph with expected centrality results)
