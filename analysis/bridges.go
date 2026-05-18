package analysis

import (
	"context"
	"sort"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
)

// BridgeResult represents an actor who bridges communities.
type BridgeResult struct {
	ActorID              entity.ActorID `json:"actor_id"`
	DisplayName          string         `json:"display_name,omitempty"`
	PrimaryCommunity     int            `json:"primary_community"`
	BridgeScore          float64        `json:"bridge_score"` // Higher = more bridging
	CommunitiesConnected int            `json:"communities_connected"`
	InternalEdges        int            `json:"internal_edges"` // Edges within community
	ExternalEdges        int            `json:"external_edges"` // Edges to other communities
}

// BridgeResults is a slice of bridge detection results.
type BridgeResults []BridgeResult

// Sort sorts results by bridge score descending.
func (r BridgeResults) Sort() {
	sort.Slice(r, func(i, j int) bool {
		return r[i].BridgeScore > r[j].BridgeScore
	})
}

// Top returns the top N bridges.
func (r BridgeResults) Top(n int) BridgeResults {
	if n >= len(r) {
		return r
	}
	return r[:n]
}

// DetectBridges identifies actors who connect different communities.
// It requires community detection to be run first.
func (a *Analyzer) DetectBridges(ctx context.Context, communities *CommunityResults) (BridgeResults, error) {
	if communities == nil || len(communities.Membership) == 0 {
		return nil, nil
	}

	// Get interactions
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Count internal and external edges for each actor
	internalEdges := make(map[entity.ActorID]int)
	externalEdges := make(map[entity.ActorID]int)
	connectedCommunities := make(map[entity.ActorID]map[int]bool)

	for _, interaction := range interactions {
		fromComm, fromOk := communities.Membership[interaction.From]
		toComm, toOk := communities.Membership[interaction.To]

		if !fromOk || !toOk {
			continue
		}

		// Initialize maps
		if connectedCommunities[interaction.From] == nil {
			connectedCommunities[interaction.From] = make(map[int]bool)
		}
		if connectedCommunities[interaction.To] == nil {
			connectedCommunities[interaction.To] = make(map[int]bool)
		}

		if fromComm == toComm {
			// Same community
			internalEdges[interaction.From]++
			internalEdges[interaction.To]++
		} else {
			// Cross-community edge
			externalEdges[interaction.From]++
			externalEdges[interaction.To]++
			connectedCommunities[interaction.From][toComm] = true
			connectedCommunities[interaction.To][fromComm] = true
		}

		// Track own community
		connectedCommunities[interaction.From][fromComm] = true
		connectedCommunities[interaction.To][toComm] = true
	}

	// Calculate bridge scores
	results := make(BridgeResults, 0)

	for actor, membership := range communities.Membership {
		internal := internalEdges[actor]
		external := externalEdges[actor]
		total := internal + external

		if total == 0 {
			continue
		}

		// Bridge score: ratio of external to total edges, weighted by number of communities
		numCommunities := len(connectedCommunities[actor])
		bridgeScore := float64(external) / float64(total) * float64(numCommunities)

		result := BridgeResult{
			ActorID:              actor,
			PrimaryCommunity:     membership,
			BridgeScore:          bridgeScore,
			CommunitiesConnected: numCommunities,
			InternalEdges:        internal,
			ExternalEdges:        external,
		}

		if a.resolver != nil {
			if actorInfo, err := a.resolver.GetActor(actor); err == nil && actorInfo != nil {
				result.DisplayName = actorInfo.DisplayName
			}
		}

		results = append(results, result)
	}

	results.Sort()
	return results, nil
}

