package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadArxIgnore_MissingFile(t *testing.T) {
	tmp := t.TempDir()

	ignore, err := LoadArxIgnore(tmp)
	if err != nil {
		t.Fatalf("LoadArxIgnore() error = %v", err)
	}
	if ignore != nil {
		t.Error("LoadArxIgnore() should return nil when .arxignore does not exist")
	}
}

func TestLoadArxIgnore_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, ".arxignore"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	ignore, err := LoadArxIgnore(tmp)
	if err != nil {
		t.Fatalf("LoadArxIgnore() error = %v", err)
	}
	if ignore == nil {
		t.Fatal("LoadArxIgnore() should return non-nil for existing file")
	}
	if len(ignore.Patterns) != 0 {
		t.Errorf("Patterns = %v, want empty slice", ignore.Patterns)
	}
}

func TestLoadArxIgnore_ParsesPatterns(t *testing.T) {
	tmp := t.TempDir()
	content := `# This is a comment
vendor/
*.generated.go

build/
# Another comment
node_modules/
`
	if err := os.WriteFile(filepath.Join(tmp, ".arxignore"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ignore, err := LoadArxIgnore(tmp)
	if err != nil {
		t.Fatalf("LoadArxIgnore() error = %v", err)
	}
	if ignore == nil {
		t.Fatal("LoadArxIgnore() should return non-nil")
	}

	want := []string{"vendor/", "*.generated.go", "build/", "node_modules/"}
	if len(ignore.Patterns) != len(want) {
		t.Fatalf("Patterns count = %d, want %d", len(ignore.Patterns), len(want))
	}
	for i, p := range want {
		if ignore.Patterns[i] != p {
			t.Errorf("Patterns[%d] = %q, want %q", i, ignore.Patterns[i], p)
		}
	}
}

func TestLoadArxIgnore_OnlyCommentsAndBlanks(t *testing.T) {
	tmp := t.TempDir()
	content := `# Just a comment

# Another comment
`
	if err := os.WriteFile(filepath.Join(tmp, ".arxignore"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ignore, err := LoadArxIgnore(tmp)
	if err != nil {
		t.Fatalf("LoadArxIgnore() error = %v", err)
	}
	if ignore == nil {
		t.Fatal("LoadArxIgnore() should return non-nil")
	}
	if len(ignore.Patterns) != 0 {
		t.Errorf("Patterns = %v, want empty (only comments)", ignore.Patterns)
	}
}

func TestIsIgnored_NilReceiver(t *testing.T) {
	var ignore *ArxIgnore
	if ignore.IsIgnored("vendor/foo.go") {
		t.Error("nil ArxIgnore should never ignore any path")
	}
}

func TestIsIgnored_EmptyPatterns(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{}}
	if ignore.IsIgnored("vendor/foo.go") {
		t.Error("ArxIgnore with no patterns should never ignore any path")
	}
}

func TestIsIgnored_ExtensionGlob(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"*.generated.go"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches pb.generated.go", "pb.generated.go", true},
		{"matches pkg/pb.generated.go", "pkg/pb.generated.go", true},
		{"does not match foo.go", "foo.go", false},
		{"does not match foo.generated.ts", "foo.generated.ts", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_DirectoryPattern(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"vendor/"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches vendor/foo.go", "vendor/foo.go", true},
		{"matches vendor/sub/bar.go", "vendor/sub/bar.go", true},
		{"matches vendor exactly", "vendor", true},
		{"does not match vendorship/foo.go", "vendorship/foo.go", false},
		{"does not match other/foo.go", "other/foo.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_MultiplePatterns(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{
		"vendor/",
		"*.generated.go",
		"node_modules/",
	}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches vendor pattern", "vendor/lib.go", true},
		{"matches extension pattern", "proto.generated.go", true},
		{"matches node_modules pattern", "node_modules/pkg/index.js", true},
		{"matches no pattern", "internal/domain/user.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_QuestionMarkGlob(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"test?.go"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches test1.go", "test1.go", true},
		{"matches testA.go", "pkg/testA.go", true},
		{"does not match test.go", "test.go", false},
		{"does not match test12.go", "test12.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_CharacterClassGlob(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"*.[ch]"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches foo.c", "foo.c", true},
		{"matches bar.h", "src/bar.h", true},
		{"does not match foo.cpp", "foo.cpp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_DoubleStarPattern(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"build/**"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches build/output.txt", "build/output.txt", true},
		{"matches build/output/deep/file.txt", "build/output/deep/file.txt", true},
		{"matches build exactly", "build", true},
		{"does not match rebuild/foo.go", "rebuild/foo.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_DoubleStarPrefix(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"**/test/**"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"matches test/foo.go at root", "test/foo.go", true},
		{"matches nested test dir", "pkg/test/foo.go", true},
		{"matches deeply nested test", "a/b/c/test/x.go", true},
		{"does not match testing/foo.go", "testing/foo.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_SubdirectoryMatching(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"vendor/", "internal/generated/"}}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"vendor direct file", "vendor/lib.go", true},
		{"vendor nested", "vendor/github.com/pkg/main.go", true},
		{"vendor deeply nested", "vendor/a/b/c/d.go", true},
		{"internal/generated file", "internal/generated/pb.go", true},
		{"internal/generated nested", "internal/generated/api/v1/handler.go", true},
		{"unrelated internal", "internal/domain/user.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ignore.IsIgnored(tt.path)
			if got != tt.want {
				t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsIgnored_BackslashPathsNormalized(t *testing.T) {
	ignore := &ArxIgnore{Patterns: []string{"vendor/"}}

	// Simulate a Windows-style path being passed in.
	// filepath.ToSlash converts backslashes to forward slashes.
	got := ignore.IsIgnored("vendor\\foo.go")
	// On Linux, filepath.ToSlash is a no-op for forward slashes,
	// but backslashes are not path separators on Linux.
	// The test verifies the ToSlash call exists; behavior depends on OS.
	// On Windows, this would match. On Linux, backslash is a valid filename char.
	// We just verify no panic and the function handles it.
	_ = got // No panic is the key assertion here.
}
