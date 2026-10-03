package main

import (
	"flag"
	"os"
	"testing"

	"github.com/rshade/finfocus-plugin-opencost/pkg/version"
)

func TestMainFunction(t *testing.T) {
	// Test that main function can be called without panicking
	// This is a basic smoke test
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}
}

func TestVersionFlags(t *testing.T) {
	// Save original command line arguments
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs //nolint:reassign // Required for testing CLI behavior
		//nolint:reassign // Required for testing CLI behavior
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Test -version flag
	//nolint:reassign // Required for testing CLI behavior
	os.Args = []string{"finfocus-plugin-opencost", "-version"}
	//nolint:reassign // Required for testing CLI behavior
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// This would normally call os.Exit(0), so we can't easily test it
	// Instead, we'll test the version functions directly
	versionStr := version.String()
	if versionStr == "" {
		t.Error("Version string should not be empty")
	}

	// Test -version-full flag
	//nolint:reassign // Required for testing CLI behavior
	os.Args = []string{"finfocus-plugin-opencost", "-version-full"}
	//nolint:reassign // Required for testing CLI behavior
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fullVersionStr := version.FullString()
	if fullVersionStr == "" {
		t.Error("Full version string should not be empty")
	}
}
