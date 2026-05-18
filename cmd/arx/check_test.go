package main

import (
	"testing"
)

func TestCheckCommand_HasWatchFlag(t *testing.T) {
	// Verify the --watch flag is registered on the check command
	flag := checkCmd.Flags().Lookup("watch")
	if flag == nil {
		t.Fatal("--watch flag not found on check command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--watch default should be false, got %q", flag.DefValue)
	}
}

func TestCheckCommand_HasIntervalFlag(t *testing.T) {
	flag := checkCmd.Flags().Lookup("interval")
	if flag == nil {
		t.Fatal("--interval flag not found on check command")
	}
	if flag.DefValue != "500ms" {
		t.Errorf("--interval default should be 500ms, got %q", flag.DefValue)
	}
}

func TestCheckCommand_HasAllOriginalFlags(t *testing.T) {
	requiredFlags := []string{"config", "ci", "format", "verbose", "no-cache", "no-baseline"}
	for _, name := range requiredFlags {
		if flag := checkCmd.Flags().Lookup(name); flag == nil {
			t.Errorf("required flag --%s not found on check command", name)
		}
	}
}

func TestCheckCommand_WatchFlagType(t *testing.T) {
	flag := checkCmd.Flags().Lookup("watch")
	if flag == nil {
		t.Fatal("--watch flag not found")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--watch should be bool type, got %q", flag.Value.Type())
	}
}

func TestCheckCommand_IntervalFlagType(t *testing.T) {
	flag := checkCmd.Flags().Lookup("interval")
	if flag == nil {
		t.Fatal("--interval flag not found")
	}
	if flag.Value.Type() != "duration" {
		t.Errorf("--interval should be duration type, got %q", flag.Value.Type())
	}
}

func TestCheckCommand_HasDiffFlag(t *testing.T) {
	flag := checkCmd.Flags().Lookup("diff")
	if flag == nil {
		t.Fatal("--diff flag not found on check command")
	}
	if flag.DefValue != "false" {
		t.Errorf("--diff default should be false, got %q", flag.DefValue)
	}
}

func TestCheckCommand_DiffFlagType(t *testing.T) {
	flag := checkCmd.Flags().Lookup("diff")
	if flag == nil {
		t.Fatal("--diff flag not found")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--diff should be bool type, got %q", flag.Value.Type())
	}
}
