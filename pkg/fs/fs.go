package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// FileInfo is a platform-neutral file information.
type FileInfo struct {
	Name          string
	Path          string
	Size          int64
	Mode          os.FileMode
	ModTime       time.Time
	IsDir         bool
	IsSymlink     bool
	SymlinkTarget string
}

// FS provides filesystem operations with cross-platform behavior.
type FS interface {
	Stat(path string) (FileInfo, error)
	Lstat(path string) (FileInfo, error)
	ReadDir(path string) ([]FileInfo, error)
	Walk(root string, walkFn WalkFunc) error
	WalkDir(root string, walkFn WalkDirFunc) error
	ReadFile(path string) ([]byte, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	EvalSymlinks(path string) (string, error)
	IsAbs(path string) bool
	Join(elem ...string) string
	Rel(basepath, targpath string) (string, error)
	Base(path string) string
	Dir(path string) string
	Clean(path string) string
	VolumeName(path string) string
}

// WalkFunc is the function type called by Walk.
type WalkFunc func(path string, info FileInfo, err error) error

// WalkDirFunc is the function type called by WalkDir.
type WalkDirFunc func(path string, d FileInfo, err error) error

// OSFS is the default implementation using the host OS filesystem.
type OSFS struct{}

func (OSFS) Stat(path string) (FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoFromOS(path, fi), nil
}

func (OSFS) Lstat(path string) (FileInfo, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoFromOS(path, fi), nil
}

func (OSFS) ReadDir(path string) ([]FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]FileInfo, len(entries))
	for i, e := range entries {
		result[i] = dirEntryToFileInfo(path, e)
	}
	return result, nil
}

func (OSFS) Walk(root string, walkFn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return walkFn(path, FileInfo{}, err)
		}
		return walkFn(path, fileInfoFromOS(path, info), nil)
	})
}

func (OSFS) WalkDir(root string, walkFn WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return walkFn(path, FileInfo{}, err)
		}
		return walkFn(path, dirEntryToFileInfo(path, d), nil)
	})
}

func (OSFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (OSFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (OSFS) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

func (OSFS) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

func (OSFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (OSFS) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

func (OSFS) Base(path string) string {
	return filepath.Base(path)
}

func (OSFS) Dir(path string) string {
	return filepath.Dir(path)
}

func (OSFS) Clean(path string) string {
	return filepath.Clean(path)
}

func (OSFS) VolumeName(path string) string {
	return filepath.VolumeName(path)
}

func fileInfoFromOS(path string, fi os.FileInfo) FileInfo {
	info := FileInfo{
		Name:    fi.Name(),
		Path:    path,
		Size:    fi.Size(),
		Mode:    fi.Mode(),
		ModTime: fi.ModTime(),
		IsDir:   fi.IsDir(),
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		info.IsSymlink = true
		if target, err := os.Readlink(path); err == nil {
			info.SymlinkTarget = target
		}
	}
	return info
}

func dirEntryToFileInfo(basePath string, d os.DirEntry) FileInfo {
	info := FileInfo{
		Name:  d.Name(),
		Path:  filepath.Join(basePath, d.Name()),
		IsDir: d.IsDir(),
	}
	if fi, err := d.Info(); err == nil {
		info.Size = fi.Size()
		info.Mode = fi.Mode()
		info.ModTime = fi.ModTime()
		if fi.Mode()&os.ModeSymlink != 0 {
			info.IsSymlink = true
			if target, err := os.Readlink(info.Path); err == nil {
				info.SymlinkTarget = target
			}
		}
	}
	return info
}

// MemFS is an in-memory filesystem for testing.
type MemFS struct {
	mu    sync.RWMutex
	files map[string]*memFile
}

type memFile struct {
	data    []byte
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func NewMemFS() *MemFS {
	return &MemFS{
		files: make(map[string]*memFile),
	}
}

func (m *MemFS) Stat(path string) (FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	path = filepath.Clean(path)
	f, ok := m.files[path]
	if !ok {
		return FileInfo{}, os.ErrNotExist
	}
	return FileInfo{
		Name:    filepath.Base(path),
		Path:    path,
		Size:    int64(len(f.data)),
		Mode:    f.mode,
		ModTime: f.modTime,
		IsDir:   f.isDir,
	}, nil
}

func (m *MemFS) Lstat(path string) (FileInfo, error) {
	return m.Stat(path)
}

func (m *MemFS) ReadDir(path string) ([]FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	path = filepath.Clean(path)
	var result []FileInfo
	for p, f := range m.files {
		if filepath.Dir(p) == path {
			result = append(result, FileInfo{
				Name:    filepath.Base(p),
				Path:    p,
				Size:    int64(len(f.data)),
				Mode:    f.mode,
				ModTime: f.modTime,
				IsDir:   f.isDir,
			})
		}
	}
	return result, nil
}

func (m *MemFS) Walk(root string, walkFn WalkFunc) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	root = filepath.Clean(root)
	for p, f := range m.files {
		if !isUnder(p, root) {
			continue
		}
		if err := walkFn(p, FileInfo{
			Name:    filepath.Base(p),
			Path:    p,
			Size:    int64(len(f.data)),
			Mode:    f.mode,
			ModTime: f.modTime,
			IsDir:   f.isDir,
		}, nil); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemFS) WalkDir(root string, walkFn WalkDirFunc) error {
	return m.Walk(root, func(path string, info FileInfo, err error) error {
		return walkFn(path, info, err)
	})
}

func (m *MemFS) ReadFile(path string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	path = filepath.Clean(path)
	f, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	if f.isDir {
		return nil, os.ErrInvalid
	}
	data := make([]byte, len(f.data))
	copy(data, f.data)
	return data, nil
}

func (m *MemFS) MkdirAll(path string, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	path = filepath.Clean(path)
	if _, ok := m.files[path]; ok {
		return nil
	}
	m.files[path] = &memFile{mode: perm | os.ModeDir, modTime: time.Now(), isDir: true}
	return nil
}

func (m *MemFS) RemoveAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	path = filepath.Clean(path)
	for p := range m.files {
		if isUnder(p, path) || p == path {
			delete(m.files, p)
		}
	}
	return nil
}

func (m *MemFS) EvalSymlinks(path string) (string, error) {
	return path, nil
}

func (m *MemFS) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

func (m *MemFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (m *MemFS) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

func (m *MemFS) Base(path string) string {
	return filepath.Base(path)
}

func (m *MemFS) Dir(path string) string {
	return filepath.Dir(path)
}

func (m *MemFS) Clean(path string) string {
	return filepath.Clean(path)
}

func (m *MemFS) VolumeName(path string) string {
	return filepath.VolumeName(path)
}

// WriteFile adds a file to the in-memory filesystem.
func (m *MemFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	path = filepath.Clean(path)
	d := make([]byte, len(data))
	copy(d, data)
	m.files[path] = &memFile{data: d, mode: perm, modTime: time.Now()}
	return nil
}

func isUnder(path, root string) bool {
	if path == root {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	if rel == ".." || len(rel) > 2 && rel[0] == '.' && rel[1] == '.' {
		return false
	}
	return true
}

// Default is the default filesystem implementation.
var Default FS = OSFS{}

// IsWindows returns true if the platform is Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}