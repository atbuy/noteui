package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Open           key.Binding
	Refresh        key.Binding
	Quit           key.Binding
	Focus          key.Binding
	NewNote        key.Binding
	Search         key.Binding
	ShowHelp       key.Binding
	CreateCategory key.Binding
	ToggleCategory key.Binding
	Delete         key.Binding
	Move           key.Binding
	Rename         key.Binding
	Pin            key.Binding
}

var keys = keyMap{
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "Search"),
	),
	Open: key.NewBinding(
		key.WithKeys("enter", "o"),
		key.WithHelp("enter/o", "Open in editor"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "Refresh"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	Focus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "Switch focused pane"),
	),
	NewNote: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "New note"),
	),
	ShowHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "Help"),
	),
	CreateCategory: key.NewBinding(
		key.WithKeys("C"),
		key.WithHelp("C", "New category"),
	),
	ToggleCategory: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "Toggle category"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "Delete"),
	),
	Move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "Move"),
	),
	Rename: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "Rename note"),
	),
	Pin: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Pin"),
	),
}
