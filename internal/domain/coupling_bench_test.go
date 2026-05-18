package domain

import (
	"testing"
)

func BenchmarkCouplingMatrix(b *testing.B) {
	calculator := NewCouplingCalculator()

	// Create test data with N layers and M dependencies
	for i := 0; i < b.N; i++ {
		layers := []Layer{
			{Name: "domain", Paths: []string{"domain/**"}},
			{Name: "application", Paths: []string{"application/**"}},
			{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
			{Name: "interfaces", Paths: []string{"interfaces/**"}},
		}

		deps := []Dependency{
			{SourceFile: "domain/entity.go", ResolvedLayer: "application"},
			{SourceFile: "application/service.go", ResolvedLayer: "infrastructure"},
			{SourceFile: "infrastructure/repo.go", ResolvedLayer: "interfaces"},
			{SourceFile: "application/handler.go", ResolvedLayer: "domain"},
			{SourceFile: "domain/value.go", ResolvedLayer: "infrastructure"},
		}

		_ = calculator.CalculateCouplingMatrix(deps, layers)
	}
}

func BenchmarkRuleEvaluation(b *testing.B) {
	rules := []Rule{
		{ID: "r1", From: "domain", To: []string{"infrastructure"}, Type: RuleTypeCannot, Severity: SeverityError},
		{ID: "r2", From: "application", To: []string{"infrastructure"}, Type: RuleTypeCannot, Severity: SeverityError},
		{ID: "r3", From: "domain", To: []string{"application"}, Type: RuleTypeCannot, Severity: SeverityError},
	}

	layers := []Layer{
		{Name: "domain", Paths: []string{"domain/**"}},
		{Name: "application", Paths: []string{"application/**"}},
		{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
	}

	deps := []Dependency{
		{SourceFile: "domain/entity.go", ResolvedLayer: "infrastructure"},
		{SourceFile: "application/service.go", ResolvedLayer: "infrastructure"},
	}

	for i := 0; i < b.N; i++ {
		EvaluateRules(deps, rules, layers, nil)
	}
}
