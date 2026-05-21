package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage arx configuration",
	Long:  `Commands for managing arx configuration files.`,
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate arx configuration file",
	Long: `Validate an arx configuration file.

If no path is specified, looks for arx.yaml in the current directory.

Exit codes:
  0 - Config is valid
  1 - Config is invalid or error occurred

Examples:
  arx config validate                    # Validate arx.yaml in current dir
  arx config validate ./custom.yaml      # Validate specific file
  arx config validate --path ./arx.yaml  # Using --path flag`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigValidate,
}

var (
	configValidatePath     string
	configValidateStrict   bool
	configValidateSchema   bool
	configValidateOverride string
)



// knownConfigKeys lists all top-level keys understood by arx config.
var knownConfigKeys = map[string]bool{
	"version": true, "layers": true, "rules": true,
	"language_overrides": true, "exclude": true, "severity_config": true,
	"max_violations": true, "severity_mapping": true, "functions": true,
	"cross_language": true, "$schema": true,
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Determine config path
	configPath := configValidatePath
	if configPath == "" {
		if len(args) > 0 {
			configPath = args[0]
		} else {
			configPath = "arx.yaml"
		}
	}

	// Resolve to absolute path if relative
	if !filepath.IsAbs(configPath) {
		var err error
		configPath, err = filepath.Abs(configPath)
		if err != nil {
			return fmt.Errorf("invalid path %q: %w", configPath, err)
		}
	}

	// --schema flag generates and prints the JSON Schema from domain.Config
	if configValidateSchema {
		gen := &config.SchemaGeneratorImpl{}
		schema, err := gen.Generate("arx-config", domain.Config{})
		if err != nil {
			return fmt.Errorf("generating schema: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(schema))
		return nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Config file not found: %s\n", configPath)
		return fmt.Errorf("config file not found: %s", configPath)
	}

	// Strict mode: check for unknown keys before reading
	if configValidateStrict {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", configPath, err)
		}

		var doc map[string]interface{}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("invalid YAML: %w", err)
		}

		var unknownKeys []string
		for key := range doc {
			if !knownConfigKeys[key] {
				unknownKeys = append(unknownKeys, key)
			}
		}
		if len(unknownKeys) > 0 {
			for _, key := range unknownKeys {
				fmt.Fprintf(cmd.ErrOrStderr(), "✗ Unknown config key: %q\n", key)
			}
			return fmt.Errorf("config has %d unknown key(s): %s", len(unknownKeys), strings.Join(unknownKeys, ", "))
		}
	}

	// Create config reader
	reader := config.NewYAMLReader()

	// Read base config through the pipeline
	cfg, err := reader.Read(configPath)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Error reading config: %v\n", err)
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Apply override if specified
	if configValidateOverride != "" {
		// Resolve override path
		overridePath := configValidateOverride
		if !filepath.IsAbs(overridePath) {
			// Resolve relative to the base config's directory
			overridePath = filepath.Join(filepath.Dir(configPath), overridePath)
		}

		// Read override file raw bytes (not parsed — merge raw YAML to avoid zero-value pollution)
		overrideData, err := os.ReadFile(overridePath)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "✗ Error reading override config: %v\n", err)
			return fmt.Errorf("failed to read override config: %w", err)
		}

		// Marshal parsed base config to YAML for merging
		baseYAML, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshaling base config: %w", err)
		}

		// Deep merge (only keys present in override replace base)
		mergedYAML, err := config.DeepMerge(baseYAML, overrideData)
		if err != nil {
			return fmt.Errorf("failed to merge configs: %w", err)
		}

		// Re-apply pipeline to merged config (for any new env vars/includes from override)
		configDir := filepath.Dir(configPath)
		mergedYAML, err = config.InterpolateEnvVars(mergedYAML)
		if err != nil {
			return fmt.Errorf("interpolating env vars in merged config: %w", err)
		}
		mergedYAML, err = config.ResolveIncludes(configDir, mergedYAML)
		if err != nil {
			return fmt.Errorf("resolving includes in merged config: %w", err)
		}
		mergedYAML, err = config.InterpolateEnvVars(mergedYAML)
		if err != nil {
			return fmt.Errorf("interpolating env vars in merged config: %w", err)
		}

		// Re-parse the merged config
		var mergedCfg domain.Config
		if err := yaml.Unmarshal(mergedYAML, &mergedCfg); err != nil {
			return fmt.Errorf("parsing merged config: %w", err)
		}
		cfg = &mergedCfg
	}

	// Validate config
	if err := reader.Validate(cfg); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Invalid config: %v\n", err)
		return fmt.Errorf("invalid config: %w", err)
	}

	// Check for deprecated fields
	warnings := domain.CheckDeprecated(cfg)
	for _, w := range warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ WARNING: %s\n", w)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Config valid: %s\n", configPath)
	return nil
}

// resolvePath splits a dotted key into path segments and validates them.
func resolvePath(key string) ([]string, error) {
	if key == "" {
		return nil, fmt.Errorf("key cannot be empty")
	}
	parts := strings.Split(key, ".")
	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("empty segment in key %q", key)
		}
	}
	return parts, nil
}

// parseValue tries to parse raw input as JSON first; falls back to raw string.
func parseValue(raw string) (interface{}, error) {
	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed, nil
	}
	// Fallback: treat as plain string
	return raw, nil
}

// setAtPath navigates doc through path segments, creating intermediate maps as needed,
// and sets value at the leaf.
func setAtPath(doc map[string]interface{}, path []string, value interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("path cannot be empty")
	}

	current := doc
	for _, segment := range path[:len(path)-1] {
		next, ok := current[segment]
		if !ok {
			// Create intermediate map
			next = make(map[string]interface{})
			current[segment] = next
		}
		m, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot traverse into %q: not a map", segment)
		}
		current = m
	}

	leaf := path[len(path)-1]
	current[leaf] = value
	return nil
}

