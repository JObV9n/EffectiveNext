package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator("/test/root")
	if g == nil {
		t.Fatal("expected non-nil generator")
	}
	if g.root != "/test/root" {
		t.Errorf("expected root /test/root, got %s", g.root)
	}
}

func TestAddRoute(t *testing.T) {
	g := NewGenerator("/test")

	g.AddRoute(RouteEntry{
		Path: "/dashboard",
		File: "app/dashboard/page.tsx",
		Type: "page",
	})

	if len(g.manifest.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(g.manifest.Routes))
	}
	if g.manifest.Routes[0].Path != "/dashboard" {
		t.Errorf("expected path /dashboard, got %s", g.manifest.Routes[0].Path)
	}
}

func TestAddAsset(t *testing.T) {
	g := NewGenerator("/test")

	g.AddAsset(AssetEntry{
		Path: "styles/main.css",
		Size: 1024,
		Type: ".css",
	})

	if len(g.manifest.Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(g.manifest.Assets))
	}
}

func TestAddDependency(t *testing.T) {
	g := NewGenerator("/test")

	g.AddDependency(DependencyEntry{
		Path: "react",
		Type: "package",
	})

	if len(g.manifest.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(g.manifest.Dependencies))
	}
}

func TestAddChunk(t *testing.T) {
	g := NewGenerator("/test")

	g.AddChunk(ChunkEntry{
		ID:    "main",
		Files: []string{"a.js", "b.js"},
	})

	if len(g.manifest.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(g.manifest.Chunks))
	}
}

func TestBuild(t *testing.T) {
	g := NewGenerator("/test")

	g.AddRoute(RouteEntry{Path: "/"})
	g.AddRoute(RouteEntry{Path: "/about"})
	g.AddAsset(AssetEntry{Path: "a.css", Size: 100})
	g.AddAsset(AssetEntry{Path: "b.css", Size: 200})
	g.AddDependency(DependencyEntry{Path: "react"})
	g.AddChunk(ChunkEntry{ID: "main"})

	m := g.Build()
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}

	if m.Stats.TotalRoutes != 2 {
		t.Errorf("expected 2 routes, got %d", m.Stats.TotalRoutes)
	}
	if m.Stats.TotalAssets != 2 {
		t.Errorf("expected 2 assets, got %d", m.Stats.TotalAssets)
	}
	if m.Stats.TotalDependencies != 1 {
		t.Errorf("expected 1 dependency, got %d", m.Stats.TotalDependencies)
	}
	if m.Stats.TotalChunks != 1 {
		t.Errorf("expected 1 chunk, got %d", m.Stats.TotalChunks)
	}
	if m.Stats.TotalSize != 300 {
		t.Errorf("expected total size 300, got %d", m.Stats.TotalSize)
	}
}

func TestBuild_SortsRoutes(t *testing.T) {
	g := NewGenerator("/test")

	g.AddRoute(RouteEntry{Path: "/z"})
	g.AddRoute(RouteEntry{Path: "/a"})
	g.AddRoute(RouteEntry{Path: "/m"})

	m := g.Build()

	if m.Routes[0].Path != "/a" {
		t.Errorf("expected first route /a, got %s", m.Routes[0].Path)
	}
	if m.Routes[1].Path != "/m" {
		t.Errorf("expected second route /m, got %s", m.Routes[1].Path)
	}
	if m.Routes[2].Path != "/z" {
		t.Errorf("expected third route /z, got %s", m.Routes[2].Path)
	}
}

func TestWriteJSON(t *testing.T) {
	g := NewGenerator("/test")
	g.AddRoute(RouteEntry{Path: "/dashboard"})

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	if err := g.WriteJSON(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	if len(m.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(m.Routes))
	}
}

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	m := &Manifest{
		Version:     "1.0",
		GeneratedAt: time.Now(),
		Routes: []RouteEntry{
			{Path: "/", File: "page.tsx"},
		},
	}

	data, _ := json.Marshal(m)
	os.WriteFile(path, data, 0o644)

	got, err := ReadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(got.Routes))
	}
}

