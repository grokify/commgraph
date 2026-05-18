package analysis

import (
	"context"
	"math"
	"sort"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
)

// Community represents a detected community of actors.
type Community struct {
	ID          int              `json:"id"`
	Members     []entity.ActorID `json:"members"`
	Size        int              `json:"size"`
	Density     float64          `json:"density"`     // Internal edge density
	Conductance float64          `json:"conductance"` // Ratio of external to total edges
	Label       string           `json:"label,omitempty"`
}

// CommunityResults contains the results of community detection.
type CommunityResults struct {
	Communities []Community        `json:"communities"`
	Modularity  float64            `json:"modularity"`
	Membership  map[entity.ActorID]int `json:"membership"` // Actor -> Community ID
}

// Louvain performs community detection using the Louvain algorithm.
// This is a greedy optimization method that maximizes modularity.
func (a *Analyzer) Louvain(ctx context.Context, resolution float64) (*CommunityResults, error) {
	// Get all interactions
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Build adjacency structure
	graph := newLouvainGraph()
	for _, interaction := range interactions {
		weight := a.profile.Weight(interaction.EdgeType)
		graph.addEdge(interaction.From, interaction.To, weight)
	}

	// Initialize: each node in its own community
	membership := make(map[entity.ActorID]int)
	communityID := 0
	for node := range graph.nodes {
		membership[node] = communityID
		communityID++
	}

	// Louvain phase 1: local moving
	improved := true
	maxIterations := 100
	iteration := 0

	for improved && iteration < maxIterations {
		improved = false
		iteration++

		for node := range graph.nodes {
			currentCommunity := membership[node]
			bestCommunity := currentCommunity
			bestDelta := 0.0

			// Try moving to each neighbor's community
			neighborCommunities := make(map[int]bool)
			for neighbor := range graph.neighbors[node] {
				neighborCommunities[membership[neighbor]] = true
			}

			for targetCommunity := range neighborCommunities {
				if targetCommunity == currentCommunity {
					continue
				}

				delta := graph.modularityDelta(node, currentCommunity, targetCommunity, membership, resolution)
				if delta > bestDelta {
					bestDelta = delta
					bestCommunity = targetCommunity
				}
			}

			if bestCommunity != currentCommunity {
				membership[node] = bestCommunity
				improved = true
			}
		}
	}

	// Renumber communities to be contiguous
	communityMap := make(map[int]int)
	nextID := 0
	for _, comm := range membership {
		if _, exists := communityMap[comm]; !exists {
			communityMap[comm] = nextID
			nextID++
		}
	}
	for node := range membership {
		membership[node] = communityMap[membership[node]]
	}

	// Build community structures
	communityMembers := make(map[int][]entity.ActorID)
	for node, comm := range membership {
		communityMembers[comm] = append(communityMembers[comm], node)
	}

	communities := make([]Community, 0, len(communityMembers))
	for id, members := range communityMembers {
		comm := Community{
			ID:      id,
			Members: members,
			Size:    len(members),
		}
		comm.Density = graph.communityDensity(members)
		comm.Conductance = graph.communityConductance(members, membership)
		communities = append(communities, comm)
	}

	// Sort by size descending
	sort.Slice(communities, func(i, j int) bool {
		return communities[i].Size > communities[j].Size
	})

	// Reassign IDs after sorting
	for i := range communities {
		communities[i].ID = i
	}

	// Calculate overall modularity
	modularity := graph.modularity(membership, resolution)

	return &CommunityResults{
		Communities: communities,
		Modularity:  modularity,
		Membership:  membership,
	}, nil
}

// louvainGraph is an internal graph structure for Louvain algorithm.
type louvainGraph struct {
	nodes     map[entity.ActorID]bool
	neighbors map[entity.ActorID]map[entity.ActorID]float64
	degree    map[entity.ActorID]float64
	totalWeight float64
}

func newLouvainGraph() *louvainGraph {
	return &louvainGraph{
		nodes:     make(map[entity.ActorID]bool),
		neighbors: make(map[entity.ActorID]map[entity.ActorID]float64),
		degree:    make(map[entity.ActorID]float64),
	}
}

func (g *louvainGraph) addEdge(from, to entity.ActorID, weight float64) {
	g.nodes[from] = true
	g.nodes[to] = true

	if g.neighbors[from] == nil {
		g.neighbors[from] = make(map[entity.ActorID]float64)
	}
	if g.neighbors[to] == nil {
		g.neighbors[to] = make(map[entity.ActorID]float64)
	}

	g.neighbors[from][to] += weight
	g.neighbors[to][from] += weight
	g.degree[from] += weight
	g.degree[to] += weight
	g.totalWeight += weight
}

func (g *louvainGraph) modularityDelta(node entity.ActorID, fromComm, toComm int, membership map[entity.ActorID]int, resolution float64) float64 {
	if g.totalWeight == 0 {
		return 0
	}

	// Sum of weights to nodes in target community
	sumIn := 0.0
	sumTot := 0.0
	ki := g.degree[node]

	for neighbor, weight := range g.neighbors[node] {
		if membership[neighbor] == toComm {
			sumIn += weight
		}
	}

	for n, comm := range membership {
		if comm == toComm {
			sumTot += g.degree[n]
		}
	}

	m2 := g.totalWeight
	delta := (sumIn - resolution*sumTot*ki/m2) / m2

	return delta
}

