package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOSFS_Stat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	fsys := OSFS{}
	fi, err := fsys.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Name != "test.txt" {
		t.Errorf("Name = %q, want %q", fi.Name, "test.txt")
	}
	if fi.Size != 5 {
		t.Errorf("Size = %d, want 5", fi.Size)
	}
}

func TestOSFS_Stat_NotExist(t *testing.T) {
	fsys := OSFS{}
	_, err := fsys.Stat("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestOSFS_ReadDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bb"), 0o644)
	os.Mkdir(filepath.Join(dir, "sub"), 0o755)

	fsys := OSFS{}
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("ReadDir returned %d entries, want 3", len(entries))
	}
}

func TestOSFS_Walk(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0o644)

	fsys := OSFS{}
	var paths []string
	err := fsys.Walk(dir, func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 4 {
		t.Errorf("Walk visited %d paths, want 4", len(paths))
	}
}

func TestOSFS_ReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	fsys := OSFS{}
	data, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile = %q, want %q", string(data), "hello")
	}
}

func TestOSFS_PathOps(t *testing.T) {
	fsys := OSFS{}

	if fsys.Join("a", "b") != filepath.Join("a", "b") {
		t.Error("Join mismatch")
	}
	if fsys.Base("/a/b") != filepath.Base("/a/b") {
		t.Error("Base mismatch")
	}
	if fsys.Dir("/a/b") != filepath.Dir("/a/b") {
		t.Error("Dir mismatch")
	}
	if fsys.Clean("/a//b") != filepath.Clean("/a//b") {
		t.Error("Clean mismatch")
	}
}

func TestMemFS_Basic(t *testing.T) {
	m := NewMemFS()

	m.WriteFile("/src/a.ts", []byte("import b from './b'"), 0o644)
	m.WriteFile("/src/b.ts", []byte("export const b = 1"), 0o644)
	m.MkdirAll("/src/sub", 0o755)
	m.WriteFile("/src/sub/c.ts", []byte("export const c = 2"), 0o644)

	fi, err := m.Stat("/src/a.ts")
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size != 19 {
		t.Errorf("Size = %d, want 19", fi.Size)
	}

	data, err := m.ReadFile("/src/a.ts")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "import b from './b'" {
		t.Errorf("ReadFile = %q", string(data))
	}

	entries, err := m.ReadDir("/src")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Errorf("ReadDir returned %d, want 3", len(entries))
	}
}

func TestMemFS_RemoveAll(t *testing.T) {
	m := NewMemFS()
	m.WriteFile("/a.txt", []byte("a"), 0o644)
	m.WriteFile("/sub/b.txt", []byte("b"), 0o644)
	m.MkdirAll("/sub", 0o755)

	m.RemoveAll("/sub")

	_, err := m.Stat("/sub/b.txt")
	if err == nil {
		t.Error("expected error after RemoveAll")
	}
}

func TestMemFS_Walk(t *testing.T) {
	m := NewMemFS()
	m.WriteFile("/a.txt", []byte("a"), 0o644)
	m.WriteFile("/b.txt", []byte("b"), 0o644)
	m.WriteFile("/sub/c.txt", []byte("c"), 0o644)
	m.MkdirAll("/sub", 0o755)

	var paths []string
	m.Walk("/", func(path string, info FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})

	if len(paths) < 3 {
		t.Errorf("Walk visited %d paths, want at least 3", len(paths))
	}
}

func TestMemFS_IsAbs(t *testing.T) {
	m := NewMemFS()
	if !m.IsAbs("/absolute") {
		t.Error("expected /absolute to be absolute")
	}
	if m.IsAbs("relative") {
		t.Error("expected relative to not be absolute")
	}
}

func TestMemFS_Rel(t *testing.T) {
	m := NewMemFS()
	rel, err := m.Rel("/a", "/a/b/c")
	if err != nil {
		t.Fatal(err)
	}
	if rel != "b/c" {
		t.Errorf("Rel = %q, want %q", rel, "b/c")
	}
}

func TestMemFS_VolumeName(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("VolumeName test only relevant on Windows")
	}
	m := NewMemFS()
	vn := m.VolumeName("C:\\Users\\test")
	if vn != "C:" {
		t.Errorf("VolumeName = %q, want %q", vn, "C:")
	}
}

func TestMemFS_EvalSymlinks(t *testing.T) {
	m := NewMemFS()
	result, err := m.EvalSymlinks("/some/path")
	if err != nil {
		t.Fatal(err)
	}
	if result != "/some/path" {
		t.Errorf("EvalSymlinks = %q, want /some/path", result)
	}
}

func TestMemFS_Clean(t *testing.T) {
	m := NewMemFS()
	result := m.Clean("/a//b/../c")
	if result != "/a/c" {
		t.Errorf("Clean = %q, want /a/c", result)
	}
}

func TestMemFS_ReadFile_NotExist(t *testing.T) {
	m := NewMemFS()
	_, err := m.ReadFile("/nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestMemFS_ReadFile_Dir(t *testing.T) {
	m := NewMemFS()
	m.MkdirAll("/dir", 0o755)
	_, err := m.ReadFile("/dir")
	if err == nil {
		t.Error("expected error for reading directory as file")
	}
}

func TestIsUnder(t *testing.T) {
	tests := []struct {
		path, root string
		want       bool
	}{
		{"/a/b", "/a", true},
		{"/a/b", "/b", false},
		{"/a/b/c", "/a", true},
		{"/a", "/a", false},
	}
	for _, tt := range tests {
		if got := isUnder(tt.path, tt.root); got != tt.want {
			t.Errorf("isUnder(%q, %q) = %v, want %v", tt.path, tt.root, got, tt.want)
		}
	}
}

func TestIsWindows(t *testing.T) {
	result := IsWindows()
	if result != (runtime.GOOS == "windows") {
		t.Errorf("IsWindows() = %v, want %v", result, runtime.GOOS == "windows")
	}
}