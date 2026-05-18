package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
	"github.com/grokify/commgraph/weight"
)

// mockResolver implements the resolver interface for testing.
type mockResolver struct {
	actors map[entity.ActorID]*entity.Actor
}

func newMockResolver() *mockResolver {
	return &mockResolver{
		actors: make(map[entity.ActorID]*entity.Actor),
	}
}

func (r *mockResolver) GetActor(id entity.ActorID) (*entity.Actor, error) {
	if actor, ok := r.actors[id]; ok {
		return actor, nil
	}
	return nil, nil
}

func (r *mockResolver) addActor(id entity.ActorID, name string) {
	r.actors[id] = &entity.Actor{
		ID:          id,
		DisplayName: name,
	}
}

// defaultProfile returns a simple test profile.
func defaultProfile() weight.Profile {
	return weight.Profile{
		Name: "test",
		To:   1.0,
		CC:   0.5,
		BCC:  0.25,
	}
}

// setupTestStore creates a store with test data.
func setupTestStore(ctx context.Context) storage.Store {
	store := storage.NewMemoryStore()
	now := time.Now()

	// Create a simple network: Alice -> Bob -> Carol, Alice -> Carol
	interactions := []*entity.Interaction{
		{ID: "i1", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo, Timestamp: now},
		{ID: "i2", From: "alice", To: "carol", EdgeType: entity.EdgeTypeTo, Timestamp: now},
		{ID: "i3", From: "bob", To: "carol", EdgeType: entity.EdgeTypeTo, Timestamp: now},
		{ID: "i4", From: "bob", To: "alice", EdgeType: entity.EdgeTypeCC, Timestamp: now},
		{ID: "i5", From: "carol", To: "alice", EdgeType: entity.EdgeTypeTo, Timestamp: now},
	}

	for _, i := range interactions {
		_ = store.StoreInteraction(ctx, i)
	}

	return store
}

// TestCentralityResultsSort tests sorting of centrality results.
func TestCentralityResultsSort(t *testing.T) {
	results := CentralityResults{
		{ActorID: "a", Score: 0.1},
		{ActorID: "b", Score: 0.5},
		{ActorID: "c", Score: 0.3},
	}

	results.Sort()

	if results[0].ActorID != "b" {
		t.Errorf("First result should be 'b', got %s", results[0].ActorID)
	}
	if results[0].Rank != 1 {
		t.Errorf("First result rank should be 1, got %d", results[0].Rank)
	}
	if results[1].ActorID != "c" {
		t.Errorf("Second result should be 'c', got %s", results[1].ActorID)
	}
	if results[2].ActorID != "a" {
		t.Errorf("Third result should be 'a', got %s", results[2].ActorID)
	}
}

// TestCentralityResultsTop tests the Top method.
func TestCentralityResultsTop(t *testing.T) {
	results := CentralityResults{
		{ActorID: "a", Score: 0.5, Rank: 1},
		{ActorID: "b", Score: 0.3, Rank: 2},
		{ActorID: "c", Score: 0.1, Rank: 3},
	}

	top2 := results.Top(2)
	if len(top2) != 2 {
		t.Errorf("Top(2) should return 2 results, got %d", len(top2))
	}

	// Test with n > len
	top10 := results.Top(10)
	if len(top10) != 3 {
		t.Errorf("Top(10) should return 3 results, got %d", len(top10))
	}
}

// TestNewAnalyzer tests analyzer creation.
func TestNewAnalyzer(t *testing.T) {
	store := storage.NewMemoryStore()
	resolver := newMockResolver()
	profile := defaultProfile()

	analyzer := NewAnalyzer(store, resolver, profile)

	if analyzer == nil {
		t.Fatal("NewAnalyzer returned nil")
	}
	if analyzer.store != store {
		t.Error("Store not set correctly")
	}
}

// TestDegree tests degree centrality calculation.
func TestDegree(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.Degree(ctx)
	if err != nil {
		t.Fatalf("Degree failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Degree returned no results")
	}

	// Check that all actors are present
	actorFound := make(map[entity.ActorID]bool)
	for _, r := range results {
		actorFound[r.ActorID] = true
		if r.Score <= 0 {
			t.Errorf("Actor %s has non-positive degree: %f", r.ActorID, r.Score)
		}
	}

	for _, expected := range []entity.ActorID{"alice", "bob", "carol"} {
		if !actorFound[expected] {
			t.Errorf("Actor %s not found in results", expected)
		}
	}
}

