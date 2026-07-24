package graph

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportDOT(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "src/a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "src/b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)

	var buf bytes.Buffer
	if err := g.ExportDOT(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "digraph dependencies") {
		t.Error("expected digraph header")
	}
	if !strings.Contains(out, `"src/a.ts"`) {
		t.Error("expected node a")
	}
	if !strings.Contains(out, `"src/b.ts"`) {
		t.Error("expected node b")
	}
	if !strings.Contains(out, `->`) {
		t.Error("expected edge")
	}
}

func TestExportDOT_DynamicEdge(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeDynamicImport, true)

	var buf bytes.Buffer
	if err := g.ExportDOT(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "dashed") {
		t.Error("expected dashed style for dynamic import")
	}
}

func TestExportJSON(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "src/a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "src/b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)

	var buf bytes.Buffer
	if err := g.ExportJSON(&buf); err != nil {
		t.Fatal(err)
	}

	var result struct {
		Nodes []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"nodes"`
		Edges []struct {
			Source  string `json:"source"`
			Target  string `json:"target"`
			Type    string `json:"type"`
			Dynamic bool   `json:"dynamic"`
		} `json:"edges"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(result.Edges))
	}
	if result.Edges[0].Source != "a" {
		t.Errorf("expected source a, got %s", result.Edges[0].Source)
	}
	if result.Edges[0].Target != "b" {
		t.Errorf("expected target b, got %s", result.Edges[0].Target)
	}
}

func TestExportMermaid(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "src/a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "src/b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)

	var buf bytes.Buffer
	if err := g.ExportMermaid(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "graph LR") {
		t.Error("expected graph LR header")
	}
	if !strings.Contains(out, `-->`) {
		t.Error("expected arrow")
	}
}

func TestExportMermaid_DynamicEdge(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddNode(Node{ID: "b", Path: "b.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeDynamicImport, true)

	var buf bytes.Buffer
	if err := g.ExportMermaid(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "-.->") {
		t.Error("expected dashed arrow for dynamic import")
	}
}

func TestExportDOTFile(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})
	g.AddEdge("a", "b", EdgeStaticImport, false)

	dir := t.TempDir()
	path := filepath.Join(dir, "graph.dot")

	if err := g.ExportDOTFile(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "digraph") {
		t.Error("expected digraph in output")
	}
}

func TestExportJSONFile(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})

	dir := t.TempDir()
	path := filepath.Join(dir, "graph.json")

	if err := g.ExportJSONFile(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "nodes") {
		t.Error("expected nodes in output")
	}
}

func TestExportMermaidFile(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Path: "a.ts", Type: NodeTypeTS})

	dir := t.TempDir()
	path := filepath.Join(dir, "graph.mmd")

	if err := g.ExportMermaidFile(path); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "graph") {
		t.Error("expected graph in output")
	}
}

func TestExportEmptyGraph(t *testing.T) {
	g := New()

	var buf bytes.Buffer
	if err := g.ExportDOT(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "digraph") {
		t.Error("expected digraph header even for empty graph")
	}
}

func TestExportJSONEmpty(t *testing.T) {
	g := New()

	var buf bytes.Buffer
	if err := g.ExportJSON(&buf); err != nil {
		t.Fatal(err)
	}

	var result struct {
		Nodes []interface{} `json:"nodes"`
		Edges []interface{} `json:"edges"`
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(result.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(result.Nodes))
	}
}

func TestSanitizeMermaidID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"src/a.ts", "src_a_ts"},
		{"path/to/file", "path_to_file"},
		{"simple", "simple"},
		{"with-dash", "with_dash"},
		{"with.dot", "with_dot"},
	}

	for _, tt := range tests {
		result := sanitizeMermaidID(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeMermaidID(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
