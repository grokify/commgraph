package analysis

import (
	"context"
	"math"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
)

// PathResult represents a path between two actors.
type PathResult struct {
	From     entity.ActorID   `json:"from"`
	To       entity.ActorID   `json:"to"`
	Path     []entity.ActorID `json:"path"`
	Distance int              `json:"distance"`
	Weight   float64          `json:"weight"`
}

// PathAnalysisResults contains the results of path analysis.
type PathAnalysisResults struct {
	Paths           []PathResult `json:"paths"`
	AverageDistance float64      `json:"average_distance"`
	MaxDistance     int          `json:"max_distance"`
	Diameter        int          `json:"diameter"` // Longest shortest path
}

// ShortestPath finds the shortest path between two actors using BFS.
func (a *Analyzer) ShortestPath(ctx context.Context, from, to entity.ActorID) (*PathResult, error) {
	// Build adjacency list
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return nil, err
	}

	// BFS
	visited := make(map[entity.ActorID]bool)
	parent := make(map[entity.ActorID]entity.ActorID)
	queue := []entity.ActorID{from}
	visited[from] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == to {
			// Reconstruct path
			path := []entity.ActorID{to}
			for path[len(path)-1] != from {
				path = append(path, parent[path[len(path)-1]])
			}
			// Reverse
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}

			return &PathResult{
				From:     from,
				To:       to,
				Path:     path,
				Distance: len(path) - 1,
			}, nil
		}

		for neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				parent[neighbor] = current
				queue = append(queue, neighbor)
			}
		}
	}

	// No path found
	return nil, nil
}

// AllPathsUpToDepth finds all paths between two actors up to a maximum depth.
func (a *Analyzer) AllPathsUpToDepth(ctx context.Context, from, to entity.ActorID, maxDepth int) ([]PathResult, error) {
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return nil, err
	}

	var results []PathResult
	visited := make(map[entity.ActorID]bool)
	path := []entity.ActorID{from}

	a.dfsAllPaths(from, to, maxDepth, adj, visited, path, &results)

	return results, nil
}

// dfsAllPaths is a helper for finding all paths using DFS.
func (a *Analyzer) dfsAllPaths(current, target entity.ActorID, depth int, adj map[entity.ActorID]map[entity.ActorID]float64, visited map[entity.ActorID]bool, path []entity.ActorID, results *[]PathResult) {
	if depth < 0 {
		return
	}

	if current == target && len(path) > 1 {
		pathCopy := make([]entity.ActorID, len(path))
		copy(pathCopy, path)
		*results = append(*results, PathResult{
			From:     path[0],
			To:       target,
			Path:     pathCopy,
			Distance: len(pathCopy) - 1,
		})
		return
	}

	visited[current] = true
	for neighbor := range adj[current] {
		if !visited[neighbor] {
			path = append(path, neighbor)
			a.dfsAllPaths(neighbor, target, depth-1, adj, visited, path, results)
			path = path[:len(path)-1]
		}
	}
	visited[current] = false
}

// AveragePathLength calculates the average shortest path length in the graph.
func (a *Analyzer) AveragePathLength(ctx context.Context, sampleSize int) (float64, error) {
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return 0, err
	}

	actors := make([]entity.ActorID, 0, len(adj))
	for actor := range adj {
		actors = append(actors, actor)
	}

	if len(actors) < 2 {
		return 0, nil
	}

	// Limit sample size for large graphs
	if sampleSize <= 0 || sampleSize > len(actors) {
		sampleSize = len(actors)
	}

	totalDistance := 0
	pathCount := 0

	// Sample source nodes
	for i := 0; i < sampleSize; i++ {
		source := actors[i]
		distances := a.bfsDistances(source, adj)

		for _, dist := range distances {
			if dist > 0 && dist < math.MaxInt {
				totalDistance += dist
				pathCount++
			}
		}
	}

	if pathCount == 0 {
		return 0, nil
	}

	return float64(totalDistance) / float64(pathCount), nil
}

