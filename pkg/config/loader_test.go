package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JobV9n/effectiveNext/internal/testutil"
)

func TestLoader_Load_Default(t *testing.T) {
	dir := testutil.TempDir(t)
	loader := NewLoader(dir)

	cfg, err := loader.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != -1 {
		t.Fatalf("expected default workers=-1, got %d", cfg.Workers)
	}
}

func TestLoader_Load_YAML(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "effective-next.yaml", `workers: 8
cache:
  enabled: true
  compression: zstd
  maxSizeMB: 1024
scanner:
  ignore:
    - node_modules
    - .next
watch:
  enabled: true
  debounceMs: 250
`)

	loader := NewLoader(dir)
	cfg, err := loader.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != 8 {
		t.Fatalf("expected workers=8, got %d", cfg.Workers)
	}
	if cfg.Cache.MaxSizeMB != 1024 {
		t.Fatalf("expected cache maxSizeMB=1024, got %d", cfg.Cache.MaxSizeMB)
	}
	if cfg.Watch.Debounce != 250 {
		t.Fatalf("expected debounce=250, got %d", cfg.Watch.Debounce)
	}
}

func TestLoader_Load_JSON(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "effective-next.json", `{"workers": 4, "cache": {"enabled": true, "compression": "zstd"}}`)

	loader := NewLoader(dir)
	cfg, err := loader.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != 4 {
		t.Fatalf("expected workers=4, got %d", cfg.Workers)
	}
}

func TestLoader_Load_PackageJSON(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "package.json", `{"name": "demo", "effective-next": {"workers": 2, "profile": "ci"}}`)

	loader := NewLoader(dir)
	cfg, err := loader.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != 2 {
		t.Fatalf("expected workers=2 from package.json, got %d", cfg.Workers)
	}
	if cfg.Profile != "ci" {
		t.Fatalf("expected profile=ci, got %q", cfg.Profile)
	}
}

func TestLoader_Load_ExplicitPath(t *testing.T) {
	dir := testutil.TempDir(t)
	path := testutil.WriteFile(t, dir, "custom.yaml", `workers: 16`)

	loader := NewLoader(dir)
	cfg, err := loader.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != 16 {
		t.Fatalf("expected workers=16, got %d", cfg.Workers)
	}
}

func TestLoader_Load_InvalidYAML(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "effective-next.yaml", `workers: [invalid`)

	loader := NewLoader(dir)
	_, err := loader.Load("")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoader_Load_InvalidConfig(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "effective-next.yaml", `workers: 0`)

	loader := NewLoader(dir)
	_, err := loader.Load("")
	if err == nil {
		t.Fatal("expected validation error for workers=0")
	}
}

func TestLoader_Load_UnknownFormat(t *testing.T) {
	dir := testutil.TempDir(t)
	path := testutil.WriteFile(t, dir, "config.toml", `workers = 1`)

	loader := NewLoader(dir)
	_, err := loader.Load(path)
	if err == nil {
		t.Fatal("expected error for unsupported config format")
	}
}

func TestDiscoverOrder(t *testing.T) {
	dir := testutil.TempDir(t)
	// Write effective-next.json and package.json; loader should prefer effective-next.json.
	testutil.WriteFile(t, dir, "effective-next.json", `{"workers": 7}`)
	testutil.WriteFile(t, dir, "package.json", `{"name": "demo"}`)

	loader := NewLoader(dir)
	path, found := loader.discover()
	if !found {
		t.Fatal("expected config discovery to find a file")
	}
	if filepath.Base(path) != "effective-next.json" {
		t.Fatalf("expected effective-next.json to be preferred, got %q", path)
	}
}

func TestLoadFromPackageJSON_NoConfig(t *testing.T) {
	cfg, err := loadFromPackageJSON([]byte(`{"name": "demo"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Workers != -1 {
		t.Fatalf("expected defaults when effective-next key absent, got workers=%d", cfg.Workers)
	}
}

func TestLoadFromPackageJSON_InvalidJSON(t *testing.T) {
	_, err := loadFromPackageJSON([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoad_NonExistentExplicit(t *testing.T) {
	dir := testutil.TempDir(t)
	loader := NewLoader(dir)
	_, err := loader.Load(filepath.Join(dir, "missing.yaml"))
	if err == nil {
		t.Fatal("expected error for missing explicit config")
	}
}

func TestLoad_OverridesDefaultIgnore(t *testing.T) {
	dir := testutil.TempDir(t)
	testutil.WriteFile(t, dir, "effective-next.yaml", `scanner:
  ignore:
    - custom_dir
`)

	loader := NewLoader(dir)
	cfg, err := loader.Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Scanner.Ignore) != 1 || cfg.Scanner.Ignore[0] != "custom_dir" {
		t.Fatalf("expected override ignore list, got %v", cfg.Scanner.Ignore)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
