package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps a SQLite database with schema migrations and typed accessors.
type DB struct {
	mu   sync.RWMutex
	conn *sql.DB
	path string
}

// Open opens or creates a SQLite database at path.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db := &DB{conn: conn, path: path}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}

// Close closes the database.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.conn.Close()
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

func (db *DB) migrate() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS files (
			path TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			mod_time TEXT NOT NULL,
			scan_time TEXT NOT NULL,
			is_dir INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS hashes (
			path TEXT NOT NULL,
			algo TEXT NOT NULL,
			value TEXT NOT NULL,
			PRIMARY KEY (path, algo)
		);`,
		`CREATE TABLE IF NOT EXISTS dependencies (
			source TEXT NOT NULL,
			target TEXT NOT NULL,
			edge_type TEXT NOT NULL,
			dynamic INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (source, target, edge_type)
		);`,
		`CREATE TABLE IF NOT EXISTS routes (
			path TEXT PRIMARY KEY,
			route_type TEXT NOT NULL,
			layout TEXT,
			metadata TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS assets (
			path TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			optimized_path TEXT,
			optimized_hash TEXT,
			optimized_size INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS cache_entries (
			key TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			compressed INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			last_access TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			build_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value REAL NOT NULL,
			recorded_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS benchmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			metric TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			recorded_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			build_id TEXT NOT NULL,
			started_at TEXT NOT NULL,
			finished_at TEXT,
			status TEXT NOT NULL,
			cold_time_ms REAL,
			incremental_time_ms REAL,
			files_scanned INTEGER,
			files_rebuilt INTEGER,
			cache_hit_ratio REAL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_files_hash ON files(hash);`,
		`CREATE INDEX IF NOT EXISTS idx_deps_source ON dependencies(source);`,
		`CREATE INDEX IF NOT EXISTS idx_deps_target ON dependencies(target);`,
		`CREATE INDEX IF NOT EXISTS idx_cache_hash ON cache_entries(hash);`,
		`CREATE INDEX IF NOT EXISTS idx_stats_build ON stats(build_id);`,
		`CREATE INDEX IF NOT EXISTS idx_history_build ON history(build_id);`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, m)
		}
	}

	return nil
}

// FileRecord represents a scanned file entry.
type FileRecord struct {
	Path     string
	Hash     string
	Size     int64
	ModTime  time.Time
	ScanTime time.Time
	IsDir    bool
}

// UpsertFile inserts or updates a file record.
func (db *DB) UpsertFile(rec FileRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO files (path, hash, size, mod_time, scan_time, is_dir) VALUES (?, ?, ?, ?, ?, ?)`,
		rec.Path, rec.Hash, rec.Size, rec.ModTime.Format(time.RFC3339), rec.ScanTime.Format(time.RFC3339), boolToInt(rec.IsDir),
	)
	return err
}

// GetFile returns a file record by path.
func (db *DB) GetFile(path string) (FileRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var rec FileRecord
	var modTime, scanTime string
	var isDir int
	err := db.conn.QueryRow(`SELECT path, hash, size, mod_time, scan_time, is_dir FROM files WHERE path = ?`, path).Scan(
		&rec.Path, &rec.Hash, &rec.Size, &modTime, &scanTime, &isDir,
	)
	if err != nil {
		return rec, err
	}
	rec.ModTime, _ = time.Parse(time.RFC3339, modTime)
	rec.ScanTime, _ = time.Parse(time.RFC3339, scanTime)
	rec.IsDir = isDir != 0
	return rec, nil
}

// ListFiles returns all file records.
func (db *DB) ListFiles() ([]FileRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT path, hash, size, mod_time, scan_time, is_dir FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []FileRecord
	for rows.Next() {
		var rec FileRecord
		var modTime, scanTime string
		var isDir int
		if err := rows.Scan(&rec.Path, &rec.Hash, &rec.Size, &modTime, &scanTime, &isDir); err != nil {
			return nil, err
		}
		rec.ModTime, _ = time.Parse(time.RFC3339, modTime)
		rec.ScanTime, _ = time.Parse(time.RFC3339, scanTime)
		rec.IsDir = isDir != 0
		result = append(result, rec)
	}
	return result, rows.Err()
}

// FileCount returns the number of files in the database.
func (db *DB) FileCount() (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&count)
	return count, err
}

// DeleteFile removes a file record.
func (db *DB) DeleteFile(path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM files WHERE path = ?`, path)
	return err
}

// DepRecord represents a dependency edge.
type DepRecord struct {
	Source   string
	Target   string
	EdgeType string
	Dynamic  bool
}

// UpsertDependency inserts or updates a dependency edge.
func (db *DB) UpsertDependency(rec DepRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO dependencies (source, target, edge_type, dynamic) VALUES (?, ?, ?, ?)`,
		rec.Source, rec.Target, rec.EdgeType, boolToInt(rec.Dynamic),
	)
	return err
}

