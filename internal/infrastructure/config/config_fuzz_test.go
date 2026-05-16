package config

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzConfigParse(f *testing.F) {
	// Seed corpus with valid configs
	f.Add([]byte(`version: "1.0"` + "\nlayers:\n  - name: domain\n    paths: [./domain]\n"))
	f.Add([]byte(`version: "1.0"`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "arx.yaml")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return
		}

		reader := NewYAMLReader()
		cfg, err := reader.Read(configPath)
		if err != nil {
			// Expected for most random input
			return
		}
		_ = reader.Validate(cfg)
	})
}
