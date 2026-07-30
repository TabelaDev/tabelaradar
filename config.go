package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// rootEntry is one line of the config file: a path to monitor, or (if
// Exclude) a path to hide from whatever root it's nested under.
type rootEntry struct {
	Path    string
	Exclude bool
}

// configPath is the settings file listing which repos/repo-groups to
// monitor. CCDI_CONFIG overrides it, same envOr pattern as CCDI_ROOT.
var configPath = envOr("CCDI_CONFIG", filepath.Join(configDir(), "ccdi", "config"))

func configDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return dir
	}
	return filepath.Join(homeDir, ".config")
}

// defaultConfig preserves ccdi's original behavior (scan CCDI_ROOT, or
// ~/codigo/pessoal if unset) for anyone who hasn't written a config file yet.
func defaultConfig() []rootEntry {
	return []rootEntry{{Path: envOr("CCDI_ROOT", filepath.Join(homeDir, "codigo", "pessoal"))}}
}

// loadRootsConfig reads configPath, one entry per line:
//
//	~/codigo/pessoal          # a repo-group root: each child dir is a project
//	~/codigo/cogu             # or a single repo root, monitored on its own
//	!~/codigo/pessoal/spotdash  # exclude this path from whatever root it's under
//
// Blank lines and lines starting with "#" are ignored. Returns a non-empty
// warning string (instead of an error) when the file exists but couldn't be
// read in full — scanning still proceeds with whatever roots were parsed
// before the failure, or the default root if none were.
func loadRootsConfig() ([]rootEntry, string) {
	f, err := os.Open(configPath)
	if err != nil {
		return defaultConfig(), ""
	}
	defer f.Close()

	var entries []rootEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		exclude := strings.HasPrefix(line, "!")
		line = strings.TrimPrefix(line, "!")
		entries = append(entries, rootEntry{Path: expandHome(strings.TrimSpace(line)), Exclude: exclude})
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Sprintf("erro lendo %s: %v", configPath, err)
	}
	if len(entries) == 0 {
		return defaultConfig(), ""
	}
	return entries, ""
}

func expandHome(path string) string {
	if path == "~" {
		return homeDir
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