func TestReadManifest_NotFound(t *testing.T) {
	_, err := ReadManifest("/nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestDiff_SameManifests(t *testing.T) {
	a := &Manifest{
		Routes: []RouteEntry{{Path: "/"}},
		Assets: []AssetEntry{{Path: "a.css", Hash: "abc"}},
	}
	b := &Manifest{
		Routes: []RouteEntry{{Path: "/"}},
		Assets: []AssetEntry{{Path: "a.css", Hash: "abc"}},
	}

	diff := Diff(a, b)
	if len(diff.AddedRoutes) != 0 || len(diff.RemovedRoutes) != 0 {
		t.Error("expected no route differences")
	}
	if len(diff.AddedAssets) != 0 || len(diff.RemovedAssets) != 0 {
		t.Error("expected no asset differences")
	}
}

func TestDiff_AddedRoutes(t *testing.T) {
	a := &Manifest{
		Routes: []RouteEntry{{Path: "/"}},
	}
	b := &Manifest{
		Routes: []RouteEntry{
			{Path: "/"},
			{Path: "/about"},
		},
	}

	diff := Diff(a, b)
	if len(diff.AddedRoutes) != 1 {
		t.Errorf("expected 1 added route, got %d", len(diff.AddedRoutes))
	}
}

func TestDiff_RemovedRoutes(t *testing.T) {
	a := &Manifest{
		Routes: []RouteEntry{
			{Path: "/"},
			{Path: "/about"},
		},
	}
	b := &Manifest{
		Routes: []RouteEntry{{Path: "/"}},
	}

	diff := Diff(a, b)
	if len(diff.RemovedRoutes) != 1 {
		t.Errorf("expected 1 removed route, got %d", len(diff.RemovedRoutes))
	}
}

func TestDiff_AddedAssets(t *testing.T) {
	a := &Manifest{
		Assets: []AssetEntry{{Path: "a.css", Hash: "abc"}},
	}
	b := &Manifest{
		Assets: []AssetEntry{
			{Path: "a.css", Hash: "abc"},
			{Path: "b.css", Hash: "def"},
		},
	}

	diff := Diff(a, b)
	if len(diff.AddedAssets) != 1 {
		t.Errorf("expected 1 added asset, got %d", len(diff.AddedAssets))
	}
}

func TestDiff_RemovedAssets(t *testing.T) {
	a := &Manifest{
		Assets: []AssetEntry{
			{Path: "a.css", Hash: "abc"},
			{Path: "b.css", Hash: "def"},
		},
	}
	b := &Manifest{
		Assets: []AssetEntry{{Path: "a.css", Hash: "abc"}},
	}

	diff := Diff(a, b)
	if len(diff.RemovedAssets) != 1 {
		t.Errorf("expected 1 removed asset, got %d", len(diff.RemovedAssets))
	}
}

func TestDiff_ChangedAssets(t *testing.T) {
	a := &Manifest{
		Assets: []AssetEntry{{Path: "a.css", Hash: "old"}},
	}
	b := &Manifest{
		Assets: []AssetEntry{{Path: "a.css", Hash: "new"}},
	}

	diff := Diff(a, b)
	if len(diff.ChangedAssets) != 1 {
		t.Errorf("expected 1 changed asset, got %d", len(diff.ChangedAssets))
	}
}

func TestFindManifest(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".effective-next"), 0o755)
	os.WriteFile(filepath.Join(dir, ".effective-next", "manifest.json"), []byte("{}"), 0o644)

	found := FindManifest(dir)
	if found == "" {
		t.Error("expected to find manifest")
	}
}

func TestFindManifest_NotFound(t *testing.T) {
	dir := t.TempDir()
	found := FindManifest(dir)
	if found != "" {
		t.Errorf("expected empty, got %s", found)
	}
}

func TestManifest_Version(t *testing.T) {
	g := NewGenerator("/test")
	m := g.Build()
	if m.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", m.Version)
	}
}

func TestManifest_GeneratedAt(t *testing.T) {
	g := NewGenerator("/test")
	m := g.Build()
	if m.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
	}
}

func TestEmptyManifest(t *testing.T) {
	g := NewGenerator("/test")
	m := g.Build()

	if m.Stats.TotalRoutes != 0 {
		t.Errorf("expected 0 routes, got %d", m.Stats.TotalRoutes)
	}
	if m.Stats.TotalAssets != 0 {
		t.Errorf("expected 0 assets, got %d", m.Stats.TotalAssets)
	}
	if m.Stats.TotalSize != 0 {
		t.Errorf("expected 0 size, got %d", m.Stats.TotalSize)
	}
}

func TestDiff_EmptyManifests(t *testing.T) {
	a := &Manifest{}
	b := &Manifest{}

	diff := Diff(a, b)
	if len(diff.AddedRoutes) != 0 || len(diff.RemovedRoutes) != 0 {
		t.Error("expected no differences for empty manifests")
	}
}
