package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaultProvidesExpectedBaseline(t *testing.T) {
	cfg := Default()

	if !cfg.Dashboard {
		t.Fatal("expected dashboard to be enabled by default")
	}
	if cfg.Theme.Name != "default" {
		t.Fatalf("expected default theme, got %q", cfg.Theme.Name)
	}
	if cfg.Theme.BorderStyle != "rounded" {
		t.Fatalf("expected rounded border style, got %q", cfg.Theme.BorderStyle)
	}
	if !cfg.Preview.RenderMarkdown {
		t.Fatal("expected markdown preview to be enabled by default")
	}
	if !cfg.Preview.SyntaxHighlight {
		t.Fatal("expected syntax highlighting to be enabled by default")
	}
	if !cfg.Preview.LineNumbers {
		t.Fatal("expected line numbers to be enabled by default")
	}
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg := Default()
	cfg.Theme.Name = "nord"
	cfg.Theme.BorderStyle = "double"
	cfg.Modal.BorderStyle = "thick"
	cfg.Preview.Style = "light"
	cfg.Preview.CodeStyle = "github"

	if err := Validate(cfg); err != nil {
		t.Fatalf("Validate returned error for valid config: %v", err)
	}
}

func TestValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "invalid theme name",
			mutate: func(cfg *Config) {
				cfg.Theme.Name = "bogus"
			},
			wantErr: `unknown theme "bogus"`,
		},
		{
			name: "invalid theme border",
			mutate: func(cfg *Config) {
				cfg.Theme.BorderStyle = "zigzag"
			},
			wantErr: `invalid theme.border_style "zigzag"`,
		},
		{
			name: "invalid modal border",
			mutate: func(cfg *Config) {
				cfg.Modal.BorderStyle = "zigzag"
			},
			wantErr: `invalid modal.border_style "zigzag"`,
		},
		{
			name: "invalid preview style",
			mutate: func(cfg *Config) {
				cfg.Preview.Style = "sepia"
			},
			wantErr: `invalid preview.style "sepia"`,
		},
		{
			name: "invalid code style",
			mutate: func(cfg *Config) {
				cfg.Preview.CodeStyle = "mystery"
			},
			wantErr: `invalid preview.code_style "mystery"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)

			err := Validate(cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err)
			}
		})
	}
}

func TestLoadReturnsDefaultWhenConfigMissing(t *testing.T) {
	t.Setenv("NOTEUI_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("expected default config, got %#v", cfg)
	}
}

func TestLoadAppliesOverridesFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := strings.Join([]string{
		"dashboard = false",
		"",
		"[theme]",
		`name = "nord"`,
		`border_style = "double"`,
		"",
		"[preview]",
		`style = "light"`,
		`code_style = "github"`,
		`line_numbers = false`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Dashboard {
		t.Fatal("expected dashboard override to be applied")
	}
	if cfg.Theme.Name != "nord" {
		t.Fatalf("expected theme override to be applied, got %q", cfg.Theme.Name)
	}
	if cfg.Theme.BorderStyle != "double" {
		t.Fatalf("expected border style override, got %q", cfg.Theme.BorderStyle)
	}
	if cfg.Preview.Style != "light" {
		t.Fatalf("expected preview style override, got %q", cfg.Preview.Style)
	}
	if cfg.Preview.CodeStyle != "github" {
		t.Fatalf("expected code style override, got %q", cfg.Preview.CodeStyle)
	}
	if cfg.Preview.LineNumbers {
		t.Fatal("expected line number override to be applied")
	}
}

func TestLoadReturnsDefaultOnParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[theme\nname = \"nord\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "config parse error") {
		t.Fatalf("expected parse error, got %q", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("expected default config on parse error, got %#v", cfg)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := strings.Join([]string{
		"[theme]",
		`name = "default"`,
		`unexpected = "value"`,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error for unknown keys, got nil")
	}
	if !strings.Contains(err.Error(), "unknown config key(s): theme.unexpected") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("expected default config on error, got %#v", cfg)
	}
}
