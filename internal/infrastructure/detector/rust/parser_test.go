package rust

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
			name:     "standard use",
			line:     "use std::collections::HashMap;",
			expected: []string{"std::collections::HashMap"},
		},
		{
			name:     "crate-relative use",
			line:     "use crate::domain::model::Order;",
			expected: []string{"crate::domain::model::Order"},
		},
		{
			name:     "self-relative use",
			line:     "use self::submodule::Helper;",
			expected: []string{"self::submodule::Helper"},
		},
		{
			name:     "super-relative use",
			line:     "use super::parent_module::Something;",
			expected: []string{"super::parent_module::Something"},
		},
		{
			name:     "re-export with pub use",
			line:     "pub use crate::domain::Model;",
			expected: []string{"crate::domain::Model"},
		},
		{
			name:     "pub use with self",
			line:     "pub use self::internal::Helper;",
			expected: []string{"self::internal::Helper"},
		},
		{
			name:     "pub mod declaration",
			line:     "pub mod models;",
			expected: []string{},
		},
		{
			name:     "use with crate and nested modules",
			line:     "use crate::infrastructure::repository::OrderRepository;",
			expected: []string{"crate::infrastructure::repository::OrderRepository"},
		},
		{
			name:     "use with super multiple levels",
			line:     "use super::super::domain::Event;",
			expected: []string{"super::super::domain::Event"},
		},
		{
			name:     "comment line",
			line:     "// use std::collections::HashMap;",
			expected: []string{},
		},
		{
			name:     "doc comment line",
			line:     "/// This module provides utilities",
			expected: []string{},
		},
		{
			name:     "attribute line",
			line:     "#[cfg(test)]",
			expected: []string{},
		},
		{
			name:     "empty line",
			line:     "",
			expected: []string{},
		},
		{
			name:     "not an import",
			line:     "fn do_something() {}",
			expected: []string{},
		},
		{
			name:     "inline comment after use",
			line:     "use crate::domain::Order; // this is a domain model",
			expected: []string{"crate::domain::Order"},
		},
		{
			name:     "use with spaces",
			line:     "  use   std::collections::HashMap  ;",
			expected: []string{"std::collections::HashMap"},
		},
		{
			name:     "use with underscore identifiers",
			line:     "use crate::my_module::MyType;",
			expected: []string{"crate::my_module::MyType"},
		},
		{
			name:     "nested module with multiple colons",
			line:     "use crate::a::b::c::d::E;",
			expected: []string{"crate::a::b::c::d::E"},
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

func Test_extractPubMod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "valid pub mod",
			line:     "pub mod models;",
			expected: "models",
		},
		{
			name:     "pub mod with underscore",
			line:     "pub mod my_module;",
			expected: "my_module",
		},
		{
			name:     "not a pub mod",
			line:     "use crate::models;",
			expected: "",
		},
		{
			name:     "comment line",
			line:     "// pub mod models;",
			expected: "",
		},
		{
			name:     "empty line",
			line:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := extractPubMod(tt.line)
			if result != tt.expected {
				t.Errorf("extractPubMod(%q) = %q, want %q", tt.line, result, tt.expected)
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
		{"std standard", "std::collections::HashMap", true},
		{"std io", "std::io::Read", true},
		{"core", "core::mem::MaybeUninit", true},
		{"alloc", "alloc::sync::Arc", true},
		{"test", "test::Bencher", true},
		{"crate-relative", "crate::domain::Order", false},
		{"self-relative", "self::submodule::Helper", false},
		{"super-relative", "super::parent::Something", false},
		{"external crate serde", "serde::Deserialize", false},
		{"external crate tokio", "tokio::runtime::Runtime", false},
		{"nested crate path", "crate::app::service::Handler", false},
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
