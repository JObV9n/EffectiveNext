package hash

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkHash64_Small(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64(data)
	}
}

func BenchmarkHash64_Medium(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64(data)
	}
}

func BenchmarkHash64_Large(b *testing.B) {
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64(data)
	}
}

func BenchmarkHash64String_Small(b *testing.B) {
	s := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64String(s)
	}
}

func BenchmarkHash64String_Medium(b *testing.B) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	s := string(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64String(s)
	}
}

func BenchmarkHash64Reader_Small(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		Hash64Reader(r)
	}
}

func BenchmarkHash64Reader_Large(b *testing.B) {
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		Hash64Reader(r)
	}
}

func BenchmarkHash64File(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "test.bin")
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(path, data, 0o644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64File(path)
	}
}

func BenchmarkHash64Bytes_Small(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64Bytes(data)
	}
}

func BenchmarkHash64Bytes_Large(b *testing.B) {
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash64Bytes(data)
	}
}

func BenchmarkHashCombiner(b *testing.B) {
	data := []byte("hello world")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc := NewHashCombiner()
		hc.Write(data)
		hc.Sum()
	}
}

func BenchmarkHashCombiner_String(b *testing.B) {
	s := "hello world"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc := NewHashCombiner()
		hc.WriteString(s)
		hc.Sum()
	}
}

func BenchmarkHashCombiner_Uint64(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hc := NewHashCombiner()
		hc.WriteUint64(uint64(i))
		hc.Sum()
	}
}

func BenchmarkFormatUint64(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatUint64(uint64(i))
	}
}
