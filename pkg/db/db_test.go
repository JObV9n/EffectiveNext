package db

import (
	"os"
	"testing"
	"time"
)

func TestDB_OpenAndMigrate(t *testing.T) {
	path := t.TempDir() + "/test.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if db.Path() != path {
		t.Errorf("Path() = %q, want %q", db.Path(), path)
	}
}

func TestDB_UpsertAndGetFile(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	rec := FileRecord{
		Path:     "src/index.ts",
		Hash:     "abc123",
		Size:     1024,
		ModTime:  time.Now(),
		ScanTime: time.Now(),
		IsDir:    false,
	}

	if err := db.UpsertFile(rec); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetFile("src/index.ts")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != rec.Hash {
		t.Errorf("Hash = %q, want %q", got.Hash, rec.Hash)
	}
	if got.Size != rec.Size {
		t.Errorf("Size = %d, want %d", got.Size, rec.Size)
	}
}

func TestDB_ListFiles(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	files := []FileRecord{
		{Path: "a.ts", Hash: "a", Size: 100},
		{Path: "b.ts", Hash: "b", Size: 200},
	}
	for _, f := range files {
		if err := db.UpsertFile(f); err != nil {
			t.Fatal(err)
		}
	}

	got, err := db.ListFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("ListFiles() returned %d files, want 2", len(got))
	}
}

func TestDB_DeleteFile(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := db.UpsertFile(FileRecord{Path: "del.ts", Hash: "d", Size: 50}); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteFile("del.ts"); err != nil {
		t.Fatal(err)
	}
	_, err := db.GetFile("del.ts")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestDB_FileCount(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	count, err := db.FileCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("FileCount() = %d, want 0", count)
	}

	db.UpsertFile(FileRecord{Path: "a.ts", Hash: "a"})
	count, _ = db.FileCount()
	if count != 1 {
		t.Errorf("FileCount() = %d, want 1", count)
	}
}

func TestDB_Dependencies(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	dep := DepRecord{
		Source:   "a.ts",
		Target:   "b.ts",
		EdgeType: "static-import",
		Dynamic:  false,
	}
	if err := db.UpsertDependency(dep); err != nil {
		t.Fatal(err)
	}

	deps, err := db.GetDependencies("a.ts")
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 1 {
		t.Fatalf("GetDependencies() returned %d, want 1", len(deps))
	}
	if deps[0].Target != "b.ts" {
		t.Errorf("Target = %q, want %q", deps[0].Target, "b.ts")
	}

	dependents, err := db.GetDependents("b.ts")
	if err != nil {
		t.Fatal(err)
	}
	if len(dependents) != 1 {
		t.Errorf("GetDependents() returned %d, want 1", len(dependents))
	}
}

func TestDB_ClearDependencies(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	db.UpsertDependency(DepRecord{Source: "a.ts", Target: "b.ts", EdgeType: "import"})
	db.UpsertDependency(DepRecord{Source: "a.ts", Target: "c.ts", EdgeType: "import"})

	if err := db.ClearDependencies("a.ts"); err != nil {
		t.Fatal(err)
	}
	deps, _ := db.GetDependencies("a.ts")
	if len(deps) != 0 {
		t.Errorf("after ClearDependencies, got %d deps, want 0", len(deps))
	}
}

func TestDB_Routes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	route := RouteRecord{
		Path:      "/about",
		RouteType: "page",
		Layout:    "/layout.tsx",
		Metadata:  `{"title":"About"}`,
	}
	if err := db.UpsertRoute(route); err != nil {
		t.Fatal(err)
	}

	routes, err := db.ListRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 {
		t.Fatalf("ListRoutes() returned %d, want 1", len(routes))
	}
	if routes[0].Path != "/about" {
		t.Errorf("Path = %q, want %q", routes[0].Path, "/about")
	}

	if err := db.ClearRoutes(); err != nil {
		t.Fatal(err)
	}
	routes, _ = db.ListRoutes()
	if len(routes) != 0 {
		t.Errorf("after ClearRoutes, got %d routes, want 0", len(routes))
	}
}

func TestDB_CacheEntries(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	entry := CacheEntry{
		Key:        "cache-key-1",
		Hash:       "abc123",
		Size:       1024,
		Compressed: true,
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
	}
	if err := db.UpsertCacheEntry(entry); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetCacheEntry("cache-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Hash != entry.Hash {
		t.Errorf("Hash = %q, want %q", got.Hash, entry.Hash)
	}
	if !got.Compressed {
		t.Error("expected Compressed to be true")
	}

	size, _ := db.CacheSize()
	if size != 1024 {
		t.Errorf("CacheSize() = %d, want 1024", size)
	}

	count, _ := db.CacheCount()
	if count != 1 {
		t.Errorf("CacheCount() = %d, want 1", count)
	}

	if err := db.PurgeCache(); err != nil {
		t.Fatal(err)
	}
	count, _ = db.CacheCount()
	if count != 0 {
		t.Errorf("after PurgeCache, CacheCount() = %d, want 0", count)
	}
}

func TestDB_Stats(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := db.RecordStat("build-1", "cold_time_ms", 1500); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordStat("build-1", "files_scanned", 500); err != nil {
		t.Fatal(err)
	}

	stats, err := db.GetStats("build-1")
	if err != nil {
		t.Fatal(err)
	}
	if stats["cold_time_ms"] != 1500 {
		t.Errorf("cold_time_ms = %v, want 1500", stats["cold_time_ms"])
	}
}

func TestDB_Benchmarks(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	if err := db.RecordBenchmark("scanner", "throughput", 10000, "files/s"); err != nil {
		t.Fatal(err)
	}

	bench, err := db.GetBenchmarks("scanner")
	if err != nil {
		t.Fatal(err)
	}
	if bench["throughput"] != 10000 {
		t.Errorf("throughput = %v, want 10000", bench["throughput"])
	}
}

func TestDB_BuildHistory(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	buildID, err := db.StartBuild()
	if err != nil {
		t.Fatal(err)
	}
	if buildID == "" {
		t.Fatal("expected non-empty build ID")
	}

	metrics := map[string]float64{
		"cold_time_ms":       1500,
		"files_scanned":      500,
		"cache_hit_ratio":    0.85,
		"files_rebuilt":      50,
		"incremental_time_ms": 200,
	}
	if err := db.FinishBuild(buildID, "success", metrics); err != nil {
		t.Fatal(err)
	}

	history, err := db.GetBuildHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("GetBuildHistory() returned %d, want 1", len(history))
	}
	if history[0].Status != "success" {
		t.Errorf("Status = %q, want %q", history[0].Status, "success")
	}
	if history[0].ColdTimeMs != 1500 {
		t.Errorf("ColdTimeMs = %v, want 1500", history[0].ColdTimeMs)
	}
}

func TestDB_PurgeAll(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	db.UpsertFile(FileRecord{Path: "a.ts", Hash: "a"})
	db.UpsertDependency(DepRecord{Source: "a.ts", Target: "b.ts", EdgeType: "import"})
	db.UpsertRoute(RouteRecord{Path: "/home", RouteType: "page"})
	db.UpsertCacheEntry(CacheEntry{Key: "k", Hash: "h", Size: 100})

	if err := db.PurgeAll(); err != nil {
		t.Fatal(err)
	}

	count, _ := db.FileCount()
	if count != 0 {
		t.Errorf("after PurgeAll, FileCount() = %d, want 0", count)
	}
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := t.TempDir() + "/test.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(path)
	})
	return db
}