package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/parser"
	"github.com/JobV9n/effectiveNext/pkg/routes"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
)

const fixtureDir = "fixtures/nextjs-app"

func TestIntegration_FullScan(t *testing.T) {
	absFixture, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(absFixture); os.IsNotExist(err) {
		t.Skipf("fixture directory not found: %s", absFixture)
	}

	s := scanner.New(scanner.Options{
		Root:    absFixture,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) < 10 {
		t.Errorf("expected at least 10 files, got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.RelPath] = true
	}

	expectedFiles := []string{
		"package.json",
		"next.config.js",
		"app/layout.tsx",
		"app/page.tsx",
		"app/loading.tsx",
		"app/error.tsx",
		"app/not-found.tsx",
		"components/Button.tsx",
		"lib/utils.ts",
		"app/globals.css",
	}

	for _, ef := range expectedFiles {
		if !paths[ef] {
			t.Errorf("expected to find %s", ef)
		}
	}
}

func TestIntegration_RouteDetection(t *testing.T) {
	absFixture, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(absFixture); os.IsNotExist(err) {
		t.Skipf("fixture directory not found: %s", absFixture)
	}

	s := routes.New(absFixture)
	manifest, err := s.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if !manifest.HasApp {
		t.Error("expected to detect app directory")
	}

	if manifest.TotalRoutes < 5 {
		t.Errorf("expected at least 5 routes, got %d", manifest.TotalRoutes)
	}

	routeTypes := make(map[routes.RouteType]int)
	for _, r := range manifest.Routes {
		routeTypes[r.Type]++
	}

	if routeTypes[routes.RouteTypePage] == 0 {
		t.Error("expected at least one page route")
	}
	if routeTypes[routes.RouteTypeLayout] == 0 {
		t.Error("expected at least one layout route")
	}
	if routeTypes[routes.RouteTypeAPI] == 0 {
		t.Error("expected at least one API route")
	}
}

func TestIntegration_ParserExtraction(t *testing.T) {
	absFixture, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(absFixture); os.IsNotExist(err) {
		t.Skipf("fixture directory not found: %s", absFixture)
	}

	results, err := parser.ParseDirectory(absFixture)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) < 5 {
		t.Errorf("expected at least 5 parsed files, got %d", len(results))
	}

	stats := parser.ParseStats(results)
	if stats.TotalImports == 0 {
		t.Error("expected at least one import")
	}
	if stats.TotalExports == 0 {
		t.Error("expected at least one export")
	}

	metadataCount := 0
	for _, r := range results {
		if r.HasMetadata {
			metadataCount++
		}
	}
	if metadataCount == 0 {
		t.Error("expected at least one file with metadata export")
	}
}

func TestIntegration_DependencyGraph(t *testing.T) {
	absFixture, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(absFixture); os.IsNotExist(err) {
		t.Skipf("fixture directory not found: %s", absFixture)
	}

	s := scanner.New(scanner.Options{
		Root:    absFixture,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	g := graph.New()
	for _, f := range files {
		g.AddNode(graph.Node{
			ID:   f.RelPath,
			Path: f.RelPath,
			Hash: f.Hash,
			Size: f.Size,
		})
	}

	if g.NodeCount() < 10 {
		t.Errorf("expected at least 10 nodes, got %d", g.NodeCount())
	}

	stats := g.Stats()
	if stats.NodeCount < 10 {
		t.Errorf("expected at least 10 graph nodes, got %d", stats.NodeCount)
	}
}

func TestIntegration_IgnorePatterns(t *testing.T) {
	absFixture, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(absFixture); os.IsNotExist(err) {
		t.Skipf("fixture directory not found: %s", absFixture)
	}

	os.MkdirAll(filepath.Join(absFixture, "node_modules", "fake-pkg"), 0o755)
	os.WriteFile(filepath.Join(absFixture, "node_modules", "fake-pkg", "index.js"), []byte("// fake"), 0o644)
	os.MkdirAll(filepath.Join(absFixture, ".next", "server"), 0o755)
	os.WriteFile(filepath.Join(absFixture, ".next", "server", "app.js"), []byte("// compiled"), 0o644)
	t.Cleanup(func() {
		os.RemoveAll(filepath.Join(absFixture, "node_modules"))
		os.RemoveAll(filepath.Join(absFixture, ".next"))
	})

	s := scanner.New(scanner.Options{
		Root:    absFixture,
		Config:  config.Default().Scanner,
		Workers: 2,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		if filepath.Base(f.RelPath) == "index.js" && filepath.Dir(f.RelPath) == "node_modules/fake-pkg" {
			t.Error("should not find files in node_modules")
		}
		if filepath.Base(f.RelPath) == "app.js" && filepath.Dir(f.RelPath) == ".next/server" {
			t.Error("should not find files in .next")
		}
	}
}

func TestIntegration_ConfigLoading(t *testing.T) {
	absFixture, err := filepath.Abs(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(absFixture); os.IsNotExist(err) {
		t.Skipf("fixture directory not found: %s", absFixture)
	}

	loader := config.NewLoader(absFixture)
	cfg, err := loader.Load("")
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Cache.Enabled {
		t.Error("expected cache to be enabled by default")
	}
	if cfg.Cache.Compression != "zstd" {
		t.Errorf("expected zstd compression, got %q", cfg.Cache.Compression)
	}
	if len(cfg.Scanner.Ignore) == 0 {
		t.Error("expected default ignore patterns")
	}
}