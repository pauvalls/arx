package csharp

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzCSharpParse(f *testing.F) {
	// Seed corpus with valid C# using directives
	f.Add([]byte("using System;"))
	f.Add([]byte("using static System.Math;"))
	f.Add([]byte("using Alias = Namespace.Class;"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.cs")
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
