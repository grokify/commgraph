// Package config provides configuration loading for CommGraph.
// Configuration is loaded from YAML files, environment variables, and command-line flags.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for CommGraph.
type Config struct {
	Source   SourceConfig   `mapstructure:"source"`
	Analysis AnalysisConfig `mapstructure:"analysis"`
	Identity IdentityConfig `mapstructure:"identity"`
	Session  SessionConfig  `mapstructure:"session"`
	Export   ExportConfig   `mapstructure:"export"`
}

// SourceConfig configures data source settings.
type SourceConfig struct {
	Format          string   `mapstructure:"format"`
	InternalDomains []string `mapstructure:"internal_domains"`
}

// AnalysisConfig configures analysis defaults.
type AnalysisConfig struct {
	Profile             string  `mapstructure:"profile"`
	TopN                int     `mapstructure:"top_n"`
	CentralityAlgorithm string  `mapstructure:"centrality_algorithm"`
	CommunityAlgorithm  string  `mapstructure:"community_algorithm"`
	CommunityResolution float64 `mapstructure:"community_resolution"`
}

// IdentityConfig configures identity resolution.
type IdentityConfig struct {
	AutoCreate bool `mapstructure:"auto_create"`
	LoadEnron  bool `mapstructure:"load_enron"`
}

// SessionConfig configures session persistence.
type SessionConfig struct {
	Path     string `mapstructure:"path"`
	AutoSave bool   `mapstructure:"auto_save"`
}

// ExportConfig configures export defaults.
type ExportConfig struct {
	Format         string `mapstructure:"format"`
	PrettyJSON     bool   `mapstructure:"pretty_json"`
	Neo4jBatchSize int    `mapstructure:"neo4j_batch_size"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Source: SourceConfig{
			Format:          "maildir",
			InternalDomains: []string{},
		},
		Analysis: AnalysisConfig{
			Profile:             "influence",
			TopN:                20,
			CentralityAlgorithm: "pagerank",
			CommunityAlgorithm:  "louvain",
			CommunityResolution: 1.0,
		},
		Identity: IdentityConfig{
			AutoCreate: true,
			LoadEnron:  false,
		},
		Session: SessionConfig{
			Path:     ".commgraph-session.json",
			AutoSave: true,
		},
		Export: ExportConfig{
			Format:         "table",
			PrettyJSON:     true,
			Neo4jBatchSize: 500,
		},
	}
}

// Load loads configuration from file, environment, and defaults.
// Configuration is loaded in this order (later overrides earlier):
// 1. Default values
// 2. Configuration file (.commgraph.yaml)
// 3. Environment variables (COMMGRAPH_*)
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Look for config in current directory and home
		v.SetConfigName(".commgraph")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(home)
		}
	}

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		// Only error if a specific config file was requested and not found
		if configPath != "" {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("error reading config file: %w", err)
			}
		}
	}

	// Environment variable support
	v.SetEnvPrefix("COMMGRAPH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal into struct
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a specific file path.
func LoadFromFile(path string) (*Config, error) {
	return Load(path)
}

// ConfigFilePath returns the path to the config file being used, if any.
func ConfigFilePath() string {
	// Check for config in current directory
	if _, err := os.Stat(".commgraph.yaml"); err == nil {
		return ".commgraph.yaml"
	}
	if _, err := os.Stat(".commgraph.yml"); err == nil {
		return ".commgraph.yml"
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		yamlPath := filepath.Join(home, ".commgraph.yaml")
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath
		}
		ymlPath := filepath.Join(home, ".commgraph.yml")
		if _, err := os.Stat(ymlPath); err == nil {
			return ymlPath
		}
	}

	return ""
}

func setDefaults(v *viper.Viper) {
	// Source defaults
	v.SetDefault("source.format", "maildir")
	v.SetDefault("source.internal_domains", []string{})

	// Analysis defaults
	v.SetDefault("analysis.profile", "influence")
	v.SetDefault("analysis.top_n", 20)
	v.SetDefault("analysis.centrality_algorithm", "pagerank")
	v.SetDefault("analysis.community_algorithm", "louvain")
	v.SetDefault("analysis.community_resolution", 1.0)

	// Identity defaults
	v.SetDefault("identity.auto_create", true)
	v.SetDefault("identity.load_enron", false)

	// Session defaults
	v.SetDefault("session.path", ".commgraph-session.json")
	v.SetDefault("session.auto_save", true)

	// Export defaults
	v.SetDefault("export.format", "table")
	v.SetDefault("export.pretty_json", true)
	v.SetDefault("export.neo4j_batch_size", 500)
}
