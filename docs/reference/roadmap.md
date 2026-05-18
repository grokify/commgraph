# Roadmap

This document tracks the development roadmap for CommGraph.

## Current Status

CommGraph is under active development. Core functionality is complete with ongoing improvements.

## Completed Features

### Core Infrastructure

- [x] Entity model (Actor, Interaction, Message, Thread)
- [x] In-memory storage with JSON serialization
- [x] Session persistence across CLI invocations

### Data Ingestion

- [x] Mbox adapter with header parsing
- [x] Maildir adapter with recursive scanning
- [x] Address parsing and normalization
- [x] Email threading reconstruction

### Identity Resolution

- [x] Basic identity store
- [x] Alias management
- [x] Internal/external classification
- [x] Enron employee data integration

### Analysis

- [x] PageRank centrality
- [x] Degree centrality (in/out/total)
- [x] Betweenness centrality
- [x] Louvain community detection
- [x] Label propagation community detection
- [x] Bridge detection
- [x] Weight profiles (influence, information_flow, coordination)

### Export

- [x] GEXF export for Gephi
- [x] Cypher export for Neo4j

### CLI

- [x] Pipeline command (combined ingest + analyze)
- [x] Separate ingest/analyze/export commands
- [x] Identity management commands
- [x] Multiple output formats (table, JSON, CSV)

## Planned Features

### Near Term

- [ ] **Configuration file support**: YAML configuration with environment variable overrides
- [ ] **Improved error handling**: Better error messages and recovery
- [ ] **Progress reporting**: Progress bars for long-running operations

### Medium Term

- [ ] **Additional adapters**: PST, EML folder, Gmail API
- [ ] **Temporal analysis**: Time-windowed analysis, trend detection
- [ ] **Interactive mode**: REPL-style interface for exploration
- [ ] **Caching**: LRU cache for computed results

### Long Term

- [ ] **Web UI**: Browser-based visualization and exploration
- [ ] **Database backend**: PostgreSQL/SQLite storage option
- [ ] **Distributed processing**: Support for very large corpora
- [ ] **Machine learning**: Anomaly detection, classification

## Contributing

Contributions are welcome! See the [GitHub repository](https://github.com/grokify/commgraph) for:

- Open issues
- Contribution guidelines
- Development setup instructions

## Version History

See [CHANGELOG.md](https://github.com/grokify/commgraph/blob/main/CHANGELOG.md) for detailed version history.
