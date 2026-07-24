package graph

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store provides SQLite persistence for the dependency graph.
type Store struct {
	conn *sql.DB
}

// OpenStore opens or creates a SQLite store for graph persistence.
func OpenStore(dbPath string) (*Store, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open graph store: %w", err)
	}
	s := &Store{conn: conn}
	if err := s.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate graph store: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS graph_nodes (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL,
			type TEXT NOT NULL,
			hash INTEGER NOT NULL DEFAULT 0,
			size INTEGER NOT NULL DEFAULT 0,
			saved_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS graph_edges (
			source TEXT NOT NULL,
			target TEXT NOT NULL,
			edge_type TEXT NOT NULL,
			dynamic INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (source, target, edge_type)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_graph_edges_source ON graph_edges(source);`,
		`CREATE INDEX IF NOT EXISTS idx_graph_edges_target ON graph_edges(target);`,
	}
	for _, m := range migrations {
		if _, err := s.conn.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}

// Save persists an in-memory graph to SQLite.
func (s *Store) Save(g *Graph) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	tx, err := s.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM graph_nodes`); err != nil {
		return fmt.Errorf("clear nodes: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM graph_edges`); err != nil {
		return fmt.Errorf("clear edges: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	nodeStmt, err := tx.Prepare(`INSERT INTO graph_nodes (id, path, type, hash, size, saved_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare node insert: %w", err)
	}
	defer nodeStmt.Close()

	for _, n := range g.nodes {
		if _, err := nodeStmt.Exec(n.ID, n.Path, string(n.Type), int64(n.Hash), n.Size, now); err != nil {
			return fmt.Errorf("insert node: %w", err)
		}
	}

	edgeStmt, err := tx.Prepare(`INSERT OR IGNORE INTO graph_edges (source, target, edge_type, dynamic) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare edge insert: %w", err)
	}
	defer edgeStmt.Close()

	for _, edges := range g.edges {
		for _, e := range edges {
			if _, err := edgeStmt.Exec(e.Source, e.Target, string(e.Type), boolToInt(e.Dynamic)); err != nil {
				return fmt.Errorf("insert edge: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Load restores an in-memory graph from SQLite.
func (s *Store) Load() (*Graph, error) {
	g := New()

	rows, err := s.conn.Query(`SELECT id, path, type, hash, size FROM graph_nodes`)
	if err != nil {
		return nil, fmt.Errorf("query nodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Node
		var nodeType string
		var hash int64
		if err := rows.Scan(&n.ID, &n.Path, &nodeType, &hash, &n.Size); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		n.Type = NodeType(nodeType)
		n.Hash = uint64(hash)
		g.nodes[n.ID] = &n
		g.order = append(g.order, n.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	edgeRows, err := s.conn.Query(`SELECT source, target, edge_type, dynamic FROM graph_edges`)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var e Edge
		var edgeType string
		var dynamic int
		if err := edgeRows.Scan(&e.Source, &e.Target, &edgeType, &dynamic); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		e.Type = EdgeType(edgeType)
		e.Dynamic = dynamic != 0
		g.edges[e.Source] = append(g.edges[e.Source], e)
		g.reverse[e.Target] = append(g.reverse[e.Target], e)
	}
	if err := edgeRows.Err(); err != nil {
		return nil, err
	}

	return g, nil
}

// NodeCount returns the number of persisted nodes.
func (s *Store) NodeCount() (int, error) {
	var count int
	err := s.conn.QueryRow(`SELECT COUNT(*) FROM graph_nodes`).Scan(&count)
	return count, err
}

// EdgeCount returns the number of persisted edges.
func (s *Store) EdgeCount() (int, error) {
	var count int
	err := s.conn.QueryRow(`SELECT COUNT(*) FROM graph_edges`).Scan(&count)
	return count, err
}

// Clear removes all persisted graph data.
func (s *Store) Clear() error {
	if _, err := s.conn.Exec(`DELETE FROM graph_nodes`); err != nil {
		return err
	}
	_, err := s.conn.Exec(`DELETE FROM graph_edges`)
	return err
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.conn.Close()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
