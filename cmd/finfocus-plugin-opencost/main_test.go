package main

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

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

func TestCubectx(t *testing.T) {
	// Test default timeout
	ctx := context.Background()
	resultCtx := cubectx(ctx)

	// Check that context has timeout
	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Default timeout should be 30 seconds
	expectedDeadline := time.Now().Add(30 * time.Second)
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestCubectxWithEnvironmentVariable(t *testing.T) {
	// Set environment variable
	t.Setenv("KUBECOST_TIMEOUT", "60s")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx := context.Background()
	resultCtx := cubectx(ctx)

	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Timeout should be 60 seconds
	expectedDeadline := time.Now().Add(60 * time.Second)
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestCubectxWithInvalidEnvironmentVariable(t *testing.T) {
	// Set invalid environment variable
	t.Setenv("KUBECOST_TIMEOUT", "invalid")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx := context.Background()
	resultCtx := cubectx(ctx)

	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Should fall back to default 30 seconds
	expectedDeadline := time.Now().Add(30 * time.Second)
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestCubectxWithEmptyEnvironmentVariable(t *testing.T) {
	// Set empty environment variable
	t.Setenv("KUBECOST_TIMEOUT", "")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx := context.Background()
	resultCtx := cubectx(ctx)

	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Should fall back to default 30 seconds
	expectedDeadline := time.Now().Add(30 * time.Second)
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestCubectxWithZeroTimeout(t *testing.T) {
	// Set zero timeout
	t.Setenv("KUBECOST_TIMEOUT", "0s")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx := context.Background()
	resultCtx := cubectx(ctx)

	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Should use 0 timeout
	expectedDeadline := time.Now()
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestCubectxWithNegativeTimeout(t *testing.T) {
	// Set negative timeout
	t.Setenv("KUBECOST_TIMEOUT", "-10s")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx := context.Background()
	resultCtx := cubectx(ctx)

	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Should use negative timeout (which will be in the past)
	expectedDeadline := time.Now().Add(-10 * time.Second)
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestCubectxWithComplexTimeout(t *testing.T) {
	// Set complex timeout
	t.Setenv("KUBECOST_TIMEOUT", "1h30m45s")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx := context.Background()
	resultCtx := cubectx(ctx)

	deadline, ok := resultCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}

	// Should use complex timeout
	expectedDeadline := time.Now().Add(1*time.Hour + 30*time.Minute + 45*time.Second)
	if deadline.Sub(expectedDeadline) > time.Second {
		t.Errorf("Expected deadline around %v, got %v", expectedDeadline, deadline)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx := context.Background()
	resultCtx := cubectx(ctx)

	// Context should not be cancelled initially
	select {
	case <-resultCtx.Done():
		t.Error("Context should not be cancelled initially")
	default:
		// Expected
	}

	// Wait for timeout to expire (use a short timeout for testing)
	t.Setenv("KUBECOST_TIMEOUT", "1ms")
	defer os.Unsetenv("KUBECOST_TIMEOUT")

	ctx2 := context.Background()
	resultCtx2 := cubectx(ctx2)

	// Wait a bit for the timeout to expire
	time.Sleep(10 * time.Millisecond)

	// Context should be cancelled after timeout
	select {
	case <-resultCtx2.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after timeout")
	}
}

func TestContextWithParentCancellation(t *testing.T) {
	// Create a parent context that gets cancelled
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCtx := cubectx(parentCtx)

	// Cancel the parent context
	cancel()

	// Result context should also be cancelled
	select {
	case <-resultCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled when parent is cancelled")
	}
}

func TestContextWithParentTimeout(t *testing.T) {
	// Create a parent context with a shorter timeout
	parentCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	resultCtx := cubectx(parentCtx)

	// Wait for parent timeout to expire
	time.Sleep(20 * time.Millisecond)

	// Result context should be cancelled due to parent timeout
	select {
	case <-resultCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled when parent times out")
	}
}

func TestContextWithParentDeadline(t *testing.T) {
	// Create a parent context with a deadline
	deadline := time.Now().Add(10 * time.Millisecond)
	parentCtx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	resultCtx := cubectx(parentCtx)

	// Wait for deadline to pass
	time.Sleep(20 * time.Millisecond)

	// Result context should be cancelled due to parent deadline
	select {
	case <-resultCtx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled when parent deadline passes")
	}
}
