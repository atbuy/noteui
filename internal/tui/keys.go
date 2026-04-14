package tui

import (
	"atbuy/noteui/internal/config"
	"atbuy/noteui/internal/tui/shortcuts"
)

var keys = shortcuts.DefaultMap()

func ApplyConfigKeys(cfg config.KeysConfig) {
	shortcuts.ApplyConfig(&keys, cfg)
}

func ValidateKeyCollisions() []string {
	return shortcuts.ValidateCollisions(keys)
}
