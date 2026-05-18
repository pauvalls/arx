package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCompletionBashSmoke tests that bash completion output is non-empty and starts with bash comment
func TestCompletionBashSmoke(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.AddCommand(completionCmd)

	var buf bytes.Buffer
	completionBashCmd.SetOut(&buf)

	err := completionBashCmd.RunE(completionBashCmd, []string{})
	if err != nil {
		t.Fatalf("completionBashCmd.RunE() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("bash completion output is empty")
	}

	// Bash completion scripts start with a comment
	if !strings.HasPrefix(output, "#") {
		t.Errorf("bash completion should start with '#', got: %q", output[:50])
	}

	// Should contain arx-specific completion
	if !strings.Contains(output, "arx") {
		t.Error("bash completion should contain 'arx'")
	}
}

// TestCompletionZshSmoke tests that zsh completion output is non-empty and starts with zsh comment
func TestCompletionZshSmoke(t *testing.T) {
	var buf bytes.Buffer
	completionZshCmd.SetOut(&buf)

	err := completionZshCmd.RunE(completionZshCmd, []string{})
	if err != nil {
		t.Fatalf("completionZshCmd.RunE() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("zsh completion output is empty")
	}

	// Zsh completion scripts start with a comment
	if !strings.HasPrefix(output, "#") {
		t.Errorf("zsh completion should start with '#', got: %q", output[:50])
	}

	// Should contain arx-specific completion
	if !strings.Contains(output, "arx") {
		t.Error("zsh completion should contain 'arx'")
	}
}

// TestCompletionFishSmoke tests that fish completion output is non-empty
func TestCompletionFishSmoke(t *testing.T) {
	var buf bytes.Buffer
	completionFishCmd.SetOut(&buf)

	err := completionFishCmd.RunE(completionFishCmd, []string{})
	if err != nil {
		t.Fatalf("completionFishCmd.RunE() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("fish completion output is empty")
	}

	// Fish completion scripts contain function definitions
	if !strings.Contains(output, "function") {
		t.Error("fish completion should contain 'function'")
	}
}

// TestCompletionPowerShellSmoke tests that PowerShell completion output is non-empty
func TestCompletionPowerShellSmoke(t *testing.T) {
	var buf bytes.Buffer
	completionPowerShellCmd.SetOut(&buf)

	err := completionPowerShellCmd.RunE(completionPowerShellCmd, []string{})
	if err != nil {
		t.Fatalf("completionPowerShellCmd.RunE() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("powershell completion output is empty")
	}

	// PowerShell completion scripts contain Register-ArgumentCompleter
	if !strings.Contains(output, "Register-ArgumentCompleter") {
		t.Error("powershell completion should contain 'Register-ArgumentCompleter'")
	}
}

// TestCompletionCmdHidden verifies completion command is hidden from main help
func TestCompletionCmdHidden(t *testing.T) {
	if !completionCmd.Hidden {
		t.Error("completionCmd should be Hidden=true")
	}
}

// TestCompletionSubcommandsRegistered verifies all shell subcommands are registered
func TestCompletionSubcommandsRegistered(t *testing.T) {
	subcommands := completionCmd.Commands()
	expectedShells := []string{"bash", "zsh", "fish", "powershell"}

	found := make(map[string]bool)
	for _, cmd := range subcommands {
		found[cmd.Name()] = true
	}

	for _, shell := range expectedShells {
		if !found[shell] {
			t.Errorf("missing subcommand: %s", shell)
		}
	}
}

// TestCompletionBashOutputValid verifies bash completion has required elements
func TestCompletionBashOutputValid(t *testing.T) {
	var buf bytes.Buffer
	completionBashCmd.SetOut(&buf)

	err := completionBashCmd.RunE(completionBashCmd, []string{})
	if err != nil {
		t.Fatalf("completionBashCmd.RunE() error = %v", err)
	}

	output := buf.String()

	// Check for bash completion function
	if !strings.Contains(output, "_arx") {
		t.Error("bash completion should contain '_arx'")
	}

	// Check for complete command
	if !strings.Contains(output, "complete ") {
		t.Error("bash completion should use 'complete' command")
	}
}

// TestCompletionZshOutputValid verifies zsh completion has required elements
func TestCompletionZshOutputValid(t *testing.T) {
	var buf bytes.Buffer
	completionZshCmd.SetOut(&buf)

	err := completionZshCmd.RunE(completionZshCmd, []string{})
	if err != nil {
		t.Fatalf("completionZshCmd.RunE() error = %v", err)
	}

	output := buf.String()

	// Check for zsh completion function
	if !strings.Contains(output, "_arx()") {
		t.Error("zsh completion should define _arx() function")
	}
}

// captureOutput captures stdout from a command
func captureOutput(cmd *cobra.Command) (string, error) {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	return buf.String(), err
}
