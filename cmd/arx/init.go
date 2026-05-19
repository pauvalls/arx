package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pauvalls/arx/internal/application"
	"github.com/pauvalls/arx/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Initialize Arx configuration for a project",
	Long: `Initialize Arx configuration for a project by scanning the directory
structure, detecting languages, and generating an arx.yaml file with
sensible defaults.

If no path is provided, the current directory is used.

The generated configuration includes:
  - Detected layers based on directory structure
  - Default architectural rules (domain cannot depend on infrastructure, etc.)
  - Language-specific overrides for Go and TypeScript
  - Common exclude patterns (vendor, node_modules, etc.)

Example:
  arx init                    # Initialize in current directory
  arx init ./my-project       # Initialize in specific directory
  arx init --output config/arx.yaml  # Write to custom path`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initOutput string
	initForce  bool
	initPreset string
	initDetect bool
)

func init() {
	initCmd.Flags().StringVarP(&initOutput, "output", "o", "arx.yaml", "Output file path for the generated configuration")
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing configuration file")
	initCmd.Flags().StringVarP(&initPreset, "preset", "p", "", "Use preset template (clean, hexagonal, ddd, layered, onion)")
	initCmd.Flags().BoolVarP(&initDetect, "detect", "d", false, "Scan project and show detected configuration (dry run, no file written)")
	rootCmd.AddCommand(initCmd)
}

// isDefaultConfigPath returns true when the output path is the default arx.yaml
// or a path ending in "/arx.yaml", indicating the schema reference should be injected.
func isDefaultConfigPath(path string) bool {
	base := filepath.Base(path)
	return base == "arx.yaml"
}

// ensureGitignoreEntries appends arx-specific entries to .gitignore if the
// project is a git repository and the entries are not already present.
// It is a no-op outside git repos.
func ensureGitignoreEntries(projectRoot string) error {
	gitDir := filepath.Join(projectRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil // Not a git repo, skip
	}

	entries := []string{".arx-cache/", ".arx-history/", ".arx-baseline-history/"}
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	// Read existing content
	var existing string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	}

	// Check which entries are missing
	var missing []string
	for _, entry := range entries {
		if !containsGitignoreEntry(existing, entry) {
			missing = append(missing, entry)
		}
	}

	if len(missing) == 0 {
		return nil // All entries already present
	}

	// Build content to append
	var buf strings.Builder
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		buf.WriteString("\n")
	}
	buf.WriteString("\n# Arx\n")
	for _, entry := range missing {
		buf.WriteString(entry + "\n")
	}

	// Append to .gitignore (create if not exists)
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(buf.String()); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	return nil
}

// containsGitignoreEntry checks if a .gitignore content string already contains
// the given entry (exact line match, ignoring comments and blank lines).
func containsGitignoreEntry(content, entry string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Skip comments and blank lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == entry {
			return true
		}
	}
	return false
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine project root
	projectRoot := "."
	if len(args) > 0 {
		projectRoot = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", projectRoot, err)
	}
	projectRoot = absPath

	// Check if project exists
	if _, err := os.Stat(projectRoot); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", projectRoot)
	}

	// Check if output file already exists
	outputPath := initOutput
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(projectRoot, initOutput)
	}

	// Create service
	service := newInitService()

	// Detect mode: scan and show without writing
	if initDetect {
		info, err := service.Scan(projectRoot)
		if err != nil {
			return fmt.Errorf("failed to scan project: %w", err)
		}

		cfg, err := service.Generate(info)
		if err != nil {
			return fmt.Errorf("failed to generate config: %w", err)
		}

		fmt.Printf("Project: %s\n", projectRoot)
		fmt.Printf("Languages: %s\n", strings.Join(info.Languages, ", "))
		fmt.Println()

		// Import analysis
		fmt.Println("─── Import Analysis ───")
		importSummary, err := application.ScanImports(projectRoot, cfg.Layers)
		if err == nil && importSummary != nil {
			fmt.Println(importSummary.FormatSummary())
		}
		fmt.Println()

		// Detected layers
		fmt.Println("─── Detected Layers ───")
		fmt.Printf("  %d layer(s) detected\n", len(cfg.Layers))
		for _, layer := range cfg.Layers {
			fmt.Printf("  • %s: %s\n", layer.Name, strings.Join(layer.Paths, ", "))
		}
		fmt.Println()

		// Generated rules
		fmt.Println("─── Generated Rules ───")
		fmt.Printf("  %d rule(s)\n", len(cfg.Rules))
		for _, rule := range cfg.Rules {
			if rule.Check.Raw != "" {
				fmt.Printf("  • %s: check: %s (%s)\n", rule.ID, rule.Check.Raw, rule.Severity)
			} else {
				to := strings.Join(rule.To, ", ")
				fmt.Printf("  • %s: %s → [%s] (%s)\n", rule.ID, rule.From, to, rule.Severity)
			}
		}
		fmt.Println()

		// Generated YAML
		fmt.Println("─── Generated Configuration ───")
		fmt.Println()
		yamlBytes, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		fmt.Println(string(yamlBytes))

		return nil
	}

	var config *domain.Config
	var initErr error
	if initPreset != "" {
		// Validate preset name
		validPresets := []string{"clean", "hexagonal", "ddd", "layered", "onion"}
		isValid := false
		for _, p := range validPresets {
			if p == initPreset {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("unknown preset %q, available presets: %s", initPreset, strings.Join(validPresets, ", "))
		}

		config, initErr = service.InitWithPreset(initPreset, outputPath, initForce)
		if initErr != nil {
			return fmt.Errorf("failed to initialize with preset: %w", initErr)
		}
	} else {
		config, initErr = service.Init(projectRoot, outputPath)
		if initErr != nil {
			return fmt.Errorf("failed to initialize: %w", initErr)
		}
	}

	// Inject $schema reference into the written arx.yaml when using default config path
	if isDefaultConfigPath(initOutput) {
		if err := injectSchemaField(outputPath, "./arx-schema.json"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to add $schema to config: %v\n", err)
		}
	}

	// Print success message
	fmt.Printf("✓ Written to %s\n", outputPath)
	fmt.Printf("  Detected %d layer(s): ", len(config.Layers))
	for i, layer := range config.Layers {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(layer.Name)
	}
	fmt.Println()
	fmt.Printf("  Generated %d rule(s)\n", len(config.Rules))

	fmt.Println()
	fmt.Println("Review and adjust the configuration before running 'arx check'.")

	// Manage .gitignore entries (only in git repos)
	if err := ensureGitignoreEntries(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update .gitignore: %v\n", err)
	}

	return nil
}

// injectSchemaField reads a YAML file and prepends the $schema field.
// Uses simple string insertion to avoid YAML round-trip issues.
func injectSchemaField(filePath, schemaURL string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Prepend $schema line after any leading comments/blank lines
	lines := strings.Split(string(data), "\n")
	var result []string
	injected := false
	for _, line := range lines {
		if !injected && line != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			result = append(result, fmt.Sprintf("$schema: %s", schemaURL))
			injected = true
		}
		result = append(result, line)
	}
	if !injected {
		result = append([]string{fmt.Sprintf("$schema: %s", schemaURL)}, result...)
	}

	return os.WriteFile(filePath, []byte(strings.Join(result, "\n")), 0644)
}
