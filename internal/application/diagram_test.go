package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

func TestNewDiagramService(t *testing.T) {
	svc := NewDiagramService(nil)
	if svc == nil {
		t.Fatal("NewDiagramService returned nil")
	}
}

func TestDiagramService_Generate_EmptyDetectors(t *testing.T) {
	svc := NewDiagramService(nil)
	result, err := svc.Generate("/test", []domain.Layer{}, &domain.Config{})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result == nil {
		t.Fatal("Generate() returned nil result")
	}
	if len(result.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies, got %d", len(result.Dependencies))
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
}

func TestDiagramService_Generate_WithLayers(t *testing.T) {
	svc := NewDiagramService(nil)
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"testdata/**"}},
	}
	config := &domain.Config{
		Version: "1.0",
		Layers:  layers,
		Rules: []domain.Rule{
			{ID: "R1", From: "domain", To: []string{"infrastructure"}, Type: domain.RuleTypeCannot},
		},
	}

	result, err := svc.Generate("/nonexistent", layers, config)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result == nil {
		t.Fatal("Generate() returned nil result")
	}
}

func TestDiagramService_Generate_WithDetectors(t *testing.T) {
	svc := NewDiagramService([]ports.Detector{nil}) // nil detector is skipped
	layers := []domain.Layer{
		{Name: "domain", Paths: []string{"internal/domain/**"}},
	}

	result, err := svc.Generate("/test", layers, &domain.Config{
		Version: "1.0",
		Layers:  layers,
		Rules:   []domain.Rule{},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result == nil {
		t.Fatal("Generate() returned nil result")
	}
}

func TestSortDependencies(t *testing.T) {
	deps := []domain.Dependency{
		{SourceFile: "z/file.go", ImportPath: "a/pkg"},
		{SourceFile: "a/file.go", ImportPath: "b/pkg"},
		{SourceFile: "a/file.go", ImportPath: "a/pkg"},
	}

	sortDependencies(deps)

	if deps[0].SourceFile != "a/file.go" || deps[0].ImportPath != "a/pkg" {
		t.Errorf("expected first to be a/file.go->a/pkg, got %s->%s", deps[0].SourceFile, deps[0].ImportPath)
	}
	if deps[1].SourceFile != "a/file.go" || deps[1].ImportPath != "b/pkg" {
		t.Errorf("expected second to be a/file.go->b/pkg, got %s->%s", deps[1].SourceFile, deps[1].ImportPath)
	}
	if deps[2].SourceFile != "z/file.go" || deps[2].ImportPath != "a/pkg" {
		t.Errorf("expected third to be z/file.go->a/pkg, got %s->%s", deps[2].SourceFile, deps[2].ImportPath)
	}
}

func TestSortDependencies_Empty(t *testing.T) {
	deps := []domain.Dependency{}
	sortDependencies(deps)
	// Should not panic
}
