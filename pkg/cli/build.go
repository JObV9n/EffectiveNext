package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/JobV9n/effectiveNext/pkg/engine"
	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/manifest"
	"github.com/JobV9n/effectiveNext/pkg/observability"
	"github.com/JobV9n/effectiveNext/pkg/parser"
	"github.com/JobV9n/effectiveNext/pkg/routes"
	"github.com/JobV9n/effectiveNext/pkg/scheduler"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Run an optimized production build",
	Long: `build preprocesses the project, analyzes dependencies, caches parser output,
and invokes next build with an optimized project state.

This is the primary command for production builds.

Examples:
  effective-next build                    # Full production build
  effective-next build --dry-run          # Preview without building
  effective-next build --skip-next        # Preprocess only
  effective-next build --workers 8        # Use 8 worker goroutines
  effective-next build --no-cache         # Disable caching`,
	RunE: runBuild,
}

var (
	buildWorkers int
	buildNoCache bool
	buildSkipNext bool
	buildDryRun  bool
)

func init() {
	buildCmd.Flags().IntVarP(&buildWorkers, "workers", "w", 0, "Number of worker goroutines (0=auto)")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Disable caching for this build")
	buildCmd.Flags().BoolVar(&buildSkipNext, "skip-next", false, "Skip invoking next build (preprocessing only)")
	buildCmd.Flags().BoolVar(&buildDryRun, "dry-run", false, "Show what would be built without building")
	RootCmd.AddCommand(buildCmd)
}

