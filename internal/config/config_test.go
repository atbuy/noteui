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
	require.Equal(t, []string{"ctrl+e"}, cfg.Keys.ShowSyncDebug)
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
		`show_sync_debug = ["gd"]`,
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
	require.Equal(t, []string{"gd"}, cfg.Keys.ShowSyncDebug)
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

func TestValidateMissingKindDefaultsToSSH(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "srv"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"srv": {SSHHost: "host", RemoteRoot: "/notes", RemoteBin: "noteui-sync"},
	}
	require.NoError(t, Validate(cfg))
	require.Equal(t, SyncKindSSH, ResolvedKind(cfg.Sync.Profiles["srv"]))
}

func TestValidateAcceptsWebDAVProfileWithBasicAuth(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "cloud"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"cloud": {
			Kind:        "webdav",
			WebDAVURL:   "https://cloud.example.com/dav",
			Auth:        "basic",
			UsernameEnv: "DAV_USER",
			PasswordEnv: "DAV_PASS",
		},
	}
	require.NoError(t, Validate(cfg))
}

func TestValidateAcceptsWebDAVProfileWithNoAuth(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "local"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"local": {
			Kind:      "webdav",
			WebDAVURL: "http://192.168.1.50/dav",
			Auth:      "none",
		},
	}
	require.NoError(t, Validate(cfg))
}

func TestValidateWebDAVDefaultsAuthToBasic(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "cloud"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"cloud": {
			Kind:      "webdav",
			WebDAVURL: "https://cloud.example.com/dav",
		},
	}
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing username_env")
}

func TestValidateRejectsWebDAVMissingURL(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "cloud"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"cloud": {Kind: "webdav", Auth: "none"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing webdav_url")
}

func TestValidateRejectsWebDAVBadURLScheme(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "cloud"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"cloud": {Kind: "webdav", WebDAVURL: "ftp://example.com", Auth: "none"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must start with http:// or https://")
}

func TestValidateRejectsUnknownBackendKind(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "x"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"x": {Kind: "ftp"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown kind "ftp"`)
}

func TestValidateRejectsUnknownAuthMode(t *testing.T) {
	cfg := Default()
	cfg.Sync.DefaultProfile = "x"
	cfg.Sync.Profiles = map[string]SyncProfile{
		"x": {Kind: "webdav", WebDAVURL: "https://x.com/dav", Auth: "oauth"},
	}
	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown auth mode "oauth"`)
}

func TestLoadWebDAVProfileFromTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := strings.Join([]string{
		"[sync]",
		`default_profile = "cloud"`,
		"",
		"[sync.profiles.cloud]",
		`kind = "webdav"`,
		`webdav_url = "https://cloud.example.com/remote.php/dav/files/alice"`,
		`remote_root = "/noteui/personal"`,
		`auth = "basic"`,
		`username_env = "NOTEUI_WEBDAV_USER"`,
		`password_env = "NOTEUI_WEBDAV_PASSWORD"`,
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("NOTEUI_CONFIG", path)

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "cloud", cfg.Sync.DefaultProfile)

	p := cfg.Sync.Profiles["cloud"]
	require.Equal(t, "webdav", p.Kind)
	require.Equal(t, "https://cloud.example.com/remote.php/dav/files/alice", p.WebDAVURL)
	require.Equal(t, "/noteui/personal", p.RemoteRoot)
	require.Equal(t, "basic", p.Auth)
	require.Equal(t, "NOTEUI_WEBDAV_USER", p.UsernameEnv)
	require.Equal(t, "NOTEUI_WEBDAV_PASSWORD", p.PasswordEnv)
}

func TestResolvedKindDefaultsToSSH(t *testing.T) {
	require.Equal(t, SyncKindSSH, ResolvedKind(SyncProfile{}))
	require.Equal(t, SyncKindSSH, ResolvedKind(SyncProfile{Kind: "ssh"}))
	require.Equal(t, SyncKindSSH, ResolvedKind(SyncProfile{Kind: "SSH"}))
	require.Equal(t, SyncKindWebDAV, ResolvedKind(SyncProfile{Kind: "webdav"}))
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

func TestSaveThemeWritesNewThemeAndReturnsOld(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "[theme]\nname = \"nord\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	t.Setenv("NOTEUI_CONFIG", path)

	oldName, writtenPath, err := SaveTheme("dracula")
	require.NoError(t, err)
	require.Equal(t, "nord", oldName)
	require.Equal(t, path, writtenPath)

	reloaded, err := Load()
	require.NoError(t, err)
	require.Equal(t, "dracula", reloaded.Theme.Name)
}

func TestSaveThemeReturnsDefaultWhenNoConfigExists(t *testing.T) {
	t.Setenv("NOTEUI_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))

	oldName, _, err := SaveTheme("nord")
	require.NoError(t, err)
	require.Equal(t, "default", oldName)
}

func TestValidThemeNamesIncludesNewThemes(t *testing.T) {
	names := ValidThemeNames()
	for _, want := range []string{
		"rose-pine", "rosepine", "monokai", "solarized-dark", "solarized",
		"ayu-dark", "ayu", "material", "material-dark", "nightfox",
	} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidThemeNames missing %q", want)
		}
	}
}

func TestValidateRejectsUnknownDefaultWorkspace(t *testing.T) {
	cfg := Default()
	cfg.DefaultWorkspace = "missing"
	cfg.Workspaces = map[string]WorkspaceConfig{
		"work": {Root: "/notes/work"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown default_workspace "missing"`)
}

func TestValidateRejectsWorkspaceWithoutRoot(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = map[string]WorkspaceConfig{
		"work": {},
	}

	err := Validate(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), `workspace "work" is missing root`)
}

func TestResolveStartupWorkspaceUsesOverride(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = map[string]WorkspaceConfig{
		"work": {Root: "/notes/work", Label: "Work"},
	}

	got := ResolveStartupWorkspace(cfg, "/tmp/demo-notes", "/fallback")
	require.True(t, got.Override)
	require.Equal(t, "/tmp/demo-notes", got.Root)
	require.Empty(t, got.Name)
}

func TestResolveStartupWorkspaceUsesDefaultWorkspace(t *testing.T) {
	cfg := Default()
	cfg.DefaultWorkspace = "work"
	cfg.Workspaces = map[string]WorkspaceConfig{
		"work": {Root: "/notes/work", Label: "Work"},
		"demo": {Root: "/notes/demo", Label: "Demo"},
	}

	got := ResolveStartupWorkspace(cfg, "", "/fallback")
	require.False(t, got.Override)
	require.False(t, got.NeedsSelection)
	require.Equal(t, "work", got.Name)
	require.Equal(t, "Work", got.Label)
	require.Equal(t, "/notes/work", got.Root)
}

func TestResolveStartupWorkspaceRequiresSelectionForMultipleWithoutDefault(t *testing.T) {
	cfg := Default()
	cfg.Workspaces = map[string]WorkspaceConfig{
		"work": {Root: "/notes/work"},
		"demo": {Root: "/notes/demo"},
	}

	got := ResolveStartupWorkspace(cfg, "", "/fallback")
	require.True(t, got.NeedsSelection)
	require.Empty(t, got.Root)
}

func TestResolveStartupWorkspaceUsesFallbackWithoutProfiles(t *testing.T) {
	got := ResolveStartupWorkspace(Default(), "", "/fallback")
	require.Equal(t, "/fallback", got.Root)
	require.False(t, got.Override)
	require.False(t, got.NeedsSelection)
}
