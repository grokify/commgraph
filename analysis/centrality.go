// Package analysis provides graph analysis algorithms.
package analysis

import (
	"context"
	"sort"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
	"github.com/grokify/commgraph/weight"
	omnigraph "github.com/plexusone/omniretrieve/graph"
	"github.com/plexusone/omniretrieve/memory"
)

// CentralityResult represents an actor's centrality score.
type CentralityResult struct {
	ActorID     entity.ActorID `json:"actor_id"`
	DisplayName string         `json:"display_name,omitempty"`
	Score       float64        `json:"score"`
	Rank        int            `json:"rank"`
}

// CentralityResults is a slice of centrality results.
type CentralityResults []CentralityResult

// Sort sorts results by score descending and assigns ranks.
func (r CentralityResults) Sort() {
	sort.Slice(r, func(i, j int) bool {
		return r[i].Score > r[j].Score
	})
	for i := range r {
		r[i].Rank = i + 1
	}
}

// Top returns the top N results.
func (r CentralityResults) Top(n int) CentralityResults {
	if n >= len(r) {
		return r
	}
	return r[:n]
}

// Analyzer performs centrality analysis on communication graphs.
type Analyzer struct {
	store    storage.Store
	resolver interface {
		GetActor(entity.ActorID) (*entity.Actor, error)
	}
	profile weight.Profile
}

// NewAnalyzer creates a new centrality analyzer.
func NewAnalyzer(store storage.Store, resolver interface {
	GetActor(entity.ActorID) (*entity.Actor, error)
}, profile weight.Profile) *Analyzer {
	return &Analyzer{
		store:    store,
		resolver: resolver,
		profile:  profile,
	}
}

// Degree computes degree centrality (number of connections).
func (a *Analyzer) Degree(ctx context.Context) (CentralityResults, error) {
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Count outgoing and incoming edges per actor
	outDegree := make(map[entity.ActorID]float64)
	inDegree := make(map[entity.ActorID]float64)

	for _, interaction := range interactions {
		w := a.profile.Weight(interaction.EdgeType)
		outDegree[interaction.From] += w
		inDegree[interaction.To] += w
	}

	// Combine into total degree
	totalDegree := make(map[entity.ActorID]float64)
	for actor, deg := range outDegree {
		totalDegree[actor] += deg
	}
	for actor, deg := range inDegree {
		totalDegree[actor] += deg
	}

	return a.toResults(totalDegree), nil
}

// InDegree computes in-degree centrality (incoming connections).
func (a *Analyzer) InDegree(ctx context.Context) (CentralityResults, error) {
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	inDegree := make(map[entity.ActorID]float64)
	for _, interaction := range interactions {
		w := a.profile.Weight(interaction.EdgeType)
		inDegree[interaction.To] += w
	}

	return a.toResults(inDegree), nil
}

// OutDegree computes out-degree centrality (outgoing connections).
func (a *Analyzer) OutDegree(ctx context.Context) (CentralityResults, error) {
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	outDegree := make(map[entity.ActorID]float64)
	for _, interaction := range interactions {
		w := a.profile.Weight(interaction.EdgeType)
		outDegree[interaction.From] += w
	}

	return a.toResults(outDegree), nil
}

