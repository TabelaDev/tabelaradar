package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			os.Exit(runList())
		case "ipc":
			os.Exit(runIPC(os.Args[2:]))
		case "digest":
			os.Exit(runDigest(os.Args[2:]))
		}
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

// runList prints a plain-text dump of the scan and exits — no TTY/altscreen
// needed, useful for piping into other tools or checking the scan logic
// without launching the TUI.
func runList() int {
	entries, cfgWarning := loadRootsConfig()
	if cfgWarning != "" {
		fmt.Fprintln(os.Stderr, "aviso:", cfgWarning)
	}
	projects, warnings := scanAll(entries)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "aviso:", w)
	}
	for _, p := range projects {
		fmt.Printf("%s %-20s branch=%-14s sujo=%-3s push=%-4s ativ=%-9s remoto=%s\n",
			statusGlyph(p), p.Name, p.Branch, dirtyCell(p), pushCell(p), humanizeAgo(p.LastCommitTime), remoteCell(p))
		if p.Description != "" {
			fmt.Printf("    %s: %s\n", p.DescriptionSrc, p.Description)
		}
		for _, note := range p.MemoryNotes {
			fmt.Printf("    * %s\n", note)
		}
	}
	return 0
}
