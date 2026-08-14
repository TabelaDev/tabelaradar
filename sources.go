package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// activitySource is one feed the digest gathers from, per project. Gather
// returns human-readable lines describing the project's activity within the
// window; an error is downgraded to a warning by the digest, never fatal.
type activitySource interface {
	Name() string
	Gather(ctx context.Context, p Project, windowStart time.Time) ([]string, error)
}

// enabledSources builds the configured sources in a stable order (git first,
// then memory, then sessions), skipping the ones turned off in config.
func enabledSources(cfg digestSources) []activitySource {
	var out []activitySource
	if cfg.Git {
		out = append(out, gitSource{})
	}
	if cfg.ClaudeMemory {
		out = append(out, claudeMemorySource{})
	}
	if cfg.OpencodeSessions {
		out = append(out, &opencodeSessionsSource{max: 200})
	}
	return out
}

// gitSource reports commits since the window plus the repo state the scanner
// already read — the bread and butter: "what changed, what's mid-flight".
type gitSource struct{}

func (gitSource) Name() string { return "git" }

func (gitSource) Gather(_ context.Context, p Project, windowStart time.Time) ([]string, error) {
	if !p.IsGit {
		return nil, nil
	}
	var lines []string

	if out, err := runGit(p.Path, "log", "--since="+windowStart.Format("2006-01-02 15:04:05"),
		"--oneline", "-n", "50"); err == nil && strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(line) != "" {
				lines = append(lines, "commit "+line)
			}
		}
	}

	if p.DirtyCount > 0 {
		lines = append(lines, fmt.Sprintf("work in progress: %d arquivo(s) modificado(s) (branch %s)", p.DirtyCount, p.Branch))
	}
	if p.Ahead > 0 {
		lines = append(lines, fmt.Sprintf("%d commit(s) local(is) ainda não empurrados (branch %s)", p.Ahead, p.Branch))
	}
	if !p.LastCommitTime.IsZero() {
		lines = append(lines, fmt.Sprintf("último commit: %s (branch %s, %s)", p.LastCommitMsg, p.Branch, humanizeAgo(p.LastCommitTime)))
	}
	return lines, nil
}

// claudeMemorySource reuses the scanner's memory read — the "where did I
// stop" signal, already a Project field.
type claudeMemorySource struct{}

func (claudeMemorySource) Name() string { return "claude-memory" }

func (claudeMemorySource) Gather(_ context.Context, p Project, _ time.Time) ([]string, error) {
	notes, _, nextSteps := readMemory(p.Path)
	var lines []string
	for _, note := range notes {
		lines = append(lines, "memória: "+note)
	}
	if nextSteps != "" {
		first := strings.Split(strings.TrimSpace(nextSteps), "\n")[0]
		lines = append(lines, "próximos passos: "+first)
	}
	return lines, nil
}

// opencodeSessionsSource reads recent opencode sessions via the CLI
// (`opencode session list --format json`), which already carries each
// session's working directory and updated time — no direct sqlite access,
// no dependency on opencode's internal schema. Off by default.
//
// The session list is fetched once (lazily, on the first Gather) and reused
// for every project, so a board with N projects costs one CLI call, not N.
type opencodeSessionsSource struct {
	max int

	fetched bool
	entries []opencodeSession
}

type opencodeSession struct {
	Title     string `json:"title"`
	Updated   int64  `json:"updated"` // milliseconds since epoch
	Directory string `json:"directory"`
}

func (s *opencodeSessionsSource) Name() string { return "opencode-sessions" }

func (s *opencodeSessionsSource) Gather(ctx context.Context, p Project, windowStart time.Time) ([]string, error) {
	if !s.fetched {
		entries, err := fetchOpencodeSessions(ctx, s.max)
		if err != nil {
			return nil, err
		}
		s.entries = entries
		s.fetched = true
	}

	sinceMs := windowStart.UnixMilli()
	var lines []string
	for _, e := range s.entries {
		if e.Directory != p.Path || e.Updated < sinceMs {
			continue
		}
		when := ""
		if e.Updated > 0 {
			when = " · " + humanizeAgo(time.UnixMilli(e.Updated))
		}
		title := strings.TrimSpace(e.Title)
		if title == "" {
			title = "(sessão sem título)"
		}
		lines = append(lines, "sessão: "+title+when)
	}
	return lines, nil
}

// fetchOpencodeSessions runs `opencode session list --format json -n <max>`.
// No opencode on PATH is not an error — it just yields nothing.
func fetchOpencodeSessions(ctx context.Context, max int) ([]opencodeSession, error) {
	bin := "opencode"
	if _, err := exec.LookPath(bin); err != nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "session", "list", "--format", "json", "-n", strconv.Itoa(max))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("opencode session list: %v", err)
	}

	var sessions []opencodeSession
	if err := json.Unmarshal(out, &sessions); err != nil {
		return nil, fmt.Errorf("opencode session list: json: %v", err)
	}
	return sessions, nil
}
