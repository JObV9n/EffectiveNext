package cache

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/JobV9n/effectiveNext/pkg/db"
	"github.com/JobV9n/effectiveNext/pkg/hash"
	"github.com/klauspost/compress/zstd"
)

// Cache is the binary cache backed by disk with optional Zstd compression.
type Cache struct {
	dir         string
	db          *db.DB
	mu          sync.RWMutex
	hits        int64
	misses      int64
	compress    bool
	compressor  *zstd.Encoder
	decompressor *zstd.Decoder
}

// New creates a new cache.
func New(dir string, database *db.DB, compress bool) (*Cache, error) {
	os.MkdirAll(dir, 0o755)

	c := &Cache{
		dir:      dir,
		db:       database,
		compress: compress,
	}

	if compress {
		var err error
		c.compressor, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			return nil, err
		}
		c.decompressor, err = zstd.NewReader(nil)
		if err != nil {
			c.compressor.Close()
			return nil, err
		}
	}

	return c, nil
}

// Get retrieves a cached entry by key.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, err := c.db.GetCacheEntry(key)
	if err != nil {
		c.misses++
		return nil, false
	}

	path := filepath.Join(c.dir, entry.Key+".bin")
	data, err := os.ReadFile(path)
	if err != nil {
		c.misses++
		return nil, false
	}

	if entry.Compressed && c.decompressor != nil {
		decompressed, err := c.decompressor.DecodeAll(data, nil)
		if err != nil {
			c.misses++
			return nil, false
		}
		data = decompressed
	}

	c.hits++
	return data, true
}

// Put stores data in the cache.
func (c *Cache) Put(key string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := data
	isCompressed := false

	if c.compress && c.compressor != nil && len(data) > 1024 {
		compressed := c.compressor.EncodeAll(data, make([]byte, 0, len(data)))
		if len(compressed) < len(data) {
			stored = compressed
			isCompressed = true
		}
	}

	path := filepath.Join(c.dir, key+".bin")
	if err := os.WriteFile(path, stored, 0o644); err != nil {
		return err
	}

	return c.db.UpsertCacheEntry(db.CacheEntry{
		Key:        key,
		Hash:       hash.Hash64Bytes(data),
		Size:       int64(len(stored)),
		Compressed: isCompressed,
	})
}

// Has checks if a key exists in the cache.
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, err := c.db.GetCacheEntry(key)
	return err == nil
}

// Remove removes an entry from the cache.
func (c *Cache) Remove(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := filepath.Join(c.dir, key+".bin")
	os.Remove(path)
	return c.db.DeleteCacheEntry(key)
}

// Purge removes all cache entries.
func (c *Cache) Purge() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		os.Remove(filepath.Join(c.dir, e.Name()))
	}
	return c.db.PurgeCache()
}

// Close releases resources.
func (c *Cache) Close() {
	if c.compressor != nil {
		c.compressor.Close()
	}
	if c.decompressor != nil {
		c.decompressor.Close()
	}
}

// Stats returns cache statistics.
func (c *Cache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalSize, _ := c.db.CacheSize()
	count, _ := c.db.CacheCount()

	hitRatio := float64(0)
	if c.hits+c.misses > 0 {
		hitRatio = float64(c.hits) / float64(c.hits+c.misses)
	}

	return Stats{
		Size:      totalSize,
		Entries:   count,
		Hits:      c.hits,
		Misses:    c.misses,
		HitRatio:  hitRatio,
	}
}

// Stats holds cache statistics.
type Stats struct {
	Size     int64
	Entries  int
	Hits     int64
	Misses   int64
	HitRatio float64
}

// Size returns the total cache size in bytes.
func (c *Cache) Size() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	size, _ := c.db.CacheSize()
	return size
}

// Count returns the number of cache entries.
func (c *Cache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count, _ := c.db.CacheCount()
	return count
}