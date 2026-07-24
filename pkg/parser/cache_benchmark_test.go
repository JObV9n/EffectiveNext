package parser

import (
	"fmt"
	"testing"
)

func BenchmarkParseCache_Put(b *testing.B) {
	cache := NewParseCache(10000)
	result := &Result{Path: "test.ts", Imports: []Import{{Path: "foo"}}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("file%d.ts", i%1000)
		cache.Put(path, result, uint64(i))
	}
}

func BenchmarkParseCache_Get_Hit(b *testing.B) {
	cache := NewParseCache(10000)
	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("file%d.ts", i)
		cache.Put(path, &Result{Path: path}, uint64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(fmt.Sprintf("file%d.ts", i%1000))
	}
}

func BenchmarkParseCache_Get_Miss(b *testing.B) {
	cache := NewParseCache(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(fmt.Sprintf("missing%d.ts", i))
	}
}

func BenchmarkParseCache_Invalidate(b *testing.B) {
	cache := NewParseCache(10000)
	for i := 0; i < 5000; i++ {
		path := fmt.Sprintf("file%d.ts", i)
		cache.Put(path, &Result{Path: path}, uint64(i))
	}

	currentHashes := make(map[string]uint64)
	for i := 0; i < 5000; i++ {
		path := fmt.Sprintf("file%d.ts", i)
		if i%2 == 0 {
			currentHashes[path] = uint64(i + 10000) // changed
		} else {
			currentHashes[path] = uint64(i) // unchanged
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Invalidate(currentHashes)
	}
}

func BenchmarkParseCache_Large(b *testing.B) {
	cache := NewParseCache(50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Clear()
		for j := 0; j < 10000; j++ {
			path := fmt.Sprintf("src/components/file%d.tsx", j)
			cache.Put(path, &Result{
				Path:    path,
				Imports: []Import{{Path: "react"}, {Path: "./utils"}},
			}, uint64(j))
		}
	}
}
