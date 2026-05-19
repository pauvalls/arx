package kotlin

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzKotlinParse(f *testing.F) {
	// Seed corpus with valid Kotlin import statements
	f.Add([]byte("import java.util.List"))
	f.Add([]byte("import com.example.domain.*"))
	f.Add([]byte("package com.example"))
	f.Add([]byte("import kotlinx.coroutines.*\nimport kotlinx.coroutines.flow.Flow"))
	f.Add([]byte("import com.example.dto as DTO"))
	f.Add([]byte("import java.util.*\nimport kotlin.collections.List"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.kt")
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
