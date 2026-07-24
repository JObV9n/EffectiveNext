// Package config defines and loads effective-next configuration.
package config

import (
	"fmt"
	"runtime"
)

// Config is the top-level effective-next configuration.
type Config struct {
	Workers      int              `json:"workers" yaml:"workers"`
	Cache        CacheConfig      `json:"cache" yaml:"cache"`
	Scanner      ScannerConfig    `json:"scanner" yaml:"scanner"`
	Watch        WatchConfig      `json:"watch" yaml:"watch"`
	Optimization OptimizationConfig `json:"optimization" yaml:"optimization"`
	Profile      string           `json:"profile,omitempty" yaml:"profile,omitempty"`
}

// CacheConfig controls caching behavior.
type CacheConfig struct {
	Enabled     bool   `json:"enabled" yaml:"enabled"`
	Compression string `json:"compression" yaml:"compression"`
	MaxSizeMB   int    `json:"maxSizeMB,omitempty" yaml:"maxSizeMB,omitempty"`
	Dir         string `json:"dir,omitempty" yaml:"dir,omitempty"`
}

// ScannerConfig controls filesystem scanning.
type ScannerConfig struct {
	Ignore []string `json:"ignore" yaml:"ignore"`
}

// WatchConfig controls development file watching.
type WatchConfig struct {
	Enabled  bool `json:"enabled" yaml:"enabled"`
	Debounce int  `json:"debounceMs" yaml:"debounceMs"`
}

// OptimizationConfig toggles asset and CSS optimization.
type OptimizationConfig struct {
	Assets bool `json:"assets" yaml:"assets"`
	CSS    bool `json:"css" yaml:"css"`
}

// Default returns a configuration populated with sensible defaults.
func Default() Config {
	return Config{
		Workers: defaultWorkers(),
	Cache: CacheConfig{
		Enabled:     true,
		Compression: "zstd",
		MaxSizeMB:   25600,
		Dir:         ".effective-next/cache",
	},
		Scanner: ScannerConfig{
			Ignore: []string{
				"node_modules",
				".next",
				".git",
				"dist",
				"coverage",
				"out",
			},
		},
		Watch: WatchConfig{
			Enabled:  true,
			Debounce: 100,
		},
		Optimization: OptimizationConfig{
			Assets: true,
			CSS:    true,
		},
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if c.Workers < 1 && c.Workers != -1 {
		return fmt.Errorf("workers must be >= 1 or 'auto'(-1), got %d", c.Workers)
	}
	if c.Cache.MaxSizeMB < 0 {
		return fmt.Errorf("cache.maxSizeMB must be >= 0")
	}
	if c.Watch.Debounce < 0 {
		return fmt.Errorf("watch.debounceMs must be >= 0")
	}
	return nil
}

// ApplyPreset modifies the configuration based on the named profile.
func (c *Config) ApplyPreset(name string) error {
	switch name {
	case "", "default":
		return nil
	case "ci":
		if c.Workers == -1 {
			c.Workers = runtime.NumCPU()
		}
		c.Cache.Enabled = true
	case "low-memory":
		if c.Workers == -1 || c.Workers > max(2, runtime.NumCPU()/2) {
			c.Workers = max(2, runtime.NumCPU()/2)
		}
		c.Cache.MaxSizeMB = min(c.Cache.MaxSizeMB, 5120)
	case "desktop":
		c.Workers = -1
	default:
		return fmt.Errorf("unknown profile: %q", name)
	}
	c.Profile = name
	return nil
}

func defaultWorkers() int {
	return -1 // auto
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
