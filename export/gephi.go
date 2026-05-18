package export

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/grokify/commgraph/analysis"
	"github.com/grokify/commgraph/entity"
)

// GEXFExporter exports graph data to GEXF format for Gephi.
type GEXFExporter struct{}

// NewGEXFExporter creates a new GEXF exporter.
func NewGEXFExporter() *GEXFExporter {
	return &GEXFExporter{}
}

// GEXF format structures

// GEXF is the root element.
type GEXF struct {
	XMLName xml.Name  `xml:"gexf"`
	XMLNS   string    `xml:"xmlns,attr"`
	Version string    `xml:"version,attr"`
	Meta    *GEXFMeta `xml:"meta,omitempty"`
	Graph   GEXFGraph `xml:"graph"`
}

// GEXFMeta contains metadata about the graph.
type GEXFMeta struct {
	LastModified string `xml:"lastmodifieddate,attr,omitempty"`
	Creator      string `xml:"creator,omitempty"`
	Description  string `xml:"description,omitempty"`
}

// GEXFGraph represents the graph structure.
type GEXFGraph struct {
	Mode            string          `xml:"mode,attr,omitempty"` // static or dynamic
	DefaultEdgeType string          `xml:"defaultedgetype,attr,omitempty"`
	TimeFormat      string          `xml:"timeformat,attr,omitempty"`
	Attributes      *GEXFAttributes `xml:"attributes,omitempty"`
	Nodes           GEXFNodes       `xml:"nodes"`
	Edges           GEXFEdges       `xml:"edges"`
}

// GEXFAttributes defines custom attributes.
type GEXFAttributes struct {
	Class string          `xml:"class,attr"`
	Attrs []GEXFAttribute `xml:"attribute"`
}

// GEXFAttribute defines a single attribute.
type GEXFAttribute struct {
	ID      string `xml:"id,attr"`
	Title   string `xml:"title,attr"`
	Type    string `xml:"type,attr"`
	Default string `xml:"default,omitempty"`
}

// GEXFNodes contains all nodes.
type GEXFNodes struct {
	Nodes []GEXFNode `xml:"node"`
}

// GEXFNode represents a single node.
type GEXFNode struct {
	ID        string         `xml:"id,attr"`
	Label     string         `xml:"label,attr"`
	AttValues *GEXFAttValues `xml:"attvalues,omitempty"`
	Viz       *GEXFViz       `xml:"viz:viz,omitempty"`
}

// GEXFAttValues contains attribute values for a node or edge.
type GEXFAttValues struct {
	AttValue []GEXFAttValue `xml:"attvalue"`
}

// GEXFAttValue is a single attribute value.
type GEXFAttValue struct {
	For   string `xml:"for,attr"`
	Value string `xml:"value,attr"`
}

// GEXFViz contains visualization properties.
type GEXFViz struct {
	Color    *GEXFColor    `xml:"viz:color,omitempty"`
	Position *GEXFPosition `xml:"viz:position,omitempty"`
	Size     *GEXFSize     `xml:"viz:size,omitempty"`
}

// GEXFColor represents a node or edge color.
type GEXFColor struct {
	R int `xml:"r,attr"`
	G int `xml:"g,attr"`
	B int `xml:"b,attr"`
}

// GEXFPosition represents a node position.
type GEXFPosition struct {
	X float64 `xml:"x,attr"`
	Y float64 `xml:"y,attr"`
	Z float64 `xml:"z,attr,omitempty"`
}

// GEXFSize represents a node size.
type GEXFSize struct {
	Value float64 `xml:"value,attr"`
}

// GEXFEdges contains all edges.
type GEXFEdges struct {
	Edges []GEXFEdge `xml:"edge"`
}

// GEXFEdge represents a single edge.
type GEXFEdge struct {
	ID        string         `xml:"id,attr"`
	Source    string         `xml:"source,attr"`
	Target    string         `xml:"target,attr"`
	Type      string         `xml:"type,attr,omitempty"` // directed, undirected
	Weight    float64        `xml:"weight,attr,omitempty"`
	Label     string         `xml:"label,attr,omitempty"`
	AttValues *GEXFAttValues `xml:"attvalues,omitempty"`
}

