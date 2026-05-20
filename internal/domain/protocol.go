package domain

// --- Plugin Protocol Types ---
// These types define the JSON contract for external plugin communication.
// Plugins receive a PluginRequest via stdin and write a PluginResponse to stdout.

// PluginRequest is sent to a plugin via stdin.
type PluginRequest struct {
	Action      string                 `json:"action"` // "detect" | "extract" | "capabilities"
	ProjectRoot string                 `json:"project_root"`
	Layers      []LayerInfo            `json:"layers,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// LayerInfo describes a single layer for the extract action.
type LayerInfo struct {
	Name  string   `json:"name"`
	Paths []string `json:"paths"`
}

// PluginCapabilities describes a plugin's advertised capabilities.
type PluginCapabilities struct {
	Name      string   `json:"name"`
	Languages []string `json:"languages"`
	Version   string   `json:"version,omitempty"`
}

// PluginDetectResult is the result of a detect action.
type PluginDetectResult struct {
	Detected bool `json:"detected"`
}

// PluginExtractResult is the result of an extract action.
type PluginExtractResult struct {
	Dependencies []PluginDependency `json:"dependencies"`
}

// PluginDependency is a single dependency found by a plugin.
type PluginDependency struct {
	SourceFile    string `json:"source_file"`
	SourceLine    int    `json:"source_line"`
	ImportPath    string `json:"import_path"`
	ResolvedLayer string `json:"resolved_layer,omitempty"`
}

// PluginError carries an error message from a plugin.
type PluginError struct {
	Message string `json:"message"`
}

// PluginResponse is the response from a plugin. Exactly one action field should be set.
type PluginResponse struct {
	Capabilities *PluginCapabilities  `json:"capabilities,omitempty"`
	Detect       *PluginDetectResult  `json:"detect,omitempty"`
	Extract      *PluginExtractResult `json:"extract,omitempty"`
	Error        *PluginError         `json:"error,omitempty"`
}
