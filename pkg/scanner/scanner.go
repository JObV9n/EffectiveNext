package scanner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JobV9n/effectiveNext/pkg/config"
	"github.com/JobV9n/effectiveNext/pkg/hash"
)

// File represents a scanned file with metadata.
type File struct {
	Path      string    `json:"path"`
	RelPath   string    `json:"rel_path"`
	Size      int64     `json:"size"`
	Hash      uint64    `json:"hash"`
	ModTime   time.Time `json:"mod_time"`
	IsDir     bool      `json:"is_dir"`
	IsSymlink bool      `json:"is_symlink"`
	Extension string    `json:"extension"`
}

// Scanner is a concurrent filesystem scanner.
type Scanner struct {
	root     string
	config   config.ScannerConfig
	workers  int
	ignore   *ignoreMatcher
	filesScanned atomic.Int64
}

// Options configures the scanner.
type Options struct {
	Root    string
	Config  config.ScannerConfig
	Workers int
}

// New creates a new Scanner.
func New(opts Options) *Scanner {
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	return &Scanner{
		root:    opts.Root,
		config:  opts.Config,
		workers: workers,
		ignore:  newIgnoreMatcher(opts.Config.Ignore, opts.Root),
	}
}

func (s *Scanner) walkDir(dir string, mu *sync.Mutex, files *[]File) []string {
	var subDirs []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(dir, name)

		if s.ignore.Matches(name, e.IsDir()) || s.ignore.MatchesPath(fullPath) {
			continue
		}

		if e.IsDir() {
			subDirs = append(subDirs, fullPath)
		} else {
			h, err := hash.Hash64File(fullPath)
			if err != nil {
				h = 0
			}
			info, _ := e.Info()
			relPath, _ := filepath.Rel(s.root, fullPath)
			f := File{
				Path:      fullPath,
				RelPath:   relPath,
				Size:      info.Size(),
				Hash:      h,
				ModTime:   info.ModTime(),
				IsDir:     false,
				IsSymlink: info.Mode()&os.ModeSymlink != 0,
				Extension: filepath.Ext(name),
			}
			mu.Lock()
			*files = append(*files, f)
			mu.Unlock()
			s.filesScanned.Add(1)
		}
	}
	return subDirs
}

// Scan runs the scanner and returns all files found.
func (s *Scanner) Scan() ([]File, error) {
	start := time.Now()

	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("read root directory: %w", err)
	}

	var mu sync.Mutex
	files := make([]File, 0, 64)

	for _, e := range entries {
		name := e.Name()
		if s.ignore.Matches(name, false) {
			continue
		}
		if e.IsDir() {
			continue
		}
		h, err := hash.Hash64File(filepath.Join(s.root, name))
		if err != nil {
			h = 0
		}
		info, _ := e.Info()
		files = append(files, File{
			Path:      filepath.Join(s.root, name),
			RelPath:   name,
			Size:      info.Size(),
			Hash:      h,
			ModTime:   info.ModTime(),
			IsDir:     false,
			IsSymlink: info.Mode()&os.ModeSymlink != 0,
			Extension: filepath.Ext(name),
		})
		s.filesScanned.Add(1)
	}

	var rootDirs []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || s.ignore.Matches(name, true) {
			continue
		}
		rootDirs = append(rootDirs, filepath.Join(s.root, name))
	}

	if len(rootDirs) > 0 {
		dirQueue := make(chan string, 64)
		for _, d := range rootDirs {
			dirQueue <- d
		}

		var dirWg sync.WaitGroup
		for i := 0; i < s.workers; i++ {
			dirWg.Add(1)
			go func() {
				defer dirWg.Done()
				for {
					select {
					case dir, ok := <-dirQueue:
						if !ok {
							return
						}
						subDirs := s.walkDir(dir, &mu, &files)
						for _, sd := range subDirs {
							select {
							case dirQueue <- sd:
							default:
							}
						}
					default:
						return
					}
				}
			}()
		}
		dirWg.Wait()
	}

	elapsed := time.Since(start)
	fmt.Fprintf(io.Discard, "scanned %d files in %v (%.0f files/s)\n",
		len(files), elapsed, float64(len(files))/elapsed.Seconds())

	return files, nil
}

// FilesScanned returns the number of files scanned.
func (s *Scanner) FilesScanned() int64 {
	return s.filesScanned.Load()
}

// ScanWithFilter runs the scanner with a custom filter function.
func (s *Scanner) ScanWithFilter(filter func(File) bool) ([]File, error) {
	all, err := s.Scan()
	if err != nil {
		return nil, err
	}
	var filtered []File
	for _, f := range all {
		if filter(f) {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

// ScanByExtension groups files by extension.
func (s *Scanner) ScanByExtension() (map[string][]File, error) {
	all, err := s.Scan()
	if err != nil {
		return nil, err
	}
	result := make(map[string][]File)
	for _, f := range all {
		ext := f.Extension
		if ext == "" {
			ext = "(none)"
		}
		result[ext] = append(result[ext], f)
	}
	return result, nil
}

// ScanModifiedAfter only returns files modified after the given time.
func (s *Scanner) ScanModifiedAfter(after time.Time) ([]File, error) {
	return s.ScanWithFilter(func(f File) bool {
		return f.ModTime.After(after)
	})
}

// Ignore patterns
var defaultIgnores = []string{
	"node_modules",
	".next",
	".git",
	"dist",
	"coverage",
	"out",
	".effective-next",
	".effective-next-cache",
}

type ignoreMatcher struct {
	patterns []string
	root     string
}

func newIgnoreMatcher(patterns []string, root string) *ignoreMatcher {
	all := make([]string, 0, len(defaultIgnores)+len(patterns))
	all = append(all, defaultIgnores...)
	for _, p := range patterns {
		if !contains(all, p) {
			all = append(all, p)
		}
	}
	return &ignoreMatcher{patterns: all, root: root}
}

func (m *ignoreMatcher) Matches(name string, isDir bool) bool {
	for _, p := range m.patterns {
		if p == name {
			return true
		}
		if isDir && strings.TrimSuffix(p, "/") == name {
			return true
		}
	}
	return false
}

func (m *ignoreMatcher) MatchesPath(path string) bool {
	rel, err := filepath.Rel(m.root, path)
	if err != nil {
		return false
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		if m.Matches(part, true) {
			return true
		}
	}
	return false
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// SortByPath sorts files by path.
func SortByPath(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
}

// SortBySize sorts files by size descending.
func SortBySize(files []File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Size > files[j].Size
	})
}

// TotalSize returns the total size of all files.
func TotalSize(files []File) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

// FilterByExtension returns files matching the given extension.
func FilterByExtension(files []File, ext string) []File {
	var result []File
	for _, f := range files {
		if f.Extension == ext {
			result = append(result, f)
		}
	}
	return result
}
