# Weight Profiles

Weight profiles control how different types of email interactions are weighted in analysis algorithms.

## Overview

Email messages have different recipient types:

- **TO**: Direct recipients - the primary audience
- **CC**: Carbon copy - informed but not primary
- **BCC**: Blind carbon copy - hidden recipients

Different analysis goals require different weightings of these recipient types.

## Built-in Profiles

### Influence Profile

```yaml
name: influence
to_weight: 1.0
cc_weight: 0.5
bcc_weight: 0.25
```

**Purpose**: Identify actors with organizational influence.

**Rationale**: Direct (TO) communication indicates stronger relationships. Being addressed directly suggests importance, while CC indicates awareness without direct engagement.

**Use cases**:

- Finding key decision-makers
- Identifying informal leaders
- Understanding reporting relationships

### Information Flow Profile

```yaml
name: information_flow
to_weight: 1.0
cc_weight: 1.0
bcc_weight: 1.0
```

**Purpose**: Track how information spreads through the organization.

**Rationale**: All recipients receive the information equally, regardless of addressing type. The spread of information doesn't depend on whether someone was TO'd or CC'd.

**Use cases**:

- Studying information dissemination
- Identifying information hubs
- Tracking news/announcement propagation

### Coordination Profile

```yaml
name: coordination
to_weight: 0.5
cc_weight: 1.0
bcc_weight: 0.75
```

**Purpose**: Identify actors who coordinate activities across groups.

**Rationale**: CC recipients are often stakeholders being kept informed of coordination activities. Direct recipients may be task-oriented while CC'd parties are coordinators.

**Use cases**:

- Finding project coordinators
- Identifying cross-functional connectors
- Understanding coordination patterns

## Using Profiles

### Command Line

```bash
# Use influence profile (default)
commgraph analyze centrality --profile=influence

# Use information flow profile
commgraph analyze centrality --profile=information_flow

# Use coordination profile
commgraph analyze centrality --profile=coordination
```

### Configuration File

```yaml
# .commgraph.yaml
analysis:
  profile: influence
```

## How Weights Are Applied

When building the communication graph, each interaction's edge weight is multiplied by the profile weight:

```
edge_weight = base_weight * profile_weight[recipient_type]
```

For centrality algorithms like PageRank, these weighted edges determine how "influence" flows through the network.

### Example

Consider a message from Alice to Bob (TO) and Carol (CC):

**With influence profile:**

- Alice → Bob: weight = 1.0 × 1.0 = 1.0
- Alice → Carol: weight = 1.0 × 0.5 = 0.5

**With information_flow profile:**

- Alice → Bob: weight = 1.0 × 1.0 = 1.0
- Alice → Carol: weight = 1.0 × 1.0 = 1.0

**With coordination profile:**

- Alice → Bob: weight = 1.0 × 0.5 = 0.5
- Alice → Carol: weight = 1.0 × 1.0 = 1.0

## Choosing a Profile

| If you want to find... | Use profile |
|------------------------|-------------|
| Key influencers and decision-makers | `influence` |
| Information hubs and broadcasters | `information_flow` |
| Coordinators and project managers | `coordination` |

## Impact on Results

The choice of profile can significantly affect results:

### Example: Executive Assistant

An executive assistant who is often CC'd on executive communications:

- **influence profile**: Lower ranking (CC has lower weight)
- **coordination profile**: Higher ranking (CC has higher weight)
- **information_flow**: Moderate ranking (all weights equal)

### Example: Project Manager

A project manager who CC's many stakeholders:

- **influence profile**: Moderate (their TO recipients score high)
- **coordination profile**: High (their CC recipients score high)
- **information_flow**: High (all communication counts equally)

## Advanced: Custom Profiles

For programmatic use, you can define custom weight profiles:

```go
import "github.com/grokify/commgraph/analysis"

profile := &analysis.WeightProfile{
    Name:      "custom",
    ToWeight:  1.0,
    CCWeight:  0.75,
    BCCWeight: 0.1,
}

results, err := analysis.Centrality(store, &analysis.CentralityOptions{
    Algorithm: analysis.PageRank,
    Profile:   profile,
})
```

## Best Practices

1. **Start with influence**: The default profile works well for most organizational analysis.

2. **Compare profiles**: Run analysis with different profiles to see how results change.

3. **Consider context**: Choose the profile that matches your analysis question.

4. **Document your choice**: When sharing results, note which profile was used.
