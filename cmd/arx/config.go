package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	configValidatePath   string
	configValidateStrict bool
	configValidateSchema bool
)

func init() {
	configValidateCmd.Flags().StringVarP(&configValidatePath, "path", "p", "", "Path to config file (default: arx.yaml)")
	configValidateCmd.Flags().BoolVarP(&configValidateStrict, "strict", "s", false, "Fail on unknown config keys")
	configValidateCmd.Flags().BoolVar(&configValidateSchema, "schema", false, "Show JSON Schema reference for config")
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}

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

	// --schema flag just prints the JSON Schema reference
	if configValidateSchema {
		fmt.Println(`{
  "$schema": "https://json-schema.org/draft-07/schema#",
  "title": "Arx Configuration",
  "type": "object",
  "properties": {
    "version": { "type": "string", "description": "Config version" },
    "layers": { "type": "array", "items": { "$ref": "#/definitions/Layer" } },
    "rules": { "type": "array", "items": { "$ref": "#/definitions/Rule" } },
    "functions": { "type": "object", "additionalProperties": { "type": "string" } },
    "cross_language": { "$ref": "#/definitions/CrossLanguage" },
    "exclude": { "type": "array", "items": { "type": "string" } },
    "max_violations": { "type": "integer", "minimum": 0 },
    "severity_mapping": { "type": "object" },
    "severity_config": { "type": "object" },
    "language_overrides": { "type": "object" }
  },
  "definitions": {
    "Layer": {
      "type": "object",
      "properties": {
        "name": { "type": "string" },
        "paths": { "type": "array", "items": { "type": "string" } },
        "description": { "type": "string" },
        "tags": { "type": "array", "items": { "type": "string" } }
      },
      "required": ["name", "paths"]
    },
    "Rule": {
      "type": "object",
      "properties": {
        "id": { "type": "string" },
        "severity": { "type": "string", "enum": ["error", "warning", "info"] },
        "from": { "type": "string" },
        "to": { "type": "array", "items": { "type": "string" } },
        "type": { "type": "string" },
        "check": { "oneOf": [{ "type": "string" }, { "type": "array", "items": { "type": "string" } }] },
        "exclude": { "type": "array", "items": { "type": "string" } },
        "overrides": { "type": "array" },
        "template": { "type": "string" },
        "params": { "type": "object" }
      },
      "required": ["id"]
    },
    "CrossLanguage": {
      "type": "object",
      "properties": {
        "mappings": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "source_pattern": { "type": "string" },
              "generated_pattern": { "type": "string" },
              "language": { "type": "string" },
              "match_strategy": { "type": "string", "enum": ["stem", "glob"] }
            }
          }
        }
      }
    }
  }
}`)
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

	// Read config
	cfg, err := reader.Read(configPath)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Error reading config: %v\n", err)
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Validate config
	if err := reader.Validate(cfg); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Invalid config: %v\n", err)
		return fmt.Errorf("invalid config: %w", err)
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
