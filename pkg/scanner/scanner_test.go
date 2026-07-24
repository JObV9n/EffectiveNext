package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/pkg/config"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"src/index.ts":     `import { App } from './app';`,
		"src/app.tsx":      `export const App = () => {};`,
		"src/utils.ts":     `export const util = {};`,
		"src/styles.css":   `body { margin: 0; }`,
		"package.json":     `{"name": "test"}`,
		"next.config.js":   `module.exports = {}`,
		"public/logo.png":  "fake png data",
		"README.md":        "# Test",
		"node_modules/pkg/index.js": "module.exports = {}",
		".next/server/app/page.js":  "compiled output",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestScanner_Scan(t *testing.T) {
	dir := setupTestDir(t)

	s := New(Options{
		Root:    dir,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("expected at least one file")
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.RelPath] = true
	}

	// Should find source files
	if !paths["src/index.ts"] {
		t.Error("expected to find src/index.ts")
	}
	if !paths["package.json"] {
		t.Error("expected to find package.json")
	}
	if !paths["README.md"] {
		t.Error("expected to find README.md")
	}

	// Should NOT find ignored directories
	if paths["node_modules/pkg/index.js"] {
		t.Error("should not find node_modules/pkg/index.js")
	}
	if paths[".next/server/app/page.js"] {
		t.Error("should not find .next/server/app/page.js")
	}
}

func TestScanner_CustomIgnore(t *testing.T) {
	dir := setupTestDir(t)

	s := New(Options{
		Root: dir,
		Config: config.ScannerConfig{
			Ignore: []string{"public", "README.md"},
		},
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.RelPath] = true
	}

	if paths["public/logo.png"] {
		t.Error("should not find public/logo.png with custom ignore")
	}
	if paths["README.md"] {
		t.Error("should not find README.md with custom ignore")
	}
	if !paths["src/index.ts"] {
		t.Error("expected to find src/index.ts")
	}
}

func TestScanner_FileMetadata(t *testing.T) {
	dir := setupTestDir(t)

	s := New(Options{
		Root:    dir,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Hash == 0 {
			t.Errorf("file %s has zero hash", f.Path)
		}
		if f.Size == 0 && f.Extension != ".md" {
			t.Errorf("file %s has zero size", f.Path)
		}
		if f.Extension == "" {
			t.Errorf("file %s has no extension", f.Path)
		}
	}
}

func TestScanner_ScanWithFilter(t *testing.T) {
	dir := setupTestDir(t)

	s := New(Options{
		Root:    dir,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	files, err := s.ScanWithFilter(func(f File) bool {
		return f.Extension == ".ts" || f.Extension == ".tsx"
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if f.Extension != ".ts" && f.Extension != ".tsx" {
			t.Errorf("expected only .ts/.tsx files, got %s", f.Extension)
		}
	}
}

func TestScanner_ScanByExtension(t *testing.T) {
	dir := setupTestDir(t)

	s := New(Options{
		Root:    dir,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	groups, err := s.ScanByExtension()
	if err != nil {
		t.Fatal(err)
	}

	if len(groups) == 0 {
		t.Fatal("expected at least one extension group")
	}

	for ext, files := range groups {
		for _, f := range files {
			got := f.Extension
			if got == "" {
				got = "(none)"
			}
			if got != ext {
				t.Errorf("file %s has extension %s but is in group %s", f.Path, got, ext)
			}
		}
	}
}

func TestScanner_FilesScanned(t *testing.T) {
	dir := setupTestDir(t)

	s := New(Options{
		Root:    dir,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	_, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	count := s.FilesScanned()
	if count == 0 {
		t.Error("expected FilesScanned() > 0")
	}
}

func TestScanner_Symlinks(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "target"), 0o755)
	os.WriteFile(filepath.Join(dir, "target", "file.txt"), []byte("hello"), 0o644)

	err := os.Symlink(filepath.Join(dir, "target"), filepath.Join(dir, "link"))
	if err != nil {
		t.Skip("symlinks not supported")
	}

	s := New(Options{
		Root:    dir,
		Config:  config.ScannerConfig{Ignore: []string{}},
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, f := range files {
		if f.IsSymlink {
			found = true
		}
	}
	if !found {
		t.Error("expected to find symlink")
	}
}

func TestScanner_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	s := New(Options{
		Root:    dir,
		Config:  config.ScannerConfig{Ignore: []string{}},
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}

func TestSortByPath(t *testing.T) {
	files := []File{
		{Path: "c/file.ts"},
		{Path: "a/file.ts"},
		{Path: "b/file.ts"},
	}
	SortByPath(files)
	if files[0].Path != "a/file.ts" || files[1].Path != "b/file.ts" || files[2].Path != "c/file.ts" {
		t.Error("SortByPath did not sort correctly")
	}
}

func TestSortBySize(t *testing.T) {
	files := []File{
		{Path: "small", Size: 100},
		{Path: "large", Size: 1000},
		{Path: "medium", Size: 500},
	}
	SortBySize(files)
	if files[0].Path != "large" || files[1].Path != "medium" || files[2].Path != "small" {
		t.Error("SortBySize did not sort correctly")
	}
}

func TestTotalSize(t *testing.T) {
	files := []File{
		{Size: 100},
		{Size: 200},
		{Size: 300},
	}
	total := TotalSize(files)
	if total != 600 {
		t.Errorf("TotalSize() = %d, want 600", total)
	}
}

func TestFilterByExtension(t *testing.T) {
	files := []File{
		{Path: "a.ts", Extension: ".ts"},
		{Path: "b.js", Extension: ".js"},
		{Path: "c.tsx", Extension: ".tsx"},
		{Path: "d.ts", Extension: ".ts"},
	}
	ts := FilterByExtension(files, ".ts")
	if len(ts) != 2 {
		t.Errorf("FilterByExtension(.ts) returned %d, want 2", len(ts))
	}
}

func TestIgnoreMatcher(t *testing.T) {
	m := newIgnoreMatcher([]string{"custom"}, "/root")

	if !m.Matches("node_modules", true) {
		t.Error("should match node_modules")
	}
	if !m.Matches("custom", true) {
		t.Error("should match custom")
	}
	if m.Matches("src", true) {
		t.Error("should not match src")
	}
	if !m.MatchesPath("/root/node_modules/pkg") {
		t.Error("should match node_modules/pkg path")
	}
}