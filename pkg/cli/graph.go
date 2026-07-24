package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/hash"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Export the dependency graph",
	Long: `graph exports the dependency graph in various formats:
- GraphViz DOT (for visualization)
- JSON (for programmatic use)
- Mermaid (for documentation)`,
	RunE: runGraph,
}

var (
	graphOutput string
	graphFormat string
)

func init() {
	graphCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	graphCmd.Flags().StringP("format", "f", "dot", "Output format: dot, json, mermaid")
	RootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	depGraph := graph.New()

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if name == "node_modules" || name == ".next" || name == ".git" || name == "dist" || name == ".effective-next" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
			h, _ := hash.Hash64File(path)
			relPath, _ := filepath.Rel(dir, path)
			depGraph.AddNode(graph.Node{
				ID:   path,
				Path: relPath,
				Hash: h,
				Size: info.Size(),
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	var output string
	switch graphFormat {
	case "json":
		enc := json.NewEncoder(nil)
		_ = enc
		data := fmt.Sprintf(`{"nodes":%d,"edges":%d}`, depGraph.NodeCount(), depGraph.EdgeCount())
		output = data
	case "mermaid":
		output = "graph TD\n"
		nodes := depGraph.Nodes()
		for _, n := range nodes {
			output += fmt.Sprintf("  %s[\"%s\"]\n", sanitizeID(n.ID), n.Path)
		}
		edges := depGraph.AllEdges()
		for _, e := range edges {
			output += fmt.Sprintf("  %s --> %s\n", sanitizeID(e.Source), sanitizeID(e.Target))
		}
	default:
		output = "digraph deps {\n"
		nodes := depGraph.Nodes()
		for _, n := range nodes {
			output += fmt.Sprintf("  %q [label=%q];\n", n.ID, n.Path)
		}
		edges := depGraph.AllEdges()
		for _, e := range edges {
			output += fmt.Sprintf("  %q -> %q;\n", e.Source, e.Target)
		}
		output += "}\n"
	}

	if graphOutput != "" {
		if err := os.WriteFile(graphOutput, []byte(output), 0o644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		cmd.Println(fmt.Sprintf("effective-next: graph written to %s", graphOutput))
	} else {
		cmd.Print(output)
	}

	return nil
}

func sanitizeID(s string) string {
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result[i] = c
		} else {
			result[i] = '_'
		}
	}
	return string(result)
}
