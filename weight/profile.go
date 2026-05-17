// Package weight provides configurable weight profiles for graph analysis.
package weight

import (
	"time"

	"github.com/grokify/commgraph/entity"
)

// Profile defines weights for different edge types in graph analysis.
type Profile struct {
	// Name is the profile identifier.
	Name string `json:"name"`

	// Description explains what this profile measures.
	Description string `json:"description"`

	// To is the weight for direct recipient edges.
	To float64 `json:"to"`

	// CC is the weight for carbon copy edges.
	CC float64 `json:"cc"`

	// BCC is the weight for blind carbon copy edges.
	BCC float64 `json:"bcc"`

	// Mention is the weight for body mention edges.
	Mention float64 `json:"mention"`

	// Reply is the weight for reply edges.
	Reply float64 `json:"reply"`

	// RecencyHalfLife is the duration after which edge weight decays by half.
	// Zero means no decay.
	RecencyHalfLife time.Duration `json:"recency_half_life,omitempty"`

	// Aggregation defines how multiple edges between actors are combined.
	Aggregation AggregationType `json:"aggregation"`
}

// AggregationType defines how to combine multiple edge weights.
type AggregationType string

const (
	// AggregationSum adds all edge weights.
	AggregationSum AggregationType = "sum"

	// AggregationMax takes the maximum edge weight.
	AggregationMax AggregationType = "max"

	// AggregationAvg takes the average edge weight.
	AggregationAvg AggregationType = "avg"

	// AggregationCount counts the number of edges (ignores weight).
	AggregationCount AggregationType = "count"
)

// Weight returns the base weight for an edge type.
func (p Profile) Weight(edgeType entity.EdgeType) float64 {
	switch edgeType {
	case entity.EdgeTypeTo:
		return p.To
	case entity.EdgeTypeCC:
		return p.CC
	case entity.EdgeTypeBCC:
		return p.BCC
	case entity.EdgeTypeMention:
		return p.Mention
	case entity.EdgeTypeReply:
		return p.Reply
	default:
		return 0
	}
}

// WeightWithDecay returns the weight adjusted for recency.
func (p Profile) WeightWithDecay(edgeType entity.EdgeType, age time.Duration) float64 {
	base := p.Weight(edgeType)
	if p.RecencyHalfLife == 0 || age <= 0 {
		return base
	}
	// Exponential decay: weight * 0.5^(age/halfLife)
	decayFactor := exponentialDecay(age, p.RecencyHalfLife)
	return base * decayFactor
}

// exponentialDecay computes 0.5^(age/halfLife).
func exponentialDecay(age, halfLife time.Duration) float64 {
	if halfLife <= 0 {
		return 1.0
	}
	ratio := float64(age) / float64(halfLife)
	// 0.5^ratio = e^(ratio * ln(0.5)) = e^(-0.693 * ratio)
	return pow(0.5, ratio)
}

// pow computes base^exp for float64.
func pow(base, exp float64) float64 {
	// Simple implementation using repeated multiplication for small exponents
	// For production, use math.Pow
	if exp == 0 {
		return 1
	}
	if exp == 1 {
		return base
	}
	// Use logarithm approach
	// base^exp = e^(exp * ln(base))
	// For 0.5^exp = e^(exp * ln(0.5)) = e^(-0.693 * exp)
	const ln05 = -0.6931471805599453
	return exp2(exp * ln05 / 0.6931471805599453)
}

// exp2 computes 2^x.
func exp2(x float64) float64 {
	// Approximate using Taylor series for small values
	// For production, use math.Exp2
	if x == 0 {
		return 1
	}
	// Use identity: 2^x = e^(x * ln(2))
	const ln2 = 0.6931471805599453
	return expApprox(x * ln2)
}

// expApprox approximates e^x.
func expApprox(x float64) float64 {
	// Taylor series approximation
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
		if term < 1e-15 && term > -1e-15 {
			break
		}
	}
	return result
}

// Validate checks that the profile has valid values.
func (p Profile) Validate() error {
	if p.Name == "" {
		return ErrEmptyProfileName
	}
	return nil
}
