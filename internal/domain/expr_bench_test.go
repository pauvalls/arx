package domain

import (
	"testing"
)

func BenchmarkExpressionParse(b *testing.B) {
	exprs := []string{
		"count(deps(domain, infra)) > 0",
		"all(deps(domain, infra)) && !has_circular()",
		"violations(r1) == 0 && files(domain) > 10",
		"count(deps(domain, infra)) > 5 || count(deps(application, infra)) > 3",
		"threshold(files(domain), 10, 100)",
	}
	for _, expr := range exprs {
		b.Run(expr, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := Parse(expr)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkExpressionEval(b *testing.B) {
	exprs := []string{
		"count(deps(domain, infra)) > 0",
		"all(deps(domain, infra)) && !has_circular()",
		"violations(r1) == 0",
	}
	ctx := newBenchEvalContext()
	for _, exprStr := range exprs {
		expr, err := Parse(exprStr)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(exprStr, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := expr.Eval(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFilterMap(b *testing.B) {
	ctx := newBenchEvalContext()

	b.Run("filter-single", func(b *testing.B) {
		expr, err := Parse(`filter(deps(domain, infra), "ResolvedLayer == infra")`)
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < b.N; i++ {
			_, err := expr.Eval(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("map-files", func(b *testing.B) {
		expr, err := Parse(`map(deps(domain, infra), "SourceFile")`)
		if err != nil {
			b.Fatal(err)
		}
		for i := 0; i < b.N; i++ {
			_, err := expr.Eval(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func newBenchEvalContext() EvalContext {
	return EvalContext{
		Deps: []Dependency{
			{SourceFile: "domain/service.go", SourceLine: 10, ImportPath: "infra/db", ResolvedLayer: "infra"},
			{SourceFile: "domain/handler.go", SourceLine: 25, ImportPath: "infra/cache", ResolvedLayer: "infra"},
			{SourceFile: "application/usecase.go", SourceLine: 42, ImportPath: "infra/api", ResolvedLayer: "infra"},
		},
		Layers: []Layer{
			{Name: "domain", Paths: []string{"domain/**"}},
			{Name: "application", Paths: []string{"application/**"}},
			{Name: "infrastructure", Paths: []string{"infrastructure/**"}},
		},
		Violations: []Violation{},
		LayerFiles: map[string][]string{"domain": {"a.go", "b.go"}, "application": {"c.go"}, "infrastructure": {"d.go", "e.go"}},
	}
}


