package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/JobV9n/effectiveNext/pkg/config"
)

func createTestProject(b *testing.B, numFiles int) string {
	b.Helper()
	dir := b.TempDir()

	for i := 0; i < numFiles; i++ {
		name := filepath.Join(dir, "file"+string(rune('a'+i%26))+".ts")
		content := `import { foo } from './utils';
export default function App() {}
`
		os.WriteFile(name, []byte(content), 0o644)
	}

	subDir := filepath.Join(dir, "components")
	os.MkdirAll(subDir, 0o755)
	for i := 0; i < numFiles/2; i++ {
		name := filepath.Join(subDir, "component"+string(rune('a'+i%26))+".tsx")
		content := `import React from 'react';
export default function Component() {}
`
		os.WriteFile(name, []byte(content), 0o644)
	}

	return dir
}

func BenchmarkScanSmall(b *testing.B) {
	dir := createTestProject(b, 10)
	cfg := config.Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
		s.Scan()
	}
}

func BenchmarkScanMedium(b *testing.B) {
	dir := createTestProject(b, 100)
	cfg := config.Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
		s.Scan()
	}
}

func BenchmarkScanLarge(b *testing.B) {
	dir := createTestProject(b, 1000)
	cfg := config.Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
		s.Scan()
	}
}

func BenchmarkScanWithWorkers(b *testing.B) {
	dir := createTestProject(b, 500)
	cfg := config.Default()

	workers := []int{1, 2, 4, 8}
	for _, w := range workers {
		b.Run("workers="+string(rune('0'+w)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s := New(Options{Root: dir, Config: cfg.Scanner, Workers: w})
				s.Scan()
			}
		})
	}
}

func BenchmarkScanWithIgnore(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()
	cfg.Scanner.Ignore = append(cfg.Scanner.Ignore, "components")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
		s.Scan()
	}
}

func BenchmarkScanByExtension(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
		s.ScanByExtension()
	}
}

func BenchmarkScanModifiedAfter(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
		s.ScanModifiedAfter(time.Time{})
	}
}

func BenchmarkTotalSize(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()
	s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
	files, _ := s.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TotalSize(files)
	}
}

func BenchmarkFilterByExtension(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()
	s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
	files, _ := s.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterByExtension(files, ".ts")
	}
}

func BenchmarkSortByPath(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()
	s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
	files, _ := s.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortByPath(files)
	}
}

func BenchmarkSortBySize(b *testing.B) {
	dir := createTestProject(b, 200)
	cfg := config.Default()
	s := New(Options{Root: dir, Config: cfg.Scanner, Workers: runtime.NumCPU()})
	files, _ := s.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SortBySize(files)
	}
}
