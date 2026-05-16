package ruby

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
			name:     "require gem",
			line:     "require 'rails'",
			expected: []string{"rails"},
		},
		{
			name:     "require gem with double quotes",
			line:     `require "sinatra"`,
			expected: []string{"sinatra"},
		},
		{
			name:     "require nested gem",
			line:     "require 'sinatra/base'",
			expected: []string{"sinatra/base"},
		},
		{
			name:     "require_relative local",
			line:     "require_relative '../domain/order'",
			expected: []string{"../domain/order"},
		},
		{
			name:     "require_relative with double quotes",
			line:     `require_relative "./helpers"`,
			expected: []string{"./helpers"},
		},
		{
			name:     "require_all local",
			line:     "require_all 'lib/domain'",
			expected: []string{"lib/domain"},
		},
		{
			name:     "require File.expand_path",
			line:     "require File.expand_path('../domain/order', __dir__)",
			expected: []string{"../domain/order"},
		},
		{
			name:     "require File.expand_path with double quotes",
			line:     `require File.expand_path("../helpers", __dir__)`,
			expected: []string{"../helpers"},
		},
		{
			name:     "require bundler/setup",
			line:     "require 'bundler/setup'",
			expected: []string{"bundler/setup"},
		},
		{
			name:     "comment line",
			line:     "# require 'rails'",
			expected: []string{},
		},
		{
			name:     "empty line",
			line:     "",
			expected: []string{},
		},
		{
			name:     "not an import",
			line:     "class Order; end",
			expected: []string{},
		},
		{
			name:     "inline comment after require",
			line:     "require 'rails' # main framework",
			expected: []string{"rails"},
		},
		{
			name:     "require with spaces",
			line:     "  require   'rails'  ",
			expected: []string{"rails"},
		},
		{
			name:     "require_relative with spaces",
			line:     "  require_relative   '../domain/order'  ",
			expected: []string{"../domain/order"},
		},
		{
			name:     "require_all with spaces",
			line:     "  require_all   'lib/domain'  ",
			expected: []string{"lib/domain"},
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
		{"rails gem", "rails", true},
		{"sinatra nested", "sinatra/base", true},
		{"bundler/setup", "bundler/setup", true},
		{"rubygems", "rubygems", true},
		{"bundler", "bundler", true},
		{"relative parent path", "../domain/order", false},
		{"relative current path", "./helpers", false},
		{"lib path", "lib/domain/order", false},
		{"app path", "app/services/order_service", false},
		{"require_all lib", "lib/domain", false},
		{"File.expand_path parent", "../domain/order", false},
		{"external gem sidekiq", "sidekiq", true},
		{"external gem pg", "pg", true},
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