// PageRank computes PageRank centrality.
func (a *Analyzer) PageRank(ctx context.Context, damping float64, iterations int) (CentralityResults, error) {
	if damping <= 0 || damping >= 1 {
		damping = 0.85
	}
	if iterations <= 0 {
		iterations = 100
	}

	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Build adjacency lists
	outEdges := make(map[entity.ActorID][]weightedEdge)
	allActors := make(map[entity.ActorID]bool)

	for _, interaction := range interactions {
		w := a.profile.Weight(interaction.EdgeType)
		outEdges[interaction.From] = append(outEdges[interaction.From], weightedEdge{
			to:     interaction.To,
			weight: w,
		})
		allActors[interaction.From] = true
		allActors[interaction.To] = true
	}

	n := len(allActors)
	if n == 0 {
		return nil, nil
	}

	// Initialize PageRank
	pr := make(map[entity.ActorID]float64)
	for actor := range allActors {
		pr[actor] = 1.0 / float64(n)
	}

	// Compute out-weights
	outWeight := make(map[entity.ActorID]float64)
	for actor, edges := range outEdges {
		for _, edge := range edges {
			outWeight[actor] += edge.weight
		}
	}

	// Iterate
	for i := 0; i < iterations; i++ {
		newPR := make(map[entity.ActorID]float64)

		// Initialize with random jump probability
		for actor := range allActors {
			newPR[actor] = (1 - damping) / float64(n)
		}

		// Add contributions from incoming edges
		for from, edges := range outEdges {
			if outWeight[from] == 0 {
				continue
			}
			for _, edge := range edges {
				contribution := damping * pr[from] * edge.weight / outWeight[from]
				newPR[edge.to] += contribution
			}
		}

		// Handle dangling nodes (no outgoing edges)
		danglingSum := 0.0
		for actor := range allActors {
			if outWeight[actor] == 0 {
				danglingSum += pr[actor]
			}
		}
		danglingContrib := damping * danglingSum / float64(n)
		for actor := range allActors {
			newPR[actor] += danglingContrib
		}

		pr = newPR
	}

	return a.toResults(pr), nil
}

// Betweenness computes betweenness centrality.
// This is a simplified implementation using omniretrieve's graph traversal.
func (a *Analyzer) Betweenness(ctx context.Context) (CentralityResults, error) {
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Build graph
	g := memory.NewKnowledgeGraph("betweenness")
	allActors := make(map[entity.ActorID]bool)

	for _, interaction := range interactions {
		allActors[interaction.From] = true
		allActors[interaction.To] = true

		_ = g.AddNode(ctx, omnigraph.Node{ID: string(interaction.From), Type: "actor"})
		_ = g.AddNode(ctx, omnigraph.Node{ID: string(interaction.To), Type: "actor"})
		_ = g.AddEdge(ctx, omnigraph.Edge{
			From:   string(interaction.From),
			To:     string(interaction.To),
			Type:   string(interaction.EdgeType),
			Weight: a.profile.Weight(interaction.EdgeType),
		})
	}

	// Simple betweenness: count paths through each node
	// This is an approximation - full betweenness requires all-pairs shortest paths
	betweenness := make(map[entity.ActorID]float64)
	for actor := range allActors {
		betweenness[actor] = 0
	}

	// For each pair of actors, find path and increment betweenness for intermediates
	actorList := make([]entity.ActorID, 0, len(allActors))
	for actor := range allActors {
		actorList = append(actorList, actor)
	}

	// Sample-based approximation for large graphs
	maxPairs := 1000
	pairCount := 0

	for i := 0; i < len(actorList) && pairCount < maxPairs; i++ {
		for j := i + 1; j < len(actorList) && pairCount < maxPairs; j++ {
			source := actorList[i]
			target := actorList[j]

			result, err := g.Traverse(ctx, []string{string(source)}, omnigraph.TraversalOptions{
				Depth:    5,
				MaxNodes: 100,
			})
			if err != nil {
				continue
			}

			// Check if target was reached and count intermediates
			if path, ok := result.Paths[string(target)]; ok {
				for k := 1; k < len(path)-1; k++ {
					betweenness[entity.ActorID(path[k])]++
				}
			}
			pairCount++
		}
	}

	// Normalize
	if pairCount > 0 {
		for actor := range betweenness {
			betweenness[actor] /= float64(pairCount)
		}
	}

	return a.toResults(betweenness), nil
}

// toResults converts a score map to sorted results.
func (a *Analyzer) toResults(scores map[entity.ActorID]float64) CentralityResults {
	results := make(CentralityResults, 0, len(scores))
	for actor, score := range scores {
		result := CentralityResult{
			ActorID: actor,
			Score:   score,
		}
		if a.resolver != nil {
			if actorInfo, err := a.resolver.GetActor(actor); err == nil && actorInfo != nil {
				result.DisplayName = actorInfo.DisplayName
			}
		}
		results = append(results, result)
	}
	results.Sort()
	return results
}

type weightedEdge struct {
	to     entity.ActorID
	weight float64
}
