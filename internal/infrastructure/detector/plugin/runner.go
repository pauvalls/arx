package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/pauvalls/arx/internal/domain"
)

// DefaultTimeout is the default timeout for plugin execution.
const DefaultTimeout = 30 * time.Second

// RunPlugin executes a plugin with the given request and returns the response.
// It writes the request as JSON to the plugin's stdin, reads the response from stdout,
// and captures stderr for logging. If the plugin times out or exits with an error,
// an appropriate error is returned.
func RunPlugin(cfg domain.PluginConfig, req domain.PluginRequest) (*domain.PluginResponse, error) {
	// Determine timeout duration
	timeout := DefaultTimeout
	if cfg.Timeout != "" {
		parsed, err := time.ParseDuration(cfg.Timeout)
		if err == nil {
			timeout = parsed
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Build command
	args := make([]string, len(cfg.Args))
	copy(args, cfg.Args)
	cmd := exec.CommandContext(ctx, cfg.Command, args...)

	// Marshal request to JSON
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("plugin %q: failed to marshal request: %w", cfg.Name, err)
	}

	// Set up stdin
	cmd.Stdin = bytes.NewReader(reqData)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	if err := cmd.Run(); err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("plugin %q: timeout after %v", cfg.Name, timeout)
		}
		// Log stderr on failure
		if stderr.Len() > 0 {
			log.Printf("plugin %q stderr: %s", cfg.Name, stderr.String())
		}
		return nil, fmt.Errorf("plugin %q: execution failed: %w", cfg.Name, err)
	}

	// Log stderr if present (not an error, just informational)
	if stderr.Len() > 0 {
		log.Printf("plugin %q stderr: %s", cfg.Name, stderr.String())
	}

	// Parse response
	var resp domain.PluginResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("plugin %q: failed to parse response: %w\nraw: %s", cfg.Name, err, stdout.String())
	}

	return &resp, nil
}

// GetCapabilities queries a plugin for its capabilities.
func GetCapabilities(cfg domain.PluginConfig) (*domain.PluginCapabilities, error) {
	req := domain.PluginRequest{
		Action: "capabilities",
	}

	resp, err := RunPlugin(cfg, req)
	if err != nil {
		return nil, fmt.Errorf("plugin %q: capabilities failed: %w", cfg.Name, err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("plugin %q: capabilities error: %s", cfg.Name, resp.Error.Message)
	}

	if resp.Capabilities == nil {
		return nil, fmt.Errorf("plugin %q: capabilities response missing capabilities field", cfg.Name)
	}

	return resp.Capabilities, nil
}

// runPluginWithStdin is a helper for tests that allows passing a custom stdin reader.
// It is used by the test mock plugin runner.
func runPluginWithStdin(cfg domain.PluginConfig, req domain.PluginRequest, stdin io.Reader) (*domain.PluginResponse, error) {
	timeout := DefaultTimeout
	if cfg.Timeout != "" {
		parsed, err := time.ParseDuration(cfg.Timeout)
		if err == nil {
			timeout = parsed
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := make([]string, len(cfg.Args))
	copy(args, cfg.Args)
	cmd := exec.CommandContext(ctx, cfg.Command, args...)

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("plugin %q: failed to marshal request: %w", cfg.Name, err)
	}

	if stdin != nil {
		cmd.Stdin = io.NopCloser(stdin)
	} else {
		cmd.Stdin = bytes.NewReader(reqData)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("plugin %q: timeout after %v", cfg.Name, timeout)
		}
		if stderr.Len() > 0 {
			log.Printf("plugin %q stderr: %s", cfg.Name, stderr.String())
		}
		return nil, fmt.Errorf("plugin %q: execution failed: %w", cfg.Name, err)
	}

	if stderr.Len() > 0 {
		log.Printf("plugin %q stderr: %s", cfg.Name, stderr.String())
	}

	var resp domain.PluginResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("plugin %q: failed to parse response: %w\nraw: %s", cfg.Name, err, stdout.String())
	}

	return &resp, nil
}
