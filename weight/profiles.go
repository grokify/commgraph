package weight

import "errors"

// ErrEmptyProfileName is returned when a profile has no name.
var ErrEmptyProfileName = errors.New("profile name cannot be empty")

// ErrProfileNotFound is returned when a requested profile does not exist.
var ErrProfileNotFound = errors.New("profile not found")

// Built-in weight profiles.
var (
	// Influence measures power and authority in communication patterns.
	// High weight on direct communication, low on passive CC.
	Influence = Profile{
		Name:        "influence",
		Description: "Measures power and authority in communication patterns",
		To:          1.0,
		CC:          0.2,
		BCC:         0.4,
		Mention:     0.1,
		Reply:       0.3,
		Aggregation: AggregationSum,
	}

	// InformationFlow measures how information propagates through the organization.
	// High weight on all recipient types since they all receive the information.
	InformationFlow = Profile{
		Name:        "information_flow",
		Description: "Measures how information propagates through the organization",
		To:          1.0,
		CC:          0.8,
		BCC:         0.9,
		Mention:     0.5,
		Reply:       0.1,
		Aggregation: AggregationSum,
	}

	// Coordination measures collaborative activity and project coordination.
	// High weight on CC (alignment) and Reply (active collaboration).
	Coordination = Profile{
		Name:        "coordination",
		Description: "Measures collaborative activity and project coordination",
		To:          0.5,
		CC:          0.8,
		BCC:         0.3,
		Mention:     0.2,
		Reply:       1.0,
		Aggregation: AggregationSum,
	}
)

// Registry holds named profiles.
type Registry struct {
	profiles map[string]Profile
}

// NewRegistry creates a new profile registry with built-in profiles.
func NewRegistry() *Registry {
	r := &Registry{
		profiles: make(map[string]Profile),
	}
	r.Register(Influence)
	r.Register(InformationFlow)
	r.Register(Coordination)
	return r
}

// Register adds a profile to the registry.
func (r *Registry) Register(p Profile) {
	r.profiles[p.Name] = p
}

// Get retrieves a profile by name.
func (r *Registry) Get(name string) (Profile, error) {
	p, ok := r.profiles[name]
	if !ok {
		return Profile{}, ErrProfileNotFound
	}
	return p, nil
}

// List returns all registered profile names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.profiles))
	for name := range r.profiles {
		names = append(names, name)
	}
	return names
}

// DefaultProfile returns the default analysis profile.
func DefaultProfile() Profile {
	return Influence
}
