package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	staticImportRe  = regexp.MustCompile(`(?:import|from)\s+["']([^"']+)["']`)
	dynamicImportRe = regexp.MustCompile(`import\(\s*["']([^"']+)["']\s*\)`)
	exportRe        = regexp.MustCompile(`export\s+(?:default\s+)?(?:function|class|const|let|var|type|interface|enum)\s+(\w+)`)
	requireRe       = regexp.MustCompile(`require\(\s*["']([^"']+)["']\s*\)`)
	cssImportRe     = regexp.MustCompile(`@import\s+(?:url\()?["']([^"']+)["']\)?`)
	metadataRe      = regexp.MustCompile(`export\s+const\s+metadata\s*[:=]`)
	dynamicRouteRe  = regexp.MustCompile(`\[(\w+)\]`)
	catchAllRe      = regexp.MustCompile(`\[\.\.\.(\w+)\]`)
)

// Import represents a parsed import statement.
type Import struct {
	Path    string
	Dynamic bool
}

// Export represents a parsed export.
type Export struct {
	Name string
}

// Result holds the parsed output for a file.
type Result struct {
	Path       string
	Imports    []Import
	Exports    []Export
	HasMetadata bool
	DynRoutes  []string
	Extensions []string
}

// ParseFile parses a single source file and extracts imports/exports.
func ParseFile(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseContent(path, string(data))
}

// ParseContent parses file content and extracts imports/exports.
func ParseContent(path, content string) (*Result, error) {
	result := &Result{Path: path}

	for _, match := range staticImportRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			imp := match[1]
			if !strings.HasPrefix(imp, "http") && !strings.HasPrefix(imp, "node:") {
				result.Imports = append(result.Imports, Import{Path: imp, Dynamic: false})
			}
		}
	}

	for _, match := range dynamicImportRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			imp := match[1]
			if !strings.HasPrefix(imp, "http") && !strings.HasPrefix(imp, "node:") {
				result.Imports = append(result.Imports, Import{Path: imp, Dynamic: true})
			}
		}
	}

	for _, match := range requireRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			imp := match[1]
			if !strings.HasPrefix(imp, "http") && !strings.HasPrefix(imp, "node:") {
				result.Imports = append(result.Imports, Import{Path: imp, Dynamic: true})
			}
		}
	}

	for _, match := range exportRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			result.Exports = append(result.Exports, Export{Name: match[1]})
		}
	}

	result.HasMetadata = metadataRe.MatchString(content)

	dynSegments := dynamicRouteRe.FindAllStringSubmatch(path, -1)
	for _, seg := range dynSegments {
		if len(seg) > 1 {
			result.DynRoutes = append(result.DynRoutes, seg[1])
		}
	}
	if catchAllRe.MatchString(path) {
		result.DynRoutes = append(result.DynRoutes, "...params")
	}

	return result, nil
}

// ParseDirectory parses all source files in a directory.
func ParseDirectory(dir string) ([]*Result, error) {
	var results []*Result
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".next" || name == ".git" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".mjs" || ext == ".cjs" {
			r, err := ParseFile(path)
			if err != nil {
				return nil
			}
			results = append(results, r)
		}
		return nil
	})
	return results, err
}

// ResolveImport resolves an import path relative to the importing file.
func ResolveImport(importPath, fromFile string) string {
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		dir := filepath.Dir(fromFile)
		resolved := filepath.Join(dir, importPath)
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".json"} {
			if _, err := os.Stat(resolved + ext); err == nil {
				return resolved + ext
			}
		}
		if _, err := os.Stat(resolved); err == nil {
			if info, err := os.Stat(resolved); err == nil && info.IsDir() {
				for _, idx := range []string{"index.ts", "index.tsx", "index.js", "index.jsx"} {
					if _, err := os.Stat(filepath.Join(resolved, idx)); err == nil {
						return filepath.Join(resolved, idx)
					}
				}
			}
			return resolved
		}
	}
	return importPath
}

// Stats holds parser statistics.
type Stats struct {
	TotalFiles    int
	TotalImports  int
	TotalExports  int
	StaticImports int
	DynamicImports int
}

// ParseStats returns statistics for parsed results.
func ParseStats(results []*Result) Stats {
	stats := Stats{}
	for _, r := range results {
		stats.TotalFiles++
		stats.TotalExports += len(r.Exports)
		for _, imp := range r.Imports {
			stats.TotalImports++
			if imp.Dynamic {
				stats.DynamicImports++
			} else {
				stats.StaticImports++
			}
		}
	}
	return stats
}

// BuildDependencyGraph builds a graph from parsed results.
func BuildDependencyGraph(results []*Result, rootDir string) map[string][]string {
	graph := make(map[string][]string)
	for _, r := range results {
		for _, imp := range r.Imports {
			resolved := ResolveImport(imp.Path, r.Path)
			graph[r.Path] = append(graph[r.Path], resolved)
		}
	}
	return graph
}

// Not yet implemented - just a placeholder
var _ = fmt.Sprintf