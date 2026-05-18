package analysis

import (
	"context"
	"sort"
	"strings"

	"github.com/grokify/commgraph/entity"
	"github.com/grokify/commgraph/storage"
)

// ExternalContact represents an external entity and communication metrics.
type ExternalContact struct {
	ActorID        entity.ActorID `json:"actor_id"`
	DisplayName    string         `json:"display_name,omitempty"`
	Domain         string         `json:"domain"`
	InboundCount   int            `json:"inbound_count"`  // Messages received from external
	OutboundCount  int            `json:"outbound_count"` // Messages sent to external
	TotalCount     int            `json:"total_count"`
	UniqueInternal int            `json:"unique_internal"` // Internal actors who communicated
}

// ExternalContactResults is a slice of external contact results.
type ExternalContactResults []ExternalContact

// Sort sorts by total count descending.
func (r ExternalContactResults) Sort() {
	sort.Slice(r, func(i, j int) bool {
		return r[i].TotalCount > r[j].TotalCount
	})
}

// Top returns the top N external contacts.
func (r ExternalContactResults) Top(n int) ExternalContactResults {
	if n >= len(r) {
		return r
	}
	return r[:n]
}

// DomainStats represents communication statistics for an external domain.
type DomainStats struct {
	Domain         string   `json:"domain"`
	TotalContacts  int      `json:"total_contacts"` // Unique external actors from this domain
	InboundCount   int      `json:"inbound_count"`  // Messages received from domain
	OutboundCount  int      `json:"outbound_count"` // Messages sent to domain
	TotalCount     int      `json:"total_count"`
	UniqueInternal int      `json:"unique_internal"` // Internal actors who communicated
	TopContacts    []string `json:"top_contacts,omitempty"`
}

// DomainStatsResults is a slice of domain statistics.
type DomainStatsResults []DomainStats

// Sort sorts by total count descending.
func (r DomainStatsResults) Sort() {
	sort.Slice(r, func(i, j int) bool {
		return r[i].TotalCount > r[j].TotalCount
	})
}

// InternalExternalRatio represents an internal actor's external communication ratio.
type InternalExternalRatio struct {
	ActorID       entity.ActorID `json:"actor_id"`
	DisplayName   string         `json:"display_name,omitempty"`
	InternalCount int            `json:"internal_count"`
	ExternalCount int            `json:"external_count"`
	TotalCount    int            `json:"total_count"`
	ExternalRatio float64        `json:"external_ratio"` // ExternalCount / TotalCount
	UniqueDomains int            `json:"unique_domains"` // Number of external domains
	TopDomains    []string       `json:"top_domains,omitempty"`
	OutboundHeavy bool           `json:"outbound_heavy"` // More outbound than inbound external
}

// ExternalAnalysisResults contains comprehensive external entity analysis.
type ExternalAnalysisResults struct {
	Summary           ExternalSummary         `json:"summary"`
	TopExternalActors ExternalContactResults  `json:"top_external_actors"`
	TopDomains        DomainStatsResults      `json:"top_domains"`
	InternalRatios    []InternalExternalRatio `json:"internal_ratios"`
	BoundarySpanners  []InternalExternalRatio `json:"boundary_spanners"` // High external ratio
}

// ExternalSummary contains summary statistics for external analysis.
type ExternalSummary struct {
	TotalExternalActors       int     `json:"total_external_actors"`
	TotalExternalDomains      int     `json:"total_external_domains"`
	TotalExternalInteractions int     `json:"total_external_interactions"`
	InboundCount              int     `json:"inbound_count"`
	OutboundCount             int     `json:"outbound_count"`
	ExternalRatio             float64 `json:"external_ratio"` // External / Total interactions
}

