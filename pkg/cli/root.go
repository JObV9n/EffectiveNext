// Package cli implements the command-line interface for effective-next.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

var (
	cfgFile string
	verbose bool
	quiet   bool
	jsonOut bool
)

// RootCmd is the entrypoint for all effective-next commands.
var RootCmd = &cobra.Command{
	Use:   "effective-next",
	Short: "A high-performance build accelerator for Next.js",
	Long: `effectiveNext preprocesses, caches, and orchestrates Next.js builds to reduce
cold build time, incremental build time, disk usage, and memory consumption
without requiring changes to application source code.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if quiet && verbose {
			return fmt.Errorf("cannot use both --quiet and --verbose")
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() error {
	return ExecuteContext(context.Background())
}

// ExecuteContext runs the root command with the provided context.
func ExecuteContext(ctx context.Context) error {
	RootCmd.SetIn(os.Stdin)
	RootCmd.SetOut(os.Stdout)
	RootCmd.SetErr(os.Stderr)
	return RootCmd.ExecuteContext(ctx)
}

// CreateRootCmd creates a new root command for testing purposes.
func CreateRootCmd() *cobra.Command {
	return RootCmd
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to effective-next config file")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	RootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")
	RootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output results as JSON")
}
