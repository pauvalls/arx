package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
	presetpkg "github.com/pauvalls/arx/internal/application/presets"
	"gopkg.in/yaml.v3"
)

// ProjectInfo holds information about a detected project
type ProjectInfo struct {
	Root           string
	Languages      []string
	SuggestedLayers []domain.Layer
}

// ScanProject analyzes a project root to detect languages and infer layer structure.
// It recognizes common architectural conventions (Clean Architecture, Hexagonal, DDD).
func ScanProject(projectRoot string) (*ProjectInfo, error) {
	info := &ProjectInfo{
		Root:      projectRoot,
		Languages: []string{},
	}

	// Detect Go projects
	goModPath := filepath.Join(projectRoot, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		info.Languages = append(info.Languages, "go")
	}

	// Detect TypeScript projects
	tsConfigPath := filepath.Join(projectRoot, "tsconfig.json")
	packageJSONPath := filepath.Join(projectRoot, "package.json")
	_, tsConfigExists := os.Stat(tsConfigPath)
	_, packageJSONExists := os.Stat(packageJSONPath)
	if tsConfigExists == nil || packageJSONExists == nil {
		info.Languages = append(info.Languages, "typescript")
	}

	// Infer layer structure from common conventions
	info.SuggestedLayers = inferLayers(projectRoot, info.Languages)

	return info, nil
}

// inferLayers detects common architectural patterns from directory structure.
func inferLayers(projectRoot string, languages []string) []domain.Layer {
	var layers []domain.Layer

	// Common layer patterns across architectures
	patterns := []struct {
		name        string
		paths       []string
		description string
		tags        []string
	}{
		{
			name:        "domain",
			paths:       []string{"internal/domain/**", "src/domain/**", "domain/**", "core/**", "entities/**"},
			description: "Domain layer containing business logic, entities, and domain services",
			tags:        []string{"clean-architecture", "hexagonal", "ddd"},
		},
		{
			name:        "application",
			paths:       []string{"internal/application/**", "src/application/**", "application/**", "usecases/**", "services/**", "app/**"},
			description: "Application layer orchestrating domain operations and use cases",
			tags:        []string{"clean-architecture", "hexagonal"},
		},
		{
			name:        "infrastructure",
			paths:       []string{"internal/infrastructure/**", "src/infrastructure/**", "infrastructure/**", "infra/**", "adapters/**", "persistence/**"},
			description: "Infrastructure layer implementing external concerns and adapters",
			tags:        []string{"clean-architecture", "hexagonal"},
		},
		{
			name:        "presentation",
			paths:       []string{"internal/presentation/**", "src/presentation/**", "presentation/**", "api/**", "handlers/**", "controllers/**", "cmd/**", "cli/**"},
			description: "Presentation layer handling HTTP, CLI, and UI concerns",
			tags:        []string{"clean-architecture"},
		},
		{
			name:        "ports",
			paths:       []string{"internal/ports/**", "src/ports/**", "ports/**"},
			description: "Ports (interfaces) defining contracts between layers",
			tags:        []string{"hexagonal"},
		},
	}

	// Check which directories actually exist
	for _, pattern := range patterns {
		var existingPaths []string
		for _, p := range pattern.paths {
			// Convert glob to actual directory path for existence check
			dirPath := strings.ReplaceAll(p, "/**", "")
			fullPath := filepath.Join(projectRoot, dirPath)
			if stat, err := os.Stat(fullPath); err == nil && stat.IsDir() {
				existingPaths = append(existingPaths, p)
			}
		}

		if len(existingPaths) > 0 {
			layers = append(layers, domain.Layer{
				Name:        pattern.name,
				Paths:       existingPaths,
				Description: pattern.description,
				Tags:        pattern.tags,
			})
		}
	}

	return layers
}