// GetDependencies returns all dependencies for a source file.
func (db *DB) GetDependencies(source string) ([]DepRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT source, target, edge_type, dynamic FROM dependencies WHERE source = ?`, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []DepRecord
	for rows.Next() {
		var rec DepRecord
		var dynamic int
		if err := rows.Scan(&rec.Source, &rec.Target, &rec.EdgeType, &dynamic); err != nil {
			return nil, err
		}
		rec.Dynamic = dynamic != 0
		result = append(result, rec)
	}
	return result, rows.Err()
}

// GetDependents returns all files that depend on target.
func (db *DB) GetDependents(target string) ([]DepRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT source, target, edge_type, dynamic FROM dependencies WHERE target = ?`, target)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []DepRecord
	for rows.Next() {
		var rec DepRecord
		var dynamic int
		if err := rows.Scan(&rec.Source, &rec.Target, &rec.EdgeType, &dynamic); err != nil {
			return nil, err
		}
		rec.Dynamic = dynamic != 0
		result = append(result, rec)
	}
	return result, rows.Err()
}

// ClearDependencies removes all dependencies for a source file.
func (db *DB) ClearDependencies(source string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM dependencies WHERE source = ?`, source)
	return err
}

// RouteRecord represents a Next.js route.
type RouteRecord struct {
	Path      string
	RouteType string
	Layout    string
	Metadata  string
}

// UpsertRoute inserts or updates a route record.
func (db *DB) UpsertRoute(rec RouteRecord) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO routes (path, route_type, layout, metadata) VALUES (?, ?, ?, ?)`,
		rec.Path, rec.RouteType, rec.Layout, rec.Metadata,
	)
	return err
}

// ListRoutes returns all routes.
func (db *DB) ListRoutes() ([]RouteRecord, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT path, route_type, layout, metadata FROM routes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RouteRecord
	for rows.Next() {
		var rec RouteRecord
		if err := rows.Scan(&rec.Path, &rec.RouteType, &rec.Layout, &rec.Metadata); err != nil {
			return nil, err
		}
		result = append(result, rec)
	}
	return result, rows.Err()
}

// ClearRoutes removes all routes.
func (db *DB) ClearRoutes() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM routes`)
	return err
}

// CacheEntry represents a binary cache entry.
type CacheEntry struct {
	Key          string
	Hash         string
	Size         int64
	Compressed   bool
	CreatedAt    time.Time
	LastAccess   time.Time
}

// UpsertCacheEntry inserts or updates a cache entry.
func (db *DB) UpsertCacheEntry(entry CacheEntry) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO cache_entries (key, hash, size, compressed, created_at, last_access) VALUES (?, ?, ?, ?, ?, ?)`,
		entry.Key, entry.Hash, entry.Size, boolToInt(entry.Compressed), entry.CreatedAt.Format(time.RFC3339), entry.LastAccess.Format(time.RFC3339),
	)
	return err
}

// GetCacheEntry returns a cache entry by key.
func (db *DB) GetCacheEntry(key string) (CacheEntry, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var entry CacheEntry
	var createdAt, lastAccess string
	var compressed int
	err := db.conn.QueryRow(`SELECT key, hash, size, compressed, created_at, last_access FROM cache_entries WHERE key = ?`, key).Scan(
		&entry.Key, &entry.Hash, &entry.Size, &compressed, &createdAt, &lastAccess,
	)
	if err != nil {
		return entry, err
	}
	entry.Compressed = compressed != 0
	entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	entry.LastAccess, _ = time.Parse(time.RFC3339, lastAccess)
	return entry, nil
}

// DeleteCacheEntry removes a cache entry.
func (db *DB) DeleteCacheEntry(key string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM cache_entries WHERE key = ?`, key)
	return err
}

// PurgeCache removes all cache entries.
func (db *DB) PurgeCache() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM cache_entries`)
	return err
}

// CacheSize returns the total size of all cache entries.
func (db *DB) CacheSize() (int64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var total sql.NullInt64
	err := db.conn.QueryRow(`SELECT SUM(size) FROM cache_entries`).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}

