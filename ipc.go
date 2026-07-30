package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// projectJSON is the wire format for the ipc subcommand — the same fields
// Project already carries, just with LastCommitTime rendered as RFC3339
// instead of Go's time.Time, and NextSteps/MemoryNotes as the actual
// content an LLM would want rather than table-formatted cells.
type projectJSON struct {
	Name string `json:"name"`
	Path string `json:"path"`

	IsGit     bool `json:"is_git"`
	NoCommits bool `json:"no_commits,omitempty"`

	Branch      string `json:"branch,omitempty"`
	DirtyCount  int    `json:"dirty_count"`
	StashCount  int    `json:"stash_count,omitempty"`
	HasRemote   bool   `json:"has_remote"`
	RemoteURL   string `json:"remote_url,omitempty"`
	HasUpstream bool   `json:"has_upstream,omitempty"`
	Ahead       int    `json:"ahead,omitempty"`
	Behind      int    `json:"behind,omitempty"`

	LastCommitTime string `json:"last_commit_time,omitempty"`
	LastCommitMsg  string `json:"last_commit_msg,omitempty"`

	Description    string `json:"description,omitempty"`
	DescriptionSrc string `json:"description_src,omitempty"`

	MemoryNotes []string `json:"memory_notes,omitempty"`
	// NextSteps is the full "o que falta fazer" text (see readMemory), not
	// just its one-line hook in MemoryNotes — empty if the project has no
	// memory file tagged type: next-steps yet.
	NextSteps string `json:"next_steps,omitempty"`
}

func (p Project) toIPC() projectJSON {
	out := projectJSON{
		Name:           p.Name,
		Path:           p.Path,
		IsGit:          p.IsGit,
		NoCommits:      p.NoCommits,
		Branch:         p.Branch,
		DirtyCount:     p.DirtyCount,
		StashCount:     p.StashCount,
		HasRemote:      p.HasRemote,
		RemoteURL:      p.RemoteURL,
		HasUpstream:    p.HasUpstream,
		Ahead:          p.Ahead,
		Behind:         p.Behind,
		LastCommitMsg:  p.LastCommitMsg,
		Description:    p.Description,
		DescriptionSrc: p.DescriptionSrc,
		MemoryNotes:    p.MemoryNotes,
		NextSteps:      p.NextSteps,
	}
	if !p.LastCommitTime.IsZero() {
		out.LastCommitTime = p.LastCommitTime.Format(time.RFC3339)
	}
	return out
}

// runIPC implements `ccdi ipc <método> [key=value...] --json`, the same
// scriptable-data-source convention as dcal/djobs (`<bin> ipc <método>
// --json`) — meant for an LLM (or any script) to ask "what's left to do,
// where did I stop, what could I pick up next" across every tracked repo
// without going through the TUI.
func runIPC(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "uso: ccdi ipc <método> [key=value...] --json")
		return 1
	}

	method := args[0]
	filters := map[string]string{}
	jsonOut := false
	for _, arg := range args[1:] {
		if arg == "--json" {
			jsonOut = true
			continue
		}
		if k, v, ok := strings.Cut(arg, "="); ok {
			filters[k] = v
			continue
		}
		fmt.Fprintf(os.Stderr, "argumento inválido: %q (esperado key=value ou --json)\n", arg)
		return 1
	}
	if !jsonOut {
		fmt.Fprintln(os.Stderr, "apenas saída --json é suportada por enquanto")
		return 1
	}

	entries, cfgWarning := loadRootsConfig()
	projects, warnings := scanAll(entries)
	if cfgWarning != "" {
		warnings = append([]string{cfgWarning}, warnings...)
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "aviso:", w)
	}

	switch method {
	case "projects.list":
		return ipcProjectsList(projects, filters)
	case "projects.next":
		return ipcProjectsNext(projects)
	default:
		fmt.Fprintf(os.Stderr, "método desconhecido: %q\n", method)
		return 1
	}
}

func ipcProjectsList(projects []Project, filters map[string]string) int {
	out := make([]projectJSON, 0, len(projects))
	for _, p := range projects {
		if name, ok := filters["name"]; ok && p.Name != name {
			continue
		}
		if dirty, ok := filters["dirty"]; ok && (p.DirtyCount > 0) != (dirty == "true") {
			continue
		}
		out = append(out, p.toIPC())
	}
	return writeJSON(out)
}

// ipcProjectsNext returns the single project ccdi itself would put first —
// projects come back from scanAll already ordered mid-flight (dirty) work
// first, then most recently active, the same priority the TUI's sidebar
// shows top-to-bottom.
func ipcProjectsNext(projects []Project) int {
	if len(projects) == 0 {
		return writeJSON(nil)
	}
	return writeJSON(projects[0].toIPC())
}

func writeJSON(v any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "erro ao serializar json:", err)
		return 1
	}
	return 0
}