// GenerateConfig creates a sensible default configuration based on detected project info.
// It generates 5+ default rules covering common architectural constraints.
func GenerateConfig(projectInfo *ProjectInfo) (*domain.Config, error) {
	if projectInfo == nil {
		return nil, fmt.Errorf("project info is nil")
	}

	config := &domain.Config{
		Version: "1.0",
		Layers:  projectInfo.SuggestedLayers,
	}

	// If no layers were detected, provide sensible defaults based on languages
	if len(config.Layers) == 0 {
		config.Layers = defaultLayers(projectInfo.Languages)
	}

	// Generate default rules
	config.Rules = generateDefaultRules(config.Layers)

	// Add language-specific overrides
	config.LanguageOverrides = generateLanguageOverrides(projectInfo.Languages)

	// Add default excludes
	config.Exclude = []string{
		"vendor/**",
		"node_modules/**",
		".git/**",
		"dist/**",
		"build/**",
	}

	// Validate the generated config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("generated config is invalid: %w", err)
	}

	return config, nil
}

// defaultLayers provides fallback layer definitions when no structure is detected.
func defaultLayers(languages []string) []domain.Layer {
	layers := []domain.Layer{
		{
			Name:        "domain",
			Paths:       []string{"internal/domain/**", "domain/**"},
			Description: "Domain layer containing business logic and entities",
			Tags:        []string{"core"},
		},
		{
			Name:        "application",
			Paths:       []string{"internal/application/**", "application/**"},
			Description: "Application layer orchestrating use cases",
			Tags:        []string{"orchestration"},
		},
		{
			Name:        "infrastructure",
			Paths:       []string{"internal/infrastructure/**", "infrastructure/**"},
			Description: "Infrastructure layer implementing external adapters",
			Tags:        []string{"adapters"},
		},
	}

	// Add presentation layer for web projects
	hasWeb := false
	for _, lang := range languages {
		if lang == "typescript" || lang == "go" {
			hasWeb = true
			break
		}
	}
	if hasWeb {
		layers = append(layers, domain.Layer{
			Name:        "presentation",
			Paths:       []string{"cmd/**", "api/**", "handlers/**"},
			Description: "Presentation layer handling HTTP and CLI entry points",
			Tags:        []string{"entry-point"},
		})
	}

	return layers
}

