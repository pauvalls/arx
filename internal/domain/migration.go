package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// SchemaVersion represents a config schema version in "major.minor" format.
type SchemaVersion struct {
	Major int
	Minor int
}

// Validate checks that the schema version is valid (Major >= 1, Minor >= 0).
func (v SchemaVersion) Validate() error {
	if v.Major < 1 {
		return fmt.Errorf("schema version major must be >= 1, got %d", v.Major)
	}
	if v.Minor < 0 {
		return fmt.Errorf("schema version minor must be >= 0, got %d", v.Minor)
	}
	return nil
}

// String returns the "major.minor" string representation.
func (v SchemaVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// Compare returns -1 if v < other, 0 if equal, 1 if v > other.
func (v SchemaVersion) Compare(other SchemaVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor < other.Minor {
		return -1
	}
	if v.Minor > other.Minor {
		return 1
	}
	return 0
}

// ParseSchemaVersion parses a "major.minor" or "major" string into SchemaVersion.
// "1" is equivalent to "1.0".
func ParseSchemaVersion(s string) (SchemaVersion, error) {
	if s == "" {
		return SchemaVersion{}, fmt.Errorf("invalid schema version %q: empty string", s)
	}
	parts := strings.SplitN(s, ".", 3)
	if len(parts) > 2 {
		return SchemaVersion{}, fmt.Errorf("invalid schema version %q: expected major.minor format", s)
	}
	if parts[0] == "" {
		return SchemaVersion{}, fmt.Errorf("invalid schema version %q: empty major component", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SchemaVersion{}, fmt.Errorf("invalid schema version %q: major is not an integer", s)
	}
	minor := 0
	if len(parts) == 2 {
		if parts[1] == "" {
			return SchemaVersion{}, fmt.Errorf("invalid schema version %q: empty minor component", s)
		}
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return SchemaVersion{}, fmt.Errorf("invalid schema version %q: minor is not an integer", s)
		}
	}
	v := SchemaVersion{Major: major, Minor: minor}
	if err := v.Validate(); err != nil {
		return SchemaVersion{}, fmt.Errorf("invalid schema version %q: %w", s, err)
	}
	return v, nil
}

// MarshalJSON implements json.Marshaler for SchemaVersion.
func (v SchemaVersion) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

// UnmarshalJSON implements json.Unmarshaler for SchemaVersion.
func (v *SchemaVersion) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("schema version: %w", err)
	}
	parsed, err := ParseSchemaVersion(s)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler (used by YAML v3 for scalar values).
func (v SchemaVersion) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler (used by YAML v3 for scalar values).
func (v *SchemaVersion) UnmarshalText(text []byte) error {
	parsed, err := ParseSchemaVersion(string(text))
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

// MigrationFunc transforms raw YAML bytes from one schema version to the next.
type MigrationFunc func(yamlContent []byte) ([]byte, error)

// Migration represents a single schema version migration step.
type Migration struct {
	From SchemaVersion
	To   SchemaVersion
	Func MigrationFunc
}

// schemaVersionPair is the internal key for the registry map.
type schemaVersionPair struct {
	from, to SchemaVersion
}

// Registry holds registered migrations and can resolve migration chains.
type Registry struct {
	migrations map[schemaVersionPair]MigrationFunc
	outEdges   map[SchemaVersion][]SchemaVersion
}

// NewRegistry creates an empty migration registry.
func NewRegistry() *Registry {
	return &Registry{
		migrations: make(map[schemaVersionPair]MigrationFunc),
		outEdges:   make(map[SchemaVersion][]SchemaVersion),
	}
}

// Register adds a migration to the registry. It returns an error if registering
// would create an ambiguous path (multiple outgoing edges from the same source version).
func (r *Registry) Register(m Migration) error {
	if m.Func == nil {
		return fmt.Errorf("migration function must not be nil")
	}
	if err := m.From.Validate(); err != nil {
		return fmt.Errorf("migration from %s: %w", m.From, err)
	}
	if err := m.To.Validate(); err != nil {
		return fmt.Errorf("migration to %s: %w", m.To, err)
	}

	key := schemaVersionPair{from: m.From, to: m.To}
	if _, exists := r.migrations[key]; exists {
		return fmt.Errorf("migration %s→%s already registered", m.From, m.To)
	}

	// Check for ambiguous path: if from version already has an outgoing edge,
	// registering another would create a branch (two paths from same source).
	if existing, hasEdges := r.outEdges[m.From]; hasEdges {
		for _, existingTo := range existing {
			if existingTo != m.To {
				return fmt.Errorf(
					"ambiguous migration path: %s→%s already registered, cannot also register %s→%s",
					m.From, existingTo, m.From, m.To,
				)
			}
		}
	}

	r.migrations[key] = m.Func
	r.outEdges[m.From] = append(r.outEdges[m.From], m.To)
	return nil
}

// Resolve finds the shortest migration path from `from` to `to` using BFS.
// Returns a slice of MigrationFuncs to apply in order.
func (r *Registry) Resolve(from, to SchemaVersion) ([]MigrationFunc, error) {
	if from == to {
		return nil, nil
	}

	// BFS to find shortest path
	type node struct {
		version SchemaVersion
		path    []schemaVersionPair
	}

	visited := make(map[SchemaVersion]bool)
	queue := []node{{version: from}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.version] {
			continue
		}
		visited[current.version] = true

		// Find all outgoing edges from current version
		for _, nextVer := range r.outEdges[current.version] {
			pair := schemaVersionPair{from: current.version, to: nextVer}
			if _, exists := r.migrations[pair]; !exists {
				continue
			}

			newPath := append(append([]schemaVersionPair{}, current.path...), pair)

			if nextVer == to {
				// Found target — convert path to []MigrationFunc
				funcs := make([]MigrationFunc, len(newPath))
				for i, p := range newPath {
					funcs[i] = r.migrations[p]
				}
				return funcs, nil
			}

			queue = append(queue, node{version: nextVer, path: newPath})
		}
	}

	return nil, fmt.Errorf("no migration path from %s to %s", from, to)
}

// SourceVersion returns the earliest registered source version (minimum from).
func (r *Registry) SourceVersion() SchemaVersion {
	var earliest *SchemaVersion
	for pair := range r.migrations {
		if earliest == nil || pair.from.Compare(*earliest) < 0 {
			cp := pair.from
			earliest = &cp
		}
	}
	if earliest == nil {
		return SchemaVersion{}
	}
	return *earliest
}
