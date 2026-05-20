package domain

import (
	"testing"
)

func TestPRInfo_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pr      PRInfo
		wantErr string
	}{
		{
			name: "valid PR info",
			pr: PRInfo{
				BaseSHA:  "abc123def456",
				HeadSHA:  "789012ghi345",
				BaseRef:  "main",
				HeadRef:  "feature/foo",
				RepoPath: "/home/user/project",
				PRNumber: 42,
			},
			wantErr: "",
		},
		{
			name: "empty base SHA",
			pr: PRInfo{
				BaseSHA:  "",
				HeadSHA:  "789012ghi345",
				BaseRef:  "main",
				HeadRef:  "feature/foo",
				RepoPath: "/home/user/project",
				PRNumber: 42,
			},
			wantErr: "base SHA is required",
		},
		{
			name: "empty head SHA",
			pr: PRInfo{
				BaseSHA:  "abc123def456",
				HeadSHA:  "",
				BaseRef:  "main",
				HeadRef:  "feature/foo",
				RepoPath: "/home/user/project",
				PRNumber: 42,
			},
			wantErr: "head SHA is required",
		},
		{
			name: "empty base ref",
			pr: PRInfo{
				BaseSHA:  "abc123def456",
				HeadSHA:  "789012ghi345",
				BaseRef:  "",
				HeadRef:  "feature/foo",
				RepoPath: "/home/user/project",
				PRNumber: 42,
			},
			wantErr: "base ref is required",
		},
		{
			name: "empty head ref",
			pr: PRInfo{
				BaseSHA:  "abc123def456",
				HeadSHA:  "789012ghi345",
				BaseRef:  "main",
				HeadRef:  "",
				RepoPath: "/home/user/project",
				PRNumber: 42,
			},
			wantErr: "head ref is required",
		},
		{
			name: "zero PR number",
			pr: PRInfo{
				BaseSHA:  "abc123def456",
				HeadSHA:  "789012ghi345",
				BaseRef:  "main",
				HeadRef:  "feature/foo",
				RepoPath: "/home/user/project",
				PRNumber: 0,
			},
			wantErr: "PR number must be positive",
		},
		{
			name: "negative PR number",
			pr: PRInfo{
				BaseSHA:  "abc123def456",
				HeadSHA:  "789012ghi345",
				BaseRef:  "main",
				HeadRef:  "feature/foo",
				RepoPath: "/home/user/project",
				PRNumber: -1,
			},
			wantErr: "PR number must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pr.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestCheckRunConclusion_Valid(t *testing.T) {
	tests := []struct {
		c   CheckRunConclusion
		ok  bool
	}{
		{CheckRunSuccess, true},
		{CheckRunFailure, true},
		{CheckRunNeutral, true},
		{CheckRunCancelled, true},
		{CheckRunTimedOut, true},
		{CheckRunActionReq, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.c), func(t *testing.T) {
			got := tt.c.Valid()
			if got != tt.ok {
				t.Errorf("CheckRunConclusion(%q).Valid() = %v, want %v", tt.c, got, tt.ok)
			}
		})
	}
}

func TestAnnotationLevel_Valid(t *testing.T) {
	tests := []struct {
		level string
		ok    bool
	}{
		{"notice", true},
		{"warning", true},
		{"failure", true},
		{"error", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			got := ValidAnnotationLevel(tt.level)
			if got != tt.ok {
				t.Errorf("ValidAnnotationLevel(%q) = %v, want %v", tt.level, got, tt.ok)
			}
		})
	}
}
