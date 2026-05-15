package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tmpDir := t.TempDir()

	w, err := NewWatcher([]string{tmpDir}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	if w == nil {
		t.Fatal("NewWatcher() returned nil")
	}
}

func TestWatcher_ReceivesCreateEvent(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := NewWatcher([]string{tmpDir}, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Start() error: %v", err)
		}
	}()

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Create a file
	newFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(newFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Read event with timeout
	select {
	case evt := <-w.Events():
		if evt.Path != newFile {
			t.Errorf("event path = %q, want %q", evt.Path, newFile)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for create event")
	}
}

func TestWatcher_DebounceGroupsRapidEvents(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a longer debounce to make test reliable
	w, err := NewWatcher([]string{tmpDir}, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	eventsReceived := make(chan struct{}, 10)
	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Start() error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Rapidly write to a file twice
	testFile := filepath.Join(tmpDir, "debounce.txt")

	// First write
	if err := os.WriteFile(testFile, []byte("v1"), 0644); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	// Second write within debounce window
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(testFile, []byte("v2"), 0644); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	// Read events — should get exactly one (debounced)
	go func() {
		for range w.Events() {
			eventsReceived <- struct{}{}
		}
	}()

	select {
	case <-eventsReceived:
		// Got the debounced event
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for debounced event")
	}

	// Wait a bit to check no more events
	select {
	case <-eventsReceived:
		t.Fatal("received more than one event — debounce failed")
	case <-time.After(500 * time.Millisecond):
		// Good — only one event received
	}
}

func TestWatcher_CancelViaContext(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	w, err := NewWatcher([]string{tmpDir}, 100*time.Millisecond)
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

	// Start should return within reasonable time
	done := make(chan struct{})
	go func() {
		w.Start(ctx) // should return quickly since context already cancelled
		close(done)
	}()

	select {
	case <-done:
		// Good — returned after cancel
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after context cancel")
	}
}

func TestWatcher_CloseCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := NewWatcher([]string{tmpDir}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}

	go func() {
		w.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Close should clean up
	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestWatcher_GitignoreRespected(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .gitignore
	gitignoreContent := []byte("ignored_dir/\n*.log\n")
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"), gitignoreContent, 0644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	// Create ignored and non-ignored directories
	os.MkdirAll(filepath.Join(tmpDir, "ignored_dir"), 0755)
	if err := os.MkdirAll(filepath.Join(tmpDir, "src"), 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	// Allow watcher to register the new directory
	time.Sleep(100 * time.Millisecond)

	w, err := NewWatcher([]string{tmpDir}, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		w.Start(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create file in ignored directory — should NOT produce event
	ignoredFile := filepath.Join(tmpDir, "ignored_dir", "test.txt")
	if err := os.WriteFile(ignoredFile, []byte("ignored"), 0644); err != nil {
		t.Fatalf("failed to create ignored file: %v", err)
	}

	// Create .log file — should NOT produce event
	logFile := filepath.Join(tmpDir, "debug.log")
	if err := os.WriteFile(logFile, []byte("log"), 0644); err != nil {
		t.Fatalf("failed to create log file: %v", err)
	}

	// Create a tracked file — SHOULD produce event
	srcFile := filepath.Join(tmpDir, "src", "main.go")
	if err := os.WriteFile(srcFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create src file: %v", err)
	}

	// Should get event for src/main.go
	select {
	case evt := <-w.Events():
		if evt.Path != srcFile {
			t.Errorf("expected event for %q, got %q", srcFile, evt.Path)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event from src/main.go")
	}

	// Verify no more events pending (ignored files)
	time.Sleep(200 * time.Millisecond)
	select {
	case evt := <-w.Events():
		t.Errorf("unexpected event for ignored path: %q", evt.Path)
	default:
		// Good — no additional events
	}
}

func TestWatcher_NonExistentDir(t *testing.T) {
	_, err := NewWatcher([]string{"/nonexistent/path"}, 100*time.Millisecond)
	if err == nil {
		t.Error("NewWatcher() with nonexistent dir should return error")
	}
}
