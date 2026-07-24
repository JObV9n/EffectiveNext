package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/JobV9n/effectiveNext/pkg/graph"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
)

// Change represents a file change detected by the watcher.
type Change struct {
	Path     string
	Type     ChangeType
	OldHash  uint64
	NewHash  uint64
}

// ChangeType describes the type of change.
type ChangeType int

const (
	ChangeModified ChangeType = iota
	ChangeCreated
	ChangeDeleted
	ChangeRenamed
)

// Engine is the incremental build engine.
type Engine struct {
	database   *db.DB
	graph      *graph.Graph
	baseHashes map[string]uint64
	mu         sync.RWMutex
}

// New creates a new incremental build engine.
func New(database *db.DB) *Engine {
	return &Engine{
		database:   database,
		graph:      graph.New(),
		baseHashes: make(map[string]uint64),
	}
}

// Snapshot captures the current state of all files for future diffing.
func (e *Engine) Snapshot(files []scanner.File) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, f := range files {
		e.baseHashes[f.Path] = f.Hash
	}
}

// ComputeChanges compares current files against the snapshot and returns what changed.
func (e *Engine) ComputeChanges(current []scanner.File) []Change {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var changes []Change
	currentMap := make(map[string]scanner.File)

	for _, f := range current {
		currentMap[f.Path] = f
		oldHash, exists := e.baseHashes[f.Path]
		if !exists {
			changes = append(changes, Change{
				Path:    f.Path,
				Type:    ChangeCreated,
				NewHash: f.Hash,
			})
		} else if oldHash != f.Hash {
			changes = append(changes, Change{
				Path:    f.Path,
				Type:    ChangeModified,
				OldHash: oldHash,
				NewHash: f.Hash,
			})
		}
	}

	for path, oldHash := range e.baseHashes {
		if _, exists := currentMap[path]; !exists {
			changes = append(changes, Change{
				Path:    path,
				Type:    ChangeDeleted,
				OldHash: oldHash,
			})
		}
	}

	return changes
}

// AffectedFiles returns all files affected by the given changes.
func (e *Engine) AffectedFiles(changes []Change) []string {
	affected := make(map[string]bool)

	for _, c := range changes {
		affected[c.Path] = true
		dependents := e.graph.AffectedNodes(c.Path)
		for _, d := range dependents {
			affected[d] = true
		}
	}

	result := make([]string, 0, len(affected))
	for path := range affected {
		result = append(result, path)
	}
	return result
}

// BuildStats holds statistics about an incremental build.
type BuildStats struct {
	TotalFiles   int
	ChangedFiles int
	RebuiltFiles int
	Duration     time.Duration
	CacheHits    int
	CacheMisses  int
}

// Summary returns a human-readable summary.
func (s BuildStats) String() string {
	return fmt.Sprintf("files=%d changed=%d rebuilt=%d duration=%v cache_hits=%d cache_misses=%d",
		s.TotalFiles, s.ChangedFiles, s.RebuiltFiles, s.Duration.Round(time.Millisecond), s.CacheHits, s.CacheMisses)
}

// Clear resets the engine state.
func (e *Engine) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.baseHashes = make(map[string]uint64)
	e.graph.Clear()
}