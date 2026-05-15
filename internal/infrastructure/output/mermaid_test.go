package output

import (
	"strings"
	"testing"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
)

func TestGenerateMermaid(t *testing.T) {
	tests := []struct {
		name           string
		result         *application.DiagramResult
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "empty result",
			result: &application.DiagramResult{
				Layers:     []domain.Layer{},
				Dependencies: []domain.Dependency{},
				Violations: []domain.Violation{},
			},
			wantContains: []string{"flowchart TD"},
		},
		{
			name: "single layer no dependencies",
			result: &application.DiagramResult{
				Layers: []domain.Layer{
					{Name: "domain", Paths: []string{"domain/"}},
				},
				Dependencies: []domain.Dependency{},
				Violations:   []domain.Violation{},
			},
			wantContains: []string{
				"flowchart TD",
				"subgraph domain",
				`domain["domain"]`,
			},
		},
		{
			name: "multiple layers with dependencies",
			result: &application.DiagramResult{
				Layers: []domain.Layer{
					{Name: "domain", Paths: []string{"domain/"}},
					{Name: "application", Paths: []string{"application/"}},
				},
				Dependencies: []domain.Dependency{
					{
						SourceFile:    "application/service.go",
						SourceLine:    5,
						ImportPath:    "domain/entity.go",
						ResolvedLayer: "domain",
					},
				},
				Violations: []domain.Violation{},
			},
			wantContains: []string{
				"flowchart TD",
				"subgraph domain",
				"subgraph application",
				"application -->|1 deps| domain",
			},
		},
		{
			name: "dependencies with violations",
			result: &application.DiagramResult{
				Layers: []domain.Layer{
					{Name: "domain", Paths: []string{"domain/"}},
					{Name: "application", Paths: []string{"application/"}},
				},
				Dependencies: []domain.Dependency{
					{
						SourceFile:    "domain/service.go",
						SourceLine:    10,
						ImportPath:    "application/handler.go",
						ResolvedLayer: "application",
					},
				},
				Violations: []domain.Violation{
					{
						SourceLayer: "domain",
						TargetLayer: "application",
						File:        "domain/service.go",
					},
				},
			},
			wantContains: []string{
				"flowchart TD",
				"domain -.->|VIOLATION| application",
				"stroke-dasharray: 5 5",
				"red",
			},
		},
		{
			name: "multiple dependencies between same layers",
			result: &application.DiagramResult{
				Layers: []domain.Layer{
					{Name: "domain", Paths: []string{"domain/"}},
					{Name: "application", Paths: []string{"application/"}},
				},
				Dependencies: []domain.Dependency{
					{
						SourceFile:    "application/service1.go",
						SourceLine:    5,
						ImportPath:    "domain/entity1.go",
						ResolvedLayer: "domain",
					},
					{
						SourceFile:    "application/service2.go",
						SourceLine:    10,
						ImportPath:    "domain/entity2.go",
						ResolvedLayer: "domain",
					},
					{
						SourceFile:    "application/service3.go",
						SourceLine:    15,
						ImportPath:    "domain/entity3.go",
						ResolvedLayer: "domain",
					},
				},
				Violations: []domain.Violation{},
			},
			wantContains: []string{
				"application -->|3 deps| domain",
			},
		},
		{
			name: "layer names with special characters",
			result: &application.DiagramResult{
				Layers: []domain.Layer{
					{Name: "my-domain", Paths: []string{"my-domain/"}},
					{Name: "my_app", Paths: []string{"my_app/"}},
				},
				Dependencies: []domain.Dependency{
					{
						SourceFile:    "my_app/service.go",
						SourceLine:    5,
						ImportPath:    "my-domain/entity.go",
						ResolvedLayer: "my-domain",
					},
				},
				Violations: []domain.Violation{},
			},
			wantContains: []string{
				`subgraph my_dash_domain["my-domain"]`,
				`subgraph my_app["my_app"]`,
				"my_app -->|1 deps| my_dash_domain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateMermaid(tt.result)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerateMermaid() missing expected content:\nwant substring: %q\ngot:\n%s", want, got)
				}
			}

			for _, wantNot := range tt.wantNotContain {
				if strings.Contains(got, wantNot) {
					t.Errorf("GenerateMermaid() contains unexpected content:\nunexpected substring: %q\ngot:\n%s", wantNot, got)
				}
			}
		})
	}
}

func TestGenerateMermaidColorCoding(t *testing.T) {
	result := &application.DiagramResult{
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "application", Paths: []string{"application/"}},
		},
		Dependencies: []domain.Dependency{
			{
				SourceFile:    "application/service.go",
				SourceLine:    5,
				ImportPath:    "domain/entity.go",
				ResolvedLayer: "domain",
			},
		},
		Violations: []domain.Violation{},
	}

	got := GenerateMermaid(result)

	// Clean dependencies should have green styling
	if !strings.Contains(got, "green") {
		t.Errorf("GenerateMermaid() should use green color for clean dependencies, got:\n%s", got)
	}
}

func TestGenerateMermaidSubgraphStructure(t *testing.T) {
	result := &application.DiagramResult{
		Layers: []domain.Layer{
			{Name: "domain", Paths: []string{"domain/"}},
			{Name: "application", Paths: []string{"application/"}},
			{Name: "infrastructure", Paths: []string{"infrastructure/"}},
		},
		Dependencies: []domain.Dependency{},
		Violations:   []domain.Violation{},
	}

	got := GenerateMermaid(result)

	// Each layer should be in its own subgraph
	subgraphCount := strings.Count(got, "subgraph ")
	if subgraphCount != 3 {
		t.Errorf("GenerateMermaid() expected 3 subgraphs, got %d in:\n%s", subgraphCount, got)
	}

	// Each subgraph should have a corresponding closing "end" on its own line
	endPattern := "\n  end\n"
	endCount := strings.Count(got, endPattern)
	if endCount != 3 {
		t.Errorf("GenerateMermaid() expected 3 subgraph closings, got %d in:\n%s", endCount, got)
	}
}