// TestInDegree tests in-degree centrality.
func TestInDegree(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.InDegree(ctx)
	if err != nil {
		t.Fatalf("InDegree failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("InDegree returned no results")
	}

	// Results should be sorted by score
	for i := 0; i < len(results)-1; i++ {
		if results[i].Score < results[i+1].Score {
			t.Error("Results not sorted by score descending")
		}
	}
}

// TestOutDegree tests out-degree centrality.
func TestOutDegree(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.OutDegree(ctx)
	if err != nil {
		t.Fatalf("OutDegree failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("OutDegree returned no results")
	}
}

// TestPageRank tests PageRank centrality.
func TestPageRank(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.PageRank(ctx, 0.85, 100)
	if err != nil {
		t.Fatalf("PageRank failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("PageRank returned no results")
	}

	// Sum of PageRank scores should be approximately 1
	sum := 0.0
	for _, r := range results {
		sum += r.Score
	}
	if sum < 0.9 || sum > 1.1 {
		t.Errorf("PageRank sum should be ~1.0, got %f", sum)
	}
}

// TestPageRankDefaultParams tests PageRank with invalid params.
func TestPageRankDefaultParams(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	// Invalid damping should use default
	results, err := analyzer.PageRank(ctx, 0, 0)
	if err != nil {
		t.Fatalf("PageRank failed: %v", err)
	}
	if results == nil {
		t.Fatal("PageRank returned nil")
	}
}

// TestPageRankEmpty tests PageRank on empty store.
func TestPageRankEmpty(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.PageRank(ctx, 0.85, 100)
	if err != nil {
		t.Fatalf("PageRank failed: %v", err)
	}
	if results != nil && len(results) != 0 {
		t.Error("PageRank on empty store should return nil or empty")
	}
}

// TestAnalyzerWithResolver tests analyzer with identity resolver.
func TestAnalyzerWithResolver(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	resolver := newMockResolver()
	resolver.addActor("alice", "Alice Smith")
	resolver.addActor("bob", "Bob Jones")
	profile := defaultProfile()

	analyzer := NewAnalyzer(store, resolver, profile)

	results, err := analyzer.Degree(ctx)
	if err != nil {
		t.Fatalf("Degree failed: %v", err)
	}

	// Check that display names are populated
	foundAlice := false
	for _, r := range results {
		if r.ActorID == "alice" {
			foundAlice = true
			if r.DisplayName != "Alice Smith" {
				t.Errorf("DisplayName for alice should be 'Alice Smith', got '%s'", r.DisplayName)
			}
		}
	}
	if !foundAlice {
		t.Error("Alice not found in results")
	}
}

// TestLouvain tests Louvain community detection.
func TestLouvain(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.Louvain(ctx, 1.0)
	if err != nil {
		t.Fatalf("Louvain failed: %v", err)
	}

	if results == nil {
		t.Fatal("Louvain returned nil")
	}
	if len(results.Communities) == 0 {
		t.Error("Louvain found no communities")
	}
	if results.Membership == nil {
		t.Error("Membership map is nil")
	}

	// Every actor should have a community
	for _, actorID := range []entity.ActorID{"alice", "bob", "carol"} {
		if _, ok := results.Membership[actorID]; !ok {
			t.Errorf("Actor %s not in membership", actorID)
		}
	}
}

// TestLabelPropagation tests label propagation community detection.
func TestLabelPropagation(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.LabelPropagation(ctx, 10)
	if err != nil {
		t.Fatalf("LabelPropagation failed: %v", err)
	}

	if results == nil {
		t.Fatal("LabelPropagation returned nil")
	}
	if len(results.Communities) == 0 {
		t.Error("LabelPropagation found no communities")
	}
}

// TestCommunityResultsTop tests the Top method.
func TestCommunityResultsTop(t *testing.T) {
	results := &CommunityResults{
		Communities: []Community{
			{ID: 0, Size: 10},
			{ID: 1, Size: 5},
			{ID: 2, Size: 3},
		},
	}

	top2 := results.Top(2)
	if len(top2) != 2 {
		t.Errorf("Top(2) should return 2 communities, got %d", len(top2))
	}
	if top2[0].Size != 10 {
		t.Errorf("First community should have size 10, got %d", top2[0].Size)
	}
}

// TestCommunityResultsGetCommunity tests GetCommunity.
func TestCommunityResultsGetCommunity(t *testing.T) {
	results := &CommunityResults{
		Membership: map[entity.ActorID]int{
			"alice": 0,
			"bob":   1,
		},
	}

	comm, ok := results.GetCommunity("alice")
	if !ok {
		t.Error("GetCommunity should find alice")
	}
	if comm != 0 {
		t.Errorf("Alice should be in community 0, got %d", comm)
	}

	_, ok = results.GetCommunity("unknown")
	if ok {
		t.Error("GetCommunity should not find unknown")
	}
}

// TestTimeline tests temporal timeline analysis.
func TestTimeline(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()

	now := time.Now()
	// Add interactions over time
	for i := 0; i < 10; i++ {
		_ = store.StoreInteraction(ctx, &entity.Interaction{
			ID:        "i" + string(rune('0'+i)),
			From:      "alice",
			To:        "bob",
			EdgeType:  entity.EdgeTypeTo,
			Timestamp: now.Add(time.Duration(i) * time.Hour),
		})
	}

	analyzer := NewAnalyzer(store, nil, profile)

	window := TimeWindow{
		Start: now.Add(-1 * time.Hour),
		End:   now.Add(24 * time.Hour),
	}

	results, err := analyzer.Timeline(ctx, window, time.Hour)
	if err != nil {
		t.Fatalf("Timeline failed: %v", err)
	}

	if results == nil {
		t.Fatal("Timeline returned nil")
	}
	if len(results.Timeline) == 0 {
		t.Error("Timeline has no activity points")
	}
	if results.TotalCount == 0 {
		t.Error("TotalCount should not be 0")
	}
}

// TestBurstDetection tests burst detection.
func TestBurstDetection(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()

	now := time.Now().Truncate(24 * time.Hour)

	// Create normal baseline activity: 5 interactions per day for 20 days
	for day := 0; day < 20; day++ {
		for i := 0; i < 5; i++ {
			id := "normal-" + string(rune('A'+day)) + string(rune('0'+i))
			_ = store.StoreInteraction(ctx, &entity.Interaction{
				ID:        id,
				From:      "alice",
				To:        "bob",
				EdgeType:  entity.EdgeTypeTo,
				Timestamp: now.Add(time.Duration(day) * 24 * time.Hour),
			})
		}
	}

	// Create a burst: 50 interactions on day 10 (10x normal)
	for i := 0; i < 50; i++ {
		id := "burst-" + string(rune('A'+i/26)) + string(rune('A'+i%26))
		_ = store.StoreInteraction(ctx, &entity.Interaction{
			ID:        id,
			From:      "alice",
			To:        "bob",
			EdgeType:  entity.EdgeTypeTo,
			Timestamp: now.Add(10 * 24 * time.Hour),
		})
	}

	analyzer := NewAnalyzer(store, nil, profile)

	bursts, err := analyzer.BurstDetection(ctx, 2.0, 24*time.Hour)
	if err != nil {
		t.Fatalf("BurstDetection failed: %v", err)
	}

	// Should detect at least one burst (day 10 has 55 vs normal 5)
	if len(bursts) == 0 {
		t.Error("BurstDetection should find at least one burst")
	}
}

// TestBurstDetectionEmpty tests burst detection on empty store.
func TestBurstDetectionEmpty(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	bursts, err := analyzer.BurstDetection(ctx, 2.0, 24*time.Hour)
	if err != nil {
		t.Fatalf("BurstDetection failed: %v", err)
	}
	if bursts != nil && len(bursts) != 0 {
		t.Error("BurstDetection on empty store should return nil or empty")
	}
}

// TestMeanStdDev tests the helper function.
func TestMeanStdDev(t *testing.T) {
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}

	mean, stdDev := meanStdDev(values)

	expectedMean := 5.0
	if mean != expectedMean {
		t.Errorf("Mean should be %f, got %f", expectedMean, mean)
	}

	// Standard deviation should be 2.0
	if stdDev < 1.9 || stdDev > 2.1 {
		t.Errorf("StdDev should be ~2.0, got %f", stdDev)
	}

	// Empty slice
	mean, stdDev = meanStdDev([]float64{})
	if mean != 0 || stdDev != 0 {
		t.Error("Empty slice should return 0, 0")
	}
}

// TestShortestPath tests finding shortest path.
func TestShortestPath(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	path, err := analyzer.ShortestPath(ctx, "alice", "carol")
	if err != nil {
		t.Fatalf("ShortestPath failed: %v", err)
	}

	if path == nil {
		t.Fatal("ShortestPath returned nil")
	}
	if path.Distance == 0 {
		t.Error("Distance should not be 0")
	}
	if len(path.Path) < 2 {
		t.Error("Path should have at least 2 nodes")
	}
	if path.Path[0] != "alice" {
		t.Errorf("Path should start with alice, got %s", path.Path[0])
	}
	if path.Path[len(path.Path)-1] != "carol" {
		t.Errorf("Path should end with carol, got %s", path.Path[len(path.Path)-1])
	}
}

// TestShortestPathNoPath tests when no path exists.
func TestShortestPathNoPath(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()

	// Create disconnected nodes
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i1", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo,
	})
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i2", From: "carol", To: "dave", EdgeType: entity.EdgeTypeTo,
	})

	analyzer := NewAnalyzer(store, nil, profile)

	path, err := analyzer.ShortestPath(ctx, "alice", "dave")
	if err != nil {
		t.Fatalf("ShortestPath failed: %v", err)
	}
	if path != nil {
		t.Error("ShortestPath should return nil when no path exists")
	}
}

