# Writing a Custom Plugin Detector

This tutorial walks through creating a custom plugin detector for **Dart**, a language not covered by arx's built-in detectors.

## The Plugin Protocol

Arx communicates with plugins via a JSON protocol over stdin/stdout:

1. Arx runs the plugin as a subprocess
2. Sends a JSON request on stdin with the action type
3. Plugin processes the request and writes JSON to stdout
4. Arx parses the response and integrates the results

## Step 1: Create the plugin directory

```bash
mkdir -p tools/arx-plugins/dart-detector
cd tools/arx-plugins/dart-detector
```

## Step 2: Write the plugin (Go)

```go
// tools/arx-plugins/dart-detector/main.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Request from arx
type Request struct {
	Action      string            `json:"action"`
	ProjectRoot string            `json:"project_root"`
	Layers      []json.RawMessage `json:"layers"`
}

// Response to arx
type Response struct {
	Detect       *DetectResult    `json:"detect,omitempty"`
	Extract      *ExtractResult   `json:"extract,omitempty"`
	Capabilities *Capabilities    `json:"capabilities,omitempty"`
}

type DetectResult struct {
	Detected bool `json:"detected"`
}

type ExtractResult struct {
	Dependencies []Dependency `json:"dependencies"`
}

type Capabilities struct {
	Name      string   `json:"name"`
	Languages []string `json:"languages"`
	Version   string   `json:"version"`
}

type Dependency struct {
	SourceFile    string `json:"source_file"`
	SourceLine    int    `json:"source_line"`
	ImportPath    string `json:"import_path"`
	ResolvedLayer string `json:"resolved_layer"`
}

func main() {
	var req Request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeError(fmt.Sprintf("failed to decode request: %v", err))
		return
	}

	switch req.Action {
	case "detect":
		handleDetect(req)
	case "extract":
		handleExtract(req)
	case "capabilities":
		handleCapabilities()
	default:
		writeJSON(Response{})
	}
}

func handleDetect(req Request) {
	// Check for pubspec.yaml — Dart's project marker
	pubspecPath := filepath.Join(req.ProjectRoot, "pubspec.yaml")
	_, err := os.Stat(pubspecPath)
	writeJSON(Response{
		Detect: &DetectResult{Detected: err == nil},
	})
}

func handleExtract(req Request) {
	var deps []Dependency

	filepath.Walk(req.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".dart") {
			return nil
		}

		relPath, _ := filepath.Rel(req.ProjectRoot, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import") || strings.HasPrefix(line, "export") {
				// Extract the package path from quotes
				start := strings.Index(line, "'")
				end := strings.LastIndex(line, "'")
				if start == -1 || end == -1 || start == end {
					continue
				}
				importPath := line[start+1 : end]

				// Simple layer resolution (customize for your project)
				resolvedLayer := "infrastructure"
				if strings.Contains(importPath, "domain") {
					resolvedLayer = "domain"
				} else if strings.Contains(importPath, "app") {
					resolvedLayer = "application"
				}

				deps = append(deps, Dependency{
					SourceFile:    relPath,
					SourceLine:    i + 1,
					ImportPath:    importPath,
					ResolvedLayer: resolvedLayer,
				})
			}
		}
		return nil
	})

	writeJSON(Response{
		Extract: &ExtractResult{Dependencies: deps},
	})
}

func handleCapabilities() {
	writeJSON(Response{
		Capabilities: &Capabilities{
			Name:      "dart-detector",
			Languages: []string{"dart"},
			Version:   "1.0.0",
		},
	})
}

func writeJSON(v interface{}) {
	json.NewEncoder(os.Stdout).Encode(v)
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		"error": map[string]string{"message": msg},
	})
}
```

## Step 3: Build the plugin

```bash
go build -o dart-detector main.go
```

## Step 4: Register the plugin in arx.yaml

```yaml
# arx.yaml
plugins:
  - name: dart-detector
    command: ./tools/arx-plugins/dart-detector/dart-detector
    languages: [dart]
    timeout: 30s
    extensions: [.dart]
```

## Step 5: Test the plugin

First, verify with `arx doctor`:

```bash
arx doctor
```

This checks that the plugin's command is accessible.

Then run a full check:

```bash
arx check --verbose
```

You should see:

```
Detectors:
  ✓ Go: 142 dependencies
  ✓ Dart: 37 dependencies   ← Your plugin!
```

## Alternative: Python Plugin

```python
#!/usr/bin/env python3
# tools/arx-plugins/dart-detector/detect.py
import json
import os
import sys

def handle_detect(project_root):
    return os.path.exists(os.path.join(project_root, "pubspec.yaml"))

def handle_extract(project_root, layers):
    deps = []
    for root, dirs, files in os.walk(project_root):
        for f in files:
            if not f.endswith('.dart'):
                continue
            rel_path = os.path.relpath(os.path.join(root, f), project_root)
            with open(os.path.join(root, f)) as fh:
                for i, line in enumerate(fh, 1):
                    line = line.strip()
                    if line.startswith('import') or line.startswith('export'):
                        # Extract path from quotes
                        parts = line.split("'")
                        if len(parts) >= 2:
                            import_path = parts[1]
                            resolved = "infrastructure"
                            if "domain" in import_path:
                                resolved = "domain"
                            elif "app" in import_path:
                                resolved = "application"
                            deps.append({
                                "source_file": rel_path,
                                "source_line": i,
                                "import_path": import_path,
                                "resolved_layer": resolved,
                            })
    return deps

def main():
    data = json.load(sys.stdin)
    action = data.get("action")
    project_root = data.get("project_root", "")

    if action == "detect":
        json.dump({"detect": {"detected": handle_detect(project_root)}}, sys.stdout)
    elif action == "extract":
        deps = handle_extract(project_root, data.get("layers", []))
        json.dump({"extract": {"dependencies": deps}}, sys.stdout)
    elif action == "capabilities":
        json.dump({
            "capabilities": {
                "name": "dart-detector",
                "languages": ["dart"],
                "version": "1.0.0"
            }
        }, sys.stdout)
    else:
        json.dump({}, sys.stdout)

if __name__ == "__main__":
    main()
```

## Best Practices

1. **Fast detect**: Check for a marker file (e.g., `pubspec.yaml`) — don't scan files
2. **Handle timeouts**: Keep `extract` under 30s (default timeout)
3. **Stderr for logging**: Use stderr for debugging — arx captures it
4. **Stable output**: Always return valid JSON — malformed responses skip the plugin
5. **No side effects**: Plugins should be read-only — never modify the project
6. **Version your plugin**: Use the `capabilities` action to report version

## Plugin Configuration Reference

```yaml
plugins:
  - name: my-plugin
    command: /path/to/plugin      # Required
    args: ["--flag", "value"]      # Optional
    languages: [my-lang]           # Required
    timeout: 30s                   # Optional, default: 30s
    extensions: [.ext]             # Optional
```

For the full protocol specification, see [docs/plugins.md](../plugins.md).
