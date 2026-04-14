package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

func ResolvePath() (string, error) {
	path := os.Getenv("NOTEUI_CONFIG")
	if strings.TrimSpace(path) != "" {
		return path, nil
	}

	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCfgDir, "noteui", "config.toml"), nil
}

func Save(cfg Config) error {
	if err := Validate(cfg); err != nil {
		return err
	}

	path, err := ResolvePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	encErr := toml.NewEncoder(f).Encode(cfg)
	if closeErr := f.Close(); closeErr != nil && encErr == nil {
		return closeErr
	}
	return encErr
}

// SaveTheme updates theme.name in the config file and returns the previous
// theme name and the path of the file that was written.
func SaveTheme(name string) (oldName, configPath string, err error) {
	cfg, err := Load()
	if err != nil {
		return "", "", err
	}
	oldName = strings.TrimSpace(cfg.Theme.Name)
	if oldName == "" {
		oldName = "default"
	}
	cfg.Theme.Name = name
	path, err := ResolvePath()
	if err != nil {
		return "", "", err
	}
	if err := Save(cfg); err != nil {
		return "", "", err
	}
	return oldName, path, nil
}

func SaveDefaultSyncProfile(profile string) (Config, string, error) {
	cfg, err := Load()
	if err != nil {
		return Config{}, "", err
	}
	cfg.Sync.DefaultProfile = strings.TrimSpace(profile)
	path, err := ResolvePath()
	if err != nil {
		return Config{}, "", err
	}
	if err := Save(cfg); err != nil {
		return Config{}, "", err
	}
	return cfg, path, nil
}
