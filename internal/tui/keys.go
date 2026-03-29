package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Open    key.Binding
	Refresh key.Binding
	Quit    key.Binding
	Focus   key.Binding
	NewNote key.Binding
}

var keys = keyMap{
	Open: key.NewBinding(
		key.WithKeys("enter", "o"),
		key.WithHelp("enter/o", "open in nvim"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Focus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle search/list"),
	),
	NewNote: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new note"),
	),
}