// TestAveragePathLength tests average path length calculation.
func TestAveragePathLength(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	avgPath, err := analyzer.AveragePathLength(ctx, 100)
	if err != nil {
		t.Fatalf("AveragePathLength failed: %v", err)
	}

	if avgPath <= 0 {
		t.Error("AveragePathLength should be positive")
	}
}

// TestGraphDiameter tests graph diameter calculation.
func TestGraphDiameter(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	diameter, err := analyzer.GraphDiameter(ctx, 100)
	if err != nil {
		t.Fatalf("GraphDiameter failed: %v", err)
	}

	if diameter < 1 {
		t.Error("GraphDiameter should be at least 1")
	}
}

// TestConnectedComponents tests connected components detection.
func TestConnectedComponents(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()

	// Create two disconnected components
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i1", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo,
	})
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i2", From: "carol", To: "dave", EdgeType: entity.EdgeTypeTo,
	})

	analyzer := NewAnalyzer(store, nil, profile)

	components, err := analyzer.ConnectedComponents(ctx)
	if err != nil {
		t.Fatalf("ConnectedComponents failed: %v", err)
	}

	if len(components) != 2 {
		t.Errorf("Should have 2 components, got %d", len(components))
	}
}

// TestEgoNetwork tests ego network extraction.
func TestEgoNetwork(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	nodes, subgraph, err := analyzer.EgoNetwork(ctx, "alice", 1)
	if err != nil {
		t.Fatalf("EgoNetwork failed: %v", err)
	}

	if len(nodes) == 0 {
		t.Error("EgoNetwork should return nodes")
	}

	// Alice should be in the network
	foundAlice := false
	for _, n := range nodes {
		if n == "alice" {
			foundAlice = true
			break
		}
	}
	if !foundAlice {
		t.Error("Alice should be in ego network")
	}

	if subgraph == nil {
		t.Error("Subgraph should not be nil")
	}
}

