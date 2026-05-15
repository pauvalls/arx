package application

import (
	"testing"

	"github.com/pauvalls/arx/internal/domain"
)

func TestCalculateDebt_Basic(t *testing.T) {
	tests := []struct {
		name       string
		violations []domain.Violation
		circular   int
		wantTotal  int
		wantErrors int
		wantWarns  int
		wantInfos  int
	}{
		{
			name: "only errors",
			violations: []domain.Violation{
				{Severity: domain.SeverityError},
				{Severity: domain.SeverityError},
				{Severity: domain.SeverityError},
			},
			circular:   0,
			wantTotal:  9, // 3 * 3
			wantErrors: 3,
			wantWarns:  0,
			wantInfos:  0,
		},
		{
			name: "only warnings",
			violations: []domain.Violation{
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityWarning},
			},
			circular:   0,
			wantTotal:  5, // 5 * 1
			wantErrors: 0,
			wantWarns:  5,
			wantInfos:  0,
		},
		{
			name: "mixed severities",
			violations: []domain.Violation{
				{Severity: domain.SeverityError},
				{Severity: domain.SeverityError},
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityWarning},
				{Severity: domain.SeverityInfo},
				{Severity: domain.SeverityInfo},
			},
			circular:   0,
			wantTotal:  9, // (2 * 3) + (3 * 1) + (2 * 0)
			wantErrors: 2,
			wantWarns:  3,
			wantInfos:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateDebt(tt.violations, tt.circular)

			if score.Total != tt.wantTotal {
				t.Errorf("CalculateDebt() total = %d, want %d", score.Total, tt.wantTotal)
			}

			if score.BySeverity["error"] != tt.wantErrors {
				t.Errorf("CalculateDebt() errors = %d, want %d", score.BySeverity["error"], tt.wantErrors)
			}

			if score.BySeverity["warning"] != tt.wantWarns {
				t.Errorf("CalculateDebt() warnings = %d, want %d", score.BySeverity["warning"], tt.wantWarns)
			}

			if score.BySeverity["info"] != tt.wantInfos {
				t.Errorf("CalculateDebt() infos = %d, want %d", score.BySeverity["info"], tt.wantInfos)
			}
		})
	}
}

func TestCalculateDebt_WithCircular(t *testing.T) {
	tests := []struct {
		name          string
		violations    []domain.Violation
		circularPairs int
		wantTotal     int
		wantCircular  int
	}{
		{
			name:          "no violations, one circular pair",
			violations:    []domain.Violation{},
			circularPairs: 1,
			wantTotal:     5, // 1 * 5
			wantCircular:  5,
		},
		{
			name: "violations with circular pairs",
			violations: []domain.Violation{
				{Severity: domain.SeverityError},
				{Severity: domain.SeverityError},
				{Severity: domain.SeverityWarning},
			},
			circularPairs: 2,
			wantTotal:     17, // (2 * 3) + (1 * 1) + (2 * 5)
			wantCircular:  10,
		},
		{
			name: "multiple circular pairs only",
			violations: []domain.Violation{
				{Severity: domain.SeverityWarning},
			},
			circularPairs: 3,
			wantTotal:     16, // (1 * 1) + (3 * 5)
			wantCircular:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateDebt(tt.violations, tt.circularPairs)

			if score.Total != tt.wantTotal {
				t.Errorf("CalculateDebt() total = %d, want %d", score.Total, tt.wantTotal)
			}

			if score.BySeverity["circular"] != tt.wantCircular {
				t.Errorf("CalculateDebt() circular = %d, want %d", score.BySeverity["circular"], tt.wantCircular)
			}
		})
	}
}

func TestCalculateDebt_Empty(t *testing.T) {
	score := CalculateDebt([]domain.Violation{}, 0)

	if score.Total != 0 {
		t.Errorf("CalculateDebt() with empty input: total = %d, want 0", score.Total)
	}

	if score.BySeverity == nil {
		t.Error("CalculateDebt() BySeverity map should be initialized")
	}

	// All severity counts should be 0 or not present
	if score.BySeverity["error"] != 0 {
		t.Errorf("Expected 0 errors, got %d", score.BySeverity["error"])
	}

	if score.BySeverity["warning"] != 0 {
		t.Errorf("Expected 0 warnings, got %d", score.BySeverity["warning"])
	}

	if score.BySeverity["info"] != 0 {
		t.Errorf("Expected 0 infos, got %d", score.BySeverity["info"])
	}

	if score.BySeverity["circular"] != 0 {
		t.Errorf("Expected 0 circular, got %d", score.BySeverity["circular"])
	}
}

func TestDebtScore_String(t *testing.T) {
	tests := []struct {
		name   string
		score  domain.DebtScore
		want   string
	}{
		{
			name: "with all severities",
			score: domain.DebtScore{
				Total: 17,
				BySeverity: map[string]int{
					"error":    3,
					"warning":  2,
					"info":     5,
					"circular": 10,
				},
			},
			want: "DebtScore: Total=17 (error=3, warning=2, info=5, circular=10)",
		},
		{
			name: "zero values",
			score: domain.DebtScore{
				Total:      0,
				BySeverity: map[string]int{},
			},
			want: "DebtScore: Total=0 (error=0, warning=0, info=0, circular=0)",
		},
		{
			name: "only errors",
			score: domain.DebtScore{
				Total: 9,
				BySeverity: map[string]int{
					"error": 3,
				},
			},
			want: "DebtScore: Total=9 (error=3, warning=0, info=0, circular=0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DebtScoreString(tt.score)
			if got != tt.want {
				t.Errorf("DebtScoreString() = %q, want %q", got, tt.want)
			}
		})
	}
}
