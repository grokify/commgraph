package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// Test source defaults
	if cfg.Source.Format != "maildir" {
		t.Errorf("expected format 'maildir', got %s", cfg.Source.Format)
	}
	if len(cfg.Source.InternalDomains) != 0 {
		t.Errorf("expected empty internal domains, got %v", cfg.Source.InternalDomains)
	}

	// Test analysis defaults
	if cfg.Analysis.Profile != "influence" {
		t.Errorf("expected profile 'influence', got %s", cfg.Analysis.Profile)
	}
	if cfg.Analysis.TopN != 20 {
		t.Errorf("expected top_n 20, got %d", cfg.Analysis.TopN)
	}
	if cfg.Analysis.CentralityAlgorithm != "pagerank" {
		t.Errorf("expected centrality_algorithm 'pagerank', got %s", cfg.Analysis.CentralityAlgorithm)
	}
	if cfg.Analysis.CommunityAlgorithm != "louvain" {
		t.Errorf("expected community_algorithm 'louvain', got %s", cfg.Analysis.CommunityAlgorithm)
	}
	if cfg.Analysis.CommunityResolution != 1.0 {
		t.Errorf("expected community_resolution 1.0, got %f", cfg.Analysis.CommunityResolution)
	}

	// Test identity defaults
	if !cfg.Identity.AutoCreate {
		t.Error("expected auto_create true")
	}
	if cfg.Identity.LoadEnron {
		t.Error("expected load_enron false")
	}

	// Test session defaults
	if cfg.Session.Path != ".commgraph-session.json" {
		t.Errorf("expected session path '.commgraph-session.json', got %s", cfg.Session.Path)
	}
	if !cfg.Session.AutoSave {
		t.Error("expected auto_save true")
	}

	// Test export defaults
	if cfg.Export.Format != "table" {
		t.Errorf("expected export format 'table', got %s", cfg.Export.Format)
	}
	if !cfg.Export.PrettyJSON {
		t.Error("expected pretty_json true")
	}
	if cfg.Export.Neo4jBatchSize != 500 {
		t.Errorf("expected neo4j_batch_size 500, got %d", cfg.Export.Neo4jBatchSize)
	}
}

func TestLoadDefaults(t *testing.T) {
	// Load with no config file
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have default values
	if cfg.Source.Format != "maildir" {
		t.Errorf("expected format 'maildir', got %s", cfg.Source.Format)
	}
	if cfg.Analysis.Profile != "influence" {
		t.Errorf("expected profile 'influence', got %s", cfg.Analysis.Profile)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".commgraph.yaml")

	configContent := `
source:
  format: mbox
  internal_domains:
    - company.com
    - company.org

analysis:
  profile: coordination
  top_n: 50

identity:
  load_enron: true

export:
  format: json
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config
	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test loaded values
	if cfg.Source.Format != "mbox" {
		t.Errorf("expected format 'mbox', got %s", cfg.Source.Format)
	}
	if len(cfg.Source.InternalDomains) != 2 {
		t.Errorf("expected 2 internal domains, got %d", len(cfg.Source.InternalDomains))
	}
	if cfg.Source.InternalDomains[0] != "company.com" {
		t.Errorf("expected first domain 'company.com', got %s", cfg.Source.InternalDomains[0])
	}
	if cfg.Analysis.Profile != "coordination" {
		t.Errorf("expected profile 'coordination', got %s", cfg.Analysis.Profile)
	}
	if cfg.Analysis.TopN != 50 {
		t.Errorf("expected top_n 50, got %d", cfg.Analysis.TopN)
	}
	if !cfg.Identity.LoadEnron {
		t.Error("expected load_enron true")
	}
	if cfg.Export.Format != "json" {
		t.Errorf("expected export format 'json', got %s", cfg.Export.Format)
	}

	// Defaults should still apply for unspecified values
	if !cfg.Identity.AutoCreate {
		t.Error("expected auto_create true (default)")
	}
	if cfg.Analysis.CentralityAlgorithm != "pagerank" {
		t.Errorf("expected centrality_algorithm 'pagerank' (default), got %s", cfg.Analysis.CentralityAlgorithm)
	}
}

func TestLoadWithEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("COMMGRAPH_SOURCE_FORMAT", "mbox")
	os.Setenv("COMMGRAPH_ANALYSIS_PROFILE", "information_flow")
	os.Setenv("COMMGRAPH_ANALYSIS_TOP_N", "100")
	defer func() {
		os.Unsetenv("COMMGRAPH_SOURCE_FORMAT")
		os.Unsetenv("COMMGRAPH_ANALYSIS_PROFILE")
		os.Unsetenv("COMMGRAPH_ANALYSIS_TOP_N")
	}()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Source.Format != "mbox" {
		t.Errorf("expected format 'mbox' from env, got %s", cfg.Source.Format)
	}
	if cfg.Analysis.Profile != "information_flow" {
		t.Errorf("expected profile 'information_flow' from env, got %s", cfg.Analysis.Profile)
	}
	if cfg.Analysis.TopN != 100 {
		t.Errorf("expected top_n 100 from env, got %d", cfg.Analysis.TopN)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestConfigFilePath(t *testing.T) {
	// This test just verifies the function doesn't panic
	// Actual path depends on filesystem state
	path := ConfigFilePath()
	_ = path // May be empty, that's OK
}
