package shared

import (
	"testing"
)

func TestMatchImportToLayer_DoubleAsterisk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		importPath   string
		layerPattern string
		expected     bool
	}{
		{
			name:         "exact match with double asterisk",
			importPath:   "com/example/domain",
			layerPattern: "com/example/domain/**",
			expected:     true,
		},
		{
			name:         "nested one level with double asterisk",
			importPath:   "com/example/domain/order",
			layerPattern: "com/example/domain/**",
			expected:     true,
		},
		{
			name:         "nested multiple levels with double asterisk",
			importPath:   "com/example/domain/order/item",
			layerPattern: "com/example/domain/**",
			expected:     true,
		},
		{
			name:         "no match different base",
			importPath:   "com/example/infrastructure",
			layerPattern: "com/example/domain/**",
			expected:     false,
		},
		{
			name:         "partial match not enough",
			importPath:   "com/example",
			layerPattern: "com/example/domain/**",
			expected:     false,
		},
		{
			name:         "single asterisk exact match",
			importPath:   "com/example/domain",
			layerPattern: "com/example/*",
			expected:     true,
		},
		{
			name:         "single asterisk no nested",
			importPath:   "com/example/domain/order",
			layerPattern: "com/example/*",
			expected:     false,
		},
		{
			name:         "double asterisk in middle",
			importPath:   "com/example/app/domain/order",
			layerPattern: "com/**/domain/**",
			expected:     true,
		},
		{
			name:         "internal domain pattern from arx.yaml",
			importPath:   "internal/domain/order",
			layerPattern: "internal/domain/**",
			expected:     true,
		},
		{
			name:         "internal domain nested",
			importPath:   "internal/domain/entity/valueobject",
			layerPattern: "internal/domain/**",
			expected:     true,
		},
		{
			name:         "empty import path",
			importPath:   "",
			layerPattern: "com/example/**",
			expected:     false,
		},
		{
			name:         "exact match no glob",
			importPath:   "com/example/domain",
			layerPattern: "com/example/domain",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := MatchImportToLayer(tt.importPath, tt.layerPattern)
			if result != tt.expected {
				t.Errorf("MatchImportToLayer(%q, %q) = %v, want %v",
					tt.importPath, tt.layerPattern, result, tt.expected)
			}
		})
	}
}
