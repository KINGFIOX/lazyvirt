package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Layout: top row = domain list (55% width) + detail panel; below it the
// Command Log panel (command echo + command output); last line is the
// keybinding help.
// Widths/heights received by the render functions are outer sizes including
// the border; each function subtracts 2 itself.
const (
	commandLogHeight = 8
	listWidthPct     = 55

	colNameW  = 18
	colStateW = 14
)

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}
	if m.confirm != confirmNone {
		return m.confirmView()
	}

	help := m.helpLine()
	outputH := commandLogHeight
	topH := m.height - outputH - 1 // - help line
	if topH < 5 {
		topH = 5
	}

	listW := m.width * listWidthPct / 100
	detailW := m.width - listW
	if detailW < 24 {
		detailW = 24
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top,
			panelStyle.Render(m.listView(listW, topH)),
			panelStyle.Render(m.detailView(detailW, topH)),
		),
		panelStyle.Render(m.commandLogView(m.width, outputH)),
		help,
	)
}

// listView renders the domain list: title + header + rows inside a scrolling
// window. Alignment rule: pad/truncate plain text first, color afterwards, so
// ANSI escapes are never counted into the column widths.
func (m Model) listView(w, h int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Domains (%d)", len(m.domains))))
	b.WriteByte('\n')
	b.WriteString(dimStyle.Render(fmt.Sprintf("   %-*s %-*s %s",
		colNameW, "NAME", colStateW, "STATE", "AUTOSTART")))
	b.WriteByte('\n')

	if m.err != nil {
		b.WriteString(errStyle.Render(wordWrap(m.err.Error(), w-2)))
		return b.String()
	}
	if len(m.domains) == 0 {
		b.WriteString(dimStyle.Render("(no domains)"))
		return b.String()
	}

	visible := h - 4 // title + header + top/bottom border
	if visible < 1 {
		visible = 1
	}
	offset := m.scrollOffset(visible)

	last := offset + visible - 1
	if last >= len(m.domains) {
		last = len(m.domains) - 1
	}
	for i := offset; i <= last; i++ {
		d := m.domains[i]
		marker := " "
		if i == m.cursor {
			marker = "▸"
		}
		auto := d.Autostart
		// truncate the autostart column with whatever width is left, keeping
		// the whole row inside the panel
		if spare := w - 2 - (2 + 1 + colNameW + 1 + colStateW + 1); spare >= 0 && len(auto) > spare {
			auto = auto[:spare]
		}
		row := marker + " " +
			fmt.Sprintf("%-*.*s", colNameW, colNameW, d.Name) + " " +
			stateStyle(d.State).Render(fmt.Sprintf("%-*.*s", colStateW, colStateW, d.State)) + " " +
			auto
		if i == m.cursor {
			row = selectedRowStyle.Render(row)
		}
		b.WriteString(row)
		if i != last {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// scrollOffset keeps the cursor row visible.
func (m Model) scrollOffset(visible int) int {
	if m.cursor >= visible {
		return m.cursor - visible + 1
	}
	return 0
}

// detailView renders the dominfo of the selected domain on the right.
func (m Model) detailView(w, h int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("dominfo"))
	b.WriteByte('\n')

	sel, ok := m.selected()
	if !ok || m.detailName != sel.Name || len(m.detail.Lines) == 0 {
		return b.String() + dimStyle.Render("…")
	}

	if m.detailName != sel.Name || len(m.detail.Lines) == 0 {
		return b.String() + dimStyle.Render("…")
	}

	rows := h - 3 // border + title
	if rows < 1 {
		rows = 1
	}
	for i, l := range m.detail.Lines {
		if i == rows-1 && len(m.detail.Lines) > rows {
			b.WriteString(dimStyle.Render("…"))
			break
		}
		if i >= rows {
			break
		}
		b.WriteString(fmt.Sprintf("%s %s",
			dimStyle.Render(fmt.Sprintf("%-*.*s", 14, 14, l.Key)), l.Val))
		if i != rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// commandLogView renders the Command Log: each entry is the echoed virsh
// command line, followed by its output (or error) on the next line.
func (m Model) commandLogView(w, h int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Command Log"))
	b.WriteByte('\n')

	if len(m.logs) == 0 {
		return b.String() + dimStyle.Render("(no commands yet)")
	}

	rows := h - 3 // border + title
	if rows < 2 {
		rows = 2
	}

	// Fill from the newest entry backwards until the panel is full, keeping
	// entries intact (echo + result line), then flip to chronological order.
	var lines []string
	for i := len(m.logs) - 1; i >= 0; i-- {
		if len(lines)+2 > rows {
			break
		}
		e := m.logs[i]
		echo := dimStyle.Render(e.when.Format("15:04:05")) + "  " +
			cmdStyle.Render("$ "+truncate(e.Cmdline, w-16))
		lines = append([]string{echo, logResultLine(e, w)}, lines...)
	}
	return b.String() + strings.Join(lines, "\n")
}

// logResultLine renders the second line of an entry: the first line of the
// command output, or the error message.
func logResultLine(e logEntry, w int) string {
	const indentW = 11 // width of "15:04:05" + two spaces
	indent := strings.Repeat(" ", indentW)
	if e.Err != nil {
		return indent + errStyle.Render("error: "+truncate(firstLine(e.Err.Error()), w-4-indentW))
	}
	out := firstLine(strings.TrimSpace(e.Output))
	if out == "" {
		out = "ok"
	}
	return indent + okStyle.Render(truncate(out, w-4-indentW))
}

func (m Model) helpLine() string {
	pairs := [][2]string{
		{"j/k", "select"}, {"s", "start"}, {"S", "shutdown"}, {"r", "reboot"},
		{"x", "force off"}, {"a", "autostart"}, {"d", "undefine"}, {"U", "del + disks"},
		{"R", "refresh"}, {"q", "quit"},
	}
	var parts []string
	for _, p := range pairs {
		parts = append(parts, keyStyle.Render(p[0])+" "+p[1])
	}
	return " " + strings.Join(parts, dimStyle.Render("  •  "))
}

// confirmView renders the full-screen centered confirmation dialog.
func (m Model) confirmView() string {
	var title, warn string
	switch m.confirm {
	case confirmDestroy:
		title = "destroy " + m.confirmName
		warn = "Hard power-off, like pulling the plug. Unsaved data will be lost."
	case confirmUndefine:
		title = "undefine " + m.confirmName
		warn = "Removes the domain definition (disk files are kept)."
	case confirmUndefineAll:
		title = "undefine --nvram --remove-all-storage " + m.confirmName
		warn = "DANGER: deletes the definition, UEFI NVRAM and ALL associated disk files. Irreversible!"
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		modalTitleStyle.Render(title),
		"",
		warn,
		"",
		keyStyle.Render("y")+" confirm    "+keyStyle.Render("any other key")+" cancel",
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		modalStyle.Render(body))
}

// firstLine returns everything before the first newline (CR stripped).
func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// truncate cuts to the given display width; logs may contain CJK, so the
// rune count is a close-enough approximation.
func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func wordWrap(s string, w int) string {
	if w <= 0 {
		return s
	}
	var lines []string
	for _, seg := range strings.Split(s, "\n") {
		for len([]rune(seg)) > w {
			lines = append(lines, string([]rune(seg)[:w]))
			seg = string([]rune(seg)[w:])
		}
		lines = append(lines, seg)
	}
	return strings.Join(lines, "\n")
}
