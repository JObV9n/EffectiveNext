package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/hash"
	"github.com/JobV9n/effectiveNext/pkg/routes"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze the project dependency graph",
	Long: `analyze examines the project and reports:
- Module sizes and dependencies
- Circular dependencies
- Route structure
- Cache effectiveness
- Build bottlenecks`,
	RunE: runAnalyze,
}

func init() {
	RootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	cmd.PrintErrln("effective-next: scanning project...")

	depGraph := graph.New()
	var fileCount int

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if name == "node_modules" || name == ".next" || name == ".git" || name == "dist" || name == ".effective-next" || name == "coverage" || name == "out" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".css" || ext == ".scss" || ext == ".json" {
			h, _ := hash.Hash64File(path)
			relPath, _ := filepath.Rel(dir, path)
			depGraph.AddNode(graph.Node{
				ID:   path,
				Path: relPath,
				Hash: h,
				Size: info.Size(),
			})
			fileCount++
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	cmd.PrintErrln(fmt.Sprintf("effective-next: found %d source files", fileCount))

	routeScanner := routes.New(dir)
	manifest, err := routeScanner.GenerateManifest()
	if err != nil {
		cmd.PrintErrln(fmt.Sprintf("effective-next: warning: route scan failed: %v", err))
	}

	if jsonOut {
		type analyzeResult struct {
			Files      int            `json:"files"`
			Nodes      int            `json:"graph_nodes"`
			Edges      int            `json:"graph_edges"`
			Routes     int            `json:"routes"`
			HasApp     bool           `json:"has_app_router"`
			HasPages   bool           `json:"has_pages_router"`
			Cycles     [][]string     `json:"cycles,omitempty"`
		}
		result := analyzeResult{
			Files:    fileCount,
			Nodes:    depGraph.NodeCount(),
			Edges:    depGraph.EdgeCount(),
			HasApp:   manifest != nil && manifest.HasApp,
			HasPages: manifest != nil && manifest.HasPages,
		}
		if manifest != nil {
			result.Routes = manifest.TotalRoutes
		}
		cycles := depGraph.DetectCycles()
		if len(cycles) > 0 {
			result.Cycles = cycles
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	stats := depGraph.Stats()
	cmd.Println(fmt.Sprintf("\nFiles:          %d", fileCount))
	cmd.Println(fmt.Sprintf("Graph nodes:    %d", stats.NodeCount))
	cmd.Println(fmt.Sprintf("Graph edges:    %d", stats.EdgeCount))

	if len(stats.TypeCount) > 0 {
		cmd.Println("\nFile types:")
		for t, count := range stats.TypeCount {
			cmd.Println(fmt.Sprintf("  %-8s %d", string(t), count))
		}
	}

	if manifest != nil {
		cmd.Println(fmt.Sprintf("\nRoutes:         %d", manifest.TotalRoutes))
		cmd.Println(fmt.Sprintf("App Router:     %v", manifest.HasApp))
		cmd.Println(fmt.Sprintf("Pages Router:   %v", manifest.HasPages))
	}

	cycles := depGraph.DetectCycles()
	if len(cycles) > 0 {
		cmd.PrintErrln(fmt.Sprintf("\n⚠ %d circular dependency chain(s) detected:", len(cycles)))
		for i, cycle := range cycles {
			cmd.PrintErrln(fmt.Sprintf("  %d. %s", i+1, strings.Join(cycle, " → ")))
		}
	} else {
		cmd.Println("\n✓ No circular dependencies detected")
	}

	return nil
}
