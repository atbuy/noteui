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
	require.Equal(t, []string{"S"}, cfg.Keys.ToggleSync)
	require.Equal(t, []string{"U"}, cfg.Keys.DeleteRemoteKeepLocal)
	require.Equal(t, []string{"i"}, cfg.Keys.SyncImportCurrent)
	require.Equal(t, []string{"I"}, cfg.Keys.SyncImport)
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
		`synced_note_color = "#22c55e"`,
		`unsynced_note_color = "#ef4444"`,
		`syncing_note_color = "#f59e0b"`,
		"",
		"[preview]",
		`style = "light"`,
		`code_style = "github"`,
		`line_numbers = false`,
		"",
		"[keys]",
		`toggle_sync = ["gs"]`,
		`delete_remote_keep_local = ["gu"]`,
		`sync_import_current = ["ii"]`,
		`sync_import = ["gi"]`,
		"",
		"[sync]",
		`default_profile = "homebox"`,
		"",
		"[sync.profiles.homebox]",
		`ssh_host = "notes-prod"`,
		`remote_root = "/srv/noteui"`,
		`remote_bin = "/usr/local/bin/noteui-sync"`,
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
	require.Equal(t, "#22c55e", cfg.Theme.SyncedNoteColor)
	require.Equal(t, "#ef4444", cfg.Theme.UnsyncedNoteColor)
	require.Equal(t, "#f59e0b", cfg.Theme.SyncingNoteColor)
	require.Equal(t, []string{"gs"}, cfg.Keys.ToggleSync)
	require.Equal(t, []string{"gu"}, cfg.Keys.DeleteRemoteKeepLocal)
	require.Equal(t, []string{"ii"}, cfg.Keys.SyncImportCurrent)
	require.Equal(t, []string{"gi"}, cfg.Keys.SyncImport)
	require.Equal(t, "homebox", cfg.Sync.DefaultProfile)
	require.Equal(t, "notes-prod", cfg.Sync.Profiles["homebox"].SSHHost)
}

func TestValidateRejectsUnknownDefaultSyncProfile(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "missing"

	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown sync.default_profile "missing"`)
}

func TestValidateRejectsIncompleteSyncProfile(t *testing.T) {
	cfg := Default()
	cfg.Sync.Profiles = map[string]SyncProfile{
		"homebox": {SSHHost: "notes-prod"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `sync profile "homebox" is missing remote_root`)
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

func TestSaveDefaultSyncProfileWritesConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := strings.Join([]string{
		"dashboard = false",
		"",
		"[sync]",
		`default_profile = "homebox"`,
		"",
		"[sync.profiles.homebox]",
		`ssh_host = "notes-prod"`,
		`remote_root = "/srv/homebox"`,
		`remote_bin = "noteui-sync"`,
		"",
		"[sync.profiles.backup]",
		`ssh_host = "backup-host"`,
		`remote_root = "/srv/backup"`,
		`remote_bin = "noteui-sync"`,
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, writtenPath, err := SaveDefaultSyncProfile("backup")
	require.NoError(t, err)
	require.Equal(t, path, writtenPath)
	require.Equal(t, "backup", cfg.Sync.DefaultProfile)

	reloaded, err := Load()
	require.NoError(t, err)
	require.False(t, reloaded.Dashboard)
	require.Equal(t, "backup", reloaded.Sync.DefaultProfile)
	require.Equal(t, "notes-prod", reloaded.Sync.Profiles["homebox"].SSHHost)
	require.Equal(t, "backup-host", reloaded.Sync.Profiles["backup"].SSHHost)
}
