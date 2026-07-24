package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestAssets(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"public/logo.png":     "fake png",
		"public/icon.svg":     "<svg></svg>",
		"public/photo.jpg":    "fake jpg",
		"src/style.css":       "body {}",
		"src/image.webp":      "fake webp",
		"public/old-logo.png": "same size as logo",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	return dir
}

func TestOptimizer_Scan(t *testing.T) {
	dir := setupTestAssets(t)

	o := NewOptimizer(dir, 2)
	if err := o.Scan(); err != nil {
		t.Fatal(err)
	}

	files := o.ListFiles()
	if len(files) == 0 {
		t.Fatal("expected at least one asset file")
	}
}

func TestOptimizer_FilterByType(t *testing.T) {
	dir := setupTestAssets(t)

	o := NewOptimizer(dir, 2)
	o.Scan()

	pngs := o.FilterByType(FileTypePNG)
	if len(pngs) == 0 {
		t.Error("expected at least one PNG file")
	}
}

func TestOptimizer_GetStats(t *testing.T) {
	dir := setupTestAssets(t)

	o := NewOptimizer(dir, 2)
	o.Scan()

	stats := o.GetStats()
	if stats.TotalFiles == 0 {
		t.Error("expected TotalFiles > 0")
	}
}

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		path string
		want FileType
	}{
		{"image.png", FileTypePNG},
		{"image.jpg", FileTypeJPEG},
		{"image.jpeg", FileTypeJPEG},
		{"image.svg", FileTypeSVG},
		{"image.webp", FileTypeWebP},
		{"font.woff2", FileTypeFont},
		{"video.mp4", FileTypeVideo},
		{"data.json", FileTypeOther},
	}

	for _, tt := range tests {
		got := classifyFile(tt.path)
		if got != tt.want {
			t.Errorf("classifyFile(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}