package graph

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "graph.db")
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func buildTestGraph() *Graph {
	g := New()
	g.AddNode(Node{ID: "a", Path: "src/a.ts", Type: NodeTypeTS, Hash: 100, Size: 500})
	g.AddNode(Node{ID: "b", Path: "src/b.tsx", Type: NodeTypeTSX, Hash: 200, Size: 300})
	g.AddNode(Node{ID: "c", Path: "src/c.css", Type: NodeTypeCSS, Hash: 300, Size: 100})
	g.AddEdge("a", "b", EdgeStaticImport, false)
	g.AddEdge("a", "c", EdgeCSSImport, false)
	g.AddEdge("b", "c", EdgeStaticImport, false)
	return g
}

func TestStore_SaveAndLoad(t *testing.T) {
	store := setupTestStore(t)
	g := buildTestGraph()

	if err := store.Save(g); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.NodeCount() != 3 {
		t.Errorf("NodeCount = %d, want 3", loaded.NodeCount())
	}
	if loaded.EdgeCount() != 3 {
		t.Errorf("EdgeCount = %d, want 3", loaded.EdgeCount())
	}

	nodeA, ok := loaded.GetNode("a")
	if !ok {
		t.Fatal("expected node 'a'")
	}
	if nodeA.Path != "src/a.ts" {
		t.Errorf("Path = %q, want %q", nodeA.Path, "src/a.ts")
	}
	if nodeA.Hash != 100 {
		t.Errorf("Hash = %d, want 100", nodeA.Hash)
	}
	if nodeA.Size != 500 {
		t.Errorf("Size = %d, want 500", nodeA.Size)
	}
}

func TestStore_LoadEmpty(t *testing.T) {
	store := setupTestStore(t)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.NodeCount() != 0 {
		t.Errorf("NodeCount = %d, want 0", loaded.NodeCount())
	}
}

func TestStore_Clear(t *testing.T) {
	store := setupTestStore(t)
	g := buildTestGraph()

	if err := store.Save(g); err != nil {
		t.Fatal(err)
	}

	if err := store.Clear(); err != nil {
		t.Fatal(err)
	}

	count, _ := store.NodeCount()
	if count != 0 {
		t.Errorf("after Clear, NodeCount = %d, want 0", count)
	}
}

func TestStore_NodeCount(t *testing.T) {
	store := setupTestStore(t)
	g := buildTestGraph()

	store.Save(g)

	count, err := store.NodeCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("NodeCount = %d, want 3", count)
	}
}

func TestStore_EdgeCount(t *testing.T) {
	store := setupTestStore(t)
	g := buildTestGraph()

	store.Save(g)

	count, err := store.EdgeCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("EdgeCount = %d, want 3", count)
	}
}

func TestStore_OverwriteGraph(t *testing.T) {
	store := setupTestStore(t)

	g1 := New()
	g1.AddNode(Node{ID: "x", Path: "x.ts", Type: NodeTypeTS})
	store.Save(g1)

	g2 := New()
	g2.AddNode(Node{ID: "y", Path: "y.ts", Type: NodeTypeTS})
	g2.AddNode(Node{ID: "z", Path: "z.ts", Type: NodeTypeTS})
	store.Save(g2)

	loaded, _ := store.Load()
	if loaded.NodeCount() != 2 {
		t.Errorf("NodeCount = %d, want 2 after overwrite", loaded.NodeCount())
	}
	if _, ok := loaded.GetNode("x"); ok {
		t.Error("expected node 'x' to be removed after overwrite")
	}
}

func TestStore_PreservesEdgeTypes(t *testing.T) {
	store := setupTestStore(t)
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeDynamicImport, true)

	store.Save(g)
	loaded, _ := store.Load()

	edges := loaded.Dependencies("a")
	if len(edges) != 1 {
		t.Fatalf("Dependencies(a) = %d, want 1", len(edges))
	}
	if edges[0].Type != EdgeDynamicImport {
		t.Errorf("EdgeType = %q, want %q", edges[0].Type, EdgeDynamicImport)
	}
	if !edges[0].Dynamic {
		t.Error("expected edge to be dynamic")
	}
}

func TestStore_RoundTripLargeGraph(t *testing.T) {
	store := setupTestStore(t)
	g := New()

	for i := 0; i < 100; i++ {
		id := string(rune('a' + (i % 26)))
		path := "src/" + id + ".ts"
		g.AddNode(Node{ID: id, Path: path, Type: NodeTypeTS, Hash: uint64(i)})
	}

	g.AddEdge("a", "b", EdgeStaticImport, false)
	g.AddEdge("b", "c", EdgeStaticImport, false)
	g.AddEdge("a", "d", EdgeDynamicImport, true)

	store.Save(g)
	loaded, _ := store.Load()

	if loaded.NodeCount() != 26 {
		t.Errorf("NodeCount = %d, want 26 (deduplicated)", loaded.NodeCount())
	}
}

func TestOpenStore_InvalidPath(t *testing.T) {
	_, err := OpenStore("/nonexistent/dir/graph.db")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestStore_FileExists(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected database file to exist")
	}
}
