package kotlin

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
			line:     "import org.springframework.boot.autoconfigure.SpringBootApplication",
			expected: []string{"org.springframework.boot.autoconfigure.SpringBootApplication"},
		},
		{
			name:     "standard import with semicolon",
			line:     "import java.util.List;",
			expected: []string{"java.util.List"},
		},
		{
			name:     "wildcard import",
			line:     "import com.example.domain.*",
			expected: []string{"com.example.domain"},
		},
		{
			name:     "wildcard import with semicolon",
			line:     "import com.example.domain.*;",
			expected: []string{"com.example.domain"},
		},
		{
			name:     "import alias with as",
			line:     "import com.example.domain.Order as DomainOrder",
			expected: []string{"com.example.domain.Order"},
		},
		{
			name:     "import alias with semicolon",
			line:     "import com.example.domain.Order as DomainOrder;",
			expected: []string{"com.example.domain.Order"},
		},
		{
			name:     "import with spaces",
			line:     "  import   java.util.ArrayList  ",
			expected: []string{"java.util.ArrayList"},
		},
		{
			name:     "not an import",
			line:     "class MyClass {}",
			expected: []string{},
		},
		{
			name:     "comment line",
			line:     "// import com.example.Fake",
			expected: []string{},
		},
		{
			name:     "import line in comment after code",
			line:     "val x = 1 // import com.example.Fake",
			expected: []string{},
		},
		{
			name:     "kotlin standard library import",
			line:     "import kotlin.collections.List",
			expected: []string{"kotlin.collections.List"},
		},
		{
			name:     "nested class import with semicolon",
			line:     "import com.example.domain.Order.Item;",
			expected: []string{"com.example.domain.Order.Item"},
		},
		{
			name:     "empty line",
			line:     "",
			expected: []string{},
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

func Test_extractPackage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		expected string
	}{
		{
			name:     "valid package",
			line:     "package com.example.app",
			expected: "com.example.app",
		},
		{
			name:     "package with semicolon",
			line:     "package com.example.app;",
			expected: "com.example.app",
		},
		{
			name:     "package with spaces",
			line:     "  package   com.example.app  ",
			expected: "com.example.app",
		},
		{
			name:     "not a package",
			line:     "import com.example.app",
			expected: "",
		},
		{
			name:     "comment line",
			line:     "// package com.example.app",
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
			result := extractPackage(tt.line)
			if result != tt.expected {
				t.Errorf("extractPackage(%q) = %q, want %q", tt.line, result, tt.expected)
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
		{"kotlin standard", "kotlin.collections.List", true},
		{"kotlinx", "kotlinx.coroutines.Deferred", true},
		{"kotlin stdlib", "kotlin.text.Regex", true},
		{"java standard", "java.util.List", true},
		{"javax standard", "javax.servlet.http.HttpServletRequest", true},
		{"sun internal", "sun.misc.Unsafe", true},
		{"com.sun", "com.sun.net.httpserver.HttpServer", true},
		{"custom domain", "com.example.domain.Order", false},
		{"spring framework", "org.springframework.boot.SpringApplication", false},
		{"junit", "org.junit.Assert", false},
		{"kotlin test", "kotlin.test.Test", true},
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
