package typescript_detector

import (
	"testing"
)

func Test_extractImportsFromLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected []string
	}{
		// Standard imports (backward compatibility)
		{
			name:     "standard import",
			line:     `import { User } from './user';`,
			expected: []string{"./user"},
		},
		{
			name:     "default import",
			line:     `import express from 'express';`,
			expected: []string{"express"},
		},
		{
			name:     "namespace import",
			line:     `import * as fs from 'fs';`,
			expected: []string{"fs"},
		},
		{
			name:     "require call",
			line:     `const fs = require('fs');`,
			expected: []string{"fs"},
		},
		{
			name:     "export all from",
			line:     `export * from './module';`,
			expected: []string{"./module"},
		},
		{
			name:     "export named from",
			line:     `export { foo, bar } from './module';`,
			expected: []string{"./module"},
		},
		{
			name:     "export default from",
			line:     `export default from './module';`,
			expected: []string{"./module"},
		},

		// Type-only imports (new)
		{
			name:     "import type with braces",
			line:     `import type { User } from './user';`,
			expected: []string{"./user"},
		},
		{
			name:     "import type single",
			line:     `import type User from './user';`,
			expected: []string{"./user"},
		},

		// Re-exports (new)
		{
			name:     "re-export named",
			line:     `export { X } from 'module';`,
			expected: []string{"module"},
		},
		{
			name:     "re-export with braces multiple",
			line:     `export { A, B, C } from './utils';`,
			expected: []string{"./utils"},
		},

		// Dynamic imports (new)
		{
			name:     "dynamic import",
			line:     `const mod = import('./module');`,
			expected: []string{"./module"},
		},
		{
			name:     "await dynamic import",
			line:     `const mod = await import('./module');`,
			expected: []string{"./module"},
		},
		{
			name:     "const with await dynamic import",
			line:     `const X = await import('module');`,
			expected: []string{"module"},
		},
		{
			name:     "dynamic import with double quotes",
			line:     `const mod = import("./module");`,
			expected: []string{"./module"},
		},
		{
			name:     "dynamic import in function call",
			line:     `Promise.all([import('./a'), import('./b')])`,
			expected: []string{"./a"},
		},

		// Edge cases
		{
			name:     "empty line",
			line:     "",
			expected: []string{},
		},
		{
			name:     "no import",
			line:     `const x = 42;`,
			expected: []string{},
		},
		{
			name:     "comment line",
			line:     `// import { X } from './module';`,
			expected: []string{},
		},
		{
			name:     "import without from",
			line:     `import './styles.css';`,
			expected: []string{"./styles.css"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractImportsFromLine(tt.line)
			if len(result) != len(tt.expected) {
				t.Errorf("extractImportsFromLine(%q) returned %d imports, expected %d\n  got:  %v\n  want: %v",
					tt.line, len(result), len(tt.expected), result, tt.expected)
				return
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("extractImportsFromLine(%q)[%d] = %q, want %q",
						tt.line, i, result[i], exp)
				}
			}
		})
	}
}

func Test_extractImportsFromLine_DynamicImportMultiple(t *testing.T) {
	// A single regex match returns the first dynamic import on a line.
	// This test documents current behavior: only first match per pattern.
	line := `Promise.all([import('./a'), import('./b')])`
	result := extractImportsFromLine(line)
	if len(result) != 1 {
		t.Fatalf("expected 1 import, got %d: %v", len(result), result)
	}
	if result[0] != "./a" {
		t.Errorf("expected './a', got %q", result[0])
	}
}

func Test_extractImportsFromLine_BackwardCompat(t *testing.T) {
	// Ensure existing patterns still work exactly as before
	lines := []string{
		`import { X } from 'module';`,
		`import * as Y from 'module2';`,
		`import Z from 'module3';`,
		`const A = require('module4');`,
		`export * from 'module5';`,
	}

	expected := []string{"module", "module2", "module3", "module4", "module5"}

	for i, line := range lines {
		result := extractImportsFromLine(line)
		if len(result) != 1 {
			t.Fatalf("line %d: expected 1 import, got %d: %v", i, len(result), result)
		}
		if result[0] != expected[i] {
			t.Errorf("line %d: expected %q, got %q", i, expected[i], result[0])
		}
	}
}
