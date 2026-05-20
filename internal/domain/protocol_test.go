package domain

import (
	"encoding/json"
	"testing"
)

func TestPluginRequestRoundTrip(t *testing.T) {
	req := PluginRequest{
		Action:      "detect",
		ProjectRoot: "/home/project",
		Layers: []LayerInfo{
			{Name: "domain", Paths: []string{"internal/domain"}},
		},
		Config: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal PluginRequest: %v", err)
	}

	var got PluginRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal PluginRequest: %v", err)
	}

	if got.Action != req.Action {
		t.Errorf("Action = %q, want %q", got.Action, req.Action)
	}
	if got.ProjectRoot != req.ProjectRoot {
		t.Errorf("ProjectRoot = %q, want %q", got.ProjectRoot, req.ProjectRoot)
	}
	if len(got.Layers) != len(req.Layers) {
		t.Fatalf("Layers length = %d, want %d", len(got.Layers), len(req.Layers))
	}
	if got.Layers[0].Name != req.Layers[0].Name {
		t.Errorf("Layer[0].Name = %q, want %q", got.Layers[0].Name, req.Layers[0].Name)
	}
	if len(got.Layers[0].Paths) != 1 || got.Layers[0].Paths[0] != "internal/domain" {
		t.Errorf("Layer[0].Paths = %v, want [internal/domain]", got.Layers[0].Paths)
	}
	if got.Config["key"] != "value" {
		t.Errorf("Config[key] = %v, want value", got.Config["key"])
	}
}

func TestPluginResponseRoundTrip_Detect(t *testing.T) {
	resp := PluginResponse{
		Detect: &PluginDetectResult{Detected: true},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal PluginResponse: %v", err)
	}

	var got PluginResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal PluginResponse: %v", err)
	}

	if got.Detect == nil {
		t.Fatal("Detect is nil")
	}
	if got.Detect.Detected != true {
		t.Errorf("Detect.Detected = %v, want true", got.Detect.Detected)
	}
}

func TestPluginResponseRoundTrip_Extract(t *testing.T) {
	resp := PluginResponse{
		Extract: &PluginExtractResult{
			Dependencies: []PluginDependency{
				{
					SourceFile:    "src/main.go",
					SourceLine:    42,
					ImportPath:    "fmt",
					ResolvedLayer: "stdlib",
				},
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got PluginResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Extract == nil {
		t.Fatal("Extract is nil")
	}
	if len(got.Extract.Dependencies) != 1 {
		t.Fatalf("Dependencies length = %d, want 1", len(got.Extract.Dependencies))
	}
	dep := got.Extract.Dependencies[0]
	if dep.SourceFile != "src/main.go" {
		t.Errorf("SourceFile = %q, want %q", dep.SourceFile, "src/main.go")
	}
	if dep.SourceLine != 42 {
		t.Errorf("SourceLine = %d, want 42", dep.SourceLine)
	}
	if dep.ImportPath != "fmt" {
		t.Errorf("ImportPath = %q, want %q", dep.ImportPath, "fmt")
	}
	if dep.ResolvedLayer != "stdlib" {
		t.Errorf("ResolvedLayer = %q, want %q", dep.ResolvedLayer, "stdlib")
	}
}

func TestPluginResponseRoundTrip_Capabilities(t *testing.T) {
	resp := PluginResponse{
		Capabilities: &PluginCapabilities{
			Name:      "my-detector",
			Languages: []string{"dart", "flutter"},
			Version:   "1.0.0",
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got PluginResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Capabilities == nil {
		t.Fatal("Capabilities is nil")
	}
	if got.Capabilities.Name != "my-detector" {
		t.Errorf("Name = %q, want %q", got.Capabilities.Name, "my-detector")
	}
	if len(got.Capabilities.Languages) != 2 || got.Capabilities.Languages[0] != "dart" {
		t.Errorf("Languages = %v, want [dart flutter]", got.Capabilities.Languages)
	}
	if got.Capabilities.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", got.Capabilities.Version, "1.0.0")
	}
}

func TestPluginResponseRoundTrip_Error(t *testing.T) {
	resp := PluginResponse{
		Error: &PluginError{Message: "something went wrong"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got PluginResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Error == nil {
		t.Fatal("Error is nil")
	}
	if got.Error.Message != "something went wrong" {
		t.Errorf("Error.Message = %q, want %q", got.Error.Message, "something went wrong")
	}
}

func TestPluginResponse_AllFieldsOmitted(t *testing.T) {
	// A plugin that sends an empty response (no action requested)
	resp := PluginResponse{}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got PluginResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Detect != nil {
		t.Error("Detect should be nil")
	}
	if got.Extract != nil {
		t.Error("Extract should be nil")
	}
	if got.Capabilities != nil {
		t.Error("Capabilities should be nil")
	}
	if got.Error != nil {
		t.Error("Error should be nil")
	}
}

func TestPluginRequest_EmptyLayers(t *testing.T) {
	req := PluginRequest{
		Action:      "extract",
		ProjectRoot: "/tmp/test",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got PluginRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Action != "extract" {
		t.Errorf("Action = %q, want %q", got.Action, "extract")
	}
	if got.ProjectRoot != "/tmp/test" {
		t.Errorf("ProjectRoot = %q, want %q", got.ProjectRoot, "/tmp/test")
	}
	if len(got.Layers) != 0 {
		t.Errorf("Layers = %v, want empty", got.Layers)
	}
}
