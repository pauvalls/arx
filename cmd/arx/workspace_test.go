package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestWorkspaceCommand_Registered(t *testing.T) {
	if workspaceCmd == nil {
		t.Fatal("workspace command is nil")
	}
	if workspaceCmd.Use != "workspace [path]" {
		t.Errorf("workspaceCmd.Use = %q, want %q", workspaceCmd.Use, "workspace [path]")
	}
}

func TestWorkspaceCommand_HelpContainsExpectedText(t *testing.T) {
	buf := new(bytes.Buffer)
	workspaceCmd.SetOut(buf)
	workspaceCmd.Help()
	output := buf.String()

	if !strings.Contains(output, "workspace") {
		t.Errorf("help should contain 'workspace', got: %s", output)
	}
	if !strings.Contains(output, "architecture audit") {
		t.Errorf("help should mention 'architecture audit', got: %s", output)
	}
	if !strings.Contains(output, "arx-workspace.yaml") {
		t.Errorf("help should mention 'arx-workspace.yaml', got: %s", output)
	}
}

func TestWorkspaceCommand_Flags(t *testing.T) {
	tests := []struct {
		name     string
		flagName string
		wantType string
	}{
		{"json flag", "json", "bool"},
		{"verbose flag", "verbose", "bool"},
		{"output flag", "output", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := workspaceCmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("Flag --%s not found on workspace command", tt.flagName)
			}
			if flag.Value.Type() != tt.wantType {
				t.Errorf("Flag --%s type = %q, want %q", tt.flagName, flag.Value.Type(), tt.wantType)
			}
		})
	}
}

func TestWorkspaceCommand_Defaults(t *testing.T) {
	jsonFlag := workspaceCmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("--json flag not found")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("--json default should be false, got %q", jsonFlag.DefValue)
	}

	outputFlag := workspaceCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatal("--output flag not found")
	}
	if outputFlag.DefValue != "" {
		t.Errorf("--output default should be empty, got %q", outputFlag.DefValue)
	}

	verboseFlag := workspaceCmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("--verbose flag not found")
	}
	if verboseFlag.DefValue != "false" {
		t.Errorf("--verbose default should be false, got %q", verboseFlag.DefValue)
	}
}

func TestWorkspaceCommand_Args(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no args is valid",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one arg is valid",
			args:    []string{"/some/path"},
			wantErr: false,
		},
		{
			name:    "two args is invalid",
			args:    []string{"/path1", "/path2"},
			wantErr: true,
			errMsg:  "accepts at most 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := workspaceCmd.Args(workspaceCmd, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error = %v", err)
			}
		})
	}
}
