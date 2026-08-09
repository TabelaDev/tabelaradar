package main

import (
	"github.com/charmbracelet/bubbles/key"
)

// tabelaradar's keybindings, declared once and shared by the key dispatch in
// Update (key.Matches), the footer hints (tuiui.Footer) and the help modal
// (tuiui.HelpModal) — the hints can never drift out of sync with what
// Update actually matches.
var (
	keyQuit      = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	keyHelp      = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keybindings"))
	keyRefresh   = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rescan"))
	keyOpen      = key.NewBinding(key.WithKeys("o", "enter"), key.WithHelp("o/enter", "abrir editor"))
	keyFocusList = key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "projetos"))
	keyFocusDesc = key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "descrição"))
	keyScroll    = key.NewBinding(key.WithKeys("j", "k", "up", "down"), key.WithHelp("j/k", "rolar descrição"))
)

// appKeymap is the full list of bindings the footer hints and the help modal
// render from.
var appKeymap = []key.Binding{
	keyRefresh, keyOpen, keyFocusList, keyFocusDesc, keyScroll, keyHelp, keyQuit,
}
