package identity

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/grokify/commgraph/entity"
)

// SCIMResolver resolves identities using SCIM-format user data.
type SCIMResolver struct {
	mu              sync.RWMutex
	config          Config
	actors          map[entity.ActorID]*entity.Actor
	emailToActor    map[string]entity.ActorID // lowercase email -> actor ID
	internalDomains map[string]bool
	stats           Stats
	nextID          int64
}

// NewSCIMResolver creates a new SCIM-based identity resolver.
func NewSCIMResolver(config Config) *SCIMResolver {
	internalDomains := make(map[string]bool)
	for _, domain := range config.InternalDomains {
		internalDomains[strings.ToLower(domain)] = true
	}

	return &SCIMResolver{
		config:          config,
		actors:          make(map[entity.ActorID]*entity.Actor),
		emailToActor:    make(map[string]entity.ActorID),
		internalDomains: internalDomains,
	}
}

// LoadActor adds an actor to the resolver.
// If an actor with the same ID already exists, emails are merged
// but existing display name and metadata are preserved.
func (r *SCIMResolver) LoadActor(actor *entity.Actor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existingActor, exists := r.actors[actor.ID]
	if exists {
		// Merge emails into existing actor
		existingEmails := make(map[string]bool)
		for _, email := range existingActor.Emails {
			existingEmails[strings.ToLower(email)] = true
		}

		newAliases := 0
		for _, email := range actor.Emails {
			emailLower := strings.ToLower(email)
			if !existingEmails[emailLower] {
				existingActor.Emails = append(existingActor.Emails, emailLower)
				r.emailToActor[emailLower] = actor.ID
				existingEmails[emailLower] = true
				newAliases++
			}
		}
		r.stats.TotalAliases += newAliases
		return
	}

	// New actor
	r.actors[actor.ID] = actor

	// Index all emails
	for _, email := range actor.Emails {
		r.emailToActor[strings.ToLower(email)] = actor.ID
	}
	if actor.PrimaryEmail != "" {
		r.emailToActor[strings.ToLower(actor.PrimaryEmail)] = actor.ID
	}

	// Update stats
	r.stats.TotalActors++
	r.stats.TotalAliases += len(actor.Emails)
	if actor.Internal {
		r.stats.InternalActors++
	} else {
		r.stats.ExternalActors++
	}
}

// LoadActors adds multiple actors to the resolver.
func (r *SCIMResolver) LoadActors(actors []*entity.Actor) {
	for _, actor := range actors {
		r.LoadActor(actor)
	}
}

// Resolve returns the canonical ActorID for an address.
func (r *SCIMResolver) Resolve(addr string) (entity.ActorID, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalized := r.normalizeEmail(addr)
	if id, ok := r.emailToActor[normalized]; ok {
		atomic.AddInt64(&r.stats.ResolutionHits, 1)
		return id, nil
	}

	atomic.AddInt64(&r.stats.ResolutionMisses, 1)
	return "", ErrUnknownActor
}

// ResolveOrCreate resolves an address or creates a new actor.
func (r *SCIMResolver) ResolveOrCreate(addr string) entity.ActorID {
	normalized := r.normalizeEmail(addr)

	// Try read-only first
	r.mu.RLock()
	if id, ok := r.emailToActor[normalized]; ok {
		r.mu.RUnlock()
		atomic.AddInt64(&r.stats.ResolutionHits, 1)
		return id
	}
	r.mu.RUnlock()

	// Need to create - acquire write lock
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if id, ok := r.emailToActor[normalized]; ok {
		atomic.AddInt64(&r.stats.ResolutionHits, 1)
		return id
	}

	// Create new actor
	id := r.generateID()
	internal := r.isInternalEmail(normalized)

	actor := &entity.Actor{
		ID:           id,
		DisplayName:  extractDisplayName(addr),
		Emails:       []string{normalized},
		PrimaryEmail: normalized,
		Internal:     internal,
	}

	r.actors[id] = actor
	r.emailToActor[normalized] = id
	r.stats.TotalActors++
	r.stats.TotalAliases++
	atomic.AddInt64(&r.stats.AutoCreated, 1)

	if internal {
		r.stats.InternalActors++
	} else {
		r.stats.ExternalActors++
	}

	return id
}

// IsInternal returns true if the address belongs to the organization.
func (r *SCIMResolver) IsInternal(addr string) bool {
	normalized := r.normalizeEmail(addr)

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if we have this actor
	if id, ok := r.emailToActor[normalized]; ok {
		if actor, ok := r.actors[id]; ok {
			return actor.Internal
		}
	}

	// Fall back to domain check
	return r.isInternalEmail(normalized)
}

// GetActor returns full actor details by ID.
func (r *SCIMResolver) GetActor(id entity.ActorID) (*entity.Actor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actor, ok := r.actors[id]
	if !ok {
		return nil, ErrUnknownActor
	}
	return actor, nil
}

// Aliases returns all known addresses for an actor.
func (r *SCIMResolver) Aliases(id entity.ActorID) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	actor, ok := r.actors[id]
	if !ok {
		return nil
	}
	return actor.Emails
}

// Stats returns resolution statistics.
func (r *SCIMResolver) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return Stats{
		TotalActors:      r.stats.TotalActors,
		InternalActors:   r.stats.InternalActors,
		ExternalActors:   r.stats.ExternalActors,
		TotalAliases:     r.stats.TotalAliases,
		ResolutionHits:   atomic.LoadInt64(&r.stats.ResolutionHits),
		ResolutionMisses: atomic.LoadInt64(&r.stats.ResolutionMisses),
		AutoCreated:      atomic.LoadInt64(&r.stats.AutoCreated),
	}
}

// AllActors returns all actors in the resolver.
func (r *SCIMResolver) AllActors() []*entity.Actor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entity.Actor, 0, len(r.actors))
	for _, actor := range r.actors {
		result = append(result, actor)
	}
	return result
}

// normalizeEmail normalizes an email address.
func (r *SCIMResolver) normalizeEmail(addr string) string {
	// Extract email from "Name <email>" format
	if idx := strings.Index(addr, "<"); idx >= 0 {
		if end := strings.Index(addr[idx:], ">"); end > 0 {
			addr = addr[idx+1 : idx+end]
		}
	}
	return strings.ToLower(strings.TrimSpace(addr))
}

// isInternalEmail checks if an email belongs to an internal domain.
func (r *SCIMResolver) isInternalEmail(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])
	return r.internalDomains[domain]
}

// generateID generates a unique actor ID.
func (r *SCIMResolver) generateID() entity.ActorID {
	id := atomic.AddInt64(&r.nextID, 1)
	return entity.ActorID(fmt.Sprintf("auto_%d", id))
}

// extractDisplayName extracts a display name from an email header.
func extractDisplayName(addr string) string {
	// Try to extract name from "Name <email>" format
	if idx := strings.Index(addr, "<"); idx > 0 {
		name := strings.TrimSpace(addr[:idx])
		name = strings.Trim(name, "\"'")
		if name != "" {
			return name
		}
	}
	// Fall back to local part of email
	addr = strings.TrimSpace(addr)
	if idx := strings.Index(addr, "@"); idx > 0 {
		local := addr[:idx]
		// Convert dots/underscores to spaces and title case
		local = strings.ReplaceAll(local, ".", " ")
		local = strings.ReplaceAll(local, "_", " ")
		return toTitleCase(local)
	}
	return addr
}

// toTitleCase converts a string to title case.
func toTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// Verify interface compliance.
var _ Resolver = (*SCIMResolver)(nil)
