// Package tui is lazyvirt's terminal UI: the bubbletea Model/Update/View
// trio, split by file into model (state & messages) / update (keys &
// commands) / view (rendering).
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbletea"

	"lazyvirt/internal/virsh"
)

const refreshInterval = 5 * time.Second

// logEntry is one virsh invocation shown in the Command Log panel.
type logEntry struct {
	when time.Time
	virsh.ExecLog
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
	logs []logEntry

	confirm     confirmKind
	confirmName string

	width  int
	height int
}

func New() Model { return Model{} }

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshCmd(false), tickCmd())
}

// selected returns the domain under the cursor.
func (m Model) selected() (virsh.Domain, bool) {
	if m.cursor < 0 || m.cursor >= len(m.domains) {
		return virsh.Domain{}, false
	}
	return m.domains[m.cursor], true
}

// AttachExecLogger wires virsh execution reports into the program as
// messages, so every non-quiet invocation shows up in the Command Log.
func AttachExecLogger(p *tea.Program) {
	virsh.OnExec = func(e virsh.ExecLog) {
		p.Send(execLogMsg{entry: e})
	}
}

type execLogMsg struct {
	entry virsh.ExecLog
}

// refreshMsg carries the result of a list refresh; quiet marks background
// (ticker-driven) refreshes so their follow-up detail fetch stays silent.
type refreshMsg struct {
	domains []virsh.Domain
	err     error
	quiet   bool
}

type detailMsg struct {
	name   string
	detail virsh.Detail
	err    error
}

// opDoneMsg signals that an operation finished; its echo and result have
// already been reported through execLogMsg.
type opDoneMsg struct{}

type tickMsg struct{}

// refreshCmd runs `virsh list --all`; quiet suppresses the Command Log entry
// (used by the periodic ticker).
func refreshCmd(quiet bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if quiet {
			ctx = virsh.Quiet(ctx)
		}
		domains, err := virsh.ListAll(ctx)
		return refreshMsg{domains: domains, err: err, quiet: quiet}
	}
}

func detailCmd(name string, quiet bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if quiet {
			ctx = virsh.Quiet(ctx)
		}
		d, err := virsh.DomInfo(ctx, name)
		return detailMsg{name: name, detail: d, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

// opCmd wraps a backend operation into an async command; the invocation
// itself is logged by the virsh layer, so on completion we only refresh.
func opCmd(fn func(ctx context.Context) (string, error)) tea.Cmd {
	return func() tea.Msg {
		_, _ = fn(context.Background())
		return opDoneMsg{}
	}
}
