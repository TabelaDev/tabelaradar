package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ianptkcs/tabelatuiui"
)

// rootEntry is one monitored path: a repo, a repo-group root, or (if Exclude)
// a path to hide from whatever root it's nested under.
type rootEntry struct {
	Path    string
	Exclude bool
}

// config is tabelaradar's settings schema, read from
// ~/.config/tabelaradar/config.toml. Every field falls back to defaultConfig
// when the file leaves it out.
type config struct {
	// Roots are the paths to monitor: a repo-group root whose child dirs each
	// become a project, or a single repo monitored on its own.
	Roots []string `toml:"roots"`
	// Exclude hides a path from whatever root it sits under. Order does not
	// matter — scanAll builds the exclusion set before walking.
	Exclude []string      `toml:"exclude"`
	Scanner scannerConfig `toml:"scanner"`
	Layout  layoutConfig  `toml:"layout"`
	General generalConfig `toml:"general"`
}

type scannerConfig struct {
	// DescriptionFiles is the priority order for a project's "what is this"
	// blurb. Whatever is listed first and exists wins; a *.md glob is the
	// fallback when none match.
	DescriptionFiles  []string `toml:"description_files"`
	ClaudeProjectsDir string   `toml:"claude_projects_dir"`
}

type layoutConfig struct {
	// Width ratio between the sidebar and the right column, side by side.
	SidebarWidthShare int `toml:"sidebar_width_share"`
	RightWidthShare   int `toml:"right_width_share"`
	// Height ratio between stats and description, stacked in the right column.
	StatsHeightShare int `toml:"stats_height_share"`
	DescHeightShare  int `toml:"desc_height_share"`
}

type generalConfig struct {
	// Editor overrides $EDITOR for the "open" key.
	Editor string `toml:"editor"`
}

func defaultConfig() config {
	return config{
		Roots:   []string{tuiui.EnvOr("TABELARADAR_ROOT", filepath.Join(tuiui.HomeDir(), "codigo", "pessoal"))},
		Exclude: nil,
		Scanner: scannerConfig{
			DescriptionFiles:  []string{"README.md", "PLANNING.md", "ESCOPO.md", "STACK.md", "TODO.md", "CLAUDE.md"},
			ClaudeProjectsDir: filepath.Join(tuiui.HomeDir(), ".claude", "projects"),
		},
		Layout: layoutConfig{
			SidebarWidthShare: 1,
			RightWidthShare:   4,
			StatsHeightShare:  1,
			DescHeightShare:   4,
		},
		General: generalConfig{Editor: ""}, // empty = fall back to $EDITOR, then nvim
	}
}

// configPath is resolved lazily, not in a package-level var: an init-time var
// would freeze TABELARADAR_CONFIG/XDG_CONFIG_HOME before main (or a test)
// could set them.
func configPath() string {
	return tuiui.EnvOr("TABELARADAR_CONFIG", tuiui.ConfigPath("tabelaradar", "config.toml"))
}

// legacyConfigPath is the pre-TOML file: one path per line, "!" prefixing an
// exclusion. Still read when no config.toml exists yet, so an existing
// install keeps working untouched.
func legacyConfigPath() string {
	return tuiui.ConfigPath("tabelaradar", "config")
}

var cfg *tuiui.Config[config]

// settings is the normalized snapshot the app reads from.
var settings = defaultConfig()

// normalize clamps values the layout cannot survive. A share of zero would
// collapse a panel to nothing and divide by zero in the ratio math.
func normalize(c config) config {
	d := defaultConfig()
	if c.Layout.SidebarWidthShare < 1 {
		c.Layout.SidebarWidthShare = d.Layout.SidebarWidthShare
	}
	if c.Layout.RightWidthShare < 1 {
		c.Layout.RightWidthShare = d.Layout.RightWidthShare
	}
	if c.Layout.StatsHeightShare < 1 {
		c.Layout.StatsHeightShare = d.Layout.StatsHeightShare
	}
	if c.Layout.DescHeightShare < 1 {
		c.Layout.DescHeightShare = d.Layout.DescHeightShare
	}
	if len(c.Scanner.DescriptionFiles) == 0 {
		c.Scanner.DescriptionFiles = d.Scanner.DescriptionFiles
	}
	if c.Scanner.ClaudeProjectsDir == "" {
		c.Scanner.ClaudeProjectsDir = d.Scanner.ClaudeProjectsDir
	}
	c.Scanner.ClaudeProjectsDir = tuiui.ExpandHome(c.Scanner.ClaudeProjectsDir)
	if len(c.Roots) == 0 {
		c.Roots = d.Roots
	}
	return c
}

// refreshSettings re-reads config.toml from disk and returns a warning string
// (never an error) — a bad config file must not stop the scan, same contract
// the old line-based loader had.
func refreshSettings() string {
	if cfg == nil {
		cfg = tuiui.NewConfig(configPath(), defaultConfig())
	}

	// No config.toml yet: fall back to the pre-TOML file if it's still there,
	// so an existing install keeps its roots after upgrading.
	if _, err := os.Stat(configPath()); os.IsNotExist(err) {
		if legacy, ok := loadLegacyConfig(); ok {
			settings = normalize(legacy)
			return fmt.Sprintf("lendo o config antigo %s — migre pra %s", legacyConfigPath(), configPath())
		}
	}

	_, err := cfg.Reload()
	settings = normalize(cfg.Get())
	if err != nil {
		return fmt.Sprintf("erro lendo %s: %v", configPath(), err)
	}
	return ""
}

// loadLegacyConfig reads the pre-TOML format: one path per line, "#" for
// comments, "!" prefixing an exclusion. Reports false when the file is absent
// or has no usable entries.
func loadLegacyConfig() (config, bool) {
	f, err := os.Open(legacyConfigPath())
	if err != nil {
		return config{}, false
	}
	defer f.Close()

	c := defaultConfig()
	c.Roots = nil
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if path := strings.TrimSpace(strings.TrimPrefix(line, "!")); strings.HasPrefix(line, "!") {
			c.Exclude = append(c.Exclude, path)
		} else {
			c.Roots = append(c.Roots, path)
		}
	}
	if scanner.Err() != nil || len(c.Roots) == 0 {
		return config{}, false
	}
	return c, true
}

// rootEntries flattens the configured roots and exclusions into the flat list
// scanAll walks, expanding "~" on the way.
func rootEntries() []rootEntry {
	entries := make([]rootEntry, 0, len(settings.Roots)+len(settings.Exclude))
	for _, p := range settings.Roots {
		entries = append(entries, rootEntry{Path: tuiui.ExpandHome(p)})
	}
	for _, p := range settings.Exclude {
		entries = append(entries, rootEntry{Path: tuiui.ExpandHome(p), Exclude: true})
	}
	return entries
}

// loadRootsConfig re-reads the config file and returns the roots to scan plus
// a warning string. It re-reads on every call by design: "r" (rescan) has
// always picked up an external edit to the roots without a restart.
func loadRootsConfig() ([]rootEntry, string) {
	warn := refreshSettings()
	return rootEntries(), warn
}
