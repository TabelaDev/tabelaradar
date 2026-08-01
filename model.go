package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// panelFocus selects which of the two interactive panels — the sidebar
// project list, or the description panel — currently receives key input:
// vim-style ctrl+h/l switch between them. The stats panel (top-right) is
// display-only and never a focus target.
type panelFocus int

const (
	focusList panelFocus = iota
	focusDescription
)

// Vertical overhead, in lines: header + footer + each panel's own
// border/title, plus the sidebar's own bubbles/table header row.
const (
	headerLines        = 1
	footerLines        = 1
	sidebarBoxOverhead = 2 + 1 + 1 // border + title + bubbles/table header row
	statsBoxOverhead   = 2 + 1     // border + title
	descBoxOverhead    = 2 + 1     // border + title
	minVisibleRows     = 3
	minStatsLines      = 3
	minDescLines       = 4
	// sidebar:right-column WIDTH ratio, side by side.
	sidebarWidthShare = 1
	rightWidthShare   = 4
	// stats:descrição HEIGHT ratio, stacked in the right column.
	statsHeightShare = 1
	descHeightShare  = 4
	panelGap         = 1
	minSidebarWidth  = 14
	minRightWidth    = 30
)

type appModel struct {
	projects []Project
	tbl      table.Model
	focus    panelFocus
	// detailScroll is the first visible line of the current project's
	// description panel, adjusted by j/k while focus is on the description.
	detailScroll int

	width  int
	height int

	sidebarInnerWidth int
	rightInnerWidth   int
	statsLines        int
	descMaxLines      int

	status string
}

func newModel() appModel {
	m := appModel{}
	m.rescan()
	m.tbl = table.New(table.WithFocused(true))
	m.applyStyles()
	return m
}

func (m *appModel) rescan() {
	entries, cfgWarning := loadRootsConfig()
	projects, warnings := scanAll(entries)
	if cfgWarning != "" {
		warnings = append([]string{cfgWarning}, warnings...)
	}

	m.projects = projects
	m.status = fmt.Sprintf("%d projetos", len(projects))
	if len(warnings) > 0 {
		m.status += " — " + strings.Join(warnings, "; ")
	}
}

func (m *appModel) applyStyles() {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		Foreground(colSubtext0).
		Background(colMantle).
		Bold(true)
	styles.Selected = styles.Selected.
		Foreground(colBase).
		Background(colPrimary).
		Bold(true)
	// Cell intentionally has no Background: bubbles/table renders each cell
	// individually and only wraps the whole row in Selected afterwards, so a
	// background baked into every cell's own ANSI codes would nest inside —
	// and win over — Selected's background on the row, hiding the highlight
	// entirely. Leaving Cell transparent lets the panel's own background
	// (colBase) show through for both normal and selected rows.
	m.tbl.SetStyles(styles)
}

func (m appModel) Init() tea.Cmd { return nil }

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = sizeMsg.Width, sizeMsg.Height
		m.layout()
		return m, nil
	}
	if _, ok := msg.(editorFinishedMsg); ok {
		m.rescan()
		m.layout()
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m.forwardToTable(msg)
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r":
		m.rescan()
		m.layout()
		return m, nil
	case "o", "enter":
		if p := m.current(); p != nil {
			return m, openEditor(p.Path)
		}
		return m, nil
	// vim-style pane navigation: ctrl+h focuses the project list, ctrl+l
	// focuses the description panel so j/k scroll its text instead of
	// moving the list cursor.
	case "ctrl+h":
		m.focus = focusList
		return m, nil
	case "ctrl+l":
		m.focus = focusDescription
		return m, nil
	}

	if m.focus == focusDescription {
		switch keyMsg.String() {
		case "j", "down":
			if m.detailScroll < m.maxDetailScroll() {
				m.detailScroll++
			}
		case "k", "up":
			if m.detailScroll > 0 {
				m.detailScroll--
			}
		}
		return m, nil
	}

	return m.forwardToTable(msg)
}

// forwardToTable forwards to the sidebar's table, resetting detailScroll if
// that moved the cursor to a different project — the old scroll offset
// otherwise makes no sense against new description text.
func (m appModel) forwardToTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevIdx := m.tbl.Cursor()
	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	if m.tbl.Cursor() != prevIdx {
		m.detailScroll = 0
	}
	return m, cmd
}

func (m appModel) current() *Project {
	row := m.tbl.Cursor()
	if row < 0 || row >= len(m.projects) {
		return nil
	}
	return &m.projects[row]
}

type editorFinishedMsg struct{}

// openEditor suspends the TUI to run $EDITOR (falling back to nvim) against
// the selected project's root, the same "jump straight into it" shortcut
// lazygit-style tools give you once you've spotted what needs attention.
func openEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}
	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return editorFinishedMsg{} })
}

