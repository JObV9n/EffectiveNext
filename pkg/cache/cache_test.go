package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/pkg/db"
)

func newTestCache(t *testing.T, compress bool) (*Cache, func()) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(dir, "cache")
	c, err := New(cacheDir, database, compress)
	if err != nil {
		database.Close()
		t.Fatal(err)
	}

	cleanup := func() {
		c.Close()
		database.Close()
	}

	return c, cleanup
}

func TestNew(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	if c == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestNew_Compressed(t *testing.T) {
	c, cleanup := newTestCache(t, true)
	defer cleanup()

	if c == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestPutAndGet(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "test-key"
	data := []byte("hello world")

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if string(got) != string(data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestPutAndGet_Compressed(t *testing.T) {
	c, cleanup := newTestCache(t, true)
	defer cleanup()

	key := "compressed-key"
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if len(got) != len(data) {
		t.Errorf("expected length %d, got %d", len(data), len(got))
	}
}

func TestGet_Miss(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestHas(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "exists"
	data := []byte("data")

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	if !c.Has(key) {
		t.Error("expected Has to return true")
	}

	if c.Has("nonexistent") {
		t.Error("expected Has to return false")
	}
}

func TestRemove(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "remove-me"
	data := []byte("data")

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	if err := c.Remove(key); err != nil {
		t.Fatal(err)
	}

	_, ok := c.Get(key)
	if ok {
		t.Error("expected cache miss after remove")
	}
}

func TestPurge(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	for i := 0; i < 5; i++ {
		key := "key" + string(rune('0'+i))
		if err := c.Put(key, []byte("data")); err != nil {
			t.Fatal(err)
		}
	}

	if c.Count() != 5 {
		t.Errorf("expected 5 entries, got %d", c.Count())
	}

	if err := c.Purge(); err != nil {
		t.Fatal(err)
	}

	if c.Count() != 0 {
		t.Errorf("expected 0 entries after purge, got %d", c.Count())
	}
}

func TestStats(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	c.Put("key1", []byte("data1"))
	c.Get("key1")
	c.Get("nonexistent")

	stats := c.Stats()
	if stats.Entries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.Entries)
	}
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
}

func TestSize(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	c.Put("key1", []byte("hello"))
	c.Put("key2", []byte("world"))

	size := c.Size()
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

func TestCount(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	if c.Count() != 0 {
		t.Errorf("expected 0 initially, got %d", c.Count())
	}

	c.Put("key1", []byte("data"))
	c.Put("key2", []byte("data"))

	if c.Count() != 2 {
		t.Errorf("expected 2 entries, got %d", c.Count())
	}
}

func TestOverwrite(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "overwrite"
	c.Put(key, []byte("original"))
	c.Put(key, []byte("updated"))

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if string(got) != "updated" {
		t.Errorf("expected 'updated', got %q", got)
	}
}

func TestLargeValue(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "large"
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if len(got) != len(data) {
		t.Errorf("expected length %d, got %d", len(data), len(got))
	}
}

func TestClose(t *testing.T) {
	c, _ := newTestCache(t, true)
	c.Close()
}

func TestEmptyCache(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	stats := c.Stats()
	if stats.Size != 0 {
		t.Errorf("expected 0 size, got %d", stats.Size)
	}
	if stats.Entries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.Entries)
	}
}

func TestCacheDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	cacheDir := filepath.Join(dir, "cache")
	c, err := New(cacheDir, database, false)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("cache directory was not created")
	}
}

func TestBinaryFile(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "binary"
	data := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if len(got) != len(data) {
		t.Errorf("expected length %d, got %d", len(data), len(got))
	}
}

func TestEmptyValue(t *testing.T) {
	c, cleanup := newTestCache(t, false)
	defer cleanup()

	key := "empty"
	data := []byte{}

	if err := c.Put(key, data); err != nil {
		t.Fatal(err)
	}

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	if len(got) != 0 {
		t.Errorf("expected empty, got %d bytes", len(got))
	}
}
