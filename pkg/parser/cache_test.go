package parser

import (
	"fmt"
	"testing"
)

func TestParseCache_PutAndGet(t *testing.T) {
	cache := NewParseCache(100)
	result := &Result{Path: "test.ts", Imports: []Import{{Path: "foo"}}}

	cache.Put("test.ts", result, 12345)

	got := cache.Get("test.ts")
	if got == nil {
		t.Fatal("expected cached result")
	}
	if len(got.Imports) != 1 {
		t.Errorf("Imports = %d, want 1", len(got.Imports))
	}
}

func TestParseCache_Miss(t *testing.T) {
	cache := NewParseCache(100)

	got := cache.Get("nonexistent.ts")
	if got != nil {
		t.Error("expected nil for cache miss")
	}
}

func TestParseCache_Has(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 1)

	if !cache.Has("a.ts") {
		t.Error("expected Has to return true")
	}
	if cache.Has("b.ts") {
		t.Error("expected Has to return false")
	}
}

func TestParseCache_Remove(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 1)

	if !cache.Remove("a.ts") {
		t.Error("expected Remove to return true")
	}
	if cache.Has("a.ts") {
		t.Error("expected entry to be removed")
	}
}

func TestParseCache_RemoveNonexistent(t *testing.T) {
	cache := NewParseCache(100)
	if cache.Remove("nope.ts") {
		t.Error("expected Remove to return false for nonexistent entry")
	}
}

func TestParseCache_Clear(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 1)
	cache.Put("b.ts", &Result{Path: "b.ts"}, 2)
	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Size = %d, want 0", cache.Size())
	}
}

func TestParseCache_Size(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 1)
	cache.Put("b.ts", &Result{Path: "b.ts"}, 2)

	if cache.Size() != 2 {
		t.Errorf("Size = %d, want 2", cache.Size())
	}
}

func TestParseCache_LRU_Eviction(t *testing.T) {
	cache := NewParseCache(3)

	for i := 0; i < 5; i++ {
		path := fmt.Sprintf("file%d.ts", i)
		cache.Put(path, &Result{Path: path}, uint64(i))
	}

	if cache.Size() > 3 {
		t.Errorf("Size = %d, want <= 3", cache.Size())
	}
}

func TestParseCache_Stats(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 1)

	cache.Get("a.ts") // hit
	cache.Get("b.ts") // miss

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
	if stats.Size != 1 {
		t.Errorf("Size = %d, want 1", stats.Size)
	}
}

func TestParseCache_HitRatio(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 1)

	cache.Get("a.ts") // hit
	cache.Get("a.ts") // hit
	cache.Get("b.ts") // miss

	stats := cache.Stats()
	if stats.HitRatio < 0.6 || stats.HitRatio > 0.7 {
		t.Errorf("HitRatio = %f, want ~0.67", stats.HitRatio)
	}
}

func TestParseCache_Invalidate(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts"}, 100)
	cache.Put("b.ts", &Result{Path: "b.ts"}, 200)

	// a.ts hash changed, b.ts same, c.ts is new
	currentHashes := map[string]uint64{
		"a.ts": 999,  // changed
		"b.ts": 200,  // same
	}

	removed := cache.Invalidate(currentHashes)
	if removed != 1 {
		t.Errorf("Invalidate removed %d, want 1", removed)
	}

	if cache.Has("a.ts") {
		t.Error("expected a.ts to be invalidated")
	}
	if !cache.Has("b.ts") {
		t.Error("expected b.ts to remain")
	}
}

func TestParseCache_UpdateExisting(t *testing.T) {
	cache := NewParseCache(100)
	cache.Put("a.ts", &Result{Path: "a.ts", Imports: []Import{{Path: "old"}}}, 1)
	cache.Put("a.ts", &Result{Path: "a.ts", Imports: []Import{{Path: "new"}}}, 2)

	got := cache.Get("a.ts")
	if got == nil || len(got.Imports) == 0 {
		t.Fatal("expected updated result")
	}
	if got.Imports[0].Path != "new" {
		t.Errorf("Imports[0].Path = %q, want %q", got.Imports[0].Path, "new")
	}
}

func TestParseCache_Concurrent(t *testing.T) {
	cache := NewParseCache(100)
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				path := fmt.Sprintf("file%d.ts", j)
				cache.Put(path, &Result{Path: path}, uint64(j))
				cache.Get(path)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if cache.Size() > 100 {
		t.Errorf("Size = %d, want <= 100", cache.Size())
	}
}

func TestParseCache_DefaultSize(t *testing.T) {
	cache := NewParseCache(0) // should default to 10000
	stats := cache.Stats()
	if stats.MaxSize != 10000 {
		t.Errorf("MaxSize = %d, want 10000", stats.MaxSize)
	}
}
