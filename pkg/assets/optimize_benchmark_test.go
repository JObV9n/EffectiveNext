package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkOptimize_10(b *testing.B) {
	dir := createBenchAssets(b, 10)
	benchOptimize(b, dir)
}

func BenchmarkOptimize_100(b *testing.B) {
	dir := createBenchAssets(b, 100)
	benchOptimize(b, dir)
}

func BenchmarkOptimize_500(b *testing.B) {
	dir := createBenchAssets(b, 500)
	benchOptimize(b, dir)
}

func BenchmarkDeduplicateByHash_100(b *testing.B) {
	dir := createBenchDedupAssets(b, 100)
	o := NewOptimizer(dir, 4)
	o.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o.DeduplicateByHash()
	}
}

func BenchmarkOptimize_WorkerScaling(b *testing.B) {
	dir := createBenchAssets(b, 200)
	for _, workers := range []int{1, 2, 4, 8} {
		b.Run("workers="+string(rune('0'+workers)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				o := NewOptimizer(dir, workers)
				o.Scan()
				o.Optimize(OptimizeOptions{MaxThreads: workers})
			}
		})
	}
}

func createBenchAssets(b *testing.B, count int) string {
	b.Helper()
	dir := b.TempDir()
	for i := 0; i < count; i++ {
		subdir := filepath.Join(dir, "assets")
		os.MkdirAll(subdir, 0o755)
		name := filepath.Join(subdir, "file"+string(rune('a'+i%26))+string(rune('0'+i/26))+".png")
		content := make([]byte, 1024*(i%10+1))
		for j := range content {
			content[j] = byte(j % 256)
		}
		os.WriteFile(name, content, 0o644)
	}
	return dir
}

func createBenchDedupAssets(b *testing.B, count int) string {
	b.Helper()
	dir := b.TempDir()
	content := []byte("shared content for dedup benchmark")
	for i := 0; i < count; i++ {
		subdir := filepath.Join(dir, "public")
		os.MkdirAll(subdir, 0o755)
		name := filepath.Join(subdir, "image"+string(rune('a'+i%26))+".png")
		os.WriteFile(name, content, 0o644)
	}
	return dir
}

func benchOptimize(b *testing.B, dir string) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o := NewOptimizer(dir, 4)
		o.Scan()
		o.Optimize(OptimizeOptions{MaxThreads: 4})
	}
}
