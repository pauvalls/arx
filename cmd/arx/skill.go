package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// skillCmd represents the skill command
var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage AI coding assistant integrations",
	Long: `Manage AI coding assistant integrations for arx.

Install the arx-setup skill to AI coding tools like opencode and Claude Code,
enabling them to automatically analyze and configure arx for any project.`,
}

var skillInstallCmd = &cobra.Command{
	Use:   "install [tool...]",
	Short: "Install arx-setup skill to AI coding assistants",
	Long: `Install the arx-setup skill to one or more AI coding assistants.

The skill teaches AI assistants how to:
  - Analyze codebase structure to detect architectural layers
  - Generate arx.yaml configurations from project patterns
  - Set up cross-language dependency detection
  - Run and validate arx checks

If no tools are specified, all detected tools are shown and you can select.

Supported tools:
  - opencode   (skill: ~/.config/opencode/skills/arx-setup/)
  - claude     (skill: ~/.claude/skills/arx-setup/)
  - aider      (instructions: ~/.aider.conf.yml)
  - copilot    (instructions: .github/copilot-instructions.md)

Examples:
  arx skill install                 # Interactive: select from detected tools
  arx skill install opencode        # Install to opencode only
  arx skill install opencode claude # Install to both opencode and claude
  arx skill install --all           # Install to all detected tools`,
	Args: cobra.MaximumNArgs(8),
	RunE: runSkillInstall,
}

var skillInstallAll bool

func init() {
	skillInstallCmd.Flags().BoolVar(&skillInstallAll, "all", false, "Install to all detected tools without prompting")
	skillCmd.AddCommand(skillInstallCmd)
	rootCmd.AddCommand(skillCmd)
}

// toolInfo describes an AI coding tool and where to install skills for it.
type toolInfo struct {
	Name     string
	Binary   string // executable name to check with `which`
	SkillDir string // directory where skill files go (empty if not supported)
	Notes    string // additional notes about this tool
}

var supportedTools = []toolInfo{
	{
		Name:     "opencode",
		Binary:   "opencode",
		SkillDir: "${HOME}/.config/opencode/skills/arx-setup",
		Notes:    "Agent Skills system",
	},
	{
		Name:     "claude",
		Binary:   "claude",
		SkillDir: "${HOME}/.claude/skills/arx-setup",
		Notes:    "Claude Code skills system",
	},
	{
		Name:   "copilot",
		Binary: "",
		SkillDir: "",
		Notes:  "Project-level: .github/copilot-instructions.md",
	},
	{
		Name:   "cursor",
		Binary: "cursor",
		SkillDir: "${HOME}/.cursor/rules",
		Notes:  "Cursor AI rules system",
	},
}

// skillSourcePath returns the path to the embedded skill source files.
func skillSourcePath() string {
	// Skills are embedded in the binary or alongside the config
	// Check relative to the arx binary first, then known paths
	paths := []string{
		"contrib/opencode/arx-setup",
		"/usr/local/share/arx/arx-setup",
		"/usr/share/arx/arx-setup",
	}
	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}
	return ""
}

