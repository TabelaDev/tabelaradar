package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configDir points XDG_CONFIG_HOME at a temp dir and returns
// <tmp>/tabelaradar, where both the legacy "config" and the new "config.toml"
// live. It also resets the package-level cfg so each test loads fresh.
func configDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("TABELARADAR_CONFIG", "")
	t.Setenv("TABELARADAR_ROOT", "")
	cfg = nil
	settings = defaultConfig()

	dir := filepath.Join(base, "tabelaradar")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// paths flattens rootEntries into two lists for easy comparison.
func paths(entries []rootEntry) (roots, excluded []string) {
	for _, e := range entries {
		if e.Exclude {
			excluded = append(excluded, e.Path)
		} else {
			roots = append(roots, e.Path)
		}
	}
	return roots, excluded
}

func TestLoadRootsConfigTOML(t *testing.T) {
	dir := configDir(t)
	write(t, filepath.Join(dir, "config.toml"), `
roots = ["/tmp/a", "/tmp/b"]
exclude = ["/tmp/a/skip"]
`)

	entries, warn := loadRootsConfig()
	if warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	roots, excluded := paths(entries)
	if len(roots) != 2 || roots[0] != "/tmp/a" || roots[1] != "/tmp/b" {
		t.Fatalf("roots = %v, want [/tmp/a /tmp/b]", roots)
	}
	if len(excluded) != 1 || excluded[0] != "/tmp/a/skip" {
		t.Fatalf("excluded = %v, want [/tmp/a/skip]", excluded)
	}
}

// The pre-TOML file must keep working after an upgrade: this is the only app
// of the batch with a config file that actually exists on users' disks.
func TestLoadRootsConfigFallsBackToLegacyFile(t *testing.T) {
	dir := configDir(t)
	write(t, filepath.Join(dir, "config"), `
# comentário
/tmp/pessoal
/tmp/tabeladev
!/tmp/pessoal/sigapp
`)

	entries, warn := loadRootsConfig()
	if !strings.Contains(warn, "config antigo") {
		t.Fatalf("warning = %q, want a migration hint", warn)
	}
	roots, excluded := paths(entries)
	if len(roots) != 2 || roots[0] != "/tmp/pessoal" || roots[1] != "/tmp/tabeladev" {
		t.Fatalf("roots = %v, want the 2 legacy roots", roots)
	}
	if len(excluded) != 1 || excluded[0] != "/tmp/pessoal/sigapp" {
		t.Fatalf("excluded = %v, want [/tmp/pessoal/sigapp]", excluded)
	}
}

// Once config.toml exists it wins outright — no merging of the two formats.
func TestTOMLTakesPrecedenceOverLegacy(t *testing.T) {
	dir := configDir(t)
	write(t, filepath.Join(dir, "config"), "/tmp/legacy\n")
	write(t, filepath.Join(dir, "config.toml"), `roots = ["/tmp/novo"]`+"\n")

	entries, warn := loadRootsConfig()
	if warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	roots, _ := paths(entries)
	if len(roots) != 1 || roots[0] != "/tmp/novo" {
		t.Fatalf("roots = %v, want [/tmp/novo]", roots)
	}
}

func TestLoadRootsConfigNoFilesUsesDefault(t *testing.T) {
	configDir(t)

	entries, warn := loadRootsConfig()
	if warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	roots, _ := paths(entries)
	if len(roots) != 1 {
		t.Fatalf("roots = %v, want a single default root", roots)
	}
	if !strings.HasSuffix(roots[0], filepath.Join("codigo", "pessoal")) {
		t.Fatalf("default root = %q, want ~/codigo/pessoal", roots[0])
	}
}

func TestMalformedTOMLWarnsAndKeepsScanning(t *testing.T) {
	dir := configDir(t)
	write(t, filepath.Join(dir, "config.toml"), "roots = [\nbroken")

	entries, warn := loadRootsConfig()
	if warn == "" {
		t.Fatal("warning = empty, want a parse error surfaced to the status bar")
	}
	// A bad config must not blank the scan — it degrades to defaults.
	if roots, _ := paths(entries); len(roots) == 0 {
		t.Fatal("roots = empty, want the defaults so the scan still runs")
	}
}

func TestOnlyKeysPresentOverrideDefaults(t *testing.T) {
	dir := configDir(t)
	write(t, filepath.Join(dir, "config.toml"), "[layout]\nsidebar_width_share = 2\n")

	if _, warn := loadRootsConfig(); warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	if settings.Layout.SidebarWidthShare != 2 {
		t.Fatalf("SidebarWidthShare = %d, want 2", settings.Layout.SidebarWidthShare)
	}
	if settings.Layout.RightWidthShare != 4 {
		t.Fatalf("RightWidthShare = %d, want the default 4", settings.Layout.RightWidthShare)
	}
	if len(settings.Scanner.DescriptionFiles) != 6 {
		t.Fatalf("DescriptionFiles = %v, want the 6 defaults", settings.Scanner.DescriptionFiles)
	}
}

func TestNormalizeRejectsZeroShares(t *testing.T) {
	// A zero share would divide by zero in the width/height ratio math.
	got := normalize(config{Layout: layoutConfig{SidebarWidthShare: 0, RightWidthShare: 0}})
	if got.Layout.SidebarWidthShare != 1 || got.Layout.RightWidthShare != 4 {
		t.Fatalf("layout = %+v, want the defaults restored", got.Layout)
	}
	if got.Layout.StatsHeightShare != 1 || got.Layout.DescHeightShare != 4 {
		t.Fatalf("height shares = %+v, want the defaults restored", got.Layout)
	}
}

func TestExpandHomeAppliedToRoots(t *testing.T) {
	dir := configDir(t)
	write(t, filepath.Join(dir, "config.toml"), `roots = ["~/codigo/x"]`+"\n")

	entries, _ := loadRootsConfig()
	roots, _ := paths(entries)
	if len(roots) != 1 || strings.HasPrefix(roots[0], "~") {
		t.Fatalf("roots = %v, want the ~ expanded", roots)
	}
}
