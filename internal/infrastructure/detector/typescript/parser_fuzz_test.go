package typescript_detector

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzTypeScriptDetector(f *testing.F) {
	seeds := []string{
		"import { User } from './user';\n",
		"import express from 'express';\n",
		"const fs = require('fs');\n",
		"import './styles.css';\n",
		"import type { FC } from 'react';\n",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, content string) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.ts")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return
		}
		d := New()
		// Should never panic
		deps, err := d.extractFileImports(filePath, tmpDir, nil)
		if err != nil {
			return // Parse errors expected
		}
		_ = deps
	})
}