// findTool checks if a tool is installed and returns its info.
func findTool(name string) *toolInfo {
	for _, t := range supportedTools {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// isToolInstalled checks if a tool's binary is on PATH.
func isToolInstalled(t toolInfo) bool {
	if t.Binary == "" {
		return false
	}
	_, err := exec.LookPath(t.Binary)
	return err == nil
}

// detectTools returns all supported tools that are installed.
func detectTools() []toolInfo {
	var detected []toolInfo
	for _, t := range supportedTools {
		if isToolInstalled(t) {
			detected = append(detected, t)
		}
	}
	return detected
}

// installSkillTo copies the arx-setup skill to the tool's skill directory.
func installSkillTo(tool toolInfo) error {
	if tool.SkillDir == "" {
		return fmt.Errorf("%s does not support skill installation", tool.Name)
	}

	skillDir := os.ExpandEnv(tool.SkillDir)

	// Ensure parent directory exists
	parentDir := filepath.Dir(skillDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// Create the skill directory
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	// Write SKILL.md
	skillContent := `---
name: arx-setup
description: >
  Set up and configure arx architecture audit in any project.
  Trigger: When the user says "setup arx", "configure arx", "arx init",
  "arx setup", "analizar arquitectura", or similar.
license: Apache-2.0
metadata:
  author: arx
  version: "1.0"
---

## When to Use

- Set up arx in a new project
- Generate/regenerate arx.yaml from existing code
- Analyze project architecture with arx

## Workflow

### 1. Scan the project

` + "```" + `bash
# Detect languages and structure
ls go.mod tsconfig.json package.json Cargo.toml 2>/dev/null
# Check directory patterns
ls -d internal/*/ src/*/ cmd/ proto/ 2>/dev/null
` + "```" + `

### 2. Detect with arx

` + "```" + `bash
arx init --detect
` + "```" + `

### 3. Generate configuration

Based on what's detected, write or enhance arx.yaml:

- **Domain/Application/Infrastructure** for Clean/Hexagonal projects
- **Cross-language mappings** when proto/OpenAPI specs exist
- **Expression rules** for specific constraints (thresholds, circular deps)

### 4. Validate and run

` + "```" + `bash
arx config validate
arx check
arx baseline   # For existing codebases
` + "```" + `

## Key Config Patterns

` + "```" + `yaml
# Clean Architecture layers
layers:
  - name: domain
    paths: ["internal/domain/**"]
  - name: application
    paths: ["internal/application/**"]
  - name: infrastructure
    paths: ["internal/infrastructure/**"]

# Cross-language (proto)
cross_language:
  mappings:
    - source_pattern: "proto/**/*.proto"
      generated_pattern: "**/*.pb.go"
      language: "go"
` + "```" + `
`

	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	return nil
}

func runSkillInstall(cmd *cobra.Command, args []string) error {
	detected := detectTools()

	// If no args, use interactive selection
	if len(args) == 0 && !skillInstallAll {
		if len(detected) == 0 {
			fmt.Println("No supported AI coding assistants detected.")
			fmt.Println("Install one of: opencode, claude, cursor")
			fmt.Println()
			fmt.Println("Then run: arx skill install <tool-name>")
			return nil
		}

		fmt.Println("Detected AI coding assistants:")
		fmt.Println()
		for i, t := range detected {
			fmt.Printf("  [%d] %s", i+1, t.Name)
			if t.Notes != "" {
				fmt.Printf(" (%s)", t.Notes)
			}
			fmt.Println()
		}
		fmt.Println()
		fmt.Printf("Install to which tools? (comma-separated numbers, or 'all'): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		var selected []toolInfo
		if input == "all" {
			selected = detected
		} else {
			parts := strings.Split(input, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				var idx int
				if n, err := fmt.Sscanf(p, "%d", &idx); err == nil && n == 1 {
					if idx >= 1 && idx <= len(detected) {
						selected = append(selected, detected[idx-1])
					}
				}
			}
		}

		if len(selected) == 0 {
			fmt.Println("No tools selected.")
			return nil
		}

		for _, t := range selected {
			if err := installSkillTo(t); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", t.Name, err)
			} else {
				fmt.Printf("  ✓ %s: installed\n", t.Name)
			}
		}
		return nil
	}

	// --all flag: install to all detected
	if skillInstallAll {
		if len(detected) == 0 {
			fmt.Println("No supported AI coding assistants detected.")
			return nil
		}
		for _, t := range detected {
			if err := installSkillTo(t); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", t.Name, err)
			} else {
				fmt.Printf("  ✓ %s: installed\n", t.Name)
			}
		}
		return nil
	}

	// Specific tools requested
	for _, arg := range args {
		t := findTool(arg)
		if t == nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: unknown tool (supported: opencode, claude, cursor, copilot)\n", arg)
			continue
		}
		if err := installSkillTo(*t); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", t.Name, err)
		} else {
			fmt.Printf("  ✓ %s: installed\n", t.Name)
		}
	}

	return nil
}