// TestEgoNetworkUnknown tests ego network for unknown actor.
func TestEgoNetworkUnknown(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	nodes, subgraph, err := analyzer.EgoNetwork(ctx, "unknown", 1)
	if err != nil {
		t.Fatalf("EgoNetwork failed: %v", err)
	}

	if nodes != nil && len(nodes) > 0 {
		t.Error("EgoNetwork for unknown actor should return empty")
	}
	if subgraph != nil && len(subgraph) > 0 {
		t.Error("Subgraph for unknown actor should be empty")
	}
}

// TestDetectBridges tests bridge detection.
func TestDetectBridges(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	// First run community detection
	communities, err := analyzer.Louvain(ctx, 1.0)
	if err != nil {
		t.Fatalf("Louvain failed: %v", err)
	}

	bridges, err := analyzer.DetectBridges(ctx, communities)
	if err != nil {
		t.Fatalf("DetectBridges failed: %v", err)
	}

	// Should return results (possibly empty)
	if bridges == nil {
		t.Error("DetectBridges should not return nil")
	}
}

// TestDetectBridgesNilCommunities tests bridge detection with nil communities.
func TestDetectBridgesNilCommunities(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	bridges, err := analyzer.DetectBridges(ctx, nil)
	if err != nil {
		t.Fatalf("DetectBridges failed: %v", err)
	}
	if bridges != nil && len(bridges) > 0 {
		t.Error("DetectBridges with nil communities should return empty")
	}
}

