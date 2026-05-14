package domain

import (
	"testing"
)

func TestViolation_String(t *testing.T) {
	tests := []struct {
		name       string
		violation  Violation
		want       string
	}{
		{
			name: "basic violation",
			violation: Violation{
				ID:          "D-01",
				RuleID:      "R1",
				File:        "internal/domain/user.go",
				Line:        10,
				SourceLayer: "domain",
				TargetLayer: "infrastructure",
				Import:      "github.com/example/arx/internal/infrastructure/db",
				Message:     "domain cannot depend on infrastructure",
			},
			want: "[D-01] internal/domain/user.go:10: domain -> infrastructure (domain cannot depend on infrastructure)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.violation.String()
			if got != tt.want {
				t.Errorf("Violation.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDependency_String(t *testing.T) {
	tests := []struct {
		name       string
		dependency Dependency
		want       string
	}{
		{
			name: "with resolved layer",
			dependency: Dependency{
				SourceFile:    "internal/domain/user.go",
				SourceLine:    10,
				ImportPath:    "github.com/example/arx/internal/infrastructure/db",
				ResolvedLayer: "infrastructure",
			},
			want: "internal/domain/user.go:10 -> github.com/example/arx/internal/infrastructure/db (infrastructure)",
		},
		{
			name: "without resolved layer",
			dependency: Dependency{
				SourceFile: "internal/domain/user.go",
				SourceLine: 10,
				ImportPath: "github.com/example/arx/internal/infrastructure/db",
			},
			want: "internal/domain/user.go:10 -> github.com/example/arx/internal/infrastructure/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dependency.String()
			if got != tt.want {
				t.Errorf("Dependency.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
