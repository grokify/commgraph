package identity

import (
	"testing"

	"github.com/grokify/commgraph/entity"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if len(config.InternalDomains) != 0 {
		t.Error("Default InternalDomains should be empty")
	}
	if !config.AutoCreate {
		t.Error("Default AutoCreate should be true")
	}
	if !config.NormalizeAddresses {
		t.Error("Default NormalizeAddresses should be true")
	}
}

func TestSCIMResolverNew(t *testing.T) {
	config := Config{
		InternalDomains: []string{"enron.com", "ei.enron.com"},
		AutoCreate:      true,
	}
	resolver := NewSCIMResolver(config)

	if resolver == nil {
		t.Fatal("NewSCIMResolver returned nil")
	}

	stats := resolver.Stats()
	if stats.TotalActors != 0 {
		t.Errorf("New resolver should have 0 actors: got %d", stats.TotalActors)
	}
}

func TestSCIMResolverLoadActor(t *testing.T) {
	config := DefaultConfig()
	resolver := NewSCIMResolver(config)

	actor := &entity.Actor{
		ID:           entity.ActorID("jeff.skilling"),
		DisplayName:  "Jeff Skilling",
		Emails:       []string{"jeff.skilling@enron.com", "jskilli@enron.com"},
		PrimaryEmail: "jeff.skilling@enron.com",
		Internal:     true,
	}

	resolver.LoadActor(actor)

	stats := resolver.Stats()
	if stats.TotalActors != 1 {
		t.Errorf("Should have 1 actor: got %d", stats.TotalActors)
	}
	if stats.InternalActors != 1 {
		t.Errorf("Should have 1 internal actor: got %d", stats.InternalActors)
	}
	if stats.TotalAliases != 2 {
		t.Errorf("Should have 2 aliases: got %d", stats.TotalAliases)
	}
}

func TestSCIMResolverLoadActorMerge(t *testing.T) {
	config := DefaultConfig()
	resolver := NewSCIMResolver(config)

	// Load first actor
	actor1 := &entity.Actor{
		ID:           entity.ActorID("jeff.skilling"),
		DisplayName:  "Jeff Skilling",
		Emails:       []string{"jeff.skilling@enron.com"},
		PrimaryEmail: "jeff.skilling@enron.com",
		Internal:     true,
	}
	resolver.LoadActor(actor1)

	// Load same actor with additional alias - should merge
	actor2 := &entity.Actor{
		ID:           entity.ActorID("jeff.skilling"),
		DisplayName:  "Jeff Skilling",
		Emails:       []string{"jskilli@enron.com"},
		PrimaryEmail: "jskilli@enron.com",
		Internal:     true,
	}
	resolver.LoadActor(actor2)

	stats := resolver.Stats()
	if stats.TotalActors != 1 {
		t.Errorf("Should still have 1 actor after merge: got %d", stats.TotalActors)
	}
	if stats.TotalAliases != 2 {
		t.Errorf("Should have 2 aliases after merge: got %d", stats.TotalAliases)
	}

	// Both emails should resolve to the same actor
	id1, _ := resolver.Resolve("jeff.skilling@enron.com")
	id2, _ := resolver.Resolve("jskilli@enron.com")
	if id1 != id2 {
		t.Errorf("Both emails should resolve to same actor: %s != %s", id1, id2)
	}
}

func TestSCIMResolverResolve(t *testing.T) {
	config := DefaultConfig()
	resolver := NewSCIMResolver(config)

	actor := &entity.Actor{
		ID:           entity.ActorID("alice"),
		Emails:       []string{"alice@example.com"},
		PrimaryEmail: "alice@example.com",
	}
	resolver.LoadActor(actor)

	// Test successful resolution
	id, err := resolver.Resolve("alice@example.com")
	if err != nil {
		t.Errorf("Resolve should succeed: %v", err)
	}
	if id != "alice" {
		t.Errorf("Should resolve to alice: got %s", id)
	}

	// Test case insensitivity
	id, err = resolver.Resolve("Alice@Example.COM")
	if err != nil {
		t.Errorf("Resolve should be case insensitive: %v", err)
	}
	if id != "alice" {
		t.Errorf("Should resolve to alice: got %s", id)
	}

	// Test unknown actor
	_, err = resolver.Resolve("unknown@example.com")
	if err != ErrUnknownActor {
		t.Errorf("Should return ErrUnknownActor for unknown email: got %v", err)
	}
}

func TestSCIMResolverResolveOrCreate(t *testing.T) {
	config := Config{
		InternalDomains: []string{"example.com"},
		AutoCreate:      true,
	}
	resolver := NewSCIMResolver(config)

	// Create new actor
	id1 := resolver.ResolveOrCreate("new@example.com")
	if id1 == "" {
		t.Error("ResolveOrCreate should return non-empty ID")
	}

	// Same email should return same actor
	id2 := resolver.ResolveOrCreate("new@example.com")
	if id1 != id2 {
		t.Errorf("Same email should return same actor: %s != %s", id1, id2)
	}

	// Different email should create new actor
	id3 := resolver.ResolveOrCreate("other@example.com")
	if id3 == id1 {
		t.Error("Different email should create different actor")
	}

	stats := resolver.Stats()
	if stats.TotalActors != 2 {
		t.Errorf("Should have 2 actors: got %d", stats.TotalActors)
	}
	if stats.AutoCreated != 2 {
		t.Errorf("Should have 2 auto-created actors: got %d", stats.AutoCreated)
	}
}