// StructuralHoles identifies actors who span structural holes in the network.
// Structural holes are gaps between clusters where an actor serves as the only connection.
func (a *Analyzer) StructuralHoles(ctx context.Context) (BridgeResults, error) {
	// Get interactions
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Build adjacency list
	adj := make(map[entity.ActorID]map[entity.ActorID]float64)
	for _, interaction := range interactions {
		weight := a.profile.Weight(interaction.EdgeType)

		if adj[interaction.From] == nil {
			adj[interaction.From] = make(map[entity.ActorID]float64)
		}
		if adj[interaction.To] == nil {
			adj[interaction.To] = make(map[entity.ActorID]float64)
		}

		adj[interaction.From][interaction.To] += weight
		adj[interaction.To][interaction.From] += weight
	}

	// Calculate constraint for each actor (Burt's constraint measure)
	// Lower constraint = more structural holes spanned
	results := make(BridgeResults, 0)

	for actor, neighbors := range adj {
		if len(neighbors) < 2 {
			continue
		}

		// Calculate constraint
		constraint := 0.0
		for neighbor := range neighbors {
			// Proportion of actor's network invested in this neighbor
			p := adj[actor][neighbor] / totalWeight(adj[actor])

			// Indirect constraint through mutual contacts
			indirect := 0.0
			for mutual := range neighbors {
				if mutual == neighbor {
					continue
				}
				// Does neighbor connect to mutual through actor?
				if adj[neighbor][mutual] > 0 {
					pMutual := adj[actor][mutual] / totalWeight(adj[actor])
					qMutual := adj[neighbor][mutual] / totalWeight(adj[neighbor])
					indirect += pMutual * qMutual
				}
			}

			// Constraint from this neighbor
			c := (p + indirect) * (p + indirect)
			constraint += c
		}

		// Bridge score is inverse of constraint
		// Lower constraint = spanning more structural holes = better bridge
		bridgeScore := 1.0 / (constraint + 0.001) // Add small value to avoid division by zero

		result := BridgeResult{
			ActorID:     actor,
			BridgeScore: bridgeScore,
		}

		if a.resolver != nil {
			if actorInfo, err := a.resolver.GetActor(actor); err == nil && actorInfo != nil {
				result.DisplayName = actorInfo.DisplayName
			}
		}

		results = append(results, result)
	}

	results.Sort()
	return results, nil
}

// totalWeight sums all weights in a neighbor map.
func totalWeight(neighbors map[entity.ActorID]float64) float64 {
	total := 0.0
	for _, w := range neighbors {
		total += w
	}
	return total
}

// Gatekeepers identifies actors who control information flow between groups.
// These are actors with high betweenness who connect otherwise disconnected groups.
type GatekeeperResult struct {
	ActorID         entity.ActorID   `json:"actor_id"`
	DisplayName     string           `json:"display_name,omitempty"`
	BetweennessRank int              `json:"betweenness_rank"`
	GroupsConnected int              `json:"groups_connected"`
	KeyConnections  []entity.ActorID `json:"key_connections"` // Other high-betweenness actors they connect
	GatekeeperScore float64          `json:"gatekeeper_score"`
}

// DetectGatekeepers finds actors who serve as gatekeepers between network segments.
func (a *Analyzer) DetectGatekeepers(ctx context.Context, communities *CommunityResults, betweennessTopN int) ([]GatekeeperResult, error) {
	// Get betweenness centrality
	betweenness, err := a.Betweenness(ctx)
	if err != nil {
		return nil, err
	}

	if len(betweenness) == 0 {
		return nil, nil
	}

	// Get top betweenness actors
	topActors := betweenness.Top(betweennessTopN)
	topSet := make(map[entity.ActorID]int) // Actor -> rank
	for i, result := range topActors {
		topSet[result.ActorID] = i + 1
	}

	// Get adjacency list
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]GatekeeperResult, 0)

	for _, br := range topActors {
		actor := br.ActorID

		// Find connections to other high-betweenness actors
		var keyConnections []entity.ActorID
		for neighbor := range adj[actor] {
			if _, isTop := topSet[neighbor]; isTop {
				keyConnections = append(keyConnections, neighbor)
			}
		}

		// Count communities connected (if available)
		groupsConnected := 0
		if communities != nil {
			connectedComms := make(map[int]bool)
			if comm, ok := communities.Membership[actor]; ok {
				connectedComms[comm] = true
			}
			for neighbor := range adj[actor] {
				if comm, ok := communities.Membership[neighbor]; ok {
					connectedComms[comm] = true
				}
			}
			groupsConnected = len(connectedComms)
		}

		// Gatekeeper score: combines betweenness rank, groups connected, and key connections
		score := br.Score * float64(groupsConnected+1) * (float64(len(keyConnections)) + 1)

		result := GatekeeperResult{
			ActorID:         actor,
			BetweennessRank: topSet[actor],
			GroupsConnected: groupsConnected,
			KeyConnections:  keyConnections,
			GatekeeperScore: score,
		}

		if a.resolver != nil {
			if actorInfo, err := a.resolver.GetActor(actor); err == nil && actorInfo != nil {
				result.DisplayName = actorInfo.DisplayName
			}
		}

		results = append(results, result)
	}

	// Sort by gatekeeper score
	sort.Slice(results, func(i, j int) bool {
		return results[i].GatekeeperScore > results[j].GatekeeperScore
	})

	return results, nil
}
