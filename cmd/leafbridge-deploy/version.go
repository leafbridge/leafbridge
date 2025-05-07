package main

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/leafbridge/leafbridge/internal/buildinfo"
)

const dateTimeWithZone = "2006-01-02 15:04:05 MST"

// VersionCmd shows version information about the running executable.
type VersionCmd struct{}

// Run executes the version command.
func (cmd VersionCmd) Run(ctx context.Context) error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("leafbridge-deploy build information is not available")
		return nil
	}

	// Look for build settings that are of interest.
	commit := buildinfo.ParseCommit(buildInfo.Settings)

	// Print the main module version.
	if version := buildInfo.Main.Version; version != "" {
		fmt.Printf("%s\n", version)
	}

	// Print the commit revision.
	if commit.Revision != "" {
		if commit.Modified {
			fmt.Printf("  leafbridge-deploy commit revision: %s (modified)\n", commit.Revision)
		} else {
			fmt.Printf("  leafbridge-deploy commit revision: %s\n", commit.Revision)
		}
	}

	// Print the commit date.
	if !commit.Time.IsZero() {
		fmt.Printf("  leafbridge-deploy commit date: %s\n", commit.Time.Local().Format(dateTimeWithZone))
	}

	// Print the go version.
	if version := buildInfo.GoVersion; version != "" {
		fmt.Printf("  go version: %s\n", version)
	}

	return nil
}