// ExternalAnalysis performs comprehensive analysis of external entity communication.
func (a *Analyzer) ExternalAnalysis(ctx context.Context) (*ExternalAnalysisResults, error) {
	interactions, err := a.store.GetInteractions(ctx, storage.InteractionQuery{})
	if err != nil {
		return nil, err
	}

	// Track metrics
	externalActors := make(map[entity.ActorID]*ExternalContact)
	domainStats := make(map[string]*DomainStats)
	internalMetrics := make(map[entity.ActorID]*InternalExternalRatio)

	// Track which internal actors communicate with each external
	externalToInternal := make(map[entity.ActorID]map[entity.ActorID]bool)
	// Track which internal actors communicate with each domain
	domainToInternal := make(map[string]map[entity.ActorID]bool)
	// Track which domains each internal actor communicates with
	internalToDomains := make(map[entity.ActorID]map[string]int)

	totalInteractions := 0
	externalInteractions := 0
	inboundExternal := 0
	outboundExternal := 0

	for _, interaction := range interactions {
		totalInteractions++

		fromActor, _ := a.resolver.GetActor(interaction.From)
		toActor, _ := a.resolver.GetActor(interaction.To)

		fromInternal := fromActor != nil && fromActor.Internal
		toInternal := toActor != nil && toActor.Internal

		// Initialize internal metrics
		if fromInternal {
			if internalMetrics[interaction.From] == nil {
				internalMetrics[interaction.From] = &InternalExternalRatio{
					ActorID: interaction.From,
				}
				if fromActor != nil {
					internalMetrics[interaction.From].DisplayName = fromActor.DisplayName
				}
			}
			if internalToDomains[interaction.From] == nil {
				internalToDomains[interaction.From] = make(map[string]int)
			}
		}
		if toInternal {
			if internalMetrics[interaction.To] == nil {
				internalMetrics[interaction.To] = &InternalExternalRatio{
					ActorID: interaction.To,
				}
				if toActor != nil {
					internalMetrics[interaction.To].DisplayName = toActor.DisplayName
				}
			}
			if internalToDomains[interaction.To] == nil {
				internalToDomains[interaction.To] = make(map[string]int)
			}
		}

		// Internal to Internal
		if fromInternal && toInternal {
			internalMetrics[interaction.From].InternalCount++
			internalMetrics[interaction.From].TotalCount++
			continue
		}

		externalInteractions++

		// Internal to External (outbound)
		if fromInternal && !toInternal {
			outboundExternal++
			trackExternalInteraction(
				interaction.From, interaction.To, toActor,
				true, // isOutbound
				internalMetrics, internalToDomains,
				externalActors, externalToInternal,
				domainStats, domainToInternal,
			)
		}

		// External to Internal (inbound)
		if !fromInternal && toInternal {
			inboundExternal++
			trackExternalInteraction(
				interaction.To, interaction.From, fromActor,
				false, // isOutbound
				internalMetrics, internalToDomains,
				externalActors, externalToInternal,
				domainStats, domainToInternal,
			)
		}
	}

	// Finalize external actor metrics
	externalResults := make(ExternalContactResults, 0, len(externalActors))
	for _, ext := range externalActors {
		ext.UniqueInternal = len(externalToInternal[ext.ActorID])
		externalResults = append(externalResults, *ext)
	}
	externalResults.Sort()

	// Finalize domain stats
	domainResults := make(DomainStatsResults, 0, len(domainStats))
	for domain, stats := range domainStats {
		stats.UniqueInternal = len(domainToInternal[domain])
		// Count unique external actors from this domain
		for _, ext := range externalActors {
			if ext.Domain == domain {
				stats.TotalContacts++
			}
		}
		domainResults = append(domainResults, *stats)
	}
	domainResults.Sort()

	// Finalize internal ratios
	internalResults := make([]InternalExternalRatio, 0, len(internalMetrics))
	var boundarySpanners []InternalExternalRatio
	for actorID, metrics := range internalMetrics {
		if metrics.TotalCount > 0 {
			metrics.ExternalRatio = float64(metrics.ExternalCount) / float64(metrics.TotalCount)
		}
		metrics.UniqueDomains = len(internalToDomains[actorID])

		// Get top domains for this actor
		type domainCount struct {
			domain string
			count  int
		}
		var domains []domainCount
		for d, c := range internalToDomains[actorID] {
			domains = append(domains, domainCount{d, c})
		}
		sort.Slice(domains, func(i, j int) bool {
			return domains[i].count > domains[j].count
		})
		for i, d := range domains {
			if i >= 3 {
				break
			}
			metrics.TopDomains = append(metrics.TopDomains, d.domain)
		}

		internalResults = append(internalResults, *metrics)

		// Identify boundary spanners (high external ratio, multiple domains)
		if metrics.ExternalRatio > 0.3 && metrics.UniqueDomains >= 3 && metrics.ExternalCount >= 10 {
			boundarySpanners = append(boundarySpanners, *metrics)
		}
	}

	// Sort internal results by external ratio
	sort.Slice(internalResults, func(i, j int) bool {
		return internalResults[i].ExternalRatio > internalResults[j].ExternalRatio
	})
	sort.Slice(boundarySpanners, func(i, j int) bool {
		return boundarySpanners[i].ExternalRatio > boundarySpanners[j].ExternalRatio
	})

	// Build summary
	externalRatio := 0.0
	if totalInteractions > 0 {
		externalRatio = float64(externalInteractions) / float64(totalInteractions)
	}

	summary := ExternalSummary{
		TotalExternalActors:       len(externalActors),
		TotalExternalDomains:      len(domainStats),
		TotalExternalInteractions: externalInteractions,
		InboundCount:              inboundExternal,
		OutboundCount:             outboundExternal,
		ExternalRatio:             externalRatio,
	}

	return &ExternalAnalysisResults{
		Summary:           summary,
		TopExternalActors: externalResults,
		TopDomains:        domainResults,
		InternalRatios:    internalResults,
		BoundarySpanners:  boundarySpanners,
	}, nil
}

