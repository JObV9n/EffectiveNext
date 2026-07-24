package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect and manage the binary cache",
	Long: `cache provides subcommands to view cache statistics and manage cache entries.

Use 'effective-next cache --help' to see available subcommands.`,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	RunE:  runCacheStats,
}

var cachePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Remove all cache entries",
	RunE:  runCachePurge,
}

func init() {
	cacheCmd.AddCommand(cacheStatsCmd, cachePurgeCmd)
	RootCmd.AddCommand(cacheCmd)
}

func openBuildDB() (*db.DB, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, ".effective-next", "build.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no build database found; run 'effective-next build' first")
	}
	return db.Open(dbPath)
}

func runCacheStats(cmd *cobra.Command, args []string) error {
	database, err := openBuildDB()
	if err != nil {
		return err
	}
	defer database.Close()

	count, err := database.CacheCount()
	if err != nil {
		return fmt.Errorf("cache count: %w", err)
	}

	size, err := database.CacheSize()
	if err != nil {
		return fmt.Errorf("cache size: %w", err)
	}

	if jsonOut {
		type cacheStats struct {
			Entries int   `json:"entries"`
			Size    int64 `json:"size_bytes"`
			SizeMB  float64 `json:"size_mb"`
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(cacheStats{
			Entries: count,
			Size:    size,
			SizeMB:  float64(size) / (1024 * 1024),
		})
	}

	cmd.Println(fmt.Sprintf("Cache entries: %d", count))
	cmd.Println(fmt.Sprintf("Cache size:    %.2f MB", float64(size)/(1024*1024)))
	return nil
}

func runCachePurge(cmd *cobra.Command, args []string) error {
	database, err := openBuildDB()
	if err != nil {
		return err
	}
	defer database.Close()

	if err := database.PurgeCache(); err != nil {
		return fmt.Errorf("purge cache: %w", err)
	}

	cmd.Println("effective-next: cache purged")
	return nil
}