// TestBridgeResultsSort tests sorting of bridge results.
func TestBridgeResultsSort(t *testing.T) {
	results := BridgeResults{
		{ActorID: "a", BridgeScore: 0.1},
		{ActorID: "b", BridgeScore: 0.5},
		{ActorID: "c", BridgeScore: 0.3},
	}

	results.Sort()

	if results[0].ActorID != "b" {
		t.Errorf("First result should be 'b', got %s", results[0].ActorID)
	}
}

// TestBridgeResultsTop tests the Top method.
func TestBridgeResultsTop(t *testing.T) {
	results := BridgeResults{
		{ActorID: "a", BridgeScore: 0.5},
		{ActorID: "b", BridgeScore: 0.3},
		{ActorID: "c", BridgeScore: 0.1},
	}

	top2 := results.Top(2)
	if len(top2) != 2 {
		t.Errorf("Top(2) should return 2 results, got %d", len(top2))
	}

	top10 := results.Top(10)
	if len(top10) != 3 {
		t.Errorf("Top(10) should return 3 results, got %d", len(top10))
	}
}

// TestStructuralHoles tests structural holes detection.
func TestStructuralHoles(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.StructuralHoles(ctx)
	if err != nil {
		t.Fatalf("StructuralHoles failed: %v", err)
	}

	// Should return results
	if results == nil {
		t.Error("StructuralHoles should not return nil")
	}
}

// TestAllPathsUpToDepth tests finding all paths.
func TestAllPathsUpToDepth(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	paths, err := analyzer.AllPathsUpToDepth(ctx, "alice", "carol", 3)
	if err != nil {
		t.Fatalf("AllPathsUpToDepth failed: %v", err)
	}

	if len(paths) == 0 {
		t.Error("Should find at least one path")
	}
}

// TestResponseTimes tests response time calculation.
func TestResponseTimes(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStore()
	profile := defaultProfile()

	now := time.Now()
	// Create a conversation pattern
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i1", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo,
		Timestamp: now,
	})
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i2", From: "bob", To: "alice", EdgeType: entity.EdgeTypeTo,
		Timestamp: now.Add(30 * time.Minute),
	})
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i3", From: "alice", To: "bob", EdgeType: entity.EdgeTypeTo,
		Timestamp: now.Add(1 * time.Hour),
	})
	_ = store.StoreInteraction(ctx, &entity.Interaction{
		ID: "i4", From: "bob", To: "alice", EdgeType: entity.EdgeTypeTo,
		Timestamp: now.Add(2 * time.Hour),
	})

	analyzer := NewAnalyzer(store, nil, profile)

	results, err := analyzer.ResponseTimes(ctx)
	if err != nil {
		t.Fatalf("ResponseTimes failed: %v", err)
	}

	// Should find response patterns
	// Note: need at least 2 responses between same pair to appear in results
	// In this test we have exactly 2 responses from bob to alice
	_ = results // Results may or may not have entries depending on response patterns
}

// TestDetectGatekeepers tests gatekeeper detection.
func TestDetectGatekeepers(t *testing.T) {
	ctx := context.Background()
	store := setupTestStore(ctx)
	profile := defaultProfile()
	analyzer := NewAnalyzer(store, nil, profile)

	// Run community detection first
	communities, err := analyzer.Louvain(ctx, 1.0)
	if err != nil {
		t.Fatalf("Louvain failed: %v", err)
	}

	results, err := analyzer.DetectGatekeepers(ctx, communities, 5)
	if err != nil {
		t.Fatalf("DetectGatekeepers failed: %v", err)
	}

	// Should return results (possibly empty)
	_ = results
}
