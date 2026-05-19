package ports

import "github.com/pauvalls/arx/internal/domain"

// DiagramData holds the data needed to render an architecture dependency diagram.
// This is used by output renderers (ASCII, DOT, Mermaid) to avoid direct
// dependency on the application layer, preventing circular dependencies.
type DiagramData struct {
	Layers       []domain.Layer
	Dependencies []domain.Dependency
	Violations   []domain.Violation
}

// DiagramRenderer renders architecture diagrams in various formats.
type DiagramRenderer interface {
	// Render returns a string representation of the diagram.
	Render(data DiagramData) string
}
