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
	f.Add([]byte(`version: "1.0"` + "\nlayers:\n  - name: domain\n    paths: [./domain]\n  - name: infra\n    paths: [./infra]\nrules: []\n"))
	f.Add([]byte(`version: "1.0"` + "\nlayers:\n  - name: app\n    paths: [app/**]\nrules:\n  - id: no-infra\n    template: max-deps\n    severity: error\n    params:\n      from: app\n      to: [infra]\n      max: 0\n"))
	f.Add([]byte("severity_mapping:\n  critical: error\n  minor: warning\n"))
	f.Add([]byte(`version: "1.0"` + "\nlayers:\n  - name: presentation\n    paths: [src/presentation/**]\nrules:\n  - id: R1\n    from: presentation\n    to: [infrastructure]\n    type: Cannot\n    severity: error\n    exclude: [legacy/**]\n"))
	f.Add([]byte(`version: "1.0"` + "\nlayers:\n  - name: domain\n    paths: [domain/**]\nrules: []\ncross_language:\n  mappings:\n    - source_pattern: \"**/*.proto\"\n      generated_pattern: \"**/generated/*.pb.go\"\n      language: go\n"))

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
