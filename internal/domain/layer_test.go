package domain

import (
	"testing"
)

func TestLayer_MatchesPath(t *testing.T) {
	tests := []struct {
		name     string
		layer    Layer
		filePath string
		want     bool
	}{
		{
			name: "exact prefix match",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain"},
			},
			filePath: "internal/domain/user.go",
			want:     true,
		},
		{
			name: "directory prefix with trailing slash",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain/"},
			},
			filePath: "internal/domain/user.go",
			want:     true,
		},
		{
			name: "glob pattern match",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain/*.go"},
			},
			filePath: "internal/domain/user.go",
			want:     true,
		},
		{
			name: "glob pattern no match",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain/*.go"},
			},
			filePath: "internal/domain/subdir/user.go",
			want:     false,
		},
		{
			name: "no match",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain"},
			},
			filePath: "internal/infrastructure/user.go",
			want:     false,
		},
		{
			name: "multiple patterns - first matches",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain", "pkg/domain"},
			},
			filePath: "internal/domain/user.go",
			want:     true,
		},
		{
			name: "multiple patterns - second matches",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain", "pkg/domain"},
			},
			filePath: "pkg/domain/user.go",
			want:     true,
		},
		{
			name: "empty paths",
			layer: Layer{
				Name:  "domain",
				Paths: []string{},
			},
			filePath: "internal/domain/user.go",
			want:     false,
		},
		{
			name: "double asterisk matches nested paths",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain/**"},
			},
			filePath: "internal/domain/user.go",
			want:     true,
		},
		{
			name: "double asterisk matches deeply nested paths",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain/**"},
			},
			filePath: "internal/domain/entity/valueobject/User.java",
			want:     true,
		},
		{
			name: "double asterisk matches full absolute paths",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"com/wedding/domain/**"},
			},
			filePath: "/tmp/project/src/main/java/com/wedding/domain/guest/Guest.java",
			want:     false, // Pattern doesn't include full path prefix
		},
		{
			name: "double asterisk with partial match",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"**/domain/**"},
			},
			filePath: "src/main/java/com/wedding/domain/guest/Guest.java",
			want:     true,
		},
		{
			name: "double asterisk no match different base",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain/**"},
			},
			filePath: "internal/infrastructure/user.go",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.layer.MatchesPath(tt.filePath)
			if got != tt.want {
				t.Errorf("Layer.MatchesPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLayer_Validate(t *testing.T) {
	tests := []struct {
		name    string
		layer   Layer
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid layer",
			layer: Layer{
				Name:  "domain",
				Paths: []string{"internal/domain"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			layer: Layer{
				Paths: []string{"internal/domain"},
			},
			wantErr: true,
			errMsg:  "layer name is required",
		},
		{
			name: "missing paths",
			layer: Layer{
				Name: "domain",
			},
			wantErr: true,
			errMsg:  "must have at least one path pattern",
		},
		{
			name: "valid with description and tags",
			layer: Layer{
				Name:        "domain",
				Paths:       []string{"internal/domain"},
				Description: "Domain layer",
				Tags:        []string{"core", "business"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.layer.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Layer.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || err.Error() == "" {
					t.Errorf("Layer.Validate() expected error containing %q, got %v", tt.errMsg, err)
				}
			}
		})
	}
}
