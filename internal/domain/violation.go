package domain

import "fmt"

// Violation represents an architectural rule violation
type Violation struct {
	ID          string `json:"id" yaml:"id"`
	RuleID      string `json:"rule_id" yaml:"rule_id"`
	File        string `json:"file" yaml:"file"`
	Line        int    `json:"line" yaml:"line"`
	SourceLayer string `json:"source_layer" yaml:"source_layer"`
	TargetLayer string `json:"target_layer" yaml:"target_layer"`
	Import      string `json:"import" yaml:"import"`
	Message     string `json:"message" yaml:"message"`
}

// String returns a human-readable representation of the violation for terminal output
func (v *Violation) String() string {
	return fmt.Sprintf("[%s] %s:%d: %s -> %s (%s)",
		v.ID,
		v.File,
		v.Line,
		v.SourceLayer,
		v.TargetLayer,
		v.Message,
	)
}
