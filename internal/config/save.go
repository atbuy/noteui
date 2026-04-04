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
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
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
