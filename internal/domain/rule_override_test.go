package domain

import (
	"testing"
)

func TestRule_GetEffectiveSeverity(t *testing.T) {
	tests := []struct {
		name     string
		rule     Rule
		filePath string
		wantSev  Severity
		wantOK   bool
	}{
		{
			name: "empty overrides - returns original",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityError,
			wantOK:   false,
		},
		{
			name: "non-matching override - returns original",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/migration/", Severity: SeverityWarning},
				},
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityError,
			wantOK:   false,
		},
		{
			name: "single matching override with trailing slash",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Severity: SeverityWarning},
				},
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityWarning,
			wantOK:   true,
		},
		{
			name: "single matching override without trailing slash",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy", Severity: SeverityWarning},
				},
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityWarning,
			wantOK:   true,
		},
		{
			name: "matching override with empty path matches everything",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "", Severity: SeverityInfo},
				},
			},
			filePath: "any/path/file.go",
			wantSev:  SeverityInfo,
			wantOK:   true,
		},
		{
			name: "multiple matching - longest prefix wins",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/", Severity: SeverityInfo},
					{Path: "internal/legacy/", Severity: SeverityWarning},
				},
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityWarning,
			wantOK:   true,
		},
		{
			name: "multiple matching - shorter prefix does not override longer",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Severity: SeverityWarning},
					{Path: "internal/", Severity: SeverityInfo},
				},
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityWarning,
			wantOK:   true,
		},
		{
			name: "override without severity is skipped",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Enabled: boolPtr(false)},
				},
			},
			filePath: "internal/legacy/file.go",
			wantSev:  SeverityError,
			wantOK:   false,
		},
		{
			name: "subdirectory deep match",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Severity: SeverityWarning},
				},
			},
			filePath: "internal/legacy/sub/dir/file.go",
			wantSev:  SeverityWarning,
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sev, ok := tt.rule.GetEffectiveSeverity(tt.filePath)
			if sev != tt.wantSev {
				t.Errorf("GetEffectiveSeverity() severity = %q, want %q", sev, tt.wantSev)
			}
			if ok != tt.wantOK {
				t.Errorf("GetEffectiveSeverity() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestRule_IsEnabledFor(t *testing.T) {
	disabled := false
	enabled := true

	tests := []struct {
		name     string
		rule     Rule
		filePath string
		want     bool
	}{
		{
			name: "no overrides - enabled",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
			},
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name: "empty overrides - enabled",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{},
			},
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name: "override with Enabled=false matches path",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Enabled: &disabled},
				},
			},
			filePath: "internal/legacy/file.go",
			want:     false,
		},
		{
			name: "override with Enabled=false does not match path",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/other/", Enabled: &disabled},
				},
			},
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name: "override with Enabled=nil (default enabled)",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Enabled: nil},
				},
			},
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name: "override with Enabled=true (explicitly enabled)",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/legacy/", Enabled: &enabled},
				},
			},
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name: "multiple overrides - one disables",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/", Severity: SeverityWarning},
					{Path: "internal/legacy/", Enabled: &disabled},
				},
			},
			filePath: "internal/legacy/file.go",
			want:     false,
		},
		{
			name: "multiple overrides - none disable",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "internal/", Severity: SeverityInfo},
					{Path: "internal/legacy/", Severity: SeverityWarning},
				},
			},
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name: "empty path matches everything - disabled",
			rule: Rule{
				ID:       "R1",
				Severity: SeverityError,
				Overrides: []RuleOverride{
					{Path: "", Enabled: &disabled},
				},
			},
			filePath: "any/path/file.go",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.IsEnabledFor(tt.filePath)
			if got != tt.want {
				t.Errorf("IsEnabledFor(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

// TestMatchesOverridePath tests the path matching helper directly
func TestMatchesOverridePath(t *testing.T) {
	tests := []struct {
		name     string
		override string
		filePath string
		want     bool
	}{
		{
			name:     "empty path matches everything",
			override: "",
			filePath: "some/deep/path/file.go",
			want:     true,
		},
		{
			name:     "trailing slash prefix match",
			override: "internal/legacy/",
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name:     "trailing slash deep subdirectory",
			override: "internal/legacy/",
			filePath: "internal/legacy/sub/dir/file.go",
			want:     true,
		},
		{
			name:     "no trailing slash - subdirectory with / boundary",
			override: "internal/legacy",
			filePath: "internal/legacy/file.go",
			want:     true,
		},
		{
			name:     "no trailing slash - exact file match",
			override: "internal/legacy/main.go",
			filePath: "internal/legacy/main.go",
			want:     true,
		},
		{
			name:     "no match - different prefix",
			override: "internal/legacy/",
			filePath: "internal/other/file.go",
			want:     false,
		},
		{
			name:     "partial prefix should not match without boundary",
			override: "internal/leg",
			filePath: "internal/legacy/file.go",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesOverridePath(tt.override, tt.filePath)
			if got != tt.want {
				t.Errorf("matchesOverridePath(%q, %q) = %v, want %v", tt.override, tt.filePath, got, tt.want)
			}
		})
	}
}

// boolPtr is a helper to get a *bool from a bool literal
func boolPtr(b bool) *bool {
	return &b
}
