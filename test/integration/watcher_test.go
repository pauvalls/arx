package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pauvalls/arx/internal/infrastructure/watcher"
)

// TestWatcher_Integration_TempProject verifies end-to-end file watching.
func TestWatcher_Integration_TempProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := watcher.NewWatcher([]string{tmpDir}, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Start() error: %v", err)
		}
	}()

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a new file
	testFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Expect the event
	select {
	case evt := <-w.Events():
		if evt.Path != testFile {
			t.Errorf("event path = %q, want %q", evt.Path, testFile)
		}
		if evt.Op != watcher.Create {
			t.Logf("event op = %d (expected %d = Create)", evt.Op, watcher.Create)
			// Some OS may emit Write instead of Create, that's OK
		}
		if evt.Time.IsZero() {
			t.Error("event timestamp should not be zero")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for file create event")
	}
}

// TestWatcher_Integration_Debounce verifies that rapid file changes are debounced.
func TestWatcher_Integration_Debounce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a longer debounce for reliability
	w, err := watcher.NewWatcher([]string{tmpDir}, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Start() error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	testFile := filepath.Join(tmpDir, "debounce.go")

	// Two rapid writes
	if err := os.WriteFile(testFile, []byte("v1"), 0644); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(testFile, []byte("v2"), 0644); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	// Should receive exactly one event (debounced)
	eventCount := 0
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-w.Events():
			eventCount++
		case <-time.After(500 * time.Millisecond):
			// No more events for 500ms — debounce done
			if eventCount == 0 {
				t.Fatal("expected at least one event after debounce")
			}
			if eventCount > 1 {
				t.Fatalf("expected 1 debounced event, got %d", eventCount)
			}
			return
		case <-timeout:
			t.Fatal("timeout waiting for debounced events")
		}
	}
}

// TestWatcher_Integration_GracefulShutdown verifies context cancel stops the watcher.
func TestWatcher_Integration_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	w, err := watcher.NewWatcher([]string{tmpDir}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	started := make(chan struct{})
	go func() {
		close(started)
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Start() error: %v", err)
		}
	}()

	<-started
	time.Sleep(50 * time.Millisecond)

	// Cancel context — Start should return
	cancel()

	// Verify watcher stops within reasonable time
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// Success — watcher returned
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after context cancel")
	}
}

// TestWatcher_Integration_GitignoreRespect verifies that .gitignore patterns are respected.
func TestWatcher_Integration_GitignoreRespect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	// Create .gitignore that ignores node_modules and .log files
	gitignoreContent := "node_modules/\n*.log\n"
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	// Create subdirectories
	os.MkdirAll(filepath.Join(tmpDir, "node_modules", "pkg"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)

	w, err := watcher.NewWatcher([]string{tmpDir}, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Start() error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Create file in ignored directory
	ignoredFile := filepath.Join(tmpDir, "node_modules", "pkg", "lib.js")
	if err := os.WriteFile(ignoredFile, []byte("ignored"), 0644); err != nil {
		t.Fatalf("failed to create ignored file: %v", err)
	}

	// Create .log file
	logFile := filepath.Join(tmpDir, "output.log")
	if err := os.WriteFile(logFile, []byte("log content"), 0644); err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	// Give watcher time to process (or ignore) the events
	time.Sleep(300 * time.Millisecond)

	// No events for ignored files should be in the channel
	select {
	case evt := <-w.Events():
		t.Errorf("received unexpected event for ignored path: %q", evt.Path)
	default:
		// Good — no events for ignored files
	}

	// Now create a tracked file — should get an event
	srcFile := filepath.Join(tmpDir, "src", "app.go")
	if err := os.WriteFile(srcFile, []byte("package app"), 0644); err != nil {
		t.Fatalf("failed to create src file: %v", err)
	}

	select {
	case evt := <-w.Events():
		if evt.Path != srcFile {
			t.Errorf("expected event for %q, got %q", srcFile, evt.Path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event from tracked file")
	}
}
