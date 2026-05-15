package domain

import (
	"fmt"
	"time"
)

// Baseline stores violation fingerprints that should be suppressed in future checks.
// It is serialized as .arx-baseline.json in the project root.
type Baseline struct {
	Version     string              `json:"version"`
	ConfigHash  string              `json:"config_hash"`
	GeneratedAt time.Time           `json:"generated_at"`
	Violations  []BaselineViolation `json:"violations"`
}

// BaselineViolation stores the fingerprint fields of a violation.
// Only the fields needed for stable fingerprinting are stored (not line number,
// since file reorganizations change paths/lines but the architectural violation remains).
type BaselineViolation struct {
	RuleID      string `json:"rule_id"`
	SourceLayer string `json:"source_layer"`
	TargetLayer string `json:"target_layer"`
	Import      string `json:"import"`
	File        string `json:"file"`
}

// Fingerprint returns a stable identifier for a baseline violation.
// Uses rule_id + source_layer + target_layer + import — NOT file path or line number,
// which change during code reorganization without changing the actual violation.
func (bv BaselineViolation) Fingerprint() string {
	return fmt.Sprintf("%s:%s:%s:%s", bv.RuleID, bv.SourceLayer, bv.TargetLayer, bv.Import)
}

// IsSuppressed checks if a violation's fingerprint matches any baseline entry.
// Returns false on nil receiver (no baseline = nothing suppressed).
func (b *Baseline) IsSuppressed(v Violation) bool {
	if b == nil {
		return false
	}

	fp := BaselineViolation{
		RuleID:      v.RuleID,
		SourceLayer: v.SourceLayer,
		TargetLayer: v.TargetLayer,
		Import:      v.Import,
	}.Fingerprint()

	for _, bv := range b.Violations {
		if bv.Fingerprint() == fp {
			return true
		}
	}
	return false
}

// Filter returns violations NOT present in the baseline (i.e., new ones).
// Returns all violations unchanged when baseline is nil.
func (b *Baseline) Filter(violations []Violation) []Violation {
	if b == nil {
		return violations
	}

	var newViolations []Violation
	for _, v := range violations {
		if !b.IsSuppressed(v) {
			newViolations = append(newViolations, v)
		}
	}
	return newViolations
}

// IsStale returns true if the current config hash differs from the baseline's config hash.
func (b *Baseline) IsStale(configHash string) bool {
	if b == nil {
		return true
	}
	return b.ConfigHash != configHash
}

// GenerateBaseline creates a baseline from current violations and config hash.
func GenerateBaseline(violations []Violation, configHash string) *Baseline {
	b := &Baseline{
		Version:     "1.0",
		ConfigHash:  configHash,
		GeneratedAt: time.Now(),
	}

	for _, v := range violations {
		b.Violations = append(b.Violations, BaselineViolation{
			RuleID:      v.RuleID,
			SourceLayer: v.SourceLayer,
			TargetLayer: v.TargetLayer,
			Import:      v.Import,
			File:        v.File,
		})
	}

	return b
}
