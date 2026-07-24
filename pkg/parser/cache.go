package parser

import (
	"container/heap"
	"sync"
	"time"
)

// ParsedFile holds a cached parse result with metadata.
type ParsedFile struct {
	Result      *Result
	Hash        uint64
	ParsedAt    time.Time
	AccessCount int
	index       int
}

// ParseCache is a thread-safe LRU cache for parsed AST results.
type ParseCache struct {
	mu       sync.RWMutex
	items    map[string]*ParsedFile
	order    *lruHeap
	maxSize  int
	hits     int64
	misses   int64
}

// NewParseCache creates a new parse cache with the given maximum entries.
func NewParseCache(maxSize int) *ParseCache {
	if maxSize <= 0 {
		maxSize = 10000
	}
	pc := &ParseCache{
		items:   make(map[string]*ParsedFile),
		maxSize: maxSize,
	}
	pc.order = &lruHeap{cache: pc}
	heap.Init(pc.order)
	return pc
}

// Get retrieves a cached parse result. Returns nil if not found.
func (pc *ParseCache) Get(path string) *Result {
	pc.mu.RLock()
	item, ok := pc.items[path]
	pc.mu.RUnlock()

	if !ok {
		pc.mu.Lock()
		pc.misses++
		pc.mu.Unlock()
		return nil
	}

	pc.mu.Lock()
	item.AccessCount++
	item.ParsedAt = time.Now()
	pc.hits++
	heap.Fix(pc.order, item.index)
	pc.mu.Unlock()

	return item.Result
}

// Put stores a parse result in the cache.
func (pc *ParseCache) Put(path string, result *Result, hash uint64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if existing, ok := pc.items[path]; ok {
		existing.Result = result
		existing.Hash = hash
		existing.ParsedAt = time.Now()
		existing.AccessCount++
		heap.Fix(pc.order, existing.index)
		return
	}

	item := &ParsedFile{
		Result:   result,
		Hash:     hash,
		ParsedAt: time.Now(),
	}
	heap.Push(pc.order, item)
	pc.items[path] = item

	// Evict LRU entries if over capacity
	for pc.order.Len() > pc.maxSize {
		evicted := heap.Pop(pc.order).(*ParsedFile)
		// Find and remove from map
		for path, existing := range pc.items {
			if existing == evicted {
				delete(pc.items, path)
				break
			}
		}
	}
}

// Has returns true if the path is cached.
func (pc *ParseCache) Has(path string) bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	_, ok := pc.items[path]
	return ok
}

// Remove removes a path from the cache.
func (pc *ParseCache) Remove(path string) bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	item, ok := pc.items[path]
	if !ok {
		return false
	}

	delete(pc.items, path)
	heap.Remove(pc.order, item.index)
	return true
}

// Clear removes all cached entries.
func (pc *ParseCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.items = make(map[string]*ParsedFile)
	pc.order = &lruHeap{cache: pc}
	heap.Init(pc.order)
}

// Size returns the number of cached entries.
func (pc *ParseCache) Size() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.items)
}

// Stats returns cache statistics.
type CacheStats struct {
	Size     int     `json:"size"`
	MaxSize  int     `json:"max_size"`
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	HitRatio float64 `json:"hit_ratio"`
}

// Stats returns current cache statistics.
func (pc *ParseCache) Stats() CacheStats {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	var ratio float64
	total := pc.hits + pc.misses
	if total > 0 {
		ratio = float64(pc.hits) / float64(total)
	}

	return CacheStats{
		Size:     len(pc.items),
		MaxSize:  pc.maxSize,
		Hits:     pc.hits,
		Misses:   pc.misses,
		HitRatio: ratio,
	}
}

// Invalidate removes all entries whose hash no longer matches.
func (pc *ParseCache) Invalidate(currentHashes map[string]uint64) int {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	removed := 0
	for path, item := range pc.items {
		if expectedHash, ok := currentHashes[path]; ok {
			if item.Hash != expectedHash {
				delete(pc.items, path)
				heap.Remove(pc.order, item.index)
				removed++
			}
		} else {
			delete(pc.items, path)
			heap.Remove(pc.order, item.index)
			removed++
		}
	}
	return removed
}

// lruHeap is a min-heap ordered by access time (least recently used first).
type lruHeap struct {
	items []*ParsedFile
	cache *ParseCache
}

func (h *lruHeap) Len() int { return len(h.items) }

func (h *lruHeap) Less(i, j int) bool {
	return h.items[i].AccessCount < h.items[j].AccessCount
}

func (h *lruHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].index = i
	h.items[j].index = j
}

func (h *lruHeap) Push(x interface{}) {
	item := x.(*ParsedFile)
	item.index = len(h.items)
	h.items = append(h.items, item)
}

func (h *lruHeap) Pop() interface{} {
	old := h.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	h.items = old[:n-1]
	return item
}
