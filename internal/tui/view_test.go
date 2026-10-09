package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"lazyvirt/internal/virsh"
)

// step asserts Update's interface return value back to Model for easy
// chaining in tests.
func step(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func key(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func fakeModel() Model {
	m := New()
	m.width, m.height = 100, 30
	m.domains = []virsh.Domain{
		{ID: "1", Name: "resolute", State: "running", Autostart: "disable"},
		{ID: "-", Name: "demo", State: "shut off", Autostart: "enable"},
	}
	return m
}

func TestViewLayout(t *testing.T) {
	m := fakeModel()
	m.logs = []logEntry{{
		when: time.Now(),
		ExecLog: virsh.ExecLog{
			Cmdline: "virsh start resolute",
			Output:  "Domain resolute started\n",
		},
	}}
	v := m.View()
	for _, want := range []string{
		"Domains (2)", "resolute", "shut off", "AUTOSTART",
		"Command Log", "quit", "▸",
		"$ virsh start resolute", "Domain resolute started",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("View() missing %q\n%s", want, v)
		}
	}
	if lines := strings.Count(v, "\n") + 1; lines > 30 {
		t.Errorf("View() spans %d lines, exceeding window height 30", lines)
	}
}

func TestKeysNavigationAndConfirm(t *testing.T) {
	m := fakeModel()

	m = step(m, tea.KeyMsg{Type: tea.KeyDown}) // cursor onto demo
	m = step(m, key('x'))
	if m.confirm != confirmNone {
		t.Error("x on a shut-off domain must not open the destroy confirm")
	}

	m = step(m, tea.KeyMsg{Type: tea.KeyUp}) // back to resolute
	m = step(m, key('x'))
	if m.confirm != confirmDestroy || m.confirmName != "resolute" {
		t.Error("x on a running domain must open the destroy confirm")
	}
	if !strings.Contains(m.View(), "destroy resolute") {
		t.Error("confirm dialog should show 'destroy resolute'")
	}

	m = step(m, tea.KeyMsg{Type: tea.KeyEscape}) // any non-confirm key closes
	if m.confirm != confirmNone {
		t.Error("esc should close the confirm dialog")
	}

	m = step(m, key('d'))
	if m.confirm != confirmUndefine {
		t.Error("d must open the undefine confirm")
	}
	// confirming returns the pending async command (not executed here, to
	// avoid touching the real virsh)
	next, cmd := m.Update(key('y'))
	if cmd == nil {
		t.Error("confirming should return the operation command")
	}
	if n := next.(Model); n.confirm != confirmNone {
		t.Error("the dialog should be closed after confirming")
	}
}

func TestRefreshRestoresSelectionByName(t *testing.T) {
	m := fakeModel()
	m.cursor = 1 // demo selected

	// simulate a refresh returning the rows in reversed order
	m = step(m, refreshMsg{domains: []virsh.Domain{
		{ID: "-", Name: "demo", State: "shut off"},
		{ID: "1", Name: "resolute", State: "running"},
	}})
	if m.cursor != 0 {
		t.Errorf("selection should be restored to demo by name, cursor = %d", m.cursor)
	}
}

func TestScrollOffset(t *testing.T) {
	m := fakeModel()
	m.domains = make([]virsh.Domain, 20)
	// offset semantics: hidden lines above the cursor, keeping cursor within
	// [offset, offset+visible)
	cases := []struct{ cursor, want int }{{0, 0}, {5, 0}, {10, 5}, {19, 14}}
	for _, c := range cases {
		m.cursor = c.cursor
		if got := m.scrollOffset(6); got != c.want {
			t.Errorf("scrollOffset(cursor=%d) = %d, want %d", c.cursor, got, c.want)
		}
	}
}