// generateDefaultRules creates the core architectural rules for the detected layers.
func generateDefaultRules(layers []domain.Layer) []domain.Rule {
	var rules []domain.Rule

	// Build a set of layer names for quick lookup
	layerNames := make(map[string]bool)
	for _, layer := range layers {
		layerNames[layer.Name] = true
	}

	// Rule 1: Domain cannot depend on infrastructure
	if layerNames["domain"] && layerNames["infrastructure"] {
		rules = append(rules, domain.Rule{
			ID:          "domain-imports-infrastructure",
			From:        "domain",
			To:          []string{"infrastructure"},
			Type:        domain.RuleTypeCannot,
			Severity:    domain.SeverityError,
			Explanation: GetExplanation("domain-imports-infrastructure"),
		})
	}

	// Rule 2: Domain cannot depend on application
	if layerNames["domain"] && layerNames["application"] {
		rules = append(rules, domain.Rule{
			ID:          "domain-imports-application",
			From:        "domain",
			To:          []string{"application"},
			Type:        domain.RuleTypeCannot,
			Severity:    domain.SeverityError,
			Explanation: GetExplanation("domain-imports-application"),
		})
	}

	// Rule 3: Application cannot depend on infrastructure
	if layerNames["application"] && layerNames["infrastructure"] {
		rules = append(rules, domain.Rule{
			ID:          "application-imports-infrastructure",
			From:        "application",
			To:          []string{"infrastructure"},
			Type:        domain.RuleTypeCannot,
			Severity:    domain.SeverityError,
			Explanation: GetExplanation("application-imports-infrastructure"),
		})
	}

	// Rule 4: Presentation cannot depend on infrastructure
	if layerNames["presentation"] && layerNames["infrastructure"] {
		rules = append(rules, domain.Rule{
			ID:          "presentation-imports-infrastructure",
			From:        "presentation",
			To:          []string{"infrastructure"},
			Type:        domain.RuleTypeCannot,
			Severity:    domain.SeverityWarning,
			Explanation: GetExplanation("presentation-imports-infrastructure"),
		})
	}

	// Rule 5: Presentation cannot depend on domain directly
	if layerNames["presentation"] && layerNames["domain"] {
		rules = append(rules, domain.Rule{
			ID:          "presentation-imports-domain",
			From:        "presentation",
			To:          []string{"domain"},
			Type:        domain.RuleTypeCannot,
			Severity:    domain.SeverityWarning,
			Explanation: GetExplanation("presentation-imports-domain"),
		})
	}

	// Rule 6: No circular dependencies between layers
	if len(layers) >= 2 {
		// Only reference layers that actually exist in the config
		var otherLayers []string
		for _, l := range layers {
			if l.Name != "domain" {
				otherLayers = append(otherLayers, l.Name)
			}
		}
		rules = append(rules, domain.Rule{
			ID:          "layer-circular",
			From:        "domain",
			To:          otherLayers,
			Type:        domain.RuleTypeMustNotCircular,
			Severity:    domain.SeverityError,
			Explanation: GetExplanation("layer-circular"),
		})
	}

	// Rule 7: Application can depend on domain (informational)
	if layerNames["application"] && layerNames["domain"] {
		rules = append(rules, domain.Rule{
			ID:       "application-imports-domain",
			From:     "application",
			To:       []string{"domain"},
			Type:     domain.RuleTypeCan,
			Severity: domain.SeverityInfo,
			Explanation: "Application services coordinate domain operations. This is the expected flow.",
		})
	}

	// Rule 8: Infrastructure can depend on domain (informational)
	if layerNames["infrastructure"] && layerNames["domain"] {
		rules = append(rules, domain.Rule{
			ID:       "infrastructure-imports-domain",
			From:     "infrastructure",
			To:       []string{"domain"},
			Type:     domain.RuleTypeCan,
			Severity: domain.SeverityInfo,
			Explanation: "Infrastructure adapters implement interfaces defined by the domain. This is correct.",
		})
	}

	return rules
}

// generateLanguageOverrides creates language-specific configuration overrides.
func generateLanguageOverrides(languages []string) map[string]domain.LanguageOverride {
	overrides := make(map[string]domain.LanguageOverride)

	for _, lang := range languages {
		switch lang {
		case "go":
			overrides["go"] = domain.LanguageOverride{
				Extensions: []string{".go"},
				Comment:    "//",
				Import:     "import",
			}
		case "typescript":
			overrides["typescript"] = domain.LanguageOverride{
				Extensions: []string{".ts", ".tsx"},
				Comment:    "//",
				Import:     "import",
			}
		}
	}

	return overrides
}

// WriteConfig serializes a Config to YAML and writes it using the provided FileWriter.
func WriteConfig(config *domain.Config, outputPath string, writer ports.FileWriter) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Add a header comment with preset info if applicable
	header := "# Arx Architecture Configuration\n# Generated automatically — edit to customize\n\n"
	content := append([]byte(header), yamlBytes...)

	// Write using the port interface
	if err := writer.Write(outputPath, content); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GenerateConfigWithPreset creates a configuration from a preset template.
// If presetName is empty, falls back to GenerateConfig based on project detection.
func GenerateConfigWithPreset(projectInfo *ProjectInfo, presetName string) (*domain.Config, error) {
	if presetName == "" {
		// No preset specified, use existing detection-based logic
		return GenerateConfig(projectInfo)
	}

	// Load preset template
	template, err := presetpkg.LoadPreset(presetName)
	if err != nil {
		return nil, fmt.Errorf("failed to load preset %q: %w", presetName, err)
	}

	// Apply preset with project-specific customization
	config, err := presetpkg.ApplyPreset(template, projectInfo.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to apply preset %q: %w", presetName, err)
	}

	// Add language overrides based on detected languages if not present in preset
	if len(config.LanguageOverrides) == 0 && len(projectInfo.Languages) > 0 {
		config.LanguageOverrides = generateLanguageOverrides(projectInfo.Languages)
	}

	return config, nil
}
