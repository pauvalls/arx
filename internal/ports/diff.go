package ports

import (
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
)

// DiffResultData holds the data needed to render architecture diff results.
// This is used by diff renderers to avoid direct dependency on the
// application layer, preventing circular dependencies.
type DiffResultData struct {
	Added         []domain.Violation
	Resolved      []domain.Violation
	Unchanged     []domain.Violation
	RefBefore     string
	RefAfter      string
	ConfigChanged bool
}

// HasChanges returns true if there are added or resolved violations.
func (d DiffResultData) HasChanges() bool {
	return len(d.Added) > 0 || len(d.Resolved) > 0
}

// Summary returns a human-readable summary string.
// Example: "+3 violations, -1 resolved, 12 unchanged"
func (d DiffResultData) Summary() string {
	return fmt.Sprintf("+%d violations, -%d resolved, %d unchanged",
		len(d.Added), len(d.Resolved), len(d.Unchanged))
}
