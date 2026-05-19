package swift

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzSwiftParse(f *testing.F) {
	// Seed corpus with valid Swift import statements
	f.Add([]byte("import Foundation"))
	f.Add([]byte("import struct SwiftUI.View"))
	f.Add([]byte("@_exported import UIKit"))
	f.Add([]byte("import UIKit\nimport SwiftUI\nimport Combine"))
	f.Add([]byte("import class UIKit.UIView\nimport protocol SwiftUI.View"))
	f.Add([]byte("@testable import MyApp"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.swift")
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
