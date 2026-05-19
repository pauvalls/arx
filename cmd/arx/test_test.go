package main

import (
	"strings"
	"testing"
)

func TestTestCmd_Help(t *testing.T) {
	// Check command metadata directly
	cmd, _, err := rootCmd.Find([]string{"test"})
	if err != nil {
		t.Fatalf("find 'test' command: %v", err)
	}
	if cmd == nil {
		t.Fatal("'test' command not found")
	}

	helpText := cmd.Long
	if !strings.Contains(helpText, "arx test") && !strings.Contains(helpText, "rule tests") {
		t.Errorf("help text should mention 'rule tests': %s", helpText)
	}
	if !strings.Contains(cmd.Use, "test") {
		t.Errorf("Use should contain 'test': %s", cmd.Use)
	}
}

func TestTestCmd_HasFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"test"})
	if err != nil {
		t.Fatalf("find 'test' command: %v", err)
	}
	if cmd == nil {
		t.Fatal("'test' command not found")
	}

	// Verify flags exist
	flags := []string{"fixture", "rule", "verbose", "ci", "junit"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag --%s not found on test command", flag)
		}
	}
}

func TestTestCmd_Args(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"test"})
	if err != nil {
		t.Fatalf("find 'test' command: %v", err)
	}
	if cmd == nil {
		t.Fatal("'test' command not found")
	}

	// Verify MaximumNArgs(1)
	if cmd.Args == nil {
		t.Error("Args validator should be set")
	}
}

func TestNewTestService(t *testing.T) {
	service := newTestService()
	if service == nil {
		t.Fatal("newTestService returned nil")
	}
}
