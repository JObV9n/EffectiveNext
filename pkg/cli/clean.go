package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove effectiveNext cache and build state",
	Long: `clean removes all effectiveNext-managed state:
- Binary cache (.effective-next-cache/)
- Build database (.effective-next.db)
- Temporary files

It does NOT remove node_modules, .next, or application source code.`,
	RunE: runClean,
}

var (
	cleanAll bool
)

func init() {
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all effectiveNext state including .effective-next directory")
	RootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	var removed []string

	effectiveNextDir := filepath.Join(dir, ".effective-next")
	if _, err := os.Stat(effectiveNextDir); err == nil {
		if err := os.RemoveAll(effectiveNextDir); err != nil {
			return fmt.Errorf("remove .effective-next: %w", err)
		}
		removed = append(removed, ".effective-next/")
	}

	cacheDir := filepath.Join(dir, ".effective-next-cache")
	if _, err := os.Stat(cacheDir); err == nil {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("remove .effective-next-cache: %w", err)
		}
		removed = append(removed, ".effective-next-cache/")
	}

	if cleanAll {
		candidates := []string{"effective-next.db", "effective-next-manifest.json"}
		for _, name := range candidates {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("remove %s: %w", name, err)
				}
				removed = append(removed, name)
			}
		}
	}

	if len(removed) == 0 {
		cmd.Println("effective-next: nothing to clean")
	} else {
		cmd.Println(fmt.Sprintf("effective-next: removed %s", strings.Join(removed, ", ")))
	}

	return nil
}
