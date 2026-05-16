package swift

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
		{
			name:     "standard import",
			line:     "import Foundation",
			expected: []string{"Foundation"},
		},
		{
			name:     "standard import custom module",
			line:     "import MyModule",
			expected: []string{"MyModule"},
		},
		{
			name:     "import struct member",
			line:     "import struct Foundation.URL",
			expected: []string{"Foundation"},
		},
		{
			name:     "import class member",
			line:     "import class UIKit.UIView",
			expected: []string{"UIKit"},
		},
		{
			name:     "import enum member",
			line:     "import enum SwiftUI.LayoutPriority",
			expected: []string{"SwiftUI"},
		},
		{
			name:     "import protocol member",
			line:     "import protocol Foundation.Encodable",
			expected: []string{"Foundation"},
		},
		{
			name:     "import typealias member",
			line:     "import typealias Foundation.Data",
			expected: []string{"Foundation"},
		},
		{
			name:     "import let member",
			line:     "import let SomeModule.constant",
			expected: []string{"SomeModule"},
		},
		{
			name:     "import var member",
			line:     "import var SomeModule.variable",
			expected: []string{"SomeModule"},
		},
		{
			name:     "import func member",
			line:     "import func Foundation.print",
			expected: []string{"Foundation"},
		},
		{
			name:     "exported import",
			line:     "@_exported import Foundation",
			expected: []string{"Foundation"},
		},
		{
			name:     "exported import custom module",
			line:     "@_exported import MyModule",
			expected: []string{"MyModule"},
		},
		{
			name:     "comment line",
			line:     "// import Foundation",
			expected: []string{},
		},
		{
			name:     "empty line",
			line:     "",
			expected: []string{},
		},
		{
			name:     "not an import",
			line:     "class Order {}",
			expected: []string{},
		},
		{
			name:     "inline comment after import",
			line:     "import Foundation // main framework",
			expected: []string{"Foundation"},
		},
		{
			name:     "import with leading spaces",
			line:     "  import Foundation  ",
			expected: []string{"Foundation"},
		},
		{
			name:     "exported import with spaces",
			line:     "  @_exported import MyModule  ",
			expected: []string{"MyModule"},
		},
		{
			name:     "import struct with spaces",
			line:     "  import struct Foundation.URL  ",
			expected: []string{"Foundation"},
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

func Test_isExternalDependency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		importPath string
		expected   bool
	}{
		{"Foundation", "Foundation", true},
		{"UIKit", "UIKit", true},
		{"SwiftUI", "SwiftUI", true},
		{"AppKit", "AppKit", true},
		{"CoreData", "CoreData", true},
		{"Combine", "Combine", true},
		{"Dispatch", "Dispatch", true},
		{"os", "os", true},
		{"CoreGraphics", "CoreGraphics", true},
		{"QuartzCore", "QuartzCore", true},
		{"custom module Domain", "Domain", false},
		{"custom module Application", "Application", false},
		{"custom module Infrastructure", "Infrastructure", false},
		{"custom module MyPackage", "MyPackage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := isExternalDependency(tt.importPath)
			if result != tt.expected {
				t.Errorf("isExternalDependency(%q) = %v, want %v",
					tt.importPath, result, tt.expected)
			}
		})
	}
}