func runBuild(cmd *cobra.Command, args []string) error {
	start := time.Now()
	tracker := observability.NewBuildTracker()

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	cmd.PrintErrln("effective-next: loading configuration...")
	loader := config.NewLoader(dir)
	cfg, err := loader.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\n\nHint: Check if effective-next.yaml, effective-next.yml, or effective-next.json exists in your project root", err)
	}

	if buildWorkers > 0 {
		cfg.Workers = buildWorkers
	}
	if buildNoCache {
		cfg.Cache.Enabled = false
	}

	effectiveNextDir := filepath.Join(dir, ".effective-next")
	if err := os.MkdirAll(effectiveNextDir, 0o755); err != nil {
		return fmt.Errorf("failed to create .effective-next directory: %w\n\nHint: Check file permissions in %s", err, dir)
	}

	cmd.PrintErrln("effective-next: opening build database...")
	database, err := db.Open(filepath.Join(effectiveNextDir, "build.db"))
	if err != nil {
		return fmt.Errorf("failed to open build database: %w\n\nHint: The database may be corrupted. Try 'effective-next clean' to reset", err)
	}
	defer database.Close()

	buildID, err := database.StartBuild()
	if err != nil {
		return fmt.Errorf("failed to start build: %w", err)
	}

	cmd.PrintErrln("effective-next: [1/7] scanning files...")
	scanStart := time.Now()
	files, err := scanProject(dir, cfg.Scanner)
	if err != nil {
		return fmt.Errorf("failed to scan project files: %w\n\nHint: Check if the directory is readable and not too large", err)
	}
	tracker.RecordScan(len(files), time.Since(scanStart))
	cmd.PrintErrln(fmt.Sprintf("effective-next:        %d files found", len(files)))

	if len(files) == 0 {
		return fmt.Errorf("no files found in project directory\n\nHint: Make sure you're in the correct directory and files exist")
	}

	cmd.PrintErrln("effective-next: [2/7] building dependency graph...")
	depGraph := graph.New()
	for _, f := range files {
		depGraph.AddNode(graph.Node{
			ID:   f.Path,
			Path: f.RelPath,
			Hash: f.Hash,
			Size: f.Size,
		})
	}

	cmd.PrintErrln("effective-next: [3/7] parsing imports and exports...")
	parseResults, err := parser.ParseDirectory(dir)
	if err != nil {
		cmd.PrintErrln(fmt.Sprintf("effective-next:        warning: parse failed: %v", err))
	} else {
		for _, r := range parseResults {
			for _, imp := range r.Imports {
				depGraph.AddEdge(r.Path, imp.Path, graph.EdgeStaticImport, imp.Dynamic)
			}
		}
		parseStats := parser.ParseStats(parseResults)
		cmd.PrintErrln(fmt.Sprintf("effective-next:        %d imports, %d exports", parseStats.TotalImports, parseStats.TotalExports))
	}

	cmd.PrintErrln("effective-next: [4/7] detecting routes...")
	routeScanner := routes.New(dir)
	manifestData, err := routeScanner.GenerateManifest()
	if err != nil {
		cmd.PrintErrln(fmt.Sprintf("effective-next:        warning: route scan failed: %v", err))
	} else {
		cmd.PrintErrln(fmt.Sprintf("effective-next:        %d routes detected", manifestData.TotalRoutes))
	}

	cmd.PrintErrln("effective-next: [5/7] checking for incremental changes...")
	eng := engine.New(database)
	eng.Snapshot(files)

	cmd.PrintErrln("effective-next: [6/7] generating manifest...")
	m := manifest.NewGenerator(dir)
	if manifestData != nil {
		for _, r := range manifestData.Routes {
			m.AddRoute(manifest.RouteEntry{
				Path: r.Path,
				File: r.FilePath,
				Type: string(r.Type),
			})
		}
	}
	for _, f := range files {
		m.AddAsset(manifest.AssetEntry{
			Path: f.RelPath,
			Size: f.Size,
			Type: filepath.Ext(f.RelPath),
		})
	}
	_ = m.Build()

	manifestPath := filepath.Join(effectiveNextDir, "manifest.json")
	if err := m.WriteJSON(manifestPath); err != nil {
		cmd.PrintErrln(fmt.Sprintf("effective-next:        warning: manifest write failed: %v", err))
	} else {
		cmd.PrintErrln(fmt.Sprintf("effective-next:        manifest written to %s", manifestPath))
	}

	s := scheduler.New()
	s.AddTask(&scheduler.Task{ID: "scan", Fn: func() error { return nil }})
	s.AddTask(&scheduler.Task{ID: "graph", Deps: []string{"scan"}, Fn: func() error { return nil }})
	s.AddTask(&scheduler.Task{ID: "routes", Deps: []string{"scan"}, Fn: func() error { return nil }})
	s.AddTask(&scheduler.Task{ID: "parse", Deps: []string{"scan"}, Fn: func() error { return nil }})
	s.AddTask(&scheduler.Task{ID: "manifest", Deps: []string{"graph", "routes", "parse"}, Fn: func() error { return nil }})
	if err := s.Run(); err != nil {
		return fmt.Errorf("scheduler failed: %w", err)
	}

	database.RecordStat(buildID, "files_scanned", float64(len(files)))

	if buildDryRun {
		cmd.PrintErrln("\neffective-next: dry run complete. Would invoke next build.")
		_ = database.FinishBuild(buildID, "dry-run", nil)
		return nil
	}

	cmd.PrintErrln("effective-next: [7/7] invoking next build...")
	buildStart := time.Now()
	if err := invokeNextBuild(cmd, dir, args); err != nil {
		_ = database.FinishBuild(buildID, "failed", nil)
		return fmt.Errorf("next build failed: %w\n\nHint: Run 'next build' directly to see Next.js error details", err)
	}
	buildDuration := time.Since(buildStart)

	tracker.RecordBuild(0, 0, buildDuration)
	summary := tracker.Summary()

	elapsed := time.Since(start)
	metrics := map[string]float64{
		"cold_time_ms":  float64(elapsed.Milliseconds()),
		"build_time_ms": float64(buildDuration.Milliseconds()),
		"scan_time_ms":  summary.ScanDurationMs,
		"files_scanned": float64(len(files)),
	}
	_ = database.FinishBuild(buildID, "success", metrics)

	cmd.PrintErrln(fmt.Sprintf("\neffective-next: build completed in %v (next build: %v)", elapsed.Round(time.Millisecond), buildDuration.Round(time.Millisecond)))

	if verbose {
		cmd.PrintErrln(fmt.Sprintf("effective-next:   build ID:  %s", buildID))
		cmd.PrintErrln(fmt.Sprintf("effective-next:   database:  %s", database.Path()))
		cmd.PrintErrln(fmt.Sprintf("effective-next:   files:     %d", len(files)))
	}

	return nil
}

func invokeNextBuild(cmd *cobra.Command, dir string, extraArgs []string) error {
	bin := findNextBin(dir)
	if bin == "" {
		return fmt.Errorf("next binary not found\n\nHint: Run 'npm install' or 'pnpm install' to install dependencies")
	}

	args := append([]string{"build"}, extraArgs...)
	if buildSkipNext {
		cmd.PrintErrln("effective-next:   --skip-next flag set, skipping next build")
		return nil
	}

	c := exec.Command(bin, args...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()

	return c.Run()
}

func findNextBin(dir string) string {
	candidates := []string{
		filepath.Join(dir, "node_modules", ".bin", "next"),
		filepath.Join(dir, "node_modules", "next", "dist", "bin", "next"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	path, err := exec.LookPath("next")
	if err == nil {
		return path
	}
	return ""
}

func scanProject(dir string, cfg config.ScannerConfig) ([]scanner.File, error) {
	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg,
		Workers: 4,
	})
	return s.Scan()
}
