package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Theme ThemeConfig `toml:"theme"`
}

type ThemeConfig struct {
	Name string `toml:"name"`

	BgColor         string `toml:"bg_color"`
	PanelBgColor    string `toml:"panel_bg_color"`
	BorderColor     string `toml:"border_color"`
	AccentColor     string `toml:"accent_color"`
	AccentSoftColor string `toml:"accent_soft_color"`
	TextColor       string `toml:"text_color"`
	MutedColor      string `toml:"muted_color"`
	SubtleColor     string `toml:"subtle_color"`
	ChipBgColor     string `toml:"chip_bg_color"`
	ErrorColor      string `toml:"error_color"`
	SuccessColor    string `toml:"success_color"`
	SelectedBgColor string `toml:"selected_bg_color"`
	SelectedFgColor string `toml:"selected_fg_color"`

	BorderStyle   string `toml:"border_style"` // rounded, normal, double, thick, hidden
	AppPaddingX   int    `toml:"app_padding_x"`
	AppPaddingY   int    `toml:"app_padding_y"`
	PanelPaddingX int    `toml:"panel_padding_x"`
	PanelPaddingY int    `toml:"panel_padding_y"`
}

func Default() Config {
	return Config{
		Theme: ThemeConfig{
			Name:          "default",
			BorderStyle:   "rounded",
			AppPaddingX:   2,
			AppPaddingY:   1,
			PanelPaddingX: 1,
			PanelPaddingY: 0,
		},
	}
}

func Load() (Config, error) {
	cfg := Default()

	path := os.Getenv("NOTEUI_CONFIG")
	if strings.TrimSpace(path) == "" {
		userCfgDir, err := os.UserConfigDir()
		if err != nil {
			return cfg, err
		}
		path = filepath.Join(userCfgDir, "noteui", "config.toml")
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	} else if err != nil {
		return cfg, err
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