// TopExternalContacts returns the top N external contacts by interaction volume.
func (a *Analyzer) TopExternalContacts(ctx context.Context, n int) (ExternalContactResults, error) {
	results, err := a.ExternalAnalysis(ctx)
	if err != nil {
		return nil, err
	}
	return results.TopExternalActors.Top(n), nil
}

// TopExternalDomains returns the top N external domains by interaction volume.
func (a *Analyzer) TopExternalDomains(ctx context.Context, n int) (DomainStatsResults, error) {
	results, err := a.ExternalAnalysis(ctx)
	if err != nil {
		return nil, err
	}
	if n >= len(results.TopDomains) {
		return results.TopDomains, nil
	}
	return results.TopDomains[:n], nil
}

// trackExternalInteraction tracks metrics for an external interaction.
// internalActor is the internal participant, externalActor is the external participant.
// isOutbound indicates direction: true = internal->external, false = external->internal.
func trackExternalInteraction(
	internalActorID, externalActorID entity.ActorID,
	externalActor *entity.Actor,
	isOutbound bool,
	internalMetrics map[entity.ActorID]*InternalExternalRatio,
	internalToDomains map[entity.ActorID]map[string]int,
	externalActors map[entity.ActorID]*ExternalContact,
	externalToInternal map[entity.ActorID]map[entity.ActorID]bool,
	domainStats map[string]*DomainStats,
	domainToInternal map[string]map[entity.ActorID]bool,
) {
	internalMetrics[internalActorID].ExternalCount++
	internalMetrics[internalActorID].TotalCount++

	domain := extractDomain(externalActor)
	internalToDomains[internalActorID][domain]++

	// Track external actor
	if externalActors[externalActorID] == nil {
		externalActors[externalActorID] = &ExternalContact{
			ActorID: externalActorID,
			Domain:  domain,
		}
		if externalActor != nil {
			externalActors[externalActorID].DisplayName = externalActor.DisplayName
		}
	}
	if isOutbound {
		externalActors[externalActorID].OutboundCount++
	} else {
		externalActors[externalActorID].InboundCount++
	}
	externalActors[externalActorID].TotalCount++

	// Track internal-external relationship
	if externalToInternal[externalActorID] == nil {
		externalToInternal[externalActorID] = make(map[entity.ActorID]bool)
	}
	externalToInternal[externalActorID][internalActorID] = true

	// Track domain stats
	if domainStats[domain] == nil {
		domainStats[domain] = &DomainStats{Domain: domain}
	}
	if isOutbound {
		domainStats[domain].OutboundCount++
	} else {
		domainStats[domain].InboundCount++
	}
	domainStats[domain].TotalCount++
	if domainToInternal[domain] == nil {
		domainToInternal[domain] = make(map[entity.ActorID]bool)
	}
	domainToInternal[domain][internalActorID] = true
}

// extractDomain extracts the domain from an actor's email.
func extractDomain(actor *entity.Actor) string {
	if actor == nil {
		return "unknown"
	}
	email := actor.PrimaryEmail
	if email == "" && len(actor.Emails) > 0 {
		email = actor.Emails[0]
	}
	if email == "" {
		return "unknown"
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "unknown"
	}
	return strings.ToLower(parts[1])
}