func (g *louvainGraph) modularity(membership map[entity.ActorID]int, resolution float64) float64 {
	if g.totalWeight == 0 {
		return 0
	}

	m2 := g.totalWeight
	q := 0.0

	for i := range g.nodes {
		for j, weight := range g.neighbors[i] {
			if membership[i] == membership[j] {
				expected := g.degree[i] * g.degree[j] / m2
				q += weight - resolution*expected
			}
		}
	}

	return q / m2
}

func (g *louvainGraph) communityDensity(members []entity.ActorID) float64 {
	if len(members) < 2 {
		return 1.0
	}

	memberSet := make(map[entity.ActorID]bool)
	for _, m := range members {
		memberSet[m] = true
	}

	internalEdges := 0.0
	for _, m := range members {
		for neighbor, weight := range g.neighbors[m] {
			if memberSet[neighbor] {
				internalEdges += weight
			}
		}
	}
	internalEdges /= 2 // Counted twice

	maxEdges := float64(len(members) * (len(members) - 1) / 2)
	if maxEdges == 0 {
		return 1.0
	}

	return internalEdges / maxEdges
}

func (g *louvainGraph) communityConductance(members []entity.ActorID, membership map[entity.ActorID]int) float64 {
	if len(members) == 0 {
		return 0
	}

	memberSet := make(map[entity.ActorID]bool)
	for _, m := range members {
		memberSet[m] = true
	}

	internal := 0.0
	external := 0.0

	for _, m := range members {
		for neighbor, weight := range g.neighbors[m] {
			if memberSet[neighbor] {
				internal += weight
			} else {
				external += weight
			}
		}
	}
	internal /= 2 // Counted twice

	total := internal + external
	if total == 0 {
		return 0
	}

	return external / total
}

// LabelPropagation performs community detection using label propagation.
// This is faster than Louvain but may produce less optimal results.
func (a *Analyzer) LabelPropagation(ctx context.Context, maxIterations int) (*CommunityResults, error) {
	// Get all interactions
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Build adjacency structure
	neighbors := make(map[entity.ActorID]map[entity.ActorID]float64)
	nodes := make(map[entity.ActorID]bool)

	for _, interaction := range interactions {
		weight := a.profile.Weight(interaction.EdgeType)
		nodes[interaction.From] = true
		nodes[interaction.To] = true

		if neighbors[interaction.From] == nil {
			neighbors[interaction.From] = make(map[entity.ActorID]float64)
		}
		if neighbors[interaction.To] == nil {
			neighbors[interaction.To] = make(map[entity.ActorID]float64)
		}
		neighbors[interaction.From][interaction.To] += weight
		neighbors[interaction.To][interaction.From] += weight
	}

	// Initialize: each node gets its own label
	labels := make(map[entity.ActorID]int)
	labelID := 0
	nodeList := make([]entity.ActorID, 0, len(nodes))
	for node := range nodes {
		labels[node] = labelID
		labelID++
		nodeList = append(nodeList, node)
	}

	// Iterate until convergence or max iterations
	for iter := 0; iter < maxIterations; iter++ {
		changed := false

		// Random order would be better, but deterministic for reproducibility
		for _, node := range nodeList {
			// Count weighted votes for each label
			labelVotes := make(map[int]float64)
			for neighbor, weight := range neighbors[node] {
				labelVotes[labels[neighbor]] += weight
			}

			// Find label with most votes
			bestLabel := labels[node]
			bestVotes := 0.0
			for label, votes := range labelVotes {
				if votes > bestVotes {
					bestVotes = votes
					bestLabel = label
				}
			}

			if bestLabel != labels[node] {
				labels[node] = bestLabel
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	// Renumber labels to be contiguous
	labelMap := make(map[int]int)
	nextID := 0
	for _, label := range labels {
		if _, exists := labelMap[label]; !exists {
			labelMap[label] = nextID
			nextID++
		}
	}
	membership := make(map[entity.ActorID]int)
	for node, label := range labels {
		membership[node] = labelMap[label]
	}

	// Build community structures
	communityMembers := make(map[int][]entity.ActorID)
	for node, comm := range membership {
		communityMembers[comm] = append(communityMembers[comm], node)
	}

	communities := make([]Community, 0, len(communityMembers))
	for id, members := range communityMembers {
		communities = append(communities, Community{
			ID:      id,
			Members: members,
			Size:    len(members),
		})
	}

	// Sort by size descending
	sort.Slice(communities, func(i, j int) bool {
		return communities[i].Size > communities[j].Size
	})

	// Reassign IDs after sorting
	for i := range communities {
		communities[i].ID = i
	}

	// Calculate modularity (approximate)
	graph := newLouvainGraph()
	for _, interaction := range interactions {
		weight := a.profile.Weight(interaction.EdgeType)
		graph.addEdge(interaction.From, interaction.To, weight)
	}
	modularity := graph.modularity(membership, 1.0)

	return &CommunityResults{
		Communities: communities,
		Modularity:  modularity,
		Membership:  membership,
	}, nil
}

// Top returns the top N communities by size.
func (r *CommunityResults) Top(n int) []Community {
	if n >= len(r.Communities) {
		return r.Communities
	}
	return r.Communities[:n]
}

// GetCommunity returns the community for a given actor.
func (r *CommunityResults) GetCommunity(actor entity.ActorID) (int, bool) {
	comm, ok := r.Membership[actor]
	return comm, ok
}

// Unused but kept for potential future use
var _ = math.Sqrt
