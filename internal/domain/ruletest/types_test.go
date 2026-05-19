package ruletest

import (
	"testing"
)

func TestMatchMode_String(t *testing.T) {
	tests := []struct {
		mode MatchMode
		want string
	}{
		{MatchModeCount, "count"},
		{MatchModeFiles, "files"},
		{MatchModeLayers, "layers"},
		{MatchModePatterns, "patterns"},
		{MatchMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("MatchMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpectation_HasExpectations(t *testing.T) {
	tests := []struct {
		name string
		exp  Expectation
		want bool
	}{
		{
			name: "count set",
			exp:  Expectation{Violations: IntPtr(3)},
			want: true,
		},
		{
			name: "files set",
			exp:  Expectation{Files: []string{"internal/**"}},
			want: true,
		},
		{
			name: "layers set",
			exp:  Expectation{Layers: []LayerExpectation{{Source: "domain", Target: "infra"}}},
			want: true,
		},
		{
			name: "patterns set",
			exp:  Expectation{Patterns: []string{"import cycle"}},
			want: true,
		},
		{
			name: "all fields zero",
			exp:  Expectation{},
			want: false,
		},
		{
			name: "empty slices",
			exp:  Expectation{Files: []string{}, Layers: []LayerExpectation{}, Patterns: []string{}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.exp.HasExpectations(); got != tt.want {
				t.Errorf("Expectation.HasExpectations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestCase_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tc      TestCase
		wantErr bool
	}{
		{
			name: "valid test case",
			tc: TestCase{
				Name:   "domain should not depend on infra",
				Expect: Expectation{Violations: IntPtr(2)},
			},
			wantErr: false,
		},
		{
			name:    "empty name",
			tc:      TestCase{Name: "", Expect: Expectation{Violations: IntPtr(1)}},
			wantErr: true,
		},
		{
			name:    "no expectations",
			tc:      TestCase{Name: "no expect", Expect: Expectation{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TestCase.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTestSuite_Validate(t *testing.T) {
	tests := []struct {
		name    string
		suite   TestSuite
		wantErr bool
	}{
		{
			name: "valid suite",
			suite: TestSuite{
				Name: "suite1",
				Tests: []TestCase{
					{Name: "test1", Expect: Expectation{Violations: IntPtr(1)}},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate test names",
			suite: TestSuite{
				Name: "suite1",
				Tests: []TestCase{
					{Name: "test1", Expect: Expectation{Violations: IntPtr(1)}},
					{Name: "test1", Expect: Expectation{Violations: IntPtr(2)}},
				},
			},
			wantErr: true,
		},
		{
			name: "empty tests",
			suite: TestSuite{
				Name:  "empty",
				Tests: []TestCase{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.suite.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TestSuite.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
