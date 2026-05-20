package plugin

import (
	"context"
	"fmt"

	"github.com/pauvalls/arx/internal/domain"
	"github.com/pauvalls/arx/internal/ports"
)

// Compile-time check: PluginDetector implements ports.Detector
var _ ports.Detector = (*PluginDetector)(nil)

// PluginDetector wraps an external plugin as a ports.Detector.
// It communicates with the plugin process via stdin/stdout JSON protocol.
type PluginDetector struct {
	cfg domain.PluginConfig
}

// NewPluginDetector creates a new PluginDetector from a plugin configuration.
// The cfg must be validated before calling this function.
func NewPluginDetector(cfg domain.PluginConfig) *PluginDetector {
	return &PluginDetector{cfg: cfg}
}

// Name returns the plugin's configured name.
func (d *PluginDetector) Name() string {
	return d.cfg.Name
}

// Detect checks if the plugin's target language is present in the project.
// It sends a "detect" action to the plugin and returns the result.
func (d *PluginDetector) Detect(ctx context.Context, projectRoot string) (bool, error) {
	req := domain.PluginRequest{
		Action:      "detect",
		ProjectRoot: projectRoot,
	}

	resp, err := RunPlugin(d.cfg, req)
	if err != nil {
		return false, fmt.Errorf("plugin %q: detect failed: %w", d.cfg.Name, err)
	}

	if resp.Error != nil {
		return false, fmt.Errorf("plugin %q: detect error: %s", d.cfg.Name, resp.Error.Message)
	}

	if resp.Detect == nil {
		return false, fmt.Errorf("plugin %q: detect response missing detect field", d.cfg.Name)
	}

	return resp.Detect.Detected, nil
}

// ExtractImports extracts dependencies from the project using the plugin.
// It sends an "extract" action and converts PluginDependency results to domain.Dependency.
func (d *PluginDetector) ExtractImports(ctx context.Context, projectRoot string, layers []domain.Layer) ([]domain.Dependency, error) {
	// Convert domain.Layer to domain.LayerInfo
	layerInfos := make([]domain.LayerInfo, len(layers))
	for i, l := range layers {
		layerInfos[i] = domain.LayerInfo{
			Name:  l.Name,
			Paths: l.Paths,
		}
	}

	req := domain.PluginRequest{
		Action:      "extract",
		ProjectRoot: projectRoot,
		Layers:      layerInfos,
	}

	resp, err := RunPlugin(d.cfg, req)
	if err != nil {
		return nil, fmt.Errorf("plugin %q: extract failed: %w", d.cfg.Name, err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("plugin %q: extract error: %s", d.cfg.Name, resp.Error.Message)
	}

	if resp.Extract == nil {
		return nil, fmt.Errorf("plugin %q: extract response missing extract field", d.cfg.Name)
	}

	// Convert PluginDependency to domain.Dependency
	deps := make([]domain.Dependency, len(resp.Extract.Dependencies))
	for i, pd := range resp.Extract.Dependencies {
		deps[i] = domain.Dependency{
			SourceFile:    pd.SourceFile,
			SourceLine:    pd.SourceLine,
			ImportPath:    pd.ImportPath,
			ResolvedLayer: pd.ResolvedLayer,
		}
	}

	return deps, nil
}