// ExportGraph exports actors and interactions to GEXF format.
func (e *GEXFExporter) ExportGraph(w io.Writer, actors []*entity.Actor, interactions []*entity.Interaction, meta Metadata) error {
	gexf := GEXF{
		XMLNS:   "http://www.gexf.net/1.2draft",
		Version: "1.2",
		Meta: &GEXFMeta{
			LastModified: time.Now().Format("2006-01-02"),
			Creator:      "CommGraph",
			Description:  fmt.Sprintf("Communication graph - %s", meta.Profile),
		},
		Graph: GEXFGraph{
			Mode:            "static",
			DefaultEdgeType: "directed",
			Attributes: &GEXFAttributes{
				Class: "node",
				Attrs: []GEXFAttribute{
					{ID: "0", Title: "internal", Type: "boolean"},
					{ID: "1", Title: "department", Type: "string"},
					{ID: "2", Title: "title", Type: "string"},
				},
			},
		},
	}

	// Add nodes
	actorIndex := make(map[entity.ActorID]int)
	for i, actor := range actors {
		actorIndex[actor.ID] = i

		internal := "false"
		if actor.Internal {
			internal = "true"
		}

		node := GEXFNode{
			ID:    string(actor.ID),
			Label: actor.DisplayName,
			AttValues: &GEXFAttValues{
				AttValue: []GEXFAttValue{
					{For: "0", Value: internal},
					{For: "1", Value: actor.Department},
					{For: "2", Value: actor.Title},
				},
			},
		}
		gexf.Graph.Nodes.Nodes = append(gexf.Graph.Nodes.Nodes, node)
	}

	// Add edges
	for i, interaction := range interactions {
		edge := GEXFEdge{
			ID:     fmt.Sprintf("e%d", i),
			Source: string(interaction.From),
			Target: string(interaction.To),
			Type:   "directed",
			Label:  string(interaction.EdgeType),
		}
		gexf.Graph.Edges.Edges = append(gexf.Graph.Edges.Edges, edge)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(gexf, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	_, err = w.Write(output)
	return err
}

// ExportGraphWithCentrality exports graph with centrality scores as node sizes.
func (e *GEXFExporter) ExportGraphWithCentrality(w io.Writer, actors []*entity.Actor, interactions []*entity.Interaction, centrality analysis.CentralityResults, meta Metadata) error {
	// Build centrality lookup
	centralityMap := make(map[entity.ActorID]float64)
	maxScore := 0.0
	for _, r := range centrality {
		centralityMap[r.ActorID] = r.Score
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}

	gexf := GEXF{
		XMLNS:   "http://www.gexf.net/1.2draft",
		Version: "1.2",
		Meta: &GEXFMeta{
			LastModified: time.Now().Format("2006-01-02"),
			Creator:      "CommGraph",
			Description:  fmt.Sprintf("Communication graph with %s centrality - %s profile", meta.Algorithm, meta.Profile),
		},
		Graph: GEXFGraph{
			Mode:            "static",
			DefaultEdgeType: "directed",
			Attributes: &GEXFAttributes{
				Class: "node",
				Attrs: []GEXFAttribute{
					{ID: "0", Title: "internal", Type: "boolean"},
					{ID: "1", Title: "department", Type: "string"},
					{ID: "2", Title: "title", Type: "string"},
					{ID: "3", Title: "centrality", Type: "float"},
				},
			},
		},
	}

	// Add nodes with centrality-based sizing
	for _, actor := range actors {
		internal := "false"
		if actor.Internal {
			internal = "true"
		}

		score := centralityMap[actor.ID]
		normalizedSize := 1.0
		if maxScore > 0 {
			normalizedSize = 1.0 + (score/maxScore)*49.0 // Size 1-50
		}

		node := GEXFNode{
			ID:    string(actor.ID),
			Label: actor.DisplayName,
			AttValues: &GEXFAttValues{
				AttValue: []GEXFAttValue{
					{For: "0", Value: internal},
					{For: "1", Value: actor.Department},
					{For: "2", Value: actor.Title},
					{For: "3", Value: fmt.Sprintf("%.6f", score)},
				},
			},
			Viz: &GEXFViz{
				Size: &GEXFSize{Value: normalizedSize},
			},
		}
		gexf.Graph.Nodes.Nodes = append(gexf.Graph.Nodes.Nodes, node)
	}

	// Add edges
	for i, interaction := range interactions {
		edge := GEXFEdge{
			ID:     fmt.Sprintf("e%d", i),
			Source: string(interaction.From),
			Target: string(interaction.To),
			Type:   "directed",
			Label:  string(interaction.EdgeType),
		}
		gexf.Graph.Edges.Edges = append(gexf.Graph.Edges.Edges, edge)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(gexf, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	_, err = w.Write(output)
	return err
}

// ExportGraphWithCommunities exports graph with community colors.
func (e *GEXFExporter) ExportGraphWithCommunities(w io.Writer, actors []*entity.Actor, interactions []*entity.Interaction, communities *analysis.CommunityResults, meta Metadata) error {
	// Generate colors for communities
	colors := generateColors(len(communities.Communities))

	gexf := GEXF{
		XMLNS:   "http://www.gexf.net/1.2draft",
		Version: "1.2",
		Meta: &GEXFMeta{
			LastModified: time.Now().Format("2006-01-02"),
			Creator:      "CommGraph",
			Description:  fmt.Sprintf("Communication graph with communities (modularity: %.4f)", communities.Modularity),
		},
		Graph: GEXFGraph{
			Mode:            "static",
			DefaultEdgeType: "directed",
			Attributes: &GEXFAttributes{
				Class: "node",
				Attrs: []GEXFAttribute{
					{ID: "0", Title: "internal", Type: "boolean"},
					{ID: "1", Title: "community", Type: "integer"},
				},
			},
		},
	}

	// Add nodes with community colors
	for _, actor := range actors {
		internal := "false"
		if actor.Internal {
			internal = "true"
		}

		commID, _ := communities.GetCommunity(actor.ID)
		color := colors[commID%len(colors)]

		node := GEXFNode{
			ID:    string(actor.ID),
			Label: actor.DisplayName,
			AttValues: &GEXFAttValues{
				AttValue: []GEXFAttValue{
					{For: "0", Value: internal},
					{For: "1", Value: fmt.Sprintf("%d", commID)},
				},
			},
			Viz: &GEXFViz{
				Color: &GEXFColor{R: color[0], G: color[1], B: color[2]},
			},
		}
		gexf.Graph.Nodes.Nodes = append(gexf.Graph.Nodes.Nodes, node)
	}

	// Add edges
	for i, interaction := range interactions {
		edge := GEXFEdge{
			ID:     fmt.Sprintf("e%d", i),
			Source: string(interaction.From),
			Target: string(interaction.To),
			Type:   "directed",
		}
		gexf.Graph.Edges.Edges = append(gexf.Graph.Edges.Edges, edge)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(gexf, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	_, err = w.Write(output)
	return err
}

// ExportGraphToFile exports graph to a GEXF file.
func (e *GEXFExporter) ExportGraphToFile(path string, actors []*entity.Actor, interactions []*entity.Interaction, meta Metadata) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return e.ExportGraph(f, actors, interactions, meta)
}

// generateColors generates n distinct colors.
func generateColors(n int) [][3]int {
	if n <= 0 {
		return [][3]int{{128, 128, 128}} // Gray default
	}

	colors := make([][3]int, n)
	for i := 0; i < n; i++ {
		// Use HSV to RGB conversion for distinct colors
		h := float64(i) / float64(n) * 360
		s := 0.7
		v := 0.9
		r, g, b := hsvToRGB(h, s, v)
		colors[i] = [3]int{r, g, b}
	}
	return colors
}

// hsvToRGB converts HSV to RGB.
func hsvToRGB(h, s, v float64) (int, int, int) {
	c := v * s
	x := c * (1 - abs((float64(int(h/60)%2) - 1)))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return int((r + m) * 255), int((g + m) * 255), int((b + m) * 255)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
