package config

import (
	"runtime"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Workers != -1 {
		t.Fatalf("expected workers=-1 (auto), got %d", cfg.Workers)
	}
	if !cfg.Cache.Enabled {
		t.Fatal("expected cache to be enabled by default")
	}
	if cfg.Cache.Compression != "zstd" {
		t.Fatalf("expected zstd compression, got %q", cfg.Cache.Compression)
	}
	if cfg.Watch.Debounce != 100 {
		t.Fatalf("expected debounce 100ms, got %d", cfg.Watch.Debounce)
	}
	if len(cfg.Scanner.Ignore) == 0 {
		t.Fatal("expected default scanner ignore list")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid auto workers",
			cfg: func() Config {
				c := Default()
				c.Workers = -1
				return c
			}(),
			wantErr: false,
		},
		{
			name: "valid explicit workers",
			cfg: func() Config {
				c := Default()
				c.Workers = 4
				return c
			}(),
			wantErr: false,
		},
		{
			name: "invalid workers",
			cfg: func() Config {
				c := Default()
				c.Workers = 0
				return c
			}(),
			wantErr: true,
		},
		{
			name: "negative cache size",
			cfg: func() Config {
				c := Default()
				c.Cache.MaxSizeMB = -1
				return c
			}(),
			wantErr: true,
		},
		{
			name: "negative debounce",
			cfg: func() Config {
				c := Default()
				c.Watch.Debounce = -10
				return c
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApplyPreset(t *testing.T) {
	t.Run("ci", func(t *testing.T) {
		cfg := Default()
		if err := cfg.ApplyPreset("ci"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Workers != runtime.NumCPU() {
			t.Fatalf("expected workers=%d for ci preset, got %d", runtime.NumCPU(), cfg.Workers)
		}
		if !cfg.Cache.Enabled {
			t.Fatal("expected cache enabled for ci preset")
		}
	})

	t.Run("low-memory caps workers", func(t *testing.T) {
		cfg := Default()
		cfg.Workers = -1
		if err := cfg.ApplyPreset("low-memory"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Workers < 1 {
			t.Fatalf("expected workers >= 1, got %d", cfg.Workers)
		}
		if cfg.Cache.MaxSizeMB > 5120 {
			t.Fatalf("expected cache capped at 5120MB, got %d", cfg.Cache.MaxSizeMB)
		}
	})

	t.Run("default is no-op", func(t *testing.T) {
		cfg := Default()
		if err := cfg.ApplyPreset(""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		cfg2 := Default()
		if cfg.Workers != cfg2.Workers {
			t.Fatal("empty preset should not change config")
		}
	})

	t.Run("unknown preset errors", func(t *testing.T) {
		cfg := Default()
		err := cfg.ApplyPreset("magic")
		if err == nil {
			t.Fatal("expected error for unknown preset")
		}
	})
}
