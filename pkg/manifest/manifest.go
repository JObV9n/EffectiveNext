package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RouteEntry represents a route in the manifest.
type RouteEntry struct {
	Path     string `json:"path"`
	File     string `json:"file"`
	Type     string `json:"type"`
	Layout   string `json:"layout,omitempty"`
	Children []string `json:"children,omitempty"`
}

// AssetEntry represents an asset in the manifest.
type AssetEntry struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

// DependencyEntry represents a dependency in the manifest.
type DependencyEntry struct {
	Path   string   `json:"path"`
	Hash   string   `json:"hash"`
	Size   int64    `json:"size"`
	Deps   []string `json:"deps,omitempty"`
	Type   string   `json:"type"`
}

// ChunkEntry represents a build chunk.
type ChunkEntry struct {
	ID       string   `json:"id"`
	Files    []string `json:"files"`
	Size     int64    `json:"size"`
	Parents  []string `json:"parents,omitempty"`
}

// Manifest is the complete project manifest.
type Manifest struct {
	Version      string             `json:"version"`
	GeneratedAt  time.Time          `json:"generated_at"`
	Routes       []RouteEntry       `json:"routes"`
	Assets       []AssetEntry       `json:"assets"`
	Dependencies []DependencyEntry  `json:"dependencies"`
	Chunks       []ChunkEntry       `json:"chunks"`
	Stats        Stats              `json:"stats"`
}

// Stats holds manifest statistics.
type Stats struct {
	TotalRoutes      int   `json:"total_routes"`
	TotalAssets      int   `json:"total_assets"`
	TotalDependencies int  `json:"total_dependencies"`
	TotalChunks      int   `json:"total_chunks"`
	TotalSize        int64 `json:"total_size"`
}

// Generator builds manifests.
type Generator struct {
	root     string
	manifest *Manifest
}

// NewGenerator creates a new manifest generator.
func NewGenerator(root string) *Generator {
	return &Generator{
		root: root,
		manifest: &Manifest{
			Version:     "1.0",
			GeneratedAt: time.Now(),
		},
	}
}

// AddRoute adds a route to the manifest.
func (g *Generator) AddRoute(entry RouteEntry) {
	g.manifest.Routes = append(g.manifest.Routes, entry)
}

// AddAsset adds an asset to the manifest.
func (g *Generator) AddAsset(entry AssetEntry) {
	g.manifest.Assets = append(g.manifest.Assets, entry)
}

// AddDependency adds a dependency to the manifest.
func (g *Generator) AddDependency(entry DependencyEntry) {
	g.manifest.Dependencies = append(g.manifest.Dependencies, entry)
}

// AddChunk adds a chunk to the manifest.
func (g *Generator) AddChunk(entry ChunkEntry) {
	g.manifest.Chunks = append(g.manifest.Chunks, entry)
}

// Build finalizes the manifest with statistics.
func (g *Generator) Build() *Manifest {
	g.manifest.Stats = Stats{
		TotalRoutes:       len(g.manifest.Routes),
		TotalAssets:       len(g.manifest.Assets),
		TotalDependencies: len(g.manifest.Dependencies),
		TotalChunks:       len(g.manifest.Chunks),
	}

	for _, a := range g.manifest.Assets {
		g.manifest.Stats.TotalSize += a.Size
	}

	sort.Slice(g.manifest.Routes, func(i, j int) bool {
		return g.manifest.Routes[i].Path < g.manifest.Routes[j].Path
	})

	return g.manifest
}

// WriteJSON writes the manifest as JSON.
func (g *Generator) WriteJSON(path string) error {
	g.manifest.GeneratedAt = time.Now()
	g.manifest.Stats = Stats{
		TotalRoutes:       len(g.manifest.Routes),
		TotalAssets:       len(g.manifest.Assets),
		TotalDependencies: len(g.manifest.Dependencies),
		TotalChunks:       len(g.manifest.Chunks),
	}
	for _, a := range g.manifest.Assets {
		g.manifest.Stats.TotalSize += a.Size
	}

	data, err := json.MarshalIndent(g.manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadManifest reads a manifest from a JSON file.
func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Diff compares two manifests and returns the differences.
func Diff(a, b *Manifest) DiffResult {
	result := DiffResult{}

	aRoutes := make(map[string]RouteEntry)
	for _, r := range a.Routes {
		aRoutes[r.Path] = r
	}
	bRoutes := make(map[string]RouteEntry)
	for _, r := range b.Routes {
		bRoutes[r.Path] = r
	}

	for path, r := range bRoutes {
		if _, ok := aRoutes[path]; !ok {
			result.AddedRoutes = append(result.AddedRoutes, r)
		}
	}
	for path, r := range aRoutes {
		if _, ok := bRoutes[path]; !ok {
			result.RemovedRoutes = append(result.RemovedRoutes, r)
		}
	}

	aAssets := make(map[string]AssetEntry)
	for _, a := range a.Assets {
		aAssets[a.Path] = a
	}
	bAssets := make(map[string]AssetEntry)
	for _, a := range b.Assets {
		bAssets[a.Path] = a
	}

	for path, a := range bAssets {
		if existing, ok := aAssets[path]; !ok {
			result.AddedAssets = append(result.AddedAssets, a)
		} else if existing.Hash != a.Hash {
			result.ChangedAssets = append(result.ChangedAssets, a)
		}
	}
	for path := range aAssets {
		if _, ok := bAssets[path]; !ok {
			result.RemovedAssets = append(result.RemovedAssets, aAssets[path])
		}
	}

	return result
}

// DiffResult holds manifest differences.
type DiffResult struct {
	AddedRoutes    []RouteEntry
	RemovedRoutes  []RouteEntry
	AddedAssets    []AssetEntry
	RemovedAssets  []AssetEntry
	ChangedAssets  []AssetEntry
}

// FindManifest finds effective-next manifest files in the project.
func FindManifest(root string) string {
	candidates := []string{
		filepath.Join(root, ".effective-next", "manifest.json"),
		filepath.Join(root, "effective-next-manifest.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}