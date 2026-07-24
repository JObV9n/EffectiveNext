package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func setupOptimizeAssets(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"public/logo.png":     "fake png content for hashing",
		"public/icon.svg":     "<svg><circle r='10'/></svg>",
		"public/photo.jpg":    "fake jpg content for hashing",
		"src/style.css":       "body { margin: 0; }",
		"public/old-logo.png": "fake png content for hashing",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0o755)
		os.WriteFile(fullPath, []byte(content), 0o644)
	}

	return dir
}

func TestOptimize_Hashes(t *testing.T) {
	dir := setupOptimizeAssets(t)
	o := NewOptimizer(dir, 2)
	o.Scan()

	results, err := o.Optimize(OptimizeOptions{MaxThreads: 2})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected optimization results")
	}

	for _, r := range results {
		if r.Hash == "" {
			t.Errorf("expected hash for %s", r.OriginalPath)
		}
	}
}

func TestOptimize_ResultSizes(t *testing.T) {
	dir := setupOptimizeAssets(t)
	o := NewOptimizer(dir, 2)
	o.Scan()

	results, err := o.Optimize(OptimizeOptions{MaxThreads: 2})
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range results {
		if r.OriginalSize == 0 {
			t.Errorf("expected OriginalSize > 0 for %s", r.OriginalPath)
		}
	}
}

func TestDeduplicateByHash(t *testing.T) {
	dir := t.TempDir()

	// Two files with same content
	content := "identical content for dedup test"
	os.MkdirAll(filepath.Join(dir, "a"), 0o755)
	os.MkdirAll(filepath.Join(dir, "b"), 0o755)
	os.WriteFile(filepath.Join(dir, "a", "file1.png"), []byte(content), 0o644)
	os.WriteFile(filepath.Join(dir, "b", "file2.png"), []byte(content), 0o644)
	os.WriteFile(filepath.Join(dir, "a", "unique.txt"), []byte("unique"), 0o644)

	o := NewOptimizer(dir, 2)
	o.Scan()

	dups := o.DeduplicateByHash()
	if len(dups) == 0 {
		t.Error("expected to find duplicate files")
	}

	found := false
	for _, group := range dups {
		if len(group) >= 2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a group of at least 2 identical files")
	}
}

func TestWriteMetadata(t *testing.T) {
	dir := setupOptimizeAssets(t)
	o := NewOptimizer(dir, 2)
	o.Scan()

	outDir := filepath.Join(dir, ".effective-next", "assets")
	if err := o.WriteMetadata(outDir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "assets.json"))
	if err != nil {
		t.Fatalf("expected metadata file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty metadata")
	}
}

func TestOptimize_EmptyProject(t *testing.T) {
	dir := t.TempDir()
	o := NewOptimizer(dir, 2)
	o.Scan()

	results, err := o.Optimize(OptimizeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty project, got %d", len(results))
	}
}

func TestOptimize_DefaultWorkers(t *testing.T) {
	dir := setupOptimizeAssets(t)
	o := NewOptimizer(dir, 0) // default workers
	o.Scan()

	results, err := o.Optimize(OptimizeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Error("expected results")
	}
}

func TestDeduplicateByHash_NoDuplicates(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "src", "a.png"), []byte("content a"), 0o644)
	os.WriteFile(filepath.Join(dir, "src", "b.png"), []byte("content b"), 0o644)

	o := NewOptimizer(dir, 2)
	o.Scan()

	dups := o.DeduplicateByHash()
	if len(dups) != 0 {
		t.Errorf("expected no duplicates, got %d groups", len(dups))
	}
}

func TestOptimize_ConcurrentSafety(t *testing.T) {
	dir := setupOptimizeAssets(t)
	o := NewOptimizer(dir, 8)
	o.Scan()

	done := make(chan error, 2)
	go func() {
		_, err := o.Optimize(OptimizeOptions{MaxThreads: 8})
		done <- err
	}()
	go func() {
		o.DeduplicateByHash()
		done <- nil
	}()

	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent error: %v", err)
		}
	}
}
