package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Loader discovers and loads effective-next configuration files.
type Loader struct {
	searchDir string
}

// NewLoader creates a Loader that searches for config in searchDir.
func NewLoader(searchDir string) *Loader {
	return &Loader{searchDir: searchDir}
}

// Load discovers and loads configuration. If explicit is non-empty, that file is
// used. Otherwise, Load searches for effective-next.yaml, effective-next.yml, effective-next.json or a
// "effective-next" key in package.json.
func (l *Loader) Load(explicit string) (Config, error) {
	cfg := Default()

	path := explicit
	if path == "" {
		var found bool
		path, found = l.discover()
		if !found {
			return cfg, nil
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("cannot read config %q: %w", path, err)
	}

	ext := filepath.Ext(path)
	// Handle package.json specially before extension-based dispatch.
	if filepath.Base(path) == "package.json" {
		cfg, err = loadFromPackageJSON(data)
		if err != nil {
			return cfg, fmt.Errorf("cannot load effective-next config from package.json: %w", err)
		}
	} else {
		switch ext {
		case ".yaml", ".yml":
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return cfg, fmt.Errorf("invalid YAML in %q: %w", path, err)
			}
		case ".json":
			if err := json.Unmarshal(data, &cfg); err != nil {
				return cfg, fmt.Errorf("invalid JSON in %q: %w", path, err)
			}
		default:
			return cfg, fmt.Errorf("unsupported config format: %s", path)
		}
	}

	if err := cfg.ApplyPreset(cfg.Profile); err != nil {
		return cfg, err
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// discover returns the first existing config file path and true, or an empty
// string and false if none is found.
func (l *Loader) discover() (string, bool) {
	candidates := []string{
		filepath.Join(l.searchDir, "effective-next.yaml"),
		filepath.Join(l.searchDir, "effective-next.yml"),
		filepath.Join(l.searchDir, "effective-next.json"),
		filepath.Join(l.searchDir, "package.json"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// loadFromPackageJSON extracts the "effective-next" field from package.json.
func loadFromPackageJSON(data []byte) (Config, error) {
	var wrapper struct {
		EffectiveNext Config `json:"effective-next"`
	}
	cfg := Default()
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return cfg, err
	}
	if reflect.DeepEqual(wrapper.EffectiveNext, Config{}) {
		return cfg, nil
	}
	return wrapper.EffectiveNext, nil
}
