package hash

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestHash64(t *testing.T) {
	// Known test vector for xxHash64
	data := []byte("test")
	expected := uint64(0x4fdcca5ddb678139)
	result := Hash64(data)
	if result != expected {
		t.Errorf("Hash64(%q) = 0x%x, want 0x%x", data, result, expected)
	}
}

func TestHash64String(t *testing.T) {
	expected := uint64(0x4fdcca5ddb678139)
	result := Hash64String("test")
	if result != expected {
		t.Errorf("Hash64String(%q) = 0x%x, want 0x%x", "test", result, expected)
	}
}

func TestHash64File(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.txt"
	content := []byte("file content")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Hash64File(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := Hash64(content)
	if result != expected {
		t.Errorf("Hash64File = 0x%x, want 0x%x", result, expected)
	}
}

func TestHash64Reader(t *testing.T) {
	data := []byte("reader test")
	r := bytes.NewReader(data)
	result, err := Hash64Reader(r)
	if err != nil {
		t.Fatal(err)
	}
	expected := Hash64(data)
	if result != expected {
		t.Errorf("Hash64Reader = 0x%x, want 0x%x", result, expected)
	}
}

func TestHasher64(t *testing.T) {
	h := Hasher64()
	h.Write([]byte("test"))
	result := h.Sum64()
	expected := Hash64([]byte("test"))
	if result != expected {
		t.Errorf("Hasher64 = 0x%x, want 0x%x", result, expected)
	}
}

func TestHasher64WithSeed(t *testing.T) {
	h1 := Hasher64WithSeed(12345)
	h1.Write([]byte("test"))
	h2 := Hasher64WithSeed(12345)
	h2.Write([]byte("test"))
	if h1.Sum64() != h2.Sum64() {
		t.Error("Hasher64WithSeed produces different results for same seed")
	}
}

func TestHash64Bytes(t *testing.T) {
	data := []byte("test")
	result := Hash64Bytes(data)
	expected := "4fdcca5ddb678139"
	if result != expected {
		t.Errorf("Hash64Bytes = %q, want %q", result, expected)
	}
}

func TestHashCombiner(t *testing.T) {
	hc := NewHashCombiner()
	hc.Write([]byte("part1"))
	hc.Write([]byte("part2"))
	hc.WriteString("part3")
	hc.WriteUint64(42)

	// Verify same operations produce same result
	hc2 := NewHashCombiner()
	hc2.Write([]byte("part1"))
	hc2.Write([]byte("part2"))
	hc2.WriteString("part3")
	hc2.WriteUint64(42)

	if hc.Sum() != hc2.Sum() {
		t.Error("HashCombiner produces different results for same inputs")
	}
}

func TestHashCombinerReset(t *testing.T) {
	hc := NewHashCombiner()
	hc.Write([]byte("test"))
	sum1 := hc.Sum()

	hc.Reset()
	hc.Write([]byte("test"))
	sum2 := hc.Sum()

	if sum1 != sum2 {
		t.Error("HashCombiner Reset failed")
	}
}

func TestHashCombiner_ImplementsWriter(t *testing.T) {
	hc := NewHashCombiner()
	var w io.Writer = hc
	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("Write returned %d, want 4", n)
	}
}