package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// Build-time variables - set via ldflags during build
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// VersionInfo holds all version information
 type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// GetVersionInfo returns the current version information
func GetVersionInfo() VersionInfo {
	goVersion := runtime.Version()

	// Try to read from debug info if not set via ldflags
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				version = info.Main.Version
			}
		}
	}

	return VersionInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		GoVersion: goVersion,
	}
}

// VersionString returns a formatted version string for display
// Note: Cobra adds "arx version " prefix automatically, so we only return the version details
func VersionString() string {
	info := GetVersionInfo()
	return fmt.Sprintf("%s (commit: %s, built: %s, %s)",
		info.Version, info.Commit, info.BuildDate, info.GoVersion)
}
