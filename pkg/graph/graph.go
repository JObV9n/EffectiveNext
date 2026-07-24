package graph

import (
	"fmt"
	"sort"
	"sync"
)

// NodeType represents the type of a graph node.
type NodeType string

const (
	NodeTypeJS     NodeType = "js"
	NodeTypeTS     NodeType = "ts"
	NodeTypeJSX    NodeType = "jsx"
	NodeTypeTSX    NodeType = "tsx"
	NodeTypeCSS    NodeType = "css"
	NodeTypeSCSS   NodeType = "scss"
	NodeTypeJSON   NodeType = "json"
	NodeTypeImage  NodeType = "image"
	NodeTypeRoute  NodeType = "route"
	NodeTypeLayout NodeType = "layout"
	NodeTypeOther  NodeType = "other"
)

// EdgeType represents the type of a graph edge.
type EdgeType string

const (
	EdgeStaticImport  EdgeType = "static-import"
	EdgeDynamicImport EdgeType = "dynamic-import"
	EdgeCSSImport     EdgeType = "css-import"
	EdgeAssetRef      EdgeType = "asset-reference"
	EdgeRouteParent   EdgeType = "route-parent"
	EdgeMetadata      EdgeType = "metadata"
)

// Node represents a node in the dependency graph.
type Node struct {
	ID       string
	Path     string
	Type     NodeType
	Hash     uint64
	Size     int64
	Metadata map[string]string
}

// Edge represents a dependency edge.
type Edge struct {
	Source  string
	Target  string
	Type    EdgeType
	Dynamic bool
}

// Graph is a concurrent dependency graph.
type Graph struct {
	mu      sync.RWMutex
	nodes   map[string]*Node
	edges   map[string][]Edge // source -> edges
	reverse map[string][]Edge // target -> edges (reverse index)
	order   []string          // insertion order
}

// New creates a new empty dependency graph.
func New() *Graph {
	return &Graph{
		nodes:   make(map[string]*Node),
		edges:   make(map[string][]Edge),
		reverse: make(map[string][]Edge),
	}
}

// AddNode adds a node to the graph.
func (g *Graph) AddNode(node Node) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[node.ID] = &node
	g.order = append(g.order, node.ID)
}

// GetNode returns a node by ID.
func (g *Graph) GetNode(id string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[id]
	return n, ok
}

// AddEdge adds an edge to the graph.
func (g *Graph) AddEdge(source, target string, edgeType EdgeType, dynamic bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	edge := Edge{
		Source:  source,
		Target:  target,
		Type:    edgeType,
		Dynamic: dynamic,
	}
	g.edges[source] = append(g.edges[source], edge)
	g.reverse[target] = append(g.reverse[target], edge)
}

// Dependencies returns all dependencies of a node (what it imports).
func (g *Graph) Dependencies(id string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edges[id]
}

// Dependents returns all nodes that depend on a node (what imports it).
func (g *Graph) Dependents(id string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.reverse[id]
}

// NodeCount returns the number of nodes.
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// EdgeCount returns the number of edges.
func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	count := 0
	for _, edges := range g.edges {
		count += len(edges)
	}
	return count
}

// Nodes returns all nodes sorted by ID.
func (g *Graph) Nodes() []Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, *n)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// AllEdges returns all edges.
func (g *Graph) AllEdges() []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var result []Edge
	for _, edges := range g.edges {
		result = append(result, edges...)
	}
	return result
}

// TopologicalSort returns nodes in topological order.
// Returns error if there is a cycle.
func (g *Graph) TopologicalSort() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}
	for _, edges := range g.edges {
		for _, e := range edges {
			inDegree[e.Target]++
		}
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, edge := range g.edges[node] {
			inDegree[edge.Target]--
			if inDegree[edge.Target] == 0 {
				queue = append(queue, edge.Target)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return result, nil
}

// DetectCycles finds all cycles in the graph.
func (g *Graph) DetectCycles() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var cycles [][]string
	visited := make(map[string]int) // 0=unvisited, 1=in progress, 2=done
	path := make(map[string]int)
	var stack []string

	var dfs func(id string)
	dfs = func(id string) {
		visited[id] = 1
		path[id] = len(stack)
		stack = append(stack, id)

		for _, edge := range g.edges[id] {
			if visited[edge.Target] == 0 {
				dfs(edge.Target)
			} else if visited[edge.Target] == 1 {
				cycle := make([]string, 0)
				start := path[edge.Target]
				cycle = append(cycle, stack[start:]...)
				cycle = append(cycle, edge.Target)
				cycles = append(cycles, cycle)
			}
		}

		stack = stack[:len(stack)-1]
		visited[id] = 2
		delete(path, id)
	}

	for id := range g.nodes {
		if visited[id] == 0 {
			dfs(id)
		}
	}

	return cycles
}

// AffectedNodes returns all nodes affected by a change to the given node.
func (g *Graph) AffectedNodes(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	affected := make(map[string]bool)
	var queue []string
	queue = append(queue, id)
	affected[id] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, edge := range g.reverse[current] {
			if !affected[edge.Source] {
				affected[edge.Source] = true
				queue = append(queue, edge.Source)
			}
		}
	}

	result := make([]string, 0, len(affected))
	for id := range affected {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// Diff computes the difference between two graphs, returning added/removed nodes and edges.
func (g *Graph) Diff(other *Graph) DiffResult {
	g.mu.RLock()
	defer g.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	result := DiffResult{}

	for id, node := range g.nodes {
		if _, ok := other.nodes[id]; !ok {
			result.RemovedNodes = append(result.RemovedNodes, *node)
		}
	}
	for id, node := range other.nodes {
		if _, ok := g.nodes[id]; !ok {
			result.AddedNodes = append(result.AddedNodes, *node)
		}
	}

	for source, edges := range g.edges {
		for _, edge := range edges {
			found := false
			for _, otherEdge := range other.edges[source] {
				if otherEdge.Target == edge.Target && otherEdge.Type == edge.Type {
					found = true
					break
				}
			}
			if !found {
				result.RemovedEdges = append(result.RemovedEdges, edge)
			}
		}
	}
	for source, edges := range other.edges {
		for _, edge := range edges {
			found := false
			for _, gEdge := range g.edges[source] {
				if gEdge.Target == edge.Target && gEdge.Type == edge.Type {
					found = true
					break
				}
			}
			if !found {
				result.AddedEdges = append(result.AddedEdges, edge)
			}
		}
	}

	return result
}

// DiffResult holds the difference between two graphs.
type DiffResult struct {
	AddedNodes   []Node
	RemovedNodes []Node
	AddedEdges   []Edge
	RemovedEdges []Edge
}

// Stats returns graph statistics.
func (g *Graph) Stats() Stats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	typeCount := make(map[NodeType]int)
	for _, n := range g.nodes {
		typeCount[n.Type]++
	}

	edgeTypeCount := make(map[EdgeType]int)
	for _, edges := range g.edges {
		for _, e := range edges {
			edgeTypeCount[e.Type]++
		}
	}

	return Stats{
		NodeCount:    len(g.nodes),
		EdgeCount:    g.EdgeCount(),
		TypeCount:    typeCount,
		EdgeTypeCount: edgeTypeCount,
	}
}

// Stats holds graph statistics.
type Stats struct {
	NodeCount     int
	EdgeCount     int
	TypeCount     map[NodeType]int
	EdgeTypeCount map[EdgeType]int
}

// Clear removes all nodes and edges.
func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes = make(map[string]*Node)
	g.edges = make(map[string][]Edge)
	g.reverse = make(map[string][]Edge)
	g.order = nil
}