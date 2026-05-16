package rust

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzRustParse(f *testing.F) {
	// Seed corpus with valid Rust use statements
	f.Add([]byte("use std::collections::HashMap;"))
	f.Add([]byte("use crate::domain::Entity;"))
	f.Add([]byte("pub use self::helper::format;"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.rs")
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
