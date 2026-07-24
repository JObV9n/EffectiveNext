package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/internal/testutil"
)

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		input      string
		wantMajor  int
		wantMinor  int
		wantErr    bool
	}{
		{"go1.26.0", 1, 26, false},
		{"1.24.1", 1, 24, false},
		{"go1.23", 1, 23, false},
		{"go2.0", 2, 0, false},
		{"go1", 0, 0, true},
		{"devel", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, err := parseGoVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseGoVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if major != tt.wantMajor || minor != tt.wantMinor {
				t.Fatalf("parseGoVersion(%q) = (%d, %d), want (%d, %d)", tt.input, major, minor, tt.wantMajor, tt.wantMinor)
			}
		})
	}
}

func TestParseSemverMajor(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"14.2.0", 14, false},
		{"v14.2.0", 14, false},
		{"13", 13, false},
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSemverMajor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseSemverMajor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parseSemverMajor(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestChecker_ProjectDirectory(t *testing.T) {
	dir := testutil.TempDir(t)

	checker := NewChecker(dir)
	results := checker.Run()

	var found bool
	for _, r := range results {
		if r.Name == "project-directory" {
			found = true
			if !r.Passed {
				t.Fatalf("expected project-directory check to pass, got %v", r)
			}
		}
	}
	if !found {
		t.Fatal("project-directory check not found")
	}
}

func TestChecker_MissingPackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)

	// Override execCommand so package manager and node checks fail cleanly.
	original := execCommand
	execCommand = func(name string, args ...string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	t.Cleanup(func() { execCommand = original })

	checker := NewChecker(dir)
	results := checker.Run()

	var nextjs Check
	for _, r := range results {
		if r.Name == "nextjs" {
			nextjs = r
		}
	}
	if nextjs.Passed {
		t.Fatal("expected nextjs check to fail when package.json is missing")
	}
}

func TestChecker_NextJSDetected(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name":"demo","dependencies":{"next":"^14.0.0"}}`)

	original := execCommand
	execCommand = func(name string, args ...string) (string, error) {
		if name == "node" && len(args) > 0 && args[0] == "-e" {
			return "14.2.0\n", nil
		}
		return "", fmt.Errorf("not found")
	}
	t.Cleanup(func() { execCommand = original })

	checker := NewChecker(dir)
	results := checker.Run()

	var nextjs Check
	for _, r := range results {
		if r.Name == "nextjs" {
			nextjs = r
		}
	}
	if nextjs.Name == "" {
		t.Fatal("nextjs check not found")
	}
	if !nextjs.Passed {
		t.Fatalf("expected nextjs check to pass, got %+v", nextjs)
	}
}

func TestWriteProjectConfig(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "next.config.js", "module.exports = {}\n")

	checker := NewChecker(dir)
	results := checker.Run()

	var found bool
	for _, r := range results {
		if r.Name == "project-config" {
			found = true
			if !r.Passed {
				t.Fatalf("expected project-config to pass, got %v", r)
			}
		}
	}
	if !found {
		t.Fatal("project-config check not found")
	}
}

func TestChecker_NonExistentDir(t *testing.T) {
	dir := filepath.Join(testutil.TempDir(t), "missing")
	checker := NewChecker(dir)
	results := checker.Run()

	var pd Check
	for _, r := range results {
		if r.Name == "project-directory" {
			pd = r
		}
	}
	if pd.Passed {
		t.Fatal("expected project-directory check to fail for missing dir")
	}
}

func TestChecker_PackageManager(t *testing.T) {
	original := execCommand
	execCommand = func(name string, args ...string) (string, error) {
		if name == "pnpm" {
			return "8.0.0\n", nil
		}
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { execCommand = original })

	checker := NewChecker(".")
	results := checker.Run()

	var pm Check
	for _, r := range results {
		if r.Name == "package-manager" {
			pm = r
		}
	}
	if pm.Name == "" {
		t.Fatal("package-manager check not found")
	}
	if !pm.Passed {
		t.Fatalf("expected package-manager check to pass, got %+v", pm)
	}
	if pm.Message != "pnpm 8.0.0" {
		t.Fatalf("unexpected message: %q", pm.Message)
	}
}
