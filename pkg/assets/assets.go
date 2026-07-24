package assets

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileType represents the type of asset file.
type FileType string

const (
	FileTypePNG   FileType = "png"
	FileTypeJPEG  FileType = "jpeg"
	FileTypeJPG   FileType = "jpg"
	FileTypeSVG   FileType = "svg"
	FileTypeWebP  FileType = "webp"
	FileTypeAVIF  FileType = "avif"
	FileTypeGIF   FileType = "gif"
	FileTypeICO   FileType = "ico"
	FileTypeFont  FileType = "font"
	FileTypeVideo FileType = "video"
	FileTypeOther FileType = "other"
)

// File represents an asset file with metadata.
type File struct {
	Path       string   `json:"path"`
	RelPath    string   `json:"rel_path"`
	Type       FileType `json:"type"`
	Size       int64    `json:"size"`
	Hash       uint64   `json:"hash"`
	Optimized  bool     `json:"optimized"`
	OptPath    string   `json:"opt_path,omitempty"`
	OptSize    int64    `json:"opt_size,omitempty"`
}

// Optimizer handles parallel asset optimization.
type Optimizer struct {
	root    string
	files   []*File
	mu      sync.Mutex
	workers int
}

// NewOptimizer creates a new asset optimizer.
func NewOptimizer(root string, workers int) *Optimizer {
	if workers <= 0 {
		workers = 4
	}
	return &Optimizer{
		root:    root,
		workers: workers,
	}
}

// Scan discovers all asset files in the project.
func (o *Optimizer) Scan() error {
	return filepath.Walk(o.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".next" || name == ".git" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		ft := classifyFile(path)
		if ft == FileTypeOther {
			return nil
		}

		relPath, _ := filepath.Rel(o.root, path)
		o.files = append(o.files, &File{
			Path:    path,
			RelPath: relPath,
			Type:    ft,
			Size:    info.Size(),
		})
		return nil
	})
}

// Deduplicate finds duplicate assets by size and name pattern.
func (o *Optimizer) Deduplicate() [][]*File {
	sizeMap := make(map[int64][]*File)
	for _, f := range o.files {
		sizeMap[f.Size] = append(sizeMap[f.Size], f)
	}

	var duplicates [][]*File
	for _, group := range sizeMap {
		if len(group) > 1 {
			duplicates = append(duplicates, group)
		}
	}
	return duplicates
}

// ListFiles returns all discovered asset files sorted by path.
func (o *Optimizer) ListFiles() []*File {
	result := make([]*File, len(o.files))
	copy(result, o.files)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result
}

// FilterByType returns files matching the given type.
func (o *Optimizer) FilterByType(ft FileType) []*File {
	var result []*File
	for _, f := range o.files {
		if f.Type == ft {
			result = append(result, f)
		}
	}
	return result
}

// Stats returns asset statistics.
type Stats struct {
	TotalFiles int
	TotalSize  int64
	TypeCount  map[FileType]int
	TypeSize   map[FileType]int64
}

// GetStats returns statistics about discovered assets.
func (o *Optimizer) GetStats() Stats {
	stats := Stats{
		TypeCount: make(map[FileType]int),
		TypeSize:  make(map[FileType]int64),
	}
	for _, f := range o.files {
		stats.TotalFiles++
		stats.TotalSize += f.Size
		stats.TypeCount[f.Type]++
		stats.TypeSize[f.Type] += f.Size
	}
	return stats
}

func classifyFile(path string) FileType {
	ext := filepath.Ext(path)
	switch ext {
	case ".png":
		return FileTypePNG
	case ".jpeg", ".jpg":
		return FileTypeJPEG
	case ".svg":
		return FileTypeSVG
	case ".webp":
		return FileTypeWebP
	case ".avif":
		return FileTypeAVIF
	case ".gif":
		return FileTypeGIF
	case ".ico":
		return FileTypeICO
	case ".woff", ".woff2", ".ttf", ".otf", ".eot":
		return FileTypeFont
	case ".mp4", ".webm", ".avi", ".mov":
		return FileTypeVideo
	default:
		return FileTypeOther
	}
}