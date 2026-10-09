// Package tui is lazyvirt's terminal UI: the bubbletea Model/Update/View
// trio, split by file into model (state & messages) / update (keys &
// commands) / view (rendering).
package tui

import (
	"time"

	"github.com/charmbracelet/bubbletea"

	"lazyvirt/internal/virsh"
)

const refreshInterval = 5 * time.Second

type logLine struct {
	when time.Time
	text string
	err  bool
}

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmDestroy
	confirmUndefine
	confirmUndefineAll
)

// Model holds all UI state; the list remembers the selection by name and
// restores it after refreshes.
type Model struct {
	domains []virsh.Domain
	cursor  int

	detailName string // domain the detail panel refers to (guards against stale replies)
	detail     virsh.Detail

	err  error // list refresh failure (virsh unreachable etc.), shown in the list panel
	logs []logLine

	confirm     confirmKind
	confirmName string

	width  int
	height int
}

func New() Model { return Model{} }

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(), tickCmd())
}

// selected returns the domain under the cursor.
func (m Model) selected() (virsh.Domain, bool) {
	if m.cursor < 0 || m.cursor >= len(m.domains) {
		return virsh.Domain{}, false
	}
	return m.domains[m.cursor], true
}

type refreshMsg struct {
	domains []virsh.Domain
	err     error
}

type detailMsg struct {
	name   string
	detail virsh.Detail
	err    error
}

type opDoneMsg struct {
	desc   string // display description of the command, e.g. "shutdown resolute"
	output string
	err    error
}

type tickMsg struct{}

func refreshCmd() tea.Cmd {
	return func() tea.Msg {
		domains, err := virsh.ListAll()
		return refreshMsg{domains: domains, err: err}
	}
}

func detailCmd(name string) tea.Cmd {
	return func() tea.Msg {
		d, err := virsh.DomInfo(name)
		return detailMsg{name: name, detail: d, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

// opCmd wraps a backend operation into an async command; on completion the
// command echo and its result land in the Command Log panel.
func opCmd(desc string, fn func() (string, error)) tea.Cmd {
	return func() tea.Msg {
		out, err := fn()
		return opDoneMsg{desc: desc, output: out, err: err}
	}
}