// getAtPath navigates doc through path segments and returns the value at the leaf.
func getAtPath(doc map[string]interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("path cannot be empty")
	}

	var current interface{} = doc
	for _, segment := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("cannot traverse into %q: not a map", segment)
		}
		val, exists := m[segment]
		if !exists {
			return nil, fmt.Errorf("key %q not found", segment)
		}
		current = val
	}
	return current, nil
}

// configGetCmd represents the config get command
var configGetCmd = &cobra.Command{
	Use:   "get <field>",
	Short: "Get a configuration value",
	Long:  `Read a field from arx.yaml. Supports nested keys with dot notation.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := config.NewYAMLReader()
		cfg, err := reader.Read("arx.yaml")
		if err != nil {
			return fmt.Errorf("failed to read config: %w", err)
		}

		// Serialize the typed config back to a generic map for path resolution
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		field := args[0]
		path, err := resolvePath(field)
		if err != nil {
			return fmt.Errorf("invalid key: %w", err)
		}

		value, err := getAtPath(doc, path)
		if err != nil {
			return fmt.Errorf("unknown field: %s", field)
		}

		// YAML-marshal complex types for readable display
		switch v := value.(type) {
		case map[string]interface{}, []interface{}:
			out, err := yaml.Marshal(v)
			if err != nil {
				return fmt.Errorf("failed to marshal value: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(out))
		default:
			fmt.Fprintln(cmd.OutOrStdout(), value)
		}
		return nil
	},
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set <field> <value>",
	Short: "Set a configuration value",
	Long:  `Update a field in arx.yaml. Supports dotted paths (e.g. severity_mapping.critical) and JSON array values.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		field, rawValue := args[0], args[1]

		data, err := os.ReadFile("arx.yaml")
		if err != nil {
			return fmt.Errorf("failed to read arx.yaml: %w", err)
		}

		var doc map[string]interface{}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("invalid config: %w", err)
		}

		path, err := resolvePath(field)
		if err != nil {
			return fmt.Errorf("invalid key: %w", err)
		}

		value, err := parseValue(rawValue)
		if err != nil {
			return fmt.Errorf("failed to parse value: %w", err)
		}

		if err := setAtPath(doc, path, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", field, err)
		}

		out, err := yaml.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile("arx.yaml", out, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ %s set to %s\n", field, rawValue)
		return nil
	},
}

// configMigrateCmd represents the config migrate command
var configMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate config to a new schema version",
	Long: `Migrate the arx.yaml configuration file to a new schema version.

If --to is not specified, migrates to the latest supported version.
Use --dry-run to preview changes without modifying files.
The --backup flag (default: true) creates arx.yaml.bak before modifying.

Examples:
  arx config migrate                    # Auto-detect, migrate to latest
  arx config migrate --to "2.0"         # Migrate to specific version
  arx config migrate --dry-run          # Show what would change
  arx config migrate --no-backup        # Skip backup creation`,
	RunE: runConfigMigrate,
}

var (
	configMigrateTo      string
	configMigrateDryRun  bool
	configMigrateNoBackup bool
)

func init() {
	configValidateCmd.Flags().StringVarP(&configValidatePath, "path", "p", "", "Path to config file (default: arx.yaml)")
	configValidateCmd.Flags().BoolVarP(&configValidateStrict, "strict", "s", false, "Fail on unknown config keys")
	configValidateCmd.Flags().BoolVar(&configValidateSchema, "schema", false, "Show JSON Schema reference for config")
	configValidateCmd.Flags().StringVar(&configValidateOverride, "override", "", "Path to override YAML config (deep-merged into base)")
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configMigrateCmd.Flags().StringVar(&configMigrateTo, "to", "", "Target schema version (default: latest)")
	configMigrateCmd.Flags().BoolVar(&configMigrateDryRun, "dry-run", false, "Show changes without modifying files")
	configMigrateCmd.Flags().BoolVar(&configMigrateNoBackup, "no-backup", false, "Skip backup creation")
	configCmd.AddCommand(configMigrateCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigMigrate(cmd *cobra.Command, args []string) error {
	configPath := "arx.yaml"
	if len(args) > 0 {
		configPath = args[0]
	}

	if !filepath.IsAbs(configPath) {
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}
		configPath = abs
	}

	// Build registry with default migrations
	reg := domain.NewRegistry()
	for _, m := range application.DefaultMigrationFuncs {
		if err := reg.Register(m); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to register migration %s→%s: %v\n", m.From, m.To, err)
		}
	}

	svc := application.NewMigrateService(reg)

	// Determine target version
	var toVersion domain.SchemaVersion
	if configMigrateTo != "" {
		var err error
		toVersion, err = domain.ParseSchemaVersion(configMigrateTo)
		if err != nil {
			return fmt.Errorf("invalid target version %q: %w", configMigrateTo, err)
		}
	} else {
		// Auto-detect: migrate to latest (highest To version in registry)
		toVersion = domain.SchemaVersion{Major: 2, Minor: 0} // Default latest
	}

	dryRun := configMigrateDryRun

	result, err := svc.Migrate(configPath, toVersion, dryRun)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Print result
	for _, step := range result.Steps {
		fmt.Fprintln(cmd.OutOrStdout(), step)
	}

	if result.DryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run — no files modified.")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Migration complete: %s → %s\n", result.From, result.To)
		if result.BackupPath != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Backup created: %s\n", result.BackupPath)
		}
	}

	return nil
}
