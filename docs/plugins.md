# Arx Plugin System

> **Version**: v0.51+ | **Status**: Stable

Arx supports external plugins for detecting dependencies in languages not covered by the built-in detectors. Plugins communicate with arx via a simple JSON protocol over stdin/stdout.

## How It Works

1. Arx runs the plugin as a subprocess
2. Sends a JSON request on stdin with the action type and project info
3. Plugin processes the request and writes a JSON response to stdout
4. Arx parses the response and integrates the results with built-in detectors

## Protocol

### Request Format

Every plugin receives a JSON object on stdin:

```json
{
  "action": "detect",
  "project_root": "/path/to/project",
  "layers": [
    {"name": "domain", "paths": ["internal/domain"]}
  ],
  "config": {}
}
```

### Actions

| Action | Description |
|--------|-------------|
| `detect` | Check if the plugin's language is present in the project |
| `extract` | Extract dependency information from the project |
| `capabilities` | Advertise the plugin's capabilities (optional) |

### Response Format

```json
{
  "detect": {"detected": true},
  "extract": {
    "dependencies": [
      {
        "source_file": "src/main.dart",
        "source_line": 42,
        "import_path": "package:http/http.dart",
        "resolved_layer": "infrastructure"
      }
    ]
  },
  "capabilities": {
    "name": "dart-detector",
    "languages": ["dart"],
    "version": "1.0.0"
  }
}
```

### Error Response

```json
{
  "error": {"message": "descriptive error message"}
}
```

## Configuration

Add plugins to your `arx.yaml`:

```yaml
plugins:
  - name: dart-detector
    command: dart run bin/detect.dart
    languages: [dart]
    timeout: 30s
    extensions: [.dart]
```

### Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique plugin name (must not conflict with built-in detectors) |
| `command` | Yes | Command to execute the plugin |
| `args` | No | Additional arguments to pass to the command |
| `languages` | Yes | List of languages this plugin handles |
| `timeout` | No | Timeout duration (e.g., "30s", "5m"). Default: 30s |
| `extensions` | No | File extensions this plugin handles |

### Validation Rules

- Plugin name must match `^[a-zA-Z][a-zA-Z0-9_-]*$`
- Plugin name cannot conflict with built-in detectors: `go`, `python`, `typescript`, `java`, `kotlin`, `rust`, `csharp`, `ruby`, `swift`, `php`
- Command must not be empty
- At least one language must be specified
- Timeout must be a valid Go duration string if set

## Writing Plugins

### Go Plugin

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
)

type Request struct {
    Action      string `json:"action"`
    ProjectRoot string `json:"project_root"`
}

type Response struct {
    Detect *DetectResult `json:"detect,omitempty"`
}

type DetectResult struct {
    Detected bool `json:"detected"`
}

func main() {
    data, _ := io.ReadAll(os.Stdin)
    var req Request
    json.Unmarshal(data, &req)

    switch req.Action {
    case "detect":
        // Check for language markers in the project
        _, err := os.Stat(filepath.Join(req.ProjectRoot, "pubspec.yaml"))
        resp := Response{
            Detect: &DetectResult{Detected: err == nil},
        }
        json.NewEncoder(os.Stdout).Encode(resp)

    case "capabilities":
        json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
            "capabilities": map[string]interface{}{
                "name":      "dart-detector",
                "languages": []string{"dart"},
                "version":   "1.0.0",
            },
        })

    default:
        json.NewEncoder(os.Stdout).Encode(map[string]interface{}{})
    }
}
```

### Python Plugin

```python
#!/usr/bin/env python3
import json
import os
import sys

def handle_detect(project_root):
    """Check if this project uses the target language."""
    return os.path.exists(os.path.join(project_root, "pubspec.yaml"))

def handle_extract(project_root, layers):
    """Extract dependencies from project files."""
    deps = []
    for root, dirs, files in os.walk(project_root):
        for f in files:
            if f.endswith('.dart'):
                filepath = os.path.relpath(os.path.join(root, f), project_root)
                with open(os.path.join(root, f), 'r') as fh:
                    for i, line in enumerate(fh, 1):
                        if line.strip().startswith('import'):
                            deps.append({
                                "source_file": filepath,
                                "source_line": i,
                                "import_path": line.strip().split("'")[1],
                                "resolved_layer": "infrastructure"
                            })
    return deps

def main():
    data = json.load(sys.stdin)
    action = data.get("action")

    if action == "detect":
        detected = handle_detect(data.get("project_root", ""))
        json.dump({"detect": {"detected": detected}}, sys.stdout)

    elif action == "extract":
        deps = handle_extract(
            data.get("project_root", ""),
            data.get("layers", [])
        )
        json.dump({"extract": {"dependencies": deps}}, sys.stdout)

    elif action == "capabilities":
        json.dump({
            "capabilities": {
                "name": "dart-detector",
                "languages": ["dart"],
                "version": "1.0.0"
            }
        }, sys.stdout)

if __name__ == "__main__":
    main()
```

### Shell Script Plugin

```bash
#!/bin/bash
# Read JSON from stdin
INPUT=$(cat)

# Extract action (simple parsing for basic cases)
ACTION=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['action'])" 2>/dev/null)

case "$ACTION" in
  detect)
    echo '{"detect":{"detected":true}}'
    ;;
  extract)
    echo '{"extract":{"dependencies":[]}}'
    ;;
  capabilities)
    echo '{"capabilities":{"name":"my-plugin","languages":["custom"],"version":"1.0.0"}}'
    ;;
  *)
    echo '{}'
    ;;
esac
```

## Best Practices

1. **Fast detect**: Keep `detect` fast — check for a marker file (e.g., `pubspec.yaml`, `Cargo.toml`)
2. **Handle timeouts**: Design plugins to complete within the configured timeout (default: 30s)
3. **Stderr for logging**: Use stderr for diagnostic output; arx captures and logs it
4. **Stable output**: Always output valid JSON — malformed responses cause the plugin to be skipped
5. **Graceful degradation**: Return meaningful error responses instead of crashing
6. **No side effects**: Plugins should be read-only — never modify the project

## Limitations

- Plugin marketplace is not supported
- WASM-based plugins are not supported
- Remote plugin execution is not supported
- Plugins run with user privileges — treat them as trusted code

## Troubleshooting

### Plugin Not Running
- Verify the command is in PATH or use an absolute path
- Check that the plugin file has execute permissions
- Run `arx doctor` to diagnose configuration issues

### Plugin Timeout
- Increase the `timeout` value in the plugin config
- Optimize the plugin to process files faster
- Use caching to avoid re-processing unchanged files

### Invalid Output
- Run the plugin standalone to verify it outputs valid JSON
- Check stderr for error messages
- Ensure the plugin writes JSON to stdout (not stderr)
