package graph

import (
	"testing"
)

func TestGraph_AddAndGetNode(t *testing.T) {
	g := New()

	g.AddNode(Node{ID: "a", Path: "src/a.ts", Type: NodeTypeTS, Hash: 123})
	g.AddNode(Node{ID: "b", Path: "src/b.ts", Type: NodeTypeTS, Hash: 456})

	if g.NodeCount() != 2 {
		t.Errorf("NodeCount() = %d, want 2", g.NodeCount())
	}

	node, ok := g.GetNode("a")
	if !ok {
		t.Fatal("expected to find node 'a'")
	}
	if node.Path != "src/a.ts" {
		t.Errorf("Path = %q, want %q", node.Path, "src/a.ts")
	}
}

func TestGraph_AddEdge(t *testing.T) {
	g := New()

	g.AddNode(Node{ID: "a", Path: "src/a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "src/b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)

	deps := g.Dependencies("a")
	if len(deps) != 1 {
		t.Fatalf("Dependencies(a) returned %d, want 1", len(deps))
	}
	if deps[0].Target != "b" {
		t.Errorf("Target = %q, want %q", deps[0].Target, "b")
	}

	dependents := g.Dependents("b")
	if len(dependents) != 1 {
		t.Fatalf("Dependents(b) returned %d, want 1", len(dependents))
	}
	if dependents[0].Source != "a" {
		t.Errorf("Source = %q, want %q", dependents[0].Source, "a")
	}
}

func TestGraph_TopologicalSort(t *testing.T) {
	g := New()

	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "c", Path: "c.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)
	g.AddEdge("b", "c", EdgeStaticImport, false)

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatal(err)
	}

	if len(order) != 3 {
		t.Fatalf("TopologicalSort returned %d elements, want 3", len(order))
	}

	aIdx, bIdx, cIdx := -1, -1, -1
	for i, id := range order {
		switch id {
		case "a":
			aIdx = i
		case "b":
			bIdx = i
		case "c":
			cIdx = i
		}
	}

	if aIdx >= bIdx || bIdx >= cIdx {
		t.Errorf("topological order should have a before b before c, got %v", order)
	}
}

func TestGraph_CycleDetection(t *testing.T) {
	g := New()

	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)
	g.AddEdge("b", "a", EdgeStaticImport, false)

	_, err := g.TopologicalSort()
	if err == nil {
		t.Error("expected error for cyclic graph")
	}

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected to detect cycles")
	}
}

func TestGraph_AffectedNodes(t *testing.T) {
	g := New()

	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "c", Path: "c.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "d", Path: "d.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)
	g.AddEdge("b", "c", EdgeStaticImport, false)
	g.AddEdge("d", "c", EdgeStaticImport, false)

	affected := g.AffectedNodes("c")
	if len(affected) != 4 {
		t.Fatalf("AffectedNodes(c) returned %d, want 4", len(affected))
	}
}

func TestGraph_Diff(t *testing.T) {
	g1 := New()
	g1.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g1.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g1.AddEdge("a", "b", EdgeStaticImport, false)

	g2 := New()
	g2.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g2.AddNode(Node{ID: "c", Path: "c.ts", Type: NodeTypeTS})

	diff := g1.Diff(g2)

	if len(diff.RemovedNodes) != 1 || diff.RemovedNodes[0].ID != "b" {
		t.Errorf("expected to remove node 'b', got %v", diff.RemovedNodes)
	}
	if len(diff.AddedNodes) != 1 || diff.AddedNodes[0].ID != "c" {
		t.Errorf("expected to add node 'c', got %v", diff.AddedNodes)
	}
}

func TestGraph_Stats(t *testing.T) {
	g := New()

	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.css", Type: NodeTypeCSS})
	g.AddEdge("a", "b", EdgeCSSImport, false)

	stats := g.Stats()
	if stats.NodeCount != 2 {
		t.Errorf("NodeCount = %d, want 2", stats.NodeCount)
	}
	if stats.EdgeCount != 1 {
		t.Errorf("EdgeCount = %d, want 1", stats.EdgeCount)
	}
	if stats.TypeCount[NodeTypeTS] != 1 {
		t.Errorf("TypeCount[TS] = %d, want 1", stats.TypeCount[NodeTypeTS])
	}
}

func TestGraph_Clear(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)

	g.Clear()

	if g.NodeCount() != 0 {
		t.Errorf("after Clear, NodeCount() = %d, want 0", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("after Clear, EdgeCount() = %d, want 0", g.EdgeCount())
	}
}

func TestGraph_NodesSorted(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "c", Path: "c.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})

	nodes := g.Nodes()
	if len(nodes) != 3 {
		t.Fatalf("Nodes() returned %d, want 3", len(nodes))
	}
	if nodes[0].ID != "a" || nodes[1].ID != "b" || nodes[2].ID != "c" {
		t.Errorf("Nodes() not sorted: %v", nodes)
	}
}

func TestGraph_AllEdges(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "c", Path: "c.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)
	g.AddEdge("a", "c", EdgeDynamicImport, true)

	edges := g.AllEdges()
	if len(edges) != 2 {
		t.Errorf("AllEdges() returned %d, want 2", len(edges))
	}
}

func TestGraph_DynamicImport(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeDynamicImport, true)

	edges := g.Dependencies("a")
	if len(edges) != 1 {
		t.Fatalf("Dependencies(a) returned %d, want 1", len(edges))
	}
	if !edges[0].Dynamic {
		t.Error("expected edge to be dynamic")
	}
}