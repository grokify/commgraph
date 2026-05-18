# CommGraph Product Requirements Document

## Overview

CommGraph is a communication graph analysis framework for analyzing organizational communication patterns across multiple platforms. The initial focus is email analysis with the Enron corpus as the reference dataset, with extensibility to Slack, Teams, and other platforms.

## Problem Statement

Organizations need to understand communication patterns for:

- **E-Discovery**: Legal investigations requiring defensible, reproducible analysis
- **Private Analysis**: Internal organizational intelligence and structure discovery
- **Security**: Insider threat detection, anomaly identification
- **Research**: Academic study of organizational communication dynamics

Existing tools either:

- Collapse communication into oversimplified person-to-person edges, losing context
- Are platform-specific (email-only, Slack-only)
- Lack reproducibility required for legal defensibility
- Don't support configurable analysis profiles for different use cases

## Target Users

### Primary

- **E-Discovery Analysts**: Legal professionals requiring defensible, auditable communication analysis
- **Security Analysts**: Teams investigating insider threats or anomalous communication patterns
- **Organizational Researchers**: Academic or internal researchers studying communication dynamics

### Secondary

- **Compliance Officers**: Monitoring communication policy adherence
- **IT Administrators**: Understanding information flow for access control decisions

## Use Cases

### UC-1: E-Discovery Investigation

An e-discovery analyst needs to identify key custodians and communication patterns related to a specific matter.

**Requirements:**

- Ingest email corpus with full provenance
- Identify central actors (by influence, information flow, or coordination)
- Reconstruct thread relationships even with missing headers
- Export results in standard legal formats (EDRM, Concordance)
- Maintain audit trail of all analysis steps

### UC-2: Organizational Structure Discovery

A researcher wants to understand the informal organizational structure compared to the formal hierarchy.

**Requirements:**

- Map communication patterns to organizational units
- Identify cross-department bridges
- Detect isolated silos
- Compare formal hierarchy vs. communication-based structure
- Visualize results for presentation

### UC-3: Anomaly Detection

A security analyst needs to identify unusual communication patterns that may indicate insider threat.

**Requirements:**

- Establish baseline communication patterns
- Detect deviations (sudden cross-team coordination, external entity spikes)
- Temporal analysis (burst detection, off-hours activity)
- Alert on configurable thresholds

### UC-4: Multi-Platform Analysis

An analyst needs to understand communication across email and Slack for a complete picture.

**Requirements:**

- Unified identity resolution across platforms
- Normalized interaction model
- Combined graph analysis
- Platform-specific context preserved

## Functional Requirements

### FR-1: Data Ingestion

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-1.1 | Ingest email from mbox, EML, PST formats | P0 |
| FR-1.2 | Parse MIME structure including attachments metadata | P0 |
| FR-1.3 | Extract all header fields (From, To, CC, BCC, Message-ID, References, In-Reply-To) | P0 |
| FR-1.4 | Preserve raw message for audit trail | P0 |
| FR-1.5 | Support incremental ingestion (new messages added without reprocessing) | P1 |
| FR-1.6 | Ingest Slack export format | P2 |
| FR-1.7 | Ingest Microsoft Teams export format | P2 |

### FR-2: Identity Resolution

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-2.1 | Resolve email address variants to canonical actor ID | P0 |
| FR-2.2 | Support alias definitions (multiple addresses per person) | P0 |
| FR-2.3 | Distinguish internal vs. external actors | P0 |
| FR-2.4 | Associate actors with organizational units | P1 |
| FR-2.5 | Support SCIM format for identity data | P1 |
| FR-2.6 | Cross-platform identity linking | P2 |

### FR-3: Thread Reconstruction

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-3.1 | Reconstruct threads using In-Reply-To/References headers | P0 |
| FR-3.2 | Fallback to subject+date+participant heuristics when headers missing | P0 |
| FR-3.3 | Extract embedded message hints from body for threading | P1 |
| FR-3.4 | Configurable threading parameters (max age, participant overlap) | P1 |
| FR-3.5 | Report threading confidence scores | P2 |

### FR-4: Graph Construction

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-4.1 | Model messages as first-class nodes | P0 |
| FR-4.2 | Preserve semantic edge types (TO, CC, BCC, MENTIONS) | P0 |
| FR-4.3 | Store timestamps on all edges | P0 |
| FR-4.4 | Support configurable weight profiles | P0 |
| FR-4.5 | Model threads and conversations as nodes | P1 |
| FR-4.6 | Support external entity nodes (domains, organizations) | P1 |

