package csharp

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
			name:     "standard using",
			line:     "using System;",
			expected: []string{"System"},
		},
		{
			name:     "standard using with namespace",
			line:     "using System.Collections.Generic;",
			expected: []string{"System.Collections.Generic"},
		},
		{
			name:     "static using",
			line:     "using static System.Math;",
			expected: []string{"System.Math"},
		},
		{
			name:     "static using Console",
			line:     "using static System.Console;",
			expected: []string{"System.Console"},
		},
		{
			name:     "alias using",
			line:     "using Alias = Namespace.Class;",
			expected: []string{"Namespace.Class"},
		},
		{
			name:     "alias using with generics",
			line:     "using StringList = System.Collections.Generic.List<string>;",
			expected: []string{"System.Collections.Generic.List<string>"},
		},
		{
			name:     "namespace declaration",
			line:     "namespace MyApp.Domain",
			expected: []string{},
		},
		{
			name:     "comment line",
			line:     "// using System;",
			expected: []string{},
		},
		{
			name:     "doc comment line",
			line:     "/// This is a summary",
			expected: []string{},
		},
		{
			name:     "block comment start",
			line:     "/* using System; */",
			expected: []string{},
		},
		{
			name:     "block comment line",
			line:     "* This is a comment",
			expected: []string{},
		},
		{
			name:     "empty line",
			line:     "",
			expected: []string{},
		},
		{
			name:     "not an import",
			line:     "class Program {}",
			expected: []string{},
		},
		{
			name:     "inline comment after using",
			line:     "using MyApp.Domain; // this is domain",
			expected: []string{"MyApp.Domain"},
		},
		{
			name:     "using with spaces",
			line:     "  using   System.Collections.Generic  ;",
			expected: []string{"System.Collections.Generic"},
		},
		{
			name:     "using with underscore identifiers",
			line:     "using MyApp.My_Module.MyType;",
			expected: []string{"MyApp.My_Module.MyType"},
		},
		{
			name:     "nested namespace",
			line:     "using MyApp.Domain.Entities.User;",
			expected: []string{"MyApp.Domain.Entities.User"},
		},
		{
			name:     "alias with complex type",
			line:     "using Dict = System.Collections.Generic.Dictionary<string, object>;",
			expected: []string{"System.Collections.Generic.Dictionary<string, object>"},
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

func Test_extractNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "valid namespace",
			line:     "namespace MyApp.Domain",
			expected: "MyApp.Domain",
		},
		{
			name:     "namespace with semicolon",
			line:     "namespace MyApp.Domain;",
			expected: "MyApp.Domain",
		},
		{
			name:     "namespace with underscore",
			line:     "namespace MyApp.My_Domain",
			expected: "MyApp.My_Domain",
		},
		{
			name:     "not a namespace",
			line:     "using MyApp.Domain;",
			expected: "",
		},
		{
			name:     "comment line",
			line:     "// namespace MyApp.Domain",
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
			result := extractNamespace(tt.line)
			if result != tt.expected {
				t.Errorf("extractNamespace(%q) = %q, want %q", tt.line, result, tt.expected)
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
		// Standard library
		{"System standard", "System", true},
		{"System Collections", "System.Collections.Generic", true},
		{"System IO", "System.IO", true},
		{"System Linq", "System.Linq", true},
		
		// Microsoft
		{"Microsoft EntityFramework", "Microsoft.EntityFrameworkCore", true},
		{"Microsoft ASP.NET", "Microsoft.AspNetCore.Mvc", true},
		{"Microsoft Extensions", "Microsoft.Extensions.DependencyInjection", true},
		
		// Mono
		{"Mono Posix", "Mono.Posix", true},
		
		// Unity
		{"UnityEditor", "UnityEditor", true},
		{"UnityEngine", "UnityEngine", true},
		
		// Xamarin
		{"Xamarin Forms", "Xamarin.Forms", true},
		
		// Windows
		{"Windows UI", "Windows.UI.Xaml", true},
		
		// Internal namespaces (should NOT be external)
		{"MyApp domain", "MyApp.Domain", false},
		{"MyApp infrastructure", "MyApp.Infrastructure", false},
		{"MyApp application", "MyApp.Application", false},
		{"Company project", "Company.Project.Module", false},
		{"underscore namespace", "My_App.Domain", false},
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
