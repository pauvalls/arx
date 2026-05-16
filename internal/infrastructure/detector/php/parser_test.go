package php

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
			name:     "use standard class",
			line:     "use App\\Domain\\Order;",
			expected: []string{"App\\Domain\\Order"},
		},
		{
			name:     "use standard with double backslash",
			line:     "use App\\Infrastructure\\OrderRepository;",
			expected: []string{"App\\Infrastructure\\OrderRepository"},
		},
		{
			name:     "use with alias",
			line:     "use App\\Domain\\Order as DomainOrder;",
			expected: []string{"App\\Domain\\Order"},
		},
		{
			name:     "use function",
			line:     "use function App\\Helpers\\format_money;",
			expected: []string{"App\\Helpers\\format_money"},
		},
		{
			name:     "use const",
			line:     "use const App\\Constants\\MAX_ITEMS;",
			expected: []string{"App\\Constants\\MAX_ITEMS"},
		},
		{
			name:     "require_once relative",
			line:     "require_once __DIR__ . '/../Domain/Order.php';",
			expected: []string{"../Domain/Order.php"},
		},
		{
			name:     "require_once with double quotes",
			line:     `require_once __DIR__ . "/helpers.php";`,
			expected: []string{"helpers.php"},
		},
		{
			name:     "comment line with //",
			line:     "// use App\\Domain\\Order;",
			expected: []string{},
		},
		{
			name:     "comment line with #",
			line:     "# use App\\Domain\\Order;",
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
			name:     "inline comment after use",
			line:     "use App\\Domain\\Order; // domain entity",
			expected: []string{"App\\Domain\\Order"},
		},
		{
			name:     "use with leading whitespace",
			line:     "  use App\\Domain\\Order;  ",
			expected: []string{"App\\Domain\\Order"},
		},
		{
			name:     "require_once with leading whitespace",
			line:     "  require_once __DIR__ . '/../Domain/Order.php';  ",
			expected: []string{"../Domain/Order.php"},
		},
		{
			name:     "use alias with leading whitespace",
			line:     "  use App\\Domain\\Order as DomainOrder;  ",
			expected: []string{"App\\Domain\\Order"},
		},
		{
			name:     "use function with leading whitespace",
			line:     "  use function App\\Helpers\\format_money;  ",
			expected: []string{"App\\Helpers\\format_money"},
		},
		{
			name:     "use const with leading whitespace",
			line:     "  use const App\\Constants\\MAX_ITEMS;  ",
			expected: []string{"App\\Constants\\MAX_ITEMS"},
		},
		{
			name:     "require_once without semicolon",
			line:     "require_once __DIR__ . '/../Domain/Order.php'",
			expected: []string{"../Domain/Order.php"},
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
		{"Symfony component", "Symfony\\Component\\HttpFoundation\\Request", true},
		{"Doctrine ORM", "Doctrine\\ORM\\EntityManager", true},
		{"PSR interface", "Psr\\Log\\LoggerInterface", true},
		{"Monolog", "Monolog\\Logger", true},
		{"Composer autoload", "Composer\\Autoload\\ClassLoader", true},
		{"relative parent path", "../Domain/Order.php", false},
		{"relative current path", "./helpers.php", false},
		{"vendor path", "vendor/symfony/http-foundation/Request.php", true},
		{"App namespace", "App\\Domain\\Order", false},
		{"Domain namespace", "Domain\\Order", false},
		{"Application namespace", "Application\\OrderService", false},
		{"Infrastructure namespace", "Infrastructure\\OrderRepository", false},
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
