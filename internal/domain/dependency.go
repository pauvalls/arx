package domain

import "fmt"

// Dependency represents a detected dependency between files/layers
type Dependency struct {
	SourceFile    string `json:"source_file" yaml:"source_file"`
	SourceLine    int    `json:"source_line" yaml:"source_line"`
	ImportPath    string `json:"import_path" yaml:"import_path"`
	ResolvedLayer string `json:"resolved_layer,omitempty" yaml:"resolved_layer,omitempty"`
}

// String returns a human-readable representation of the dependency
func (d *Dependency) String() string {
	if d.ResolvedLayer != "" {
		return fmt.Sprintf("%s:%d -> %s (%s)",
			d.SourceFile,
			d.SourceLine,
			d.ImportPath,
			d.ResolvedLayer,
		)
	}
	return fmt.Sprintf("%s:%d -> %s",
		d.SourceFile,
		d.SourceLine,
		d.ImportPath,
	)
}
