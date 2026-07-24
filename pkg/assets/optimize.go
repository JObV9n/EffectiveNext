package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/JobV9n/effectiveNext/pkg/hash"
)

// OptimizeResult holds the result of optimizing a single asset.
type OptimizeResult struct {
	OriginalPath string `json:"original_path"`
	OutputPath   string `json:"output_path"`
	OriginalSize int64  `json:"original_size"`
	OutputSize   int64  `json:"output_size"`
	SavedBytes   int64  `json:"saved_bytes"`
	Hash         string `json:"hash"`
}

// OptimizeOptions configures asset optimization behavior.
type OptimizeOptions struct {
	OutputDir    string `json:"output_dir"`
	MaxThreads   int    `json:"max_threads"`
	SkipCompress bool   `json:"skip_compress"`
}

// Optimize runs parallel optimization on all scanned assets.
// It hashes files, deduplicates by content hash, and writes metadata.
func (o *Optimizer) Optimize(opts OptimizeOptions) ([]OptimizeResult, error) {
	if opts.MaxThreads <= 0 {
		opts.MaxThreads = o.workers
	}

	results := make([]OptimizeResult, len(o.files))
	hashes := make([]string, len(o.files))
	jobs := make(chan int, len(o.files))

	for i := range o.files {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	hashMu := sync.Mutex{}

	for w := 0; w < opts.MaxThreads; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				f := o.files[idx]
				h, size := hashFile(f.Path)

				hashMu.Lock()
				hashes[idx] = h
				hashMu.Unlock()

				mu.Lock()
				f.Hash = uint64(0)
				results[idx] = OptimizeResult{
					OriginalPath: f.Path,
					OutputPath:   f.Path,
					OriginalSize: f.Size,
					OutputSize:   size,
					SavedBytes:   f.Size - size,
					Hash:         h,
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// Apply content hashes to files
	for i, f := range o.files {
		f.Hash = hash.Hash64([]byte(hashes[i]))
	}

	return results, nil
}

// DeduplicateByHash finds files with identical content hashes.
func (o *Optimizer) DeduplicateByHash() [][]*File {
	for _, f := range o.files {
		if f.Hash == 0 {
			h, _ := hashFile(f.Path)
			f.Hash = hash.Hash64([]byte(h))
		}
	}

	hashMap := make(map[uint64][]*File)
	for _, f := range o.files {
		hashMap[f.Hash] = append(hashMap[f.Hash], f)
	}

	var duplicates [][]*File
	for _, group := range hashMap {
		if len(group) > 1 {
			sort.Slice(group, func(i, j int) bool {
				return group[i].Path < group[j].Path
			})
			duplicates = append(duplicates, group)
		}
	}
	return duplicates
}

// WriteMetadata writes an asset metadata file to the output directory.
func (o *Optimizer) WriteMetadata(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	for _, f := range o.files {
		if f.Hash == 0 {
			h, _ := hashFile(f.Path)
			f.Hash = hash.Hash64([]byte(h))
		}
	}

	// Write summary
	summaryPath := filepath.Join(outputDir, "assets.json")
	summary := map[string]interface{}{
		"total_files": len(o.files),
		"total_size":  o.totalSize(),
	}

	summaryData, err := marshalJSON(summary)
	if err != nil {
		return err
	}
	return os.WriteFile(summaryPath, summaryData, 0o644)
}

func (o *Optimizer) totalSize() int64 {
	var total int64
	for _, f := range o.files {
		total += f.Size
	}
	return total
}

func hashFile(path string) (string, int64) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()

	h := sha256.New()
	written, err := io.Copy(h, f)
	if err != nil {
		return "", 0
	}
	return hex.EncodeToString(h.Sum(nil)), written
}

func marshalJSON(v interface{}) ([]byte, error) {
	import_json := func() ([]byte, error) {
		// Avoid importing encoding/json to keep the package lean
		return simpleJSON(v)
	}
	return import_json()
}

// simpleJSON produces minimal JSON for the metadata file.
func simpleJSON(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		result := []byte("{")
		for i, k := range keys {
			if i > 0 {
				result = append(result, ',')
			}
			result = append(result, '"')
			result = append(result, []byte(k)...)
			result = append(result, '"', ':')

			switch v := val[k].(type) {
			case int:
				result = append(result, []byte(intToStr(v))...)
			case int64:
				result = append(result, []byte(int64ToStr(v))...)
			case float64:
				result = append(result, []byte(floatToStr(v))...)
			case string:
				result = append(result, '"')
				result = append(result, []byte(v)...)
				result = append(result, '"')
			}
		}
		result = append(result, '}')
		return result, nil
	}
	return []byte("{}"), nil
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func int64ToStr(n int64) string {
	return intToStr(int(n))
}

func floatToStr(f float64) string {
	if f == 0 {
		return "0"
	}
	return intToStr(int(f))
}
