package domain

import (
	"testing"
)

// FuzzParseExpression fuzzes the expression parser with random inputs.
func FuzzParseExpression(f *testing.F) {
	// Seed corpus with valid expressions (hand-crafted from real rules)
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
		// Additional seeds from real fixture code
		"((count(deps(a,b)) > 0) && (files(domain) < 10))",
		"my_custom_check(domain, infra) > 0",
		"violations(no_infra_import) == 0 && layers() >= 2",
		"!has_circular()",
		"ratio(deps(domain, infra), deps(domain, all)) < 0.5",
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
		// Additional seeds from real fixture code
		"files(domain) >= 1",
		"layers() > 0",
		"ratio(deps(domain, infra), deps(domain, all)) < 0.5",
		"violations(test) == 0",
		`filter(deps(a,b), "ResolvedLayer == infra")`,
		"count(map(deps(a,b), \"SourceFile\"))",
		"all(deps(domain, infra)) || any(deps(domain, all))",
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
