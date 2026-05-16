package ruby

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzRubyParse(f *testing.F) {
	// Seed corpus with valid Ruby require statements
	f.Add([]byte("require 'json'"))
	f.Add([]byte("require_relative 'lib/domain/order'"))
	f.Add([]byte("require_all 'lib/'"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.rb")
		if err := os.WriteFile(srcPath, data, 0644); err != nil {
			return
		}

		content, err := os.ReadFile(srcPath)
		if err != nil {
			return
		}

		// extractImportsFromLine never panics — just returns []string
		_ = extractImportsFromLine(string(content))
	})
}
