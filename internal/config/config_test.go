package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestDefaultProvidesExpectedBaseline(t *testing.T) {
	cfg := Default()

	require.True(t, cfg.Dashboard)
	require.Equal(t, "default", cfg.Theme.Name)
	require.Equal(t, "rounded", cfg.Theme.BorderStyle)
	require.True(t, cfg.Preview.RenderMarkdown)
	require.True(t, cfg.Preview.SyntaxHighlight)
	require.True(t, cfg.Preview.LineNumbers)
}

func TestValidateAcceptsValidConfig(t *testing.T) {
	cfg := Default()
	cfg.Theme.Name = "nord"
	cfg.Theme.BorderStyle = "double"
	cfg.Modal.BorderStyle = "thick"
	cfg.Preview.Style = "light"
	cfg.Preview.CodeStyle = "github"

	require.NoError(t, Validate(cfg))
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
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoadReturnsDefaultWhenConfigMissing(t *testing.T) {
	t.Setenv("NOTEUI_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))

	cfg, err := Load()
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(Default(), cfg))
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
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.Dashboard)
	require.Equal(t, "nord", cfg.Theme.Name)
	require.Equal(t, "double", cfg.Theme.BorderStyle)
	require.Equal(t, "light", cfg.Preview.Style)
	require.Equal(t, "github", cfg.Preview.CodeStyle)
	require.False(t, cfg.Preview.LineNumbers)
}

func TestLoadReturnsDefaultOnParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[theme\nname = \"nord\"\n"), 0o644))
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "config parse error")
	require.Empty(t, cmp.Diff(Default(), cfg))
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := strings.Join([]string{
		"[theme]",
		`name = "default"`,
		`unexpected = "value"`,
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown config key(s): theme.unexpected")
	require.Empty(t, cmp.Diff(Default(), cfg))
}
