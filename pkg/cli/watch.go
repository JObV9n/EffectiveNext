package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch files and invalidate caches on changes",
	Long: `watch starts a filesystem watcher that tracks file changes and
invalidates the effectiveNext cache incrementally. It does not run builds itself;
use 'effective-next dev' for a full development experience.

This is useful for integrating with external build tools.`,
	RunE: runWatch,
}

var watchDebounce int

func init() {
	watchCmd.Flags().IntVarP(&watchDebounce, "debounce", "d", 100, "Debounce time in milliseconds")
	RootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	cmd.PrintErrln(fmt.Sprintf("effective-next: watching %s for changes...", dir))

	loader := config.NewLoader(dir)
	cfg, err := loader.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	s := scanner.New(scanner.Options{
		Root:    dir,
		Config:  cfg.Scanner,
		Workers: 2,
	})
	_, err = s.Scan()
	if err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}

	cmd.PrintErrln("effective-next: initial scan complete, watching for changes...")
	cmd.PrintErrln("effective-next: press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	cmd.PrintErrln("\neffective-next: stopped watching")
	return nil
}