// layout recomputes the sidebar/stats/descrição panel widths and heights so
// the whole layout always fits exactly within m.height — the stats and
// descrição panels get a fixed line budget (statsLines/descMaxLines) that
// their content is padded or clipped to, instead of growing with whatever
// text happens to be in them. A panel emitting more lines than its budget
// is exactly what used to push the sidebar off the top of the screen for
// projects with a long description or many memory notes.
func (m *appModel) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	totalRowWidth := m.width - panelGap
	if minRow := (minSidebarWidth + 4) + (minRightWidth + 4); totalRowWidth < minRow {
		totalRowWidth = minRow
	}
	sidebarBoxWidth := totalRowWidth * sidebarWidthShare / (sidebarWidthShare + rightWidthShare)
	rightBoxWidth := totalRowWidth - sidebarBoxWidth

	m.sidebarInnerWidth = sidebarBoxWidth - 4
	if m.sidebarInnerWidth < minSidebarWidth {
		m.sidebarInnerWidth = minSidebarWidth
	}
	m.rightInnerWidth = rightBoxWidth - 4
	if m.rightInnerWidth < minRightWidth {
		m.rightInnerWidth = minRightWidth
	}

	// -2: bubbles/table's default Header/Cell styles each carry their own
	// Padding(0,1), added on top of the column's Width — the exact off-by-2
	// this project already hit once before with a 7-column table.
	nameColWidth := m.sidebarInnerWidth - 2
	if nameColWidth < 1 {
		nameColWidth = 1
	}
	m.tbl.SetColumns([]table.Column{{Title: "Projeto", Width: nameColWidth}})
	m.tbl.SetRows(sidebarRows(m.projects))
	m.tbl.SetWidth(m.sidebarInnerWidth)

	bodyHeight := m.height - headerLines - footerLines
	minBody := statsBoxOverhead + minStatsLines + descBoxOverhead + minDescLines
	if minSidebarBody := sidebarBoxOverhead + minVisibleRows; minSidebarBody > minBody {
		minBody = minSidebarBody
	}
	if bodyHeight < minBody {
		bodyHeight = minBody
	}

	statsBoxHeight := bodyHeight * statsHeightShare / (statsHeightShare + descHeightShare)
	if statsBoxHeight < statsBoxOverhead+minStatsLines {
		statsBoxHeight = statsBoxOverhead + minStatsLines
	}
	descBoxHeight := bodyHeight - statsBoxHeight
	if descBoxHeight < descBoxOverhead+minDescLines {
		descBoxHeight = descBoxOverhead + minDescLines
	}

	m.statsLines = statsBoxHeight - statsBoxOverhead
	m.descMaxLines = descBoxHeight - descBoxOverhead

	// The sidebar spans both right-column boxes stacked together, so its
	// row budget must match their combined (post-clamp) height exactly or
	// the borders won't line up at the bottom.
	sidebarRowsHeight := (statsBoxHeight + descBoxHeight) - sidebarBoxOverhead
	if sidebarRowsHeight < minVisibleRows {
		sidebarRowsHeight = minVisibleRows
	}
	m.tbl.SetHeight(sidebarRowsHeight)
}

func sidebarRows(projects []Project) []table.Row {
	rows := make([]table.Row, 0, len(projects))
	for _, p := range projects {
		rows = append(rows, table.Row{fmt.Sprintf("%s %s", statusGlyph(p), p.Name)})
	}
	return rows
}

func statusGlyph(p Project) string {
	switch {
	case !p.IsGit:
		return "○"
	case p.DirtyCount > 0:
		return "●"
	case p.Ahead > 0:
		return "▲"
	case p.IsGit && !p.HasRemote:
		return "✕"
	default:
		return "✓"
	}
}

func dirtyCell(p Project) string {
	if !p.IsGit || p.DirtyCount == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", p.DirtyCount)
}

func pushCell(p Project) string {
	if !p.IsGit || !p.HasUpstream {
		return "-"
	}
	if p.Ahead == 0 {
		return "-"
	}
	return fmt.Sprintf("+%d", p.Ahead)
}

func remoteCell(p Project) string {
	if !p.IsGit {
		return "n/a"
	}
	if p.HasRemote {
		return "sim"
	}
	return "não"
}

func humanizeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return "agora"
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh atrás", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd atrás", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dm atrás", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy atrás", int(d.Hours()/24/365))
	}
}