// GraphDiameter calculates the diameter (longest shortest path) of the graph.
func (a *Analyzer) GraphDiameter(ctx context.Context, sampleSize int) (int, error) {
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return 0, err
	}

	actors := make([]entity.ActorID, 0, len(adj))
	for actor := range adj {
		actors = append(actors, actor)
	}

	if len(actors) < 2 {
		return 0, nil
	}

	// Limit sample size for large graphs
	if sampleSize <= 0 || sampleSize > len(actors) {
		sampleSize = len(actors)
	}

	maxDistance := 0

	for i := 0; i < sampleSize; i++ {
		source := actors[i]
		distances := a.bfsDistances(source, adj)

		for _, dist := range distances {
			if dist > maxDistance && dist < math.MaxInt {
				maxDistance = dist
			}
		}
	}

	return maxDistance, nil
}

// bfsDistances computes distances from source to all reachable nodes.
func (a *Analyzer) bfsDistances(source entity.ActorID, adj map[entity.ActorID]map[entity.ActorID]float64) map[entity.ActorID]int {
	distances := make(map[entity.ActorID]int)
	for actor := range adj {
		distances[actor] = math.MaxInt
	}
	distances[source] = 0

	queue := []entity.ActorID{source}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for neighbor := range adj[current] {
			if distances[neighbor] == math.MaxInt {
				distances[neighbor] = distances[current] + 1
				queue = append(queue, neighbor)
			}
		}
	}

	return distances
}

// buildAdjacencyList creates an undirected adjacency list from interactions.
func (a *Analyzer) buildAdjacencyList(ctx context.Context) (map[entity.ActorID]map[entity.ActorID]float64, error) {
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

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

	return adj, nil
}

// ConnectedComponents finds all connected components in the graph.
func (a *Analyzer) ConnectedComponents(ctx context.Context) ([][]entity.ActorID, error) {
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return nil, err
	}

	visited := make(map[entity.ActorID]bool)
	var components [][]entity.ActorID

	for actor := range adj {
		if !visited[actor] {
			component := a.bfsComponent(actor, adj, visited)
			components = append(components, component)
		}
	}

	return components, nil
}

// bfsComponent finds all nodes in a connected component using BFS.
func (a *Analyzer) bfsComponent(start entity.ActorID, adj map[entity.ActorID]map[entity.ActorID]float64, visited map[entity.ActorID]bool) []entity.ActorID {
	var component []entity.ActorID
	queue := []entity.ActorID{start}
	visited[start] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		component = append(component, current)

		for neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	return component
}

// EgoNetwork extracts the ego network (immediate neighborhood) of an actor.
func (a *Analyzer) EgoNetwork(ctx context.Context, actor entity.ActorID, depth int) ([]entity.ActorID, map[entity.ActorID]map[entity.ActorID]float64, error) {
	adj, err := a.buildAdjacencyList(ctx)
	if err != nil {
		return nil, nil, err
	}

	if _, exists := adj[actor]; !exists {
		return nil, nil, nil
	}

	// BFS to find all nodes within depth
	visited := make(map[entity.ActorID]int) // Maps to distance
	queue := []entity.ActorID{actor}
	visited[actor] = 0

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] >= depth {
			continue
		}

		for neighbor := range adj[current] {
			if _, seen := visited[neighbor]; !seen {
				visited[neighbor] = visited[current] + 1
				queue = append(queue, neighbor)
			}
		}
	}

	// Extract nodes and subgraph
	nodes := make([]entity.ActorID, 0, len(visited))
	for node := range visited {
		nodes = append(nodes, node)
	}

	subgraph := make(map[entity.ActorID]map[entity.ActorID]float64)
	for node := range visited {
		if adj[node] != nil {
			subgraph[node] = make(map[entity.ActorID]float64)
			for neighbor, weight := range adj[node] {
				if _, inSubgraph := visited[neighbor]; inSubgraph {
					subgraph[node][neighbor] = weight
				}
			}
		}
	}

	return nodes, subgraph, nil
}
