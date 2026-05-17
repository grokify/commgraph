package weight

import (
	"testing"
	"time"

	"github.com/grokify/commgraph/entity"
)

func TestProfileWeight(t *testing.T) {
	tests := []struct {
		name     string
		profile  Profile
		edgeType entity.EdgeType
		want     float64
	}{
		{
			name:     "influence TO",
			profile:  Influence,
			edgeType: entity.EdgeTypeTo,
			want:     1.0,
		},
		{
			name:     "influence CC",
			profile:  Influence,
			edgeType: entity.EdgeTypeCC,
			want:     0.2,
		},
		{
			name:     "information_flow CC",
			profile:  InformationFlow,
			edgeType: entity.EdgeTypeCC,
			want:     0.8,
		},
		{
			name:     "coordination Reply",
			profile:  Coordination,
			edgeType: entity.EdgeTypeReply,
			want:     1.0,
		},
		{
			name:     "unknown edge type",
			profile:  Influence,
			edgeType: entity.EdgeType("UNKNOWN"),
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.profile.Weight(tt.edgeType)
			if got != tt.want {
				t.Errorf("Weight() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileWeightWithDecay(t *testing.T) {
	profile := Profile{
		Name:            "test",
		To:              1.0,
		RecencyHalfLife: 24 * time.Hour,
		Aggregation:     AggregationSum,
	}

	tests := []struct {
		name string
		age  time.Duration
		want float64
	}{
		{
			name: "no age",
			age:  0,
			want: 1.0,
		},
		{
			name: "half life",
			age:  24 * time.Hour,
			want: 0.5,
		},
		{
			name: "two half lives",
			age:  48 * time.Hour,
			want: 0.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := profile.WeightWithDecay(entity.EdgeTypeTo, tt.age)
			// Allow 1% tolerance for floating point
			tolerance := tt.want * 0.01
			if got < tt.want-tolerance || got > tt.want+tolerance {
				t.Errorf("WeightWithDecay() = %v, want %v (±%v)", got, tt.want, tolerance)
			}
		})
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()

	// Test built-in profiles
	for _, name := range []string{"influence", "information_flow", "coordination"} {
		p, err := r.Get(name)
		if err != nil {
			t.Errorf("Get(%q) error = %v", name, err)
		}
		if p.Name != name {
			t.Errorf("Get(%q).Name = %q, want %q", name, p.Name, name)
		}
	}

	// Test unknown profile
	_, err := r.Get("unknown")
	if err != ErrProfileNotFound {
		t.Errorf("Get(unknown) error = %v, want ErrProfileNotFound", err)
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()

	custom := Profile{
		Name:        "custom",
		Description: "Custom profile",
		To:          0.9,
		CC:          0.1,
		Aggregation: AggregationMax,
	}

	r.Register(custom)

	got, err := r.Get("custom")
	if err != nil {
		t.Fatalf("Get(custom) error = %v", err)
	}
	if got.To != 0.9 {
		t.Errorf("custom.To = %v, want 0.9", got.To)
	}
}
