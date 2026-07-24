package engine

import (
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/JobV9n/effectiveNext/pkg/scanner"
)

func TestEngine_SnapshotAndComputeChanges_NoChanges(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	files := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
	}

	eng.Snapshot(files)

	changes := eng.ComputeChanges(files)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestEngine_ComputeChanges_ModifiedFiles(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	original := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
	}

	eng.Snapshot(original)

	modified := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 999},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
	}

	changes := eng.ComputeChanges(modified)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Type != ChangeModified {
		t.Errorf("expected ChangeModified, got %d", changes[0].Type)
	}
	if changes[0].Path != "/a.ts" {
		t.Errorf("expected /a.ts, got %s", changes[0].Path)
	}
	if changes[0].OldHash != 100 {
		t.Errorf("expected old hash 100, got %d", changes[0].OldHash)
	}
	if changes[0].NewHash != 999 {
		t.Errorf("expected new hash 999, got %d", changes[0].NewHash)
	}
}

func TestEngine_ComputeChanges_CreatedFiles(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	original := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}

	eng.Snapshot(original)

	current := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 300},
	}

	changes := eng.ComputeChanges(current)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Type != ChangeCreated {
		t.Errorf("expected ChangeCreated, got %d", changes[0].Type)
	}
	if changes[0].Path != "/b.ts" {
		t.Errorf("expected /b.ts, got %s", changes[0].Path)
	}
}

func TestEngine_ComputeChanges_DeletedFiles(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	original := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
	}

	eng.Snapshot(original)

	current := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}

	changes := eng.ComputeChanges(current)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Type != ChangeDeleted {
		t.Errorf("expected ChangeDeleted, got %d", changes[0].Type)
	}
	if changes[0].Path != "/b.ts" {
		t.Errorf("expected /b.ts, got %s", changes[0].Path)
	}
}

func TestEngine_ComputeChanges_MixedChanges(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	original := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
		{Path: "/c.ts", RelPath: "c.ts", Hash: 300},
	}

	eng.Snapshot(original)

	current := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 999},
		{Path: "/d.ts", RelPath: "d.ts", Hash: 400},
	}

	changes := eng.ComputeChanges(current)
	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	types := make(map[ChangeType]int)
	for _, c := range changes {
		types[c.Type]++
	}

	if types[ChangeModified] != 1 {
		t.Errorf("expected 1 modified, got %d", types[ChangeModified])
	}
	if types[ChangeCreated] != 1 {
		t.Errorf("expected 1 created, got %d", types[ChangeCreated])
	}
	if types[ChangeDeleted] != 1 {
		t.Errorf("expected 1 deleted, got %d", types[ChangeDeleted])
	}
}

func TestEngine_AffectedFiles(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	changes := []Change{
		{Path: "/a.ts", Type: ChangeModified},
	}

	affected := eng.AffectedFiles(changes)
	if len(affected) != 1 {
		t.Errorf("expected 1 affected file, got %d", len(affected))
	}
}

func TestEngine_Clear(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	files := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}

	eng.Snapshot(files)
	eng.Clear()

	changes := eng.ComputeChanges(files)
	if len(changes) != 1 {
		t.Errorf("expected 1 change after clear (created), got %d", len(changes))
	}
}

func TestEngine_BuildStats(t *testing.T) {
	stats := BuildStats{
		TotalFiles:   100,
		ChangedFiles: 5,
		RebuiltFiles: 3,
		Duration:     1500000000,
		CacheHits:    2,
		CacheMisses:  1,
	}

	s := stats.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestEngine_ComputeChanges_EmptySnapshot(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	current := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}

	changes := eng.ComputeChanges(current)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Type != ChangeCreated {
		t.Errorf("expected ChangeCreated, got %d", changes[0].Type)
	}
}

func TestEngine_ComputeChanges_EmptyCurrent(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	original := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}

	eng.Snapshot(original)

	changes := eng.ComputeChanges(nil)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Type != ChangeDeleted {
		t.Errorf("expected ChangeDeleted, got %d", changes[0].Type)
	}
}

func TestEngine_SnapshotOverwrite(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	first := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}
	eng.Snapshot(first)

	second := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 200},
	}
	eng.Snapshot(second)

	changes := eng.ComputeChanges(second)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes after re-snapshot, got %d", len(changes))
	}
}

func TestEngine_ConcurrentSnapshot(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	files := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			eng.Snapshot(files)
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	changes := eng.ComputeChanges(files)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestEngine_ConcurrentComputeChanges(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	files := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}
	eng.Snapshot(files)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			eng.ComputeChanges(files)
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestEngine_MultipleSnapshots(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	files1 := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
	}
	eng.Snapshot(files1)

	files2 := []scanner.File{
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
	}
	eng.Snapshot(files2)

	changes := eng.ComputeChanges(files2)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		ct   ChangeType
		want int
	}{
		{ChangeModified, 0},
		{ChangeCreated, 1},
		{ChangeDeleted, 2},
		{ChangeRenamed, 3},
	}

	for _, tt := range tests {
		if int(tt.ct) != tt.want {
			t.Errorf("ChangeType %d != %d", tt.ct, tt.want)
		}
	}
}

func TestEngine_SnapshotPreservesOrder(t *testing.T) {
	effectiveNextDir := t.TempDir()
	database, err := db.Open(filepath.Join(effectiveNextDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	eng := New(database)

	files := []scanner.File{
		{Path: "/c.ts", RelPath: "c.ts", Hash: 300},
		{Path: "/a.ts", RelPath: "a.ts", Hash: 100},
		{Path: "/b.ts", RelPath: "b.ts", Hash: 200},
	}

	eng.Snapshot(files)

	changes := eng.ComputeChanges(files)
	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}