func TestSCIMResolverResolveOrCreateFromHeader(t *testing.T) {
	config := DefaultConfig()
	resolver := NewSCIMResolver(config)

	// Test with "Name <email>" format
	id := resolver.ResolveOrCreate("John Doe <john@example.com>")
	if id == "" {
		t.Error("ResolveOrCreate should handle header format")
	}

	// Should be able to resolve just the email
	id2 := resolver.ResolveOrCreate("john@example.com")
	if id != id2 {
		t.Errorf("Should resolve to same actor: %s != %s", id, id2)
	}
}

func TestSCIMResolverIsInternal(t *testing.T) {
	config := Config{
		InternalDomains: []string{"enron.com", "ei.enron.com"},
		AutoCreate:      true,
	}
	resolver := NewSCIMResolver(config)

	// Internal domain
	if !resolver.IsInternal("jeff@enron.com") {
		t.Error("enron.com should be internal")
	}
	if !resolver.IsInternal("jeff@ei.enron.com") {
		t.Error("ei.enron.com should be internal")
	}

	// External domain
	if resolver.IsInternal("john@gmail.com") {
		t.Error("gmail.com should be external")
	}

	// Case insensitivity
	if !resolver.IsInternal("jeff@ENRON.COM") {
		t.Error("IsInternal should be case insensitive")
	}
}

func TestSCIMResolverGetActor(t *testing.T) {
	config := DefaultConfig()
	resolver := NewSCIMResolver(config)

	actor := &entity.Actor{
		ID:           entity.ActorID("alice"),
		DisplayName:  "Alice Smith",
		Emails:       []string{"alice@example.com"},
		PrimaryEmail: "alice@example.com",
	}
	resolver.LoadActor(actor)

	// Get existing actor
	retrieved, err := resolver.GetActor("alice")
	if err != nil {
		t.Errorf("GetActor should succeed: %v", err)
	}
	if retrieved.DisplayName != "Alice Smith" {
		t.Errorf("DisplayName mismatch: got %s", retrieved.DisplayName)
	}

	// Get unknown actor
	_, err = resolver.GetActor("unknown")
	if err != ErrUnknownActor {
		t.Errorf("Should return ErrUnknownActor: got %v", err)
	}
}

func TestSCIMResolverAliases(t *testing.T) {
	config := DefaultConfig()
	resolver := NewSCIMResolver(config)

	actor := &entity.Actor{
		ID:           entity.ActorID("alice"),
		Emails:       []string{"alice@example.com", "alice.smith@example.com"},
		PrimaryEmail: "alice@example.com",
	}
	resolver.LoadActor(actor)

	aliases := resolver.Aliases("alice")
	if len(aliases) != 2 {
		t.Errorf("Should have 2 aliases: got %d", len(aliases))
	}

	// Unknown actor
	aliases = resolver.Aliases("unknown")
	if len(aliases) != 0 {
		t.Errorf("Unknown actor should have no aliases: got %d", len(aliases))
	}
}

func TestSCIMResolverStats(t *testing.T) {
	config := Config{
		InternalDomains: []string{"example.com"},
		AutoCreate:      true,
	}
	resolver := NewSCIMResolver(config)

	// Load one internal actor
	resolver.LoadActor(&entity.Actor{
		ID:       entity.ActorID("internal"),
		Emails:   []string{"internal@example.com"},
		Internal: true,
	})

	// Load one external actor
	resolver.LoadActor(&entity.Actor{
		ID:       entity.ActorID("external"),
		Emails:   []string{"external@gmail.com"},
		Internal: false,
	})

	// Auto-create some actors
	resolver.ResolveOrCreate("auto1@example.com")
	resolver.ResolveOrCreate("auto2@external.com")

	// Resolve existing
	_, _ = resolver.Resolve("internal@example.com")
	_, _ = resolver.Resolve("internal@example.com")
	_, _ = resolver.Resolve("unknown@nowhere.com")

	stats := resolver.Stats()

	if stats.TotalActors != 4 {
		t.Errorf("TotalActors: got %d, want 4", stats.TotalActors)
	}
	if stats.InternalActors != 2 {
		t.Errorf("InternalActors: got %d, want 2", stats.InternalActors)
	}
	if stats.ExternalActors != 2 {
		t.Errorf("ExternalActors: got %d, want 2", stats.ExternalActors)
	}
	if stats.AutoCreated != 2 {
		t.Errorf("AutoCreated: got %d, want 2", stats.AutoCreated)
	}
	if stats.ResolutionHits < 2 {
		t.Errorf("ResolutionHits: got %d, want >= 2", stats.ResolutionHits)
	}
	if stats.ResolutionMisses < 1 {
		t.Errorf("ResolutionMisses: got %d, want >= 1", stats.ResolutionMisses)
	}
}

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"John Doe <john@example.com>", "John Doe"},
		{"\"Jane Smith\" <jane@example.com>", "Jane Smith"},
		{"john.doe@example.com", "John Doe"},
		{"john_doe@example.com", "John Doe"},
		{"simple@example.com", "Simple"},
	}

	for _, tt := range tests {
		result := extractDisplayName(tt.input)
		if result != tt.expected {
			t.Errorf("extractDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestToTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"john doe", "John Doe"},
		{"JOHN DOE", "John Doe"},
		{"jOHN dOE", "John Doe"},
		{"", ""},
		{"single", "Single"},
	}

	for _, tt := range tests {
		result := toTitleCase(tt.input)
		if result != tt.expected {
			t.Errorf("toTitleCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
