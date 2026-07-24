package hash

import (
	"hash"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// Hash64 returns the xxHash64 of the given data.
func Hash64(data []byte) uint64 {
	return xxhash.Sum64(data)
}

// Hash64String returns the xxHash64 of the given string.
func Hash64String(s string) uint64 {
	return xxhash.Sum64String(s)
}

// Hash64File returns the xxHash64 of the file at path.
func Hash64File(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return Hash64Reader(f)
}

// Hash64Reader returns the xxHash64 of the reader.
func Hash64Reader(r io.Reader) (uint64, error) {
	h := xxhash.New()
	if _, err := io.Copy(h, r); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

// Hasher64 returns a new xxHash64 hasher.
func Hasher64() hash.Hash64 {
	return xxhash.New()
}

// Hasher64WithSeed returns a new xxHash64 hasher with a custom seed.
func Hasher64WithSeed(seed uint64) hash.Hash64 {
	return xxhash.NewWithSeed(seed)
}

// Hash64Bytes is a convenience function to hash bytes and return hex string.
func Hash64Bytes(data []byte) string {
	return formatUint64(Hash64(data))
}

func formatUint64(u uint64) string {
	const hex = "0123456789abcdef"
	var buf [16]byte
	for i := 15; i >= 0; i-- {
		buf[i] = hex[u&0xf]
		u >>= 4
	}
	return string(buf[:])
}

// HashCombiner combines multiple hashes into a single hash.
type HashCombiner struct {
	h hash.Hash64
}

// NewHashCombiner creates a new hash combiner.
func NewHashCombiner() *HashCombiner {
	return &HashCombiner{h: xxhash.New()}
}

// Write adds data to the hash.
func (hc *HashCombiner) Write(p []byte) (int, error) {
	return hc.h.Write(p)
}

// WriteString adds a string to the hash.
func (hc *HashCombiner) WriteString(s string) (int, error) {
	return io.WriteString(hc.h, s)
}

// WriteUint64 adds a uint64 to the hash.
func (hc *HashCombiner) WriteUint64(u uint64) {
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(u >> (8 * i))
	}
	hc.h.Write(buf[:])
}

// Sum returns the combined hash.
func (hc *HashCombiner) Sum() uint64 {
	return hc.h.Sum64()
}

// Reset resets the hasher.
func (hc *HashCombiner) Reset() {
	hc.h.Reset()
}