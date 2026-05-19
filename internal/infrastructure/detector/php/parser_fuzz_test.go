package php

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzPHPParse(f *testing.F) {
	// Seed corpus with valid PHP use statements
	f.Add([]byte("use App\\Domain\\Order;"))
	f.Add([]byte("use App\\Infra\\Repo as RepoInterface;"))
	f.Add([]byte("use function App\\Helpers\\format;"))
	f.Add([]byte("use App\\Models\\{User, Order, Product};"))
	f.Add([]byte("use App\\Domain\\Order;\nuse App\\Domain\\Customer;\nuse App\\Infra\\Repository;"))
	f.Add([]byte("use const App\\Config\\MAX_RETRIES;"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tmpDir := t.TempDir()
		srcPath := filepath.Join(tmpDir, "Test.php")
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
