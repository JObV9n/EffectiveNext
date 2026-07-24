// Package testutil provides helpers for effectiveNext tests.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempDir creates a temporary directory and returns its path.
func TempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "effectiveNext-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// WriteFile writes content to path inside dir with the given mode.
func WriteFile(t *testing.T, dir, relPath, content string) string {
	t.Helper()
	path := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}