### FR-5: Graph Analysis

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-5.1 | Compute centrality metrics (PageRank, betweenness, degree) | P0 |
| FR-5.2 | Detect communities (Louvain, label propagation) | P0 |
| FR-5.3 | Find shortest paths between actors | P1 |
| FR-5.4 | Temporal analysis (burst detection, trend identification) | P1 |
| FR-5.5 | Anomaly detection against baseline | P2 |
| FR-5.6 | Similarity analysis via node embeddings | P2 |

### FR-6: Export and Visualization

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-6.1 | Export to Neo4j Cypher format | P0 |
| FR-6.2 | Export to Gephi (GEXF/GraphML) | P0 |
| FR-6.3 | Export analysis results as JSON/CSV | P0 |
| FR-6.4 | Export to EDRM XML for e-discovery tools | P1 |
| FR-6.5 | Generate summary reports | P1 |

## Non-Functional Requirements

### NFR-1: Performance

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-1.1 | Ingest rate | >= 10,000 messages/second |
| NFR-1.2 | In-memory graph capacity | >= 1M nodes, 10M edges |
| NFR-1.3 | Centrality computation (1M nodes) | < 60 seconds |

### NFR-2: Reproducibility

| ID | Requirement |
|----|-------------|
| NFR-2.1 | All analysis algorithms must be deterministic given same input |
| NFR-2.2 | Analysis configuration must be serializable and versioned |
| NFR-2.3 | Results must include algorithm version and parameters used |

### NFR-3: Auditability

| ID | Requirement |
|----|-------------|
| NFR-3.1 | Raw ingested data must be immutable |
| NFR-3.2 | All transformations must be logged with timestamps |
| NFR-3.3 | Hash verification for chain of custody |

### NFR-4: Extensibility

| ID | Requirement |
|----|-------------|
| NFR-4.1 | New adapters addable without core changes |
| NFR-4.2 | Custom weight profiles definable by users |
| NFR-4.3 | Analysis algorithms pluggable |

## Weight Profiles

The system supports configurable weight profiles for different analysis objectives:

### Influence Profile

Measures who has power/authority in communication patterns.

| Edge Type | Weight | Rationale |
|-----------|--------|-----------|
| TO (direct) | 1.0 | Direct communication indicates relationship |
| CC | 0.2 | Being CC'd is passive; low influence signal |
| BCC | 0.4 | BCC suggests confidential influence |
| Mention | 0.1 | Mentions are weak influence signals |
| Reply | 0.3 | Replies indicate engagement |

### Information Flow Profile

Measures how information propagates through the organization.

| Edge Type | Weight | Rationale |
|-----------|--------|-----------|
| TO (direct) | 1.0 | Direct recipient receives information |
| CC | 0.8 | CC recipients also receive full information |
| BCC | 0.9 | BCC recipients receive information secretly |
| Mention | 0.5 | Mentions spread awareness of actor |
| Reply | 0.1 | Reply doesn't add new information flow |

### Coordination Profile

Measures collaborative activity and project coordination.

| Edge Type | Weight | Rationale |
|-----------|--------|-----------|
| TO (direct) | 0.5 | Direct messages may be FYI, not coordination |
| CC | 0.8 | CC often indicates coordination/alignment |
| BCC | 0.3 | BCC is secretive, not collaborative |
| Mention | 0.2 | Mentions are weak coordination signal |
| Reply | 1.0 | Replies indicate active collaboration |

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Thread reconstruction accuracy | >= 95% | Against manually labeled subset |
| Identity resolution accuracy | >= 99% | Against known alias mappings |
| Analysis reproducibility | 100% | Same input + config = same output |
| Enron corpus full analysis | < 10 minutes | End-to-end on reference hardware |

## Out of Scope (v1)

- Real-time streaming ingestion
- Natural language processing / semantic extraction
- Built-in visualization UI (rely on Gephi/external tools)
- Cloud-hosted service
- Multi-tenant access control

## References

- [Enron Email Dataset (SNAP)](https://snap.stanford.edu/data/email-Enron.html)
- [Graph Theoretic and Spectral Analysis of Enron Email Data](https://www.researchgate.net/publication/220556827)
- [EDRM XML Specification](https://edrm.net/resources/frameworks-and-standards/edrm-xml/)
- [SCIM Protocol (RFC 7644)](https://datatracker.ietf.org/doc/html/rfc7644)
