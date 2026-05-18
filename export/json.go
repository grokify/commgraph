// Package export provides export functionality for analysis results.
package export

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/grokify/commgraph/analysis"
)

// CentralityExport represents exported centrality analysis results.
type CentralityExport struct {
	Metadata Metadata                   `json:"metadata"`
	Results  analysis.CentralityResults `json:"results"`
}

// Metadata contains export metadata.
type Metadata struct {
	ExportedAt time.Time         `json:"exported_at"`
	Profile    string            `json:"profile"`
	Algorithm  string            `json:"algorithm"`
	Parameters map[string]any    `json:"parameters,omitempty"`
	Stats      map[string]any    `json:"stats,omitempty"`
}

// JSONExporter exports results to JSON format.
type JSONExporter struct {
	Pretty bool
}

// NewJSONExporter creates a new JSON exporter.
func NewJSONExporter(pretty bool) *JSONExporter {
	return &JSONExporter{Pretty: pretty}
}

// ExportCentrality exports centrality results to JSON.
func (e *JSONExporter) ExportCentrality(w io.Writer, results analysis.CentralityResults, meta Metadata) error {
	export := CentralityExport{
		Metadata: meta,
		Results:  results,
	}
	export.Metadata.ExportedAt = time.Now()

	encoder := json.NewEncoder(w)
	if e.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(export)
}

// ExportCentralityToFile exports centrality results to a JSON file.
func (e *JSONExporter) ExportCentralityToFile(path string, results analysis.CentralityResults, meta Metadata) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return e.ExportCentrality(f, results, meta)
}

// ExportActors exports actor data to JSON.
func (e *JSONExporter) ExportActors(w io.Writer, actors any, meta Metadata) error {
	export := struct {
		Metadata Metadata `json:"metadata"`
		Actors   any      `json:"actors"`
	}{
		Metadata: meta,
		Actors:   actors,
	}
	export.Metadata.ExportedAt = time.Now()

	encoder := json.NewEncoder(w)
	if e.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(export)
}

// ExportThreads exports thread data to JSON.
func (e *JSONExporter) ExportThreads(w io.Writer, threads any, meta Metadata) error {
	export := struct {
		Metadata Metadata `json:"metadata"`
		Threads  any      `json:"threads"`
	}{
		Metadata: meta,
		Threads:  threads,
	}
	export.Metadata.ExportedAt = time.Now()

	encoder := json.NewEncoder(w)
	if e.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(export)
}
