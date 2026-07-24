package graph

import (
	"fmt"
	"testing"
)

func BenchmarkGraphStore_Save_100(b *testing.B) {
	store := setupBenchStore(b)
	g := buildBenchGraph(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save(g)
	}
}

func BenchmarkGraphStore_Save_1000(b *testing.B) {
	store := setupBenchStore(b)
	g := buildBenchGraph(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save(g)
	}
}

func BenchmarkGraphStore_Load_100(b *testing.B) {
	store := setupBenchStore(b)
	g := buildBenchGraph(100)
	store.Save(g)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Load()
	}
}

func BenchmarkGraphStore_Load_1000(b *testing.B) {
	store := setupBenchStore(b)
	g := buildBenchGraph(1000)
	store.Save(g)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Load()
	}
}

func BenchmarkGraphStore_RoundTrip_100(b *testing.B) {
	store := setupBenchStore(b)
	g := buildBenchGraph(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Save(g)
		store.Load()
	}
}

func setupBenchStore(b *testing.B) *Store {
	b.Helper()
	dir := b.TempDir()
	store, err := OpenStore(fmt.Sprintf("%s/bench.db", dir))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { store.Close() })
	return store
}

func buildBenchGraph(nodeCount int) *Graph {
	g := New()
	for i := 0; i < nodeCount; i++ {
		id := fmt.Sprintf("node-%d", i)
		g.AddNode(Node{
			ID:   id,
			Path: fmt.Sprintf("src/file%d.ts", i),
			Type: NodeTypeTS,
			Hash: uint64(i),
			Size: int64(i * 100),
		})
	}
	for i := 0; i < nodeCount-1; i++ {
		g.AddEdge(fmt.Sprintf("node-%d", i), fmt.Sprintf("node-%d", i+1), EdgeStaticImport, false)
	}
	return g
}
