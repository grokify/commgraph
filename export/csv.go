package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
)

// CSVExporter exports results to CSV format.
type CSVExporter struct{}

// NewCSVExporter creates a new CSV exporter.
func NewCSVExporter() *CSVExporter {
	return &CSVExporter{}
}

// ExportCentrality exports centrality results to CSV.
func (e *CSVExporter) ExportCentrality(w io.Writer, results analysis.CentralityResults) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"rank", "actor_id", "display_name", "score"}); err != nil {
		return err
	}

	// Write data
	for _, r := range results {
		record := []string{
			fmt.Sprintf("%d", r.Rank),
			string(r.ActorID),
			r.DisplayName,
			fmt.Sprintf("%.6f", r.Score),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportCentralityToFile exports centrality results to a CSV file.
func (e *CSVExporter) ExportCentralityToFile(path string, results analysis.CentralityResults) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return e.ExportCentrality(f, results)
}

// ExportActors exports actor data to CSV.
func (e *CSVExporter) ExportActors(w io.Writer, actors []*entity.Actor) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"id", "display_name", "primary_email", "internal", "department", "title"}); err != nil {
		return err
	}

	// Write data
	for _, a := range actors {
		internal := "false"
		if a.Internal {
			internal = "true"
		}
		record := []string{
			string(a.ID),
			a.DisplayName,
			a.PrimaryEmail,
			internal,
			a.Department,
			a.Title,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportInteractions exports interaction data to CSV.
func (e *CSVExporter) ExportInteractions(w io.Writer, interactions []*entity.Interaction) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"id", "from", "to", "edge_type", "timestamp", "message_id", "thread_id"}); err != nil {
		return err
	}

	// Write data
	for _, i := range interactions {
		record := []string{
			i.ID,
			string(i.From),
			string(i.To),
			string(i.EdgeType),
			i.Timestamp.Format("2006-01-02T15:04:05Z"),
			i.MessageID,
			i.ThreadID,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportThreads exports thread data to CSV.
func (e *CSVExporter) ExportThreads(w io.Writer, threads []*entity.Thread) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"id", "subject", "size", "depth", "start_date", "end_date", "participant_count"}); err != nil {
		return err
	}

	// Write data
	for _, t := range threads {
		record := []string{
			t.ID,
			t.Subject,
			fmt.Sprintf("%d", t.Size),
			fmt.Sprintf("%d", t.Depth),
			t.StartDate.Format("2006-01-02T15:04:05Z"),
			t.EndDate.Format("2006-01-02T15:04:05Z"),
			fmt.Sprintf("%d", len(t.Participants)),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
