// Command mockplugin is a test mock for the arx plugin system.
// It reads JSON from stdin, parses the "action" field, and returns
// the appropriate JSON response on stdout.
//
// Behavior is controlled by the binary name:
//   mockplugin-detect   → always returns detect response
//   mockplugin-full     → handles all actions
//   mockplugin-slow     → sleeps 60s then returns detect
//   mockplugin-error    → returns error response
//
// Usage: echo '{"action":"detect","project_root":"/tmp"}' | mockplugin
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	binName := filepath.Base(os.Args[0])

	// Slow plugin: sleep first
	if strings.Contains(binName, "slow") {
		time.Sleep(60 * time.Second)
	}

	// Read stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	var req struct {
		Action string `json:"action"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		// If we can't parse, use binary name to determine action
	}

	action := req.Action
	if action == "" {
		if strings.Contains(binName, "detect") {
			action = "detect"
		}
	}

	// Error plugin: always return error
	if strings.Contains(binName, "error") {
		resp := map[string]interface{}{
			"error": map[string]string{
				"message": "mock plugin error",
			},
		}
		json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	// Detect-only plugin: always detect success
	if strings.Contains(binName, "detect") {
		resp := map[string]interface{}{
			"detect": map[string]bool{"detected": true},
		}
		json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	// Full plugin: route by action
	switch action {
	case "detect":
		resp := map[string]interface{}{
			"detect": map[string]bool{"detected": true},
		}
		json.NewEncoder(os.Stdout).Encode(resp)
	case "extract":
		resp := map[string]interface{}{
			"extract": map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"source_file":    "test.py",
						"source_line":    1,
						"import_path":    "os",
						"resolved_layer": "stdlib",
					},
				},
			},
		}
		json.NewEncoder(os.Stdout).Encode(resp)
	case "capabilities":
		resp := map[string]interface{}{
			"capabilities": map[string]interface{}{
				"name":      "full-mock",
				"languages": []string{"python", "ruby"},
				"version":   "2.0.0",
			},
		}
		json.NewEncoder(os.Stdout).Encode(resp)
	default:
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{})
	}
}
