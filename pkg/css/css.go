package css

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	importRe   = regexp.MustCompile(`@import\s+(?:url\()?["']([^"']+)["']\)?`)
	tailwindRe = regexp.MustCompile(`@tailwind\s+(base|components|utilities)`)
	moduleRe   = regexp.MustCompile(`\.module\.(css|scss|sass)$`)
)

// FileType represents the type of CSS file.
type FileType string

const (
	FileTypeCSS   FileType = "css"
	FileTypeSCSS  FileType = "scss"
	FileTypeSASS  FileType = "sass"
	FileTypeLess  FileType = "less"
)

// File represents a CSS-related file with its dependencies.
type File struct {
	Path       string     `json:"path"`
	Type       FileType   `json:"type"`
	Imports    []string   `json:"imports"`
	Tailwind   bool       `json:"has_tailwind"`
	IsModule   bool       `json:"is_module"`
	Hash       uint64     `json:"hash"`
	Size       int64      `json:"size"`
}

// Tracker tracks CSS dependencies and invalidation chains.
type Tracker struct {
	files map[string]*File
}

// NewTracker creates a new CSS dependency tracker.
func NewTracker() *Tracker {
	return &Tracker{
		files: make(map[string]*File),
	}
}

// TrackFile analyzes a CSS file and records its dependencies.
func (t *Tracker) TrackFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	ext := filepath.Ext(path)

	ft := &File{
		Path: path,
		Type: extToFileType(ext),
	}

	ft.Imports = extractImports(content, filepath.Dir(path))
	ft.Tailwind = tailwindRe.MatchString(content)
	ft.IsModule = moduleRe.MatchString(path)
	ft.Size = int64(len(data))

	t.files[path] = ft
	return ft, nil
}

// GetDependencies returns all files that the given CSS file depends on.
func (t *Tracker) GetDependencies(path string) []string {
	ft, ok := t.files[path]
	if !ok {
		return nil
	}
	return ft.Imports
}

// GetDependents returns all files that depend on the given CSS file.
func (t *Tracker) GetDependents(path string) []string {
	var result []string
	for _, ft := range t.files {
		for _, imp := range ft.Imports {
			if imp == path {
				result = append(result, ft.Path)
				break
			}
		}
	}
	return result
}

// AffectedFiles returns all files affected by a change to the given file.
func (t *Tracker) AffectedFiles(path string) []string {
	affected := make(map[string]bool)
	affected[path] = true

	queue := []string{path}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range t.GetDependents(current) {
			if !affected[dep] {
				affected[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	result := make([]string, 0, len(affected))
	for p := range affected {
		result = append(result, p)
	}
	return result
}

// HasTailwind returns whether the project uses Tailwind CSS.
func (t *Tracker) HasTailwind() bool {
	for _, ft := range t.files {
		if ft.Tailwind {
			return true
		}
	}
	return false
}

// HasModules returns whether the project uses CSS Modules.
func (t *Tracker) HasModules() bool {
	for _, ft := range t.files {
		if ft.IsModule {
			return true
		}
	}
	return false
}

// ListFiles returns all tracked CSS files.
func (t *Tracker) ListFiles() []*File {
	result := make([]*File, 0, len(t.files))
	for _, ft := range t.files {
		result = append(result, ft)
	}
	return result
}

// Stats returns statistics about the tracked CSS files.
type Stats struct {
	TotalFiles   int
	TotalImports int
	HasTailwind  bool
	HasModules   bool
	HasSCSS      bool
	FileTypes    map[FileType]int
}

// GetStats returns CSS tracking statistics.
func (t *Tracker) GetStats() Stats {
	stats := Stats{
		FileTypes: make(map[FileType]int),
	}
	for _, ft := range t.files {
		stats.TotalFiles++
		stats.TotalImports += len(ft.Imports)
		stats.FileTypes[ft.Type]++
		if ft.Tailwind {
			stats.HasTailwind = true
		}
		if ft.IsModule {
			stats.HasModules = true
		}
		if ft.Type == FileTypeSCSS || ft.Type == FileTypeSASS {
			stats.HasSCSS = true
		}
	}
	return stats
}

func extractImports(content, baseDir string) []string {
	matches := importRe.FindAllStringSubmatch(content, -1)
	var result []string
	for _, m := range matches {
		if len(m) > 1 {
			dep := m[1]
			if strings.HasPrefix(dep, "http") || strings.HasPrefix(dep, "//") {
				continue
			}
			if !strings.HasPrefix(dep, "~") {
				dep = filepath.Join(baseDir, dep)
			}
			result = append(result, dep)
		}
	}
	return result
}

func extToFileType(ext string) FileType {
	switch ext {
	case ".scss":
		return FileTypeSCSS
	case ".sass":
		return FileTypeSASS
	case ".less":
		return FileTypeLess
	default:
		return FileTypeCSS
	}
}

// Module field on File struct (was missing)
// Let me fix that in the struct definition

// TrackDirectory scans a directory for CSS files and tracks them.
func (t *Tracker) TrackDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less" {
			_, err := t.TrackFile(path)
			if err != nil {
				return nil
			}
		}
		return nil
	})
}