<div align="center">

# TabelaRadar

**A TUI that audits the git health of your local repositories** — WIP, unpushed
commits, repos with no remote, projects left alone for too long.

**English** · [Português](README.pt-BR.md)

[![Go Version](https://img.shields.io/github/go-mod/go-version/TabelaDev/tabelaradar?style=flat-square&logo=go&logoColor=white&color=00ADD8)](go.mod)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)](LICENSE)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Powered by tabelatuiui](https://img.shields.io/badge/theme-tabelatuiui-d6b4f7?style=flat-square)](https://github.com/TabelaDev/tabelatuiui)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/ianptkcs)

</div>

---

## What it is

A [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI that scans the
repositories and folders-of-repositories listed in the settings (by default only
`~/codigo/pessoal`) and audits the state of each project: what is in WIP, what has
unpushed commits, what has no remote configured (and therefore no backup off the
machine) and what has not been touched in a while — a disciplinary inspection of
your repositories.

There is no separate metadata file — the whole state is inferred from each
repository's own git, plus the first paragraph of prose from its
`README.md`/`PLANNING.md`/`ESCOPO.md`/`STACK.md`/`TODO.md`/`CLAUDE.md` (whichever
exists first), plus the bullets from that project's Claude Code memory index
(`~/.claude/projects/<slug>/memory/MEMORY.md`), when a session has run in there.

The theme and the shared chrome (header/footer/panels, ANSI-aware padding, the
`ipc ... --json` helpers) come from
[`tabelatuiui`](https://github.com/TabelaDev/tabelatuiui), the shared UI library
of my Bubble Tea TUIs.

## Contents

- [Installation](#installation)
- [Layout](#layout)
- [Usage](#usage)
- [IPC](#ipc)
- [Configuration](#configuration)
- [License](#license)

## Installation

Requires Go 1.26+.

```bash
go install github.com/ianptkcs/tabelaradar@latest
```

Or building from source:

```bash
git clone https://github.com/TabelaDev/tabelaradar.git
cd tabelaradar
go build -o tabelaradar .
```

## Layout

Three panels:

- **Projects** (left, 1/5 of the width) — names only, to move quickly through the
  whole list. The glyph before the name sums up the status: `○` no git, `●`
  uncommitted changes, `▲` unpushed commits, `✕` commits but no remote, `✓` clean
  and in sync.
- **status** (top right, 1/5 of the height) — dirty (uncommitted files), push
  (unpushed commits) and activity (how long since the last commit) for the
  selected project.
- **description** (below, 4/5 of the height) — everything else: path, last commit,
  warnings (no remote, stash), the description extracted from
  README/PLANNING/etc. and the bullets from that project's Claude Code memory.
  When the content does not fit, the title shows the visible range
  (`description (1–13/27)`) and it scrolls.

## Usage

```
tabelaradar         # opens the TUI
tabelaradar list    # plain-text dump, no TTY — useful for scripting
```

Inside the TUI: `↑`/`↓` (or `j`/`k`) move through the project list,
`ctrl+h`/`ctrl+l` switch between the projects panel and the description one,
`j`/`k` (or `↑`/`↓`) scroll the description text while it is focused, `o`/`enter`
opens the selected project in `$EDITOR` (`nvim` by default), `r` rescans and `q`
quits.

## IPC

For scripts, or for an LLM to ask "what is left to do, where did I stop in each
project, what could be started" without opening the TUI, `tabelaradar` exposes a
non-interactive `ipc` subcommand, in the same spirit as
`dcal ipc <method> --json`/`djobs ipc <method> --json`:

```bash
tabelaradar ipc projects.list --json                  # every tracked project, with git status + description + next steps
tabelaradar ipc projects.list dirty=true --json       # only those with uncommitted changes
tabelaradar ipc projects.list name=tabelacal --json   # one specific project
tabelaradar ipc projects.next --json                  # the project tabelaradar itself would prioritise (mid-flight > most recent)
```

Beyond the git status fields (branch, dirty, ahead/behind, last commit), each
project in the JSON carries `description` (extracted from README/PLANNING/etc.),
`memory_notes` (the one-line hooks from the memory index) and `next_steps` — the
**entire** body (not just the truncated hook) of any memory of that project marked
`type: next-steps` in its own `~/.claude/projects/<slug>/memory/`, empty when the
project does not have one yet.

## Configuration

Everything lives in `~/.config/tabelaradar/config.toml` (overridable through
`TABELARADAR_CONFIG`). The file is optional and partial: only the keys present
override anything, the rest stay on their defaults. `f5` reloads without
restarting.

```toml
roots = ["~/codigo/pessoal", "~/codigo/tabeladev"]
exclude = ["~/codigo/pessoal/spotdash"]

[scanner]
description_files = ["README.md", "PLANNING.md", "ESCOPO.md", "STACK.md", "TODO.md", "CLAUDE.md"]
claude_projects_dir = "~/.claude/projects"

[layout]
sidebar_width_share = 1  # WIDTH ratio sidebar:right-column
right_width_share   = 4
stats_height_share  = 1  # HEIGHT ratio stats:description
desc_height_share   = 4

[general]
editor = "nvim"  # empty = use $EDITOR, then nvim
```

### Which repos to watch

`roots` accepts two kinds of path:

- a folder-of-repos: every subfolder with a `.git` becomes a row in the table
  (this is how `~/codigo/pessoal` works);
- a git repository directly: it enters on its own, as a single row (useful for a
  loose repo outside the usual folders).

`exclude` hides a specific path — whether a whole root or a child of a root listed
in `roots`. Order does not matter between the two lists.

With no file at all, it scans only `TABELARADAR_ROOT` (or `~/codigo/pessoal`).

### Migrating from the old format

Before 0.3.0 the config was `~/.config/tabelaradar/config`, one entry per line with
`!` prefixing exclusions. **That file is still read** when no `config.toml` exists,
with a warning on the status bar. The translation is direct:

```
~/codigo/pessoal              →  roots   = ["~/codigo/pessoal"]
!~/codigo/pessoal/spotdash    →  exclude = ["~/codigo/pessoal/spotdash"]
```

Once `config.toml` exists, it takes over on its own — the two formats do not mix,
and the old file can be deleted.

### Other variables

- `TABELARADAR_CONFIG` — path to `config.toml`, when it is not the default.
- `TABELARADAR_ROOT` — the directory scanned when no config exists at all
  (`~/codigo/pessoal` by default).
- `TABELARADAR_ACCENT` — a manual Catppuccin Mocha accent, used only when
  DankMaterialShell is not installed or configured (`mauve` by default).
- `TABELARADAR_DMS_SETTINGS` — path to the DMS `settings.json`, when it is not the
  default.

## Development

```bash
go test ./...
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for the version history.

## Support the project

- **Global**: [ko-fi.com/ianptkcs](https://ko-fi.com/ianptkcs)
- **Brazil (Pix)**: scan the QR below or copy the code

  <img src="pix-qr.png" alt="Pix QR" width="200" />

  <details><summary>Pix code (copy)</summary>

  ```
  00020126580014BR.GOV.BCB.PIX01365ad933b0-dcdc-4525-a736-0759902aeec65204000053039865802BR5925Ian Patrick da Costa Soar6009SAO PAULO62140510tQA85x6Dov63041FB6
  ```

  </details>

## License

[GNU AGPL-3.0](LICENSE) — free and open source. If you run a modified version of
this project, including as a network service, you also have to make the modified
source available under the same license.