func (m appModel) View() string {
	if m.width == 0 {
		return ""
	}

	header := headerStyle(m.width).Render("TabelaRadar — comissão central de inspeção disciplinar dos seus projetos")

	footerText := "↑/↓ navegar · o/enter abrir editor · ctrl+h/l trocar painel · j/k rolar descrição · r reescanear · q sair    " + m.status
	// Must be pre-truncated: footerStyle sets Width(), which word-wraps
	// overflow instead of truncating it. An untruncated wrap silently turns
	// the footer into 2 lines, which nothing in layout() accounts for.
	if avail := m.width - 4; avail > 0 {
		footerText = strings.TrimRight(padLines(footerText, avail), " ")
	}
	footer := footerStyle(m.width).Render(footerText)

	sidebarBox := panelStyle(m.focus == focusList).Render(padLines(
		titleStyle().Render("Projetos")+"\n"+m.tbl.View(), m.sidebarInnerWidth,
	))

	statsTitle := "status"
	if p := m.current(); p != nil {
		statsTitle = p.Name
	}
	statsBox := panelStyle(false).Render(padLines(
		titleStyle().Render(statsTitle)+"\n"+padToHeight(m.renderStats(), m.statsLines), m.rightInnerWidth,
	))

	descLines := m.currentDescLines()
	descTitle := "descrição"
	if total := len(descLines); total > m.descMaxLines {
		descTitle = fmt.Sprintf("descrição (%d–%d/%d)", m.detailScroll+1, min(m.detailScroll+m.descMaxLines, total), total)
	}
	descBox := panelStyle(m.focus == focusDescription).Render(padLines(
		titleStyle().Render(descTitle)+"\n"+m.renderDescBody(descLines), m.rightInnerWidth,
	))

	rightCol := lipgloss.JoinVertical(lipgloss.Left, statsBox, descBox)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, strings.Repeat(" ", panelGap), rightCol)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// renderStats is the compact top-right panel: the at-a-glance columns that
// used to live in the sidebar's table (branch/remoto dropped for now, per
// request — more fields can join sujo/push/atividade here later).
func (m appModel) renderStats() string {
	p := m.current()
	if p == nil {
		return dimStyle().Render("nenhum projeto")
	}
	return strings.Join([]string{
		fmt.Sprintf("Sujo: %s", dirtyCell(*p)),
		fmt.Sprintf("Push: %s", pushCell(*p)),
		fmt.Sprintf("Atividade: %s", humanizeAgo(p.LastCommitTime)),
	}, "\n")
}

// currentDescLines is the full (unscrolled, unclipped) descrição text for
// current(), wrapped to the panel's width and split into lines — shared by
// renderDescBody, maxDetailScroll, and the title's scroll-position indicator
// so they never disagree about line count.
func (m appModel) currentDescLines() []string {
	p := m.current()
	if p == nil {
		return []string{dimStyle().Render("nenhum projeto")}
	}

	width := m.rightInnerWidth
	var b strings.Builder
	b.WriteString(dimStyle().Render(p.Path) + "\n\n")

	switch {
	case !p.IsGit:
		b.WriteString(riskStyle().Render("sem repositório git") + "\n")
	case p.NoCommits:
		b.WriteString(riskStyle().Render("repositório sem nenhum commit ainda") + "\n")
	default:
		b.WriteString(wrapText(fmt.Sprintf("último commit: %s", p.LastCommitMsg), width) + "\n")
	}
	if p.IsGit {
		if p.StashCount > 0 {
			b.WriteString(fmt.Sprintf("%d stash(es)\n", p.StashCount))
		}
		if !p.HasRemote {
			b.WriteString(riskStyle().Render("sem remote configurado — sem backup fora desta máquina") + "\n")
		}
	}

	if p.Description != "" {
		b.WriteString("\n" + dimStyle().Render(p.DescriptionSrc) + ":\n" + wrapText(p.Description, width) + "\n")
	}

	if len(p.MemoryNotes) > 0 {
		b.WriteString("\n" + dimStyle().Render("memória do Claude Code:") + "\n")
		for _, note := range p.MemoryNotes {
			b.WriteString(wrapText("• "+note, width) + "\n")
		}
	}

	return strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
}

// maxDetailScroll is the highest detailScroll that still leaves the last
// line visible — scrolling past it would just show trailing blank space.
func (m appModel) maxDetailScroll() int {
	if n := len(m.currentDescLines()) - m.descMaxLines; n > 0 {
		return n
	}
	return 0
}

// renderDescBody clips lines to the panel's fixed descMaxLines budget,
// starting at detailScroll — this is what actually keeps the descrição
// panel's rendered height constant regardless of content length.
func (m appModel) renderDescBody(lines []string) string {
	scroll := m.detailScroll
	if max := m.maxDetailScroll(); scroll > max {
		scroll = max
	}
	end := scroll + m.descMaxLines
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[scroll:end], "\n")
}
