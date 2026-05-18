package domain

import (
	"testing"
)

// FuzzParseExpression fuzzes the expression parser with random inputs.
func FuzzParseExpression(f *testing.F) {
	// Seed corpus with valid expressions
	seeds := []string{
		"count(deps(domain, infra)) > 0",
		"all(deps(a, b))",
		"any(deps(x, y)) && !has_circular()",
		"violations(r1) == 0",
		"files(domain) > 5",
		"layers() >= 3",
		`filter(deps(a,b), "ResolvedLayer == infra")`,
		`map(deps(a,b), "SourceFile")`,
		"count(deps(domain, infra)) > 0 && !has_circular()",
		"threshold(files(domain), 1, 10)",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		expr, err := Parse(input)
		if err != nil {
			return
		}
		// Parse succeeded — just verifying no panic
		_ = expr
	})
}

// FuzzEvaluateExpression fuzzes expression evaluation with random inputs.
func FuzzEvaluateExpression(f *testing.F) {
	seeds := []string{
		"count(deps(domain, infra)) > 0",
		"all(deps(a, b))",
		"!has_circular()",
	}
	ctx := EvalContext{
		Deps: []Dependency{
			{SourceFile: "test.go", SourceLine: 1, ImportPath: "dep", ResolvedLayer: "infra"},
		},
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/**"}},
			{Name: "infra", Paths: []string{"infra/**"}},
		},
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		expr, err := Parse(input)
		if err != nil {
			return
		}
		// Evaluation should never panic, even with unexpected state
		_, _ = expr.Eval(ctx)
	})
}
