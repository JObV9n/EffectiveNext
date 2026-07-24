package css

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()
	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}
}

func TestTrackFile_CSS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.css")
	os.WriteFile(path, []byte("body { color: red; }"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if f.Type != FileTypeCSS {
		t.Errorf("expected CSS type, got %s", f.Type)
	}
	if f.Size != 20 {
		t.Errorf("expected size 20, got %d", f.Size)
	}
}

func TestTrackFile_SCSS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.scss")
	os.WriteFile(path, []byte("$color: red;"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if f.Type != FileTypeSCSS {
		t.Errorf("expected SCSS type, got %s", f.Type)
	}
}

func TestTrackFile_SASS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.sass")
	os.WriteFile(path, []byte("color: red"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if f.Type != FileTypeSASS {
		t.Errorf("expected SASS type, got %s", f.Type)
	}
}

func TestTrackFile_Less(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.less")
	os.WriteFile(path, []byte("@color: red;"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if f.Type != FileTypeLess {
		t.Errorf("expected Less type, got %s", f.Type)
	}
}

func TestTrackFile_Imports(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.css")
	content := `@import "variables.css";
@import "reset.css";
body { color: red; }
`
	os.WriteFile(path, []byte(content), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(f.Imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(f.Imports))
	}
}

func TestTrackFile_Tailwind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "globals.css")
	content := `@tailwind base;
@tailwind components;
@tailwind utilities;
`
	os.WriteFile(path, []byte(content), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !f.Tailwind {
		t.Error("expected Tailwind to be detected")
	}
}

func TestTrackFile_NoTailwind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.css")
	os.WriteFile(path, []byte("body { color: red; }"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if f.Tailwind {
		t.Error("expected no Tailwind")
	}
}

func TestTrackFile_Module(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Component.module.css")
	os.WriteFile(path, []byte(".container {}"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !f.IsModule {
		t.Error("expected CSS module to be detected")
	}
}

func TestTrackFile_NotModule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.css")
	os.WriteFile(path, []byte("body {}"), 0o644)

	tracker := NewTracker()
	f, err := tracker.TrackFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if f.IsModule {
		t.Error("expected not a module")
	}
}

func TestGetDependencies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.css")
	os.WriteFile(path, []byte(`@import "vars.css";`), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(path)

	deps := tracker.GetDependencies(path)
	if len(deps) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(deps))
	}
}

func TestGetDependencies_NotTracked(t *testing.T) {
	tracker := NewTracker()
	deps := tracker.GetDependencies("nonexistent")
	if deps != nil {
		t.Errorf("expected nil, got %v", deps)
	}
}

func TestGetDependents(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.css")
	varsPath := filepath.Join(dir, "vars.css")
	os.WriteFile(mainPath, []byte(`@import "vars.css";`), 0o644)
	os.WriteFile(varsPath, []byte(":root { --color: red; }"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(mainPath)
	tracker.TrackFile(varsPath)

	dependents := tracker.GetDependents(varsPath)
	if len(dependents) != 1 {
		t.Errorf("expected 1 dependent, got %d", len(dependents))
	}
}

func TestAffectedFiles(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.css")
	mainPath := filepath.Join(dir, "main.css")
	os.WriteFile(basePath, []byte(":root {}"), 0o644)
	os.WriteFile(mainPath, []byte(`@import "base.css";`), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(basePath)
	tracker.TrackFile(mainPath)

	affected := tracker.AffectedFiles(basePath)
	if len(affected) != 2 {
		t.Errorf("expected 2 affected files, got %d", len(affected))
	}
}

func TestHasTailwind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "globals.css")
	os.WriteFile(path, []byte("@tailwind base;"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(path)

	if !tracker.HasTailwind() {
		t.Error("expected HasTailwind to return true")
	}
}

func TestHasTailwind_False(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.css")
	os.WriteFile(path, []byte("body {}"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(path)

	if tracker.HasTailwind() {
		t.Error("expected HasTailwind to return false")
	}
}

func TestHasModules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Component.module.css")
	os.WriteFile(path, []byte(".container {}"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(path)

	if !tracker.HasModules() {
		t.Error("expected HasModules to return true")
	}
}

func TestHasModules_False(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "styles.css")
	os.WriteFile(path, []byte("body {}"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(path)

	if tracker.HasModules() {
		t.Error("expected HasModules to return false")
	}
}

func TestListFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.css"), []byte("a {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.css"), []byte("b {}"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(filepath.Join(dir, "a.css"))
	tracker.TrackFile(filepath.Join(dir, "b.css"))

	files := tracker.ListFiles()
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestGetStats(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "globals.css"), []byte("@tailwind base;"), 0o644)
	os.WriteFile(filepath.Join(dir, "Component.module.css"), []byte(".container {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.scss"), []byte("$color: red;"), 0o644)

	tracker := NewTracker()
	tracker.TrackFile(filepath.Join(dir, "globals.css"))
	tracker.TrackFile(filepath.Join(dir, "Component.module.css"))
	tracker.TrackFile(filepath.Join(dir, "main.scss"))

	stats := tracker.GetStats()
	if stats.TotalFiles != 3 {
		t.Errorf("expected 3 files, got %d", stats.TotalFiles)
	}
	if !stats.HasTailwind {
		t.Error("expected HasTailwind to be true")
	}
	if !stats.HasModules {
		t.Error("expected HasModules to be true")
	}
	if !stats.HasSCSS {
		t.Error("expected HasSCSS to be true")
	}
}

func TestTrackDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.css"), []byte("a {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.scss"), []byte("b {}"), 0o644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("not css"), 0o644)

	tracker := NewTracker()
	if err := tracker.TrackDirectory(dir); err != nil {
		t.Fatal(err)
	}

	files := tracker.ListFiles()
	if len(files) != 2 {
		t.Errorf("expected 2 CSS files, got %d", len(files))
	}
}

func TestExtractImports(t *testing.T) {
	content := `@import "a.css";
@import 'b.css';
@import url("c.css");
`
	imports := extractImports(content, "/test")
	if len(imports) != 3 {
		t.Errorf("expected 3 imports, got %d", len(imports))
	}
}

func TestExtractImports_HTTP(t *testing.T) {
	content := `@import "https://example.com/styles.css";`
	imports := extractImports(content, "/test")
	if len(imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(imports))
	}
}

func TestExtractImports_Tilde(t *testing.T) {
	content := `@import "~bootstrap/dist/css/bootstrap.css";`
	imports := extractImports(content, "/test")
	if len(imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(imports))
	}
}

func TestExtToFileType(t *testing.T) {
	tests := []struct {
		ext     string
		want    FileType
	}{
		{".css", FileTypeCSS},
		{".scss", FileTypeSCSS},
		{".sass", FileTypeSASS},
		{".less", FileTypeLess},
		{".unknown", FileTypeCSS},
	}

	for _, tt := range tests {
		got := extToFileType(tt.ext)
		if got != tt.want {
			t.Errorf("extToFileType(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}

func TestTrackFile_NotFound(t *testing.T) {
	tracker := NewTracker()
	_, err := tracker.TrackFile("/nonexistent/file.css")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestEmptyTracker(t *testing.T) {
	tracker := NewTracker()

	if tracker.HasTailwind() {
		t.Error("expected HasTailwind to return false")
	}
	if tracker.HasModules() {
		t.Error("expected HasModules to return false")
	}
	if len(tracker.ListFiles()) != 0 {
		t.Error("expected no files")
	}
}
