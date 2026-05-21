package application

import (
	"fmt"
	"os"

	"github.com/pauvalls/arx/internal/domain"
	"gopkg.in/yaml.v3"
)

// DefaultMigrationFuncs returns a list of default migrations to register.
// Extended when new schema versions are introduced.
var DefaultMigrationFuncs = []domain.Migration{} // Initially empty — migrations added when schema changes

// MigrateResult holds the outcome of a migration operation.
type MigrateResult struct {
	From        domain.SchemaVersion
	To          domain.SchemaVersion
	Steps       []string
	BackupPath  string
	DryRun      bool
}

// MigrateService handles config file schema migrations.
type MigrateService struct {
	registry *domain.Registry
}

// NewMigrateService creates a new migration service.
func NewMigrateService(registry *domain.Registry) *MigrateService {
	return &MigrateService{registry: registry}
}

// Migrate migrates a config file from its current version to the target version.
// If dryRun is true, the migration is simulated without writing to disk.
// Backup is only created when dryRun is false.
func (s *MigrateService) Migrate(path string, toVersion domain.SchemaVersion, dryRun bool) (*MigrateResult, error) {
	// Read current config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Detect current version
	fromVersion, err := DetectVersion(data)
	if err != nil {
		return nil, fmt.Errorf("detecting version: %w", err)
	}

	// Already at target?
	if fromVersion.Compare(toVersion) == 0 {
		return &MigrateResult{
			From:   fromVersion,
			To:     toVersion,
			Steps:  []string{fmt.Sprintf("already at version %s", fromVersion)},
			DryRun: dryRun,
		}, nil
	}

	// Resolve migration path
	funcs, err := s.registry.Resolve(fromVersion, toVersion)
	if err != nil {
		return nil, fmt.Errorf("resolving migration path: %w", err)
	}

	// Apply each migration step
	currentData := data
	steps := make([]string, 0, len(funcs))

	for i, fn := range funcs {
		output, err := fn(currentData)
		if err != nil {
			return nil, fmt.Errorf("migration step %d failed: %w", i+1, err)
		}
		currentData = output
		// Re-detect version after migration to track progress
		newVer, err := DetectVersion(output)
		if err == nil {
			steps = append(steps, fmt.Sprintf("migrated %s→%s", fromVersion, newVer))
		}
	}

	result := &MigrateResult{
		From:   fromVersion,
		To:     toVersion,
		Steps:  steps,
		DryRun: dryRun,
	}

	if dryRun {
		return result, nil
	}

	// Create backup
	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return nil, fmt.Errorf("creating backup: %w", err)
	}
	result.BackupPath = backupPath

	// Write migrated config
	if err := os.WriteFile(path, currentData, 0644); err != nil {
		return nil, fmt.Errorf("writing migrated config: %w", err)
	}

	return result, nil
}

// DetectVersion reads YAML bytes and extracts the schema version.
func DetectVersion(data []byte) (domain.SchemaVersion, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return domain.SchemaVersion{}, fmt.Errorf("parsing YAML: %w", err)
	}

	verStr, ok := doc["version"].(string)
	if !ok || verStr == "" {
		return domain.SchemaVersion{}, fmt.Errorf("config version not found or not a string")
	}

	return domain.ParseSchemaVersion(verStr)
}


