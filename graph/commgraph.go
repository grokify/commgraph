// Package graph provides communication graph operations using the omniretrieve abstraction.
package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/weight"
	omnigraph "github.com/plexusone/omniretrieve/graph"
)

// Node type constants for communication graphs.
const (
	NodeTypeActor   = "actor"
	NodeTypeMessage = "message"
	NodeTypeThread  = "thread"
)

// Metadata keys for nodes and edges.
const (
	MetaTimestamp  = "timestamp"
	MetaMessageID  = "message_id"
	MetaThreadID   = "thread_id"
	MetaPlatform   = "platform"
	MetaInternal   = "internal"
	MetaDepartment = "department"
)

// CommGraph wraps an omniretrieve KnowledgeGraph for communication analysis.
type CommGraph struct {
	graph    omnigraph.KnowledgeGraph
	registry *weight.Registry
}

// New creates a new CommGraph wrapping the given knowledge graph.
func New(g omnigraph.KnowledgeGraph) *CommGraph {
	return &CommGraph{
		graph:    g,
		registry: weight.NewRegistry(),
	}
}

// NewWithRegistry creates a CommGraph with a custom profile registry.
func NewWithRegistry(g omnigraph.KnowledgeGraph, registry *weight.Registry) *CommGraph {
	return &CommGraph{
		graph:    g,
		registry: registry,
	}
}

// Graph returns the underlying knowledge graph.
func (cg *CommGraph) Graph() omnigraph.KnowledgeGraph {
	return cg.graph
}

// Registry returns the weight profile registry.
func (cg *CommGraph) Registry() *weight.Registry {
	return cg.registry
}

// AddActor adds an actor node to the graph.
func (cg *CommGraph) AddActor(ctx context.Context, actor *entity.Actor) error {
	node := ActorToNode(actor)
	return cg.graph.AddNode(ctx, node)
}

// AddMessage adds a message node to the graph.
func (cg *CommGraph) AddMessage(ctx context.Context, msg *entity.Message) error {
	node := MessageToNode(msg)
	return cg.graph.AddNode(ctx, node)
}

// AddInteraction adds an interaction edge to the graph.
func (cg *CommGraph) AddInteraction(ctx context.Context, interaction *entity.Interaction, profile weight.Profile) error {
	edge := InteractionToEdge(interaction, profile)
	return cg.graph.AddEdge(ctx, edge)
}

// AddInteractionWithProfile adds an interaction using a named profile.
func (cg *CommGraph) AddInteractionWithProfile(ctx context.Context, interaction *entity.Interaction, profileName string) error {
	profile, err := cg.registry.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}
	return cg.AddInteraction(ctx, interaction, profile)
}

// ActorToNode converts an Actor to an omniretrieve Node.
func ActorToNode(actor *entity.Actor) omnigraph.Node {
	metadata := make(map[string]string)
	if actor.Internal {
		metadata[MetaInternal] = "true"
	} else {
		metadata[MetaInternal] = "false"
	}
	if actor.Department != "" {
		metadata[MetaDepartment] = actor.Department
	}
	for k, v := range actor.Metadata {
		metadata[k] = v
	}

	return omnigraph.Node{
		ID:       string(actor.ID),
		Type:     NodeTypeActor,
		Content:  actor.DisplayName,
		Source:   actor.PrimaryEmail,
		Metadata: metadata,
	}
}

// MessageToNode converts a Message to an omniretrieve Node.
func MessageToNode(msg *entity.Message) omnigraph.Node {
	metadata := map[string]string{
		MetaMessageID: msg.MessageID,
		MetaTimestamp: msg.Date.Format(time.RFC3339),
		MetaPlatform:  msg.Platform,
	}
	if msg.ThreadID != "" {
		metadata[MetaThreadID] = msg.ThreadID
	}

	return omnigraph.Node{
		ID:       msg.ID,
		Type:     NodeTypeMessage,
		Content:  msg.Subject,
		Source:   msg.SourcePath,
		Metadata: metadata,
	}
}

// InteractionToEdge converts an Interaction to an omniretrieve Edge.
func InteractionToEdge(interaction *entity.Interaction, profile weight.Profile) omnigraph.Edge {
	weight := profile.Weight(interaction.EdgeType)

	metadata := map[string]string{
		MetaTimestamp: interaction.Timestamp.Format(time.RFC3339),
		MetaMessageID: interaction.MessageID,
		MetaPlatform:  interaction.Platform,
	}
	if interaction.ThreadID != "" {
		metadata[MetaThreadID] = interaction.ThreadID
	}

	return omnigraph.Edge{
		From:     string(interaction.From),
		To:       string(interaction.To),
		Type:     string(interaction.EdgeType),
		Weight:   weight,
		Metadata: metadata,
	}
}

// TraverseFromActor traverses the graph starting from an actor.
func (cg *CommGraph) TraverseFromActor(ctx context.Context, actorID entity.ActorID, opts omnigraph.TraversalOptions) (*omnigraph.TraversalResult, error) {
	return cg.graph.Traverse(ctx, []string{string(actorID)}, opts)
}

// FindActors finds actor nodes matching the given filters.
func (cg *CommGraph) FindActors(ctx context.Context, filters map[string]string) ([]omnigraph.Node, error) {
	return cg.graph.FindNodes(ctx, NodeTypeActor, filters)
}

// FindMessages finds message nodes matching the given filters.
func (cg *CommGraph) FindMessages(ctx context.Context, filters map[string]string) ([]omnigraph.Node, error) {
	return cg.graph.FindNodes(ctx, NodeTypeMessage, filters)
}