// CacheCount returns the number of cache entries.
func (db *DB) CacheCount() (int, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM cache_entries`).Scan(&count)
	return count, err
}

// RecordStat records a build statistic.
func (db *DB) RecordStat(buildID, key string, value float64) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT INTO stats (build_id, key, value, recorded_at) VALUES (?, ?, ?, ?)`,
		buildID, key, value, time.Now().Format(time.RFC3339),
	)
	return err
}

// GetStats returns all stats for a build.
func (db *DB) GetStats(buildID string) (map[string]float64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT key, value FROM stats WHERE build_id = ?`, buildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]float64)
	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, rows.Err()
}

// RecordBenchmark records a benchmark result.
func (db *DB) RecordBenchmark(name, metric string, value float64, unit string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT INTO benchmarks (name, metric, value, unit, recorded_at) VALUES (?, ?, ?, ?, ?)`,
		name, metric, value, unit, time.Now().Format(time.RFC3339),
	)
	return err
}

// GetBenchmarks returns all benchmarks for a given name.
func (db *DB) GetBenchmarks(name string) (map[string]float64, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT metric, value FROM benchmarks WHERE name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]float64)
	for rows.Next() {
		var metric string
		var value float64
		if err := rows.Scan(&metric, &value); err != nil {
			return nil, err
		}
		result[metric] = value
	}
	return result, rows.Err()
}

// StartBuild begins a new build and returns its ID.
func (db *DB) StartBuild() (string, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	buildID := fmt.Sprintf("build-%d", time.Now().UnixNano())
	_, err := db.conn.Exec(
		`INSERT INTO history (build_id, started_at, status) VALUES (?, ?, ?)`,
		buildID, time.Now().Format(time.RFC3339), "running",
	)
	return buildID, err
}

// FinishBuild marks a build as complete with optional metrics.
func (db *DB) FinishBuild(buildID string, status string, metrics map[string]float64) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`UPDATE history SET finished_at = ?, status = ?, cold_time_ms = ?, incremental_time_ms = ?, files_scanned = ?, files_rebuilt = ?, cache_hit_ratio = ? WHERE build_id = ?`,
		time.Now().Format(time.RFC3339), status,
		metrics["cold_time_ms"], metrics["incremental_time_ms"],
		int64(metrics["files_scanned"]), int64(metrics["files_rebuilt"]),
		metrics["cache_hit_ratio"], buildID,
	)
	return err
}

// GetBuildHistory returns the last N builds.
func (db *DB) GetBuildHistory(limit int) ([]BuildHistory, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(
		`SELECT build_id, started_at, finished_at, status, cold_time_ms, incremental_time_ms, files_scanned, files_rebuilt, cache_hit_ratio FROM history ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []BuildHistory
	for rows.Next() {
		var h BuildHistory
		var startedAt, finishedAt, status string
		var coldTimeMs, incrementalTimeMs, cacheHitRatio sql.NullFloat64
		var filesScanned, filesRebuilt sql.NullInt64
		if err := rows.Scan(&h.BuildID, &startedAt, &finishedAt, &status, &coldTimeMs, &incrementalTimeMs, &filesScanned, &filesRebuilt, &cacheHitRatio); err != nil {
			return nil, err
		}
		h.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		if finishedAt != "" {
			h.FinishedAt, _ = time.Parse(time.RFC3339, finishedAt)
		}
		h.Status = status
		if coldTimeMs.Valid {
			h.ColdTimeMs = coldTimeMs.Float64
		}
		if incrementalTimeMs.Valid {
			h.IncrementalTimeMs = incrementalTimeMs.Float64
		}
		if filesScanned.Valid {
			h.FilesScanned = int(filesScanned.Int64)
		}
		if filesRebuilt.Valid {
			h.FilesRebuilt = int(filesRebuilt.Int64)
		}
		if cacheHitRatio.Valid {
			h.CacheHitRatio = cacheHitRatio.Float64
		}
		result = append(result, h)
	}
	return result, rows.Err()
}

// BuildHistory represents a completed build record.
type BuildHistory struct {
	BuildID           string
	StartedAt         time.Time
	FinishedAt        time.Time
	Status            string
	ColdTimeMs        float64
	IncrementalTimeMs float64
	FilesScanned      int
	FilesRebuilt      int
	CacheHitRatio     float64
}

// PurgeAll removes all data from the database.
func (db *DB) PurgeAll() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	tables := []string{"files", "hashes", "dependencies", "routes", "assets", "cache_entries", "stats", "benchmarks", "history"}
	for _, t := range tables {
		if _, err := db.conn.Exec(`DELETE FROM ` + t); err != nil {
			return err
		}
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}