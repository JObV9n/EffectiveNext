package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/pkg/cli"
	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/JobV9n/effectiveNext/pkg/engine"
	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/manifest"
	"github.com/JobV9n/effectiveNext/pkg/parser"
	"github.com/JobV9n/effectiveNext/pkg/routes"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
)

const testFixtureDir = "fixtures/nextjs-app"

func TestBuildIntegration_ScanFiles(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg.Scanner,
		Workers: 4,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	if len(files) < 10 {
		t.Errorf("expected at least 10 files, got %d", len(files))
	}

	hasTSX := false
	hasCSS := false
	for _, f := range files {
		if filepath.Ext(f.RelPath) == ".tsx" {
			hasTSX = true
		}
		if filepath.Ext(f.RelPath) == ".css" {
			hasCSS = true
		}
	}

	if !hasTSX {
		t.Error("expected at least one .tsx file")
	}
	if !hasCSS {
		t.Error("expected at least one .css file")
	}
}

func TestBuildIntegration_DependencyGraph(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg.Scanner,
		Workers: 4,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	g := graph.New()
	for _, f := range files {
		g.AddNode(graph.Node{
			ID:   f.Path,
			Path: f.RelPath,
			Hash: f.Hash,
			Size: f.Size,
		})
	}

	parseResults, err := parser.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range parseResults {
		for _, imp := range r.Imports {
			g.AddEdge(r.Path, imp.Path, graph.EdgeStaticImport, imp.Dynamic)
		}
	}

	if len(g.Nodes()) == 0 {
		t.Error("expected nodes in graph")
	}
}

func TestBuildIntegration_RouteDetection(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	routeScanner := routes.New(dir)
	manifest, err := routeScanner.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	if manifest.TotalRoutes < 5 {
		t.Errorf("expected at least 5 routes, got %d", manifest.TotalRoutes)
	}

	hasAppRouter := false
	for _, r := range manifest.Routes {
		if r.Type == routes.RouteTypePage {
			hasAppRouter = true
			break
		}
	}

	if !hasAppRouter {
		t.Error("expected at least one App Router route")
	}
}

func TestBuildIntegration_ParserExtraction(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	results, err := parser.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) < 5 {
		t.Errorf("expected at least 5 parse results, got %d", len(results))
	}

	totalImports := 0
	totalExports := 0
	for _, r := range results {
		totalImports += len(r.Imports)
		totalExports += len(r.Exports)
	}

	if totalImports == 0 {
		t.Error("expected at least one import")
	}
	if totalExports == 0 {
		t.Error("expected at least one export")
	}
}

func TestBuildIntegration_IgnorePatterns(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg.Scanner,
		Workers: 4,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		for _, ignore := range cfg.Scanner.Ignore {
			if matched, _ := filepath.Match(ignore, f.RelPath); matched {
				t.Errorf("file %s should be ignored by pattern %s", f.RelPath, ignore)
			}
		}
	}
}

func TestBuildIntegration_ConfigLoading(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	loader := config.NewLoader(dir)
	cfg, err := loader.Load("")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Cache.Dir != ".effective-next/cache" {
		t.Errorf("expected default cache dir, got %s", cfg.Cache.Dir)
	}

	if cfg.Cache.Compression != "zstd" {
		t.Errorf("expected zstd compression, got %s", cfg.Cache.Compression)
	}
}

func TestBuildIntegration_IncrementalEngine(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "build.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	cfg := config.Default()
	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg.Scanner,
		Workers: 4,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	eng := engine.New(database)
	eng.Snapshot(files)

	changes := eng.ComputeChanges(files)
	if len(changes) > 0 {
		t.Errorf("expected no changes on identical scan, got %d", len(changes))
	}
}

func TestBuildIntegration_Manifest(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	m := manifest.NewGenerator(dir)

	routeScanner := routes.New(dir)
	manifestData, err := routeScanner.GenerateManifest()
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range manifestData.Routes {
		m.AddRoute(manifest.RouteEntry{
			Path: r.Path,
			File: r.FilePath,
			Type: string(r.Type),
		})
	}

	cfg := config.Default()
	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg.Scanner,
		Workers: 4,
	})

	files, err := s.Scan()
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		m.AddAsset(manifest.AssetEntry{
			Path: f.RelPath,
			Size: f.Size,
			Type: filepath.Ext(f.RelPath),
		})
	}

	result := m.Build()
	if result == nil {
		t.Fatal("expected non-nil manifest result")
	}
}

func TestBuildIntegration_CLICommands(t *testing.T) {
	rootCmd := cli.CreateRootCmd()

	commands := []string{"build", "dev", "clean", "cache", "analyze", "doctor", "graph", "benchmark", "watch"}

	for _, cmd := range commands {
		found := false
		for _, c := range rootCmd.Commands() {
			if c.Name() == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command %s not found", cmd)
		}
	}
}

func TestBuildIntegration_DatabaseOperations(t *testing.T) {
	effectiveNextDir := t.TempDir()
	dbPath := filepath.Join(effectiveNextDir, "build.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	buildID, err := database.StartBuild()
	if err != nil {
		t.Fatal(err)
	}

	database.RecordStat(buildID, "files_scanned", 100)
	database.RecordStat(buildID, "total_hash", 12345)

	if err := database.FinishBuild(buildID, "success", map[string]float64{"time_ms": 1000}); err != nil {
		t.Fatal(err)
	}

	history, err := database.GetBuildHistory(10)
	if err != nil {
		t.Fatal(err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 build in history, got %d", len(history))
	}
}

func TestBuildIntegration_FileWatching(t *testing.T) {
	dir, err := filepath.Abs(testFixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()

	watchDirs := []string{dir}

	for _, d := range watchDirs {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			t.Errorf("watch directory does not exist: %s", d)
		}
	}

	if cfg.Watch.Debounce != 100 {
		t.Errorf("expected default debounce of 100ms, got %d", cfg.Watch.Debounce)
	}
}
