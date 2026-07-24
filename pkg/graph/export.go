package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ExportDOT writes the graph in GraphViz DOT format.
func (g *Graph) ExportDOT(w io.Writer) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	fmt.Fprintln(w, "digraph dependencies {")
	fmt.Fprintln(w, "  rankdir=LR;")
	fmt.Fprintln(w, "  node [shape=box];")

	for _, n := range g.Nodes() {
		label := n.Path
		if label == "" {
			label = n.ID
		}
		fmt.Fprintf(w, "  %q [label=%q];\n", n.ID, label)
	}

	for source, edges := range g.edges {
		for _, e := range edges {
			style := "solid"
			if e.Dynamic {
				style = "dashed"
			}
			fmt.Fprintf(w, "  %q -> %q [style=%s];\n", source, e.Target, style)
		}
	}

	fmt.Fprintln(w, "}")
	return nil
}

// ExportJSON writes the graph as JSON.
func (g *Graph) ExportJSON(w io.Writer) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	type jsonNode struct {
		ID   string   `json:"id"`
		Path string   `json:"path"`
		Type NodeType `json:"type"`
	}

	type jsonEdge struct {
		Source  string  `json:"source"`
		Target  string  `json:"target"`
		Type    EdgeType `json:"type"`
		Dynamic bool    `json:"dynamic"`
	}

	type jsonGraph struct {
		Nodes []jsonNode `json:"nodes"`
		Edges []jsonEdge `json:"edges"`
	}

	nodes := g.Nodes()
	jsonNodes := make([]jsonNode, len(nodes))
	for i, n := range nodes {
		jsonNodes[i] = jsonNode{ID: n.ID, Path: n.Path, Type: n.Type}
	}

	var jsonEdges []jsonEdge
	for source, edges := range g.edges {
		for _, e := range edges {
			jsonEdges = append(jsonEdges, jsonEdge{
				Source:  source,
				Target:  e.Target,
				Type:    e.Type,
				Dynamic: e.Dynamic,
			})
		}
	}

	result := jsonGraph{Nodes: jsonNodes, Edges: jsonEdges}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// ExportMermaid writes the graph in Mermaid diagram format.
func (g *Graph) ExportMermaid(w io.Writer) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	fmt.Fprintln(w, "graph LR")

	for _, n := range g.Nodes() {
		label := n.Path
		if label == "" {
			label = n.ID
		}
		fmt.Fprintf(w, "    %s[\"%s\"]\n", sanitizeMermaidID(n.ID), label)
	}

	for source, edges := range g.edges {
		for _, e := range edges {
			arrow := "-->"
			if e.Dynamic {
				arrow = "-.->"
			}
			fmt.Fprintf(w, "    %s %s %s\n", sanitizeMermaidID(source), arrow, sanitizeMermaidID(e.Target))
		}
	}

	return nil
}

func sanitizeMermaidID(id string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		".", "_",
		"-", "_",
		"[", "_",
		"]", "_",
		"(", "_",
		")", "_",
		" ", "_",
	)
	return replacer.Replace(id)
}

// ExportDOTFile writes the graph to a DOT file.
func (g *Graph) ExportDOTFile(path string) error {
	return exportToFile(g.ExportDOT, path)
}

// ExportJSONFile writes the graph to a JSON file.
func (g *Graph) ExportJSONFile(path string) error {
	return exportToFile(g.ExportJSON, path)
}

// ExportMermaidFile writes the graph to a Mermaid file.
func (g *Graph) ExportMermaidFile(path string) error {
	return exportToFile(g.ExportMermaid, path)
}

func exportToFile(fn func(io.Writer) error, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}
