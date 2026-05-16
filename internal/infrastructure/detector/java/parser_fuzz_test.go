package java

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzJavaParse(f *testing.F) {
	// Seed corpus with valid Java import/package lines
	f.Add([]byte("import java.util.List;"))
	f.Add([]byte("import static java.lang.Math.PI;"))
	f.Add([]byte("package com.example;"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.java")
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
