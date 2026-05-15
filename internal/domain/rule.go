package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// RuleType defines the type of architectural rule
type RuleType string

const (
	// Cannot: Source layer cannot depend on target layer
	RuleTypeCannot RuleType = "Cannot"
	// Must: Source layer must depend on target layer (enforced dependency)
	RuleTypeMust RuleType = "Must"
	// Can: Source layer can depend on target layer (informational, no violation)
	RuleTypeCan RuleType = "Can"
	// MustNotCircular: Prevents circular dependencies between layers
	RuleTypeMustNotCircular RuleType = "MustNotCircular"
)

// UnmarshalYAML implements yaml.Unmarshaler for RuleType (case-insensitive)
func (rt *RuleType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	// Normalize to title case for comparison
	switch strings.Title(strings.ToLower(s)) {
	case "Cannot":
		*rt = RuleTypeCannot
	case "Must":
		*rt = RuleTypeMust
	case "Can":
		*rt = RuleTypeCan
	case "MustNotCircular":
		*rt = RuleTypeMustNotCircular
	default:
		*rt = RuleType(s) // Allow custom types
	}
	return nil
}

// Severity defines the severity level of a rule violation
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// UnmarshalYAML implements yaml.Unmarshaler for Severity (case-insensitive)
func (s *Severity) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var sev string
	if err := unmarshal(&sev); err != nil {
		return err
	}
	// Normalize to lowercase
	switch strings.ToLower(sev) {
	case "error":
		*s = SeverityError
	case "warning":
		*s = SeverityWarning
	case "info":
		*s = SeverityInfo
	default:
		*s = Severity(sev) // Allow custom severities
	}
	return nil
}

// Rule represents an architectural rule between layers
type Rule struct {
	ID              string          `json:"id" yaml:"id"`
	From            string          `json:"from" yaml:"from"`
	To              []string        `json:"to" yaml:"to"`
	Type            RuleType        `json:"type" yaml:"type"`
	Severity        Severity        `json:"severity" yaml:"severity"`
	Explanation     string          `json:"explanation,omitempty" yaml:"explanation,omitempty"`
	Pattern         string          `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	compiledPattern *regexp.Regexp  `json:"-" yaml:"-"`
}

// CompilePattern compiles the Pattern field into a cached *regexp.Regexp.
// Returns nil if Pattern is empty.
func (r *Rule) CompilePattern() error {
	if r.Pattern == "" {
		r.compiledPattern = nil
		return nil
	}
	re, err := regexp.Compile(r.Pattern)
	if err != nil {
		r.compiledPattern = nil
		return err
	}
	r.compiledPattern = re
	return nil
}

// Validate checks if the rule configuration is valid
func (r *Rule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	// Compile pattern if set
	if err := r.CompilePattern(); err != nil {
		return fmt.Errorf("rule %q: invalid pattern: %w", r.ID, err)
	}

	// Pattern-only rules don't require from/to
	if r.Pattern != "" && r.From == "" && len(r.To) == 0 {
		// Validate type and severity only
		switch r.Type {
		case RuleTypeCannot, RuleTypeMust, RuleTypeCan, RuleTypeMustNotCircular:
			// valid
		default:
			return fmt.Errorf("rule %q: invalid rule type %q", r.ID, r.Type)
		}
		switch r.Severity {
		case SeverityError, SeverityWarning, SeverityInfo, "":
			// valid
		default:
			return fmt.Errorf("rule %q: invalid severity %q", r.ID, r.Severity)
		}
		return nil
	}

	if r.From == "" {
		return fmt.Errorf("rule %q: 'from' field is required", r.ID)
	}
	if len(r.To) == 0 {
		return fmt.Errorf("rule %q: 'to' field must have at least one target", r.ID)
	}
	switch r.Type {
	case RuleTypeCannot, RuleTypeMust, RuleTypeCan, RuleTypeMustNotCircular:
		// Valid rule type
	default:
		return fmt.Errorf("rule %q: invalid rule type %q", r.ID, r.Type)
	}
	switch r.Severity {
	case SeverityError, SeverityWarning, SeverityInfo, "":
		// Valid severity (empty defaults to error)
	default:
		return fmt.Errorf("rule %q: invalid severity %q", r.ID, r.Severity)
	}
	return nil
}

// Violates checks if a dependency from sourceLayer to targetLayer violates this rule
func (r *Rule) Violates(importPath, sourceLayer, targetLayer string) bool {
	// Check pattern if compiled
	if r.compiledPattern != nil {
		if !r.compiledPattern.MatchString(importPath) {
			return false
		}
		// Pattern-only rule (no from/to): pattern match determines violation
		if r.From == "" {
			switch r.Type {
			case RuleTypeCannot, RuleTypeMustNotCircular:
				return true
			default:
				return false
			}
		}
		// Combined rule: pattern matched, proceed to from/to check
	}

	// If no pattern and no from, can't determine
	if r.From == "" {
		return false
	}

	// Check if the rule applies to this source layer
	if r.From != sourceLayer {
		return false
	}

	// Check if target layer is in the rule's target list
	targetMatched := false
	for _, to := range r.To {
		if to == targetLayer {
			targetMatched = true
			break
		}
	}

	if !targetMatched {
		return false
	}

	// Evaluate based on rule type
	switch r.Type {
	case RuleTypeCannot:
		// "Cannot" rules are violated when the dependency exists
		return true
	case RuleTypeMust:
		// "Must" rules are NOT violated when the dependency exists (it's required)
		return false
	case RuleTypeCan:
		// "Can" rules are informational, never violated
		return false
	case RuleTypeMustNotCircular:
		// Circular dependency check - violated if dependency exists
		return true
	default:
		return false
	}
}
