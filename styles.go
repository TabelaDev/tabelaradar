package main

import (
	"strings"

	catppuccin "github.com/catppuccin/go"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Colors follow the official semantic guide:
// https://github.com/catppuccin/catppuccin/blob/main/docs/style-guide.md
var (
	mocha = catppuccin.Mocha

	colBase     = lipgloss.Color(mocha.Base().Hex)
	colMantle   = lipgloss.Color(mocha.Mantle().Hex)
	colSurface0 = lipgloss.Color(mocha.Surface0().Hex)
	colSurface1 = lipgloss.Color(mocha.Surface1().Hex)
	colOverlay0 = lipgloss.Color(mocha.Overlay0().Hex)
	colOverlay1 = lipgloss.Color(mocha.Overlay1().Hex)
	colText     = lipgloss.Color(mocha.Text().Hex)
	colSubtext0 = lipgloss.Color(mocha.Subtext0().Hex)
	// colPrimary mirrors the installed DankMaterialShell's own configured
	// accent (falling back to a manually chosen Catppuccin accent when DMS
	// isn't installed/configured) — see dmstheme.go. Same lookup djobs uses,
	// so the two tools' chrome matches whatever accent DMS is set to.
	colPrimary = lipgloss.Color(resolvePrimaryHex())
	colGreen   = lipgloss.Color(mocha.Green().Hex)
	colYellow  = lipgloss.Color(mocha.Yellow().Hex)
	colRed     = lipgloss.Color(mocha.Red().Hex)
	colBlue    = lipgloss.Color(mocha.Blue().Hex)
)

func headerStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(colMantle).
		Foreground(colPrimary).
		Bold(true).
		Width(width).
		Padding(0, 2)
}

func footerStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(colMantle).
		Foreground(colSubtext0).
		Width(width).
		Padding(0, 2)
}

// panelStyle intentionally has no Width(): calling Width() on a style makes
// lipgloss re-wrap its content, and that wrap logic miscounts lines that
// already carry their own nested ANSI (a table's selected-row highlight),
// breaking alignment. Content is pre-padded to a uniform width with
// padLines instead, so the border ends up sized correctly on its own.
func panelStyle(focused bool) lipgloss.Style {
	border := colSurface1
	if focused {
		border = colPrimary
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Background(colBase).
		Padding(0, 1)
}

// padLines pads or truncates every line of s so its ANSI-aware visible
// width is exactly `width` — a border box only ends up sized (and
// positioned) correctly if every line it wraps is uniform.
//
// The truncating side must use an ANSI-aware truncate (ansi.Truncate, not
// go-runewidth's): lines here carry nested SGR codes (table header/selected
// row colors), and a truncate that counts escape-sequence bytes as visible
// width cuts real content far too early, garbling it.
func padLines(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		switch w := lipgloss.Width(line); {
		case w < width:
			lines[i] = line + strings.Repeat(" ", width-w)
		case w > width:
			lines[i] = ansi.Truncate(line, width, "…")
		}
	}
	return strings.Join(lines, "\n")
}

// wrapText word-wraps plain text (no embedded ANSI — see panelStyle's
// comment on why Width()-based wrapping doesn't mix with already-styled
// content) to width, so long natural-language strings (descriptions, commit
// messages, memory notes) break onto new lines instead of being cut short
// with "…" by padLines.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}

// padToHeight pads s with blank lines until it has exactly `lines` lines (or
// truncates extra ones) — used for panels whose natural content is shorter
// than their computed box height, so a fixed-height neighbor panel's border
// still lines up with this one's bottom border.
func padToHeight(s string, lines int) string {
	got := strings.Split(s, "\n")
	for len(got) < lines {
		got = append(got, "")
	}
	if len(got) > lines {
		got = got[:lines]
	}
	return strings.Join(got, "\n")
}

func titleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(colPrimary).Background(colBase)
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colOverlay0).Background(colBase)
}

func wipStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colGreen).Bold(true)
}

func unpushedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colYellow).Bold(true)
}

func riskStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colRed).Bold(true)
}

func staleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colOverlay1)
}
