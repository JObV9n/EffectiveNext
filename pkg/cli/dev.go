package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/JobV9n/effectiveNext/pkg/engine"
	"github.com/JobV9n/effectiveNext/pkg/watch"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start development server with fast incremental builds",
	Long: `dev starts a file watcher, runs incremental builds on changes,
and proxies to next dev for hot module replacement.

Use this instead of 'next dev' for faster iteration.

Examples:
  effective-next dev                      # Start on default port 3000
  effective-next dev -p 8080              # Start on port 8080
  effective-next dev -H 0.0.0.0           # Listen on all interfaces`,
	RunE: runDev,
}

var (
	devPort     int
	devHostname string
)

func init() {
	devCmd.Flags().IntVarP(&devPort, "port", "p", 3000, "Port to listen on")
	devCmd.Flags().StringVar(&devHostname, "hostname", "localhost", "Hostname to listen on")
	RootCmd.AddCommand(devCmd)
}

func runDev(cmd *cobra.Command, args []string) error {
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

	cmd.PrintErrln("effective-next: scanning initial state...")
	files, err := scanProject(dir, cfg.Scanner)
	if err != nil {
		return fmt.Errorf("failed to scan project files: %w\n\nHint: Check if the directory is readable", err)
	}
	cmd.PrintErrln(fmt.Sprintf("effective-next: found %d files", len(files)))

	eng := engine.New(database)
	eng.Snapshot(files)

	cmd.PrintErrln(fmt.Sprintf("effective-next: starting file watcher on %s...", dir))
	w := watch.New(dir, cfg.Watch.Debounce)
	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w\n\nHint: Check if the directory is readable and inotify limits are not exceeded", err)
	}
	defer w.Stop()

	bin := findNextBin(dir)
	if bin == "" {
		cmd.PrintErrln("effective-next: next binary not found, running in watch-only mode")
		cmd.PrintErrln("effective-next: press Ctrl+C to stop")
		for e := range w.Events() {
			cmd.PrintErrln(fmt.Sprintf("effective-next: change detected: %s", e.Path))
		}
		return nil
	}

	cmd.PrintErrln(fmt.Sprintf("effective-next: starting next dev on %s:%d...", devHostname, devPort))

	args = append([]string{"dev", "-p", fmt.Sprintf("%d", devPort), "-H", devHostname}, args...)
	c := exec.Command(bin, args...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = cmd.OutOrStdout()
	c.Stderr = cmd.ErrOrStderr()

	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start next dev: %w\n\nHint: Make sure dependencies are installed and port %d is available", err, devPort)
	}

	cmd.PrintErrln("effective-next: watching for changes... (Ctrl+C to stop)")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for e := range w.Events() {
			cmd.PrintErrln(fmt.Sprintf("effective-next: change detected: %s", e.Path))

			scanStart := time.Now()
			currentFiles, err := scanProject(dir, cfg.Scanner)
			if err != nil {
				cmd.PrintErrln(fmt.Sprintf("effective-next: scan error: %v", err))
				continue
			}

			changes := eng.ComputeChanges(currentFiles)
			if len(changes) > 0 {
				affected := eng.AffectedFiles(changes)
				cmd.PrintErrln(fmt.Sprintf("effective-next: %d file(s) changed, %d affected (scan: %v)",
					len(changes), len(affected), time.Since(scanStart).Round(time.Millisecond)))
				eng.Snapshot(currentFiles)
			}
		}
	}()

	<-sigCh
	cmd.PrintErrln("\neffective-next: shutting down...")

	c.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() { done <- c.Wait() }()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		c.Process.Kill()
	}

	cmd.PrintErrln("effective-next: stopped")
	return nil
}
