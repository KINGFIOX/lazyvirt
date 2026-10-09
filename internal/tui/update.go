package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"

	"lazyvirt/internal/virsh"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(tickCmd(), refreshCmd())

	case refreshMsg:
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}
		var keep string
		if sel, ok := m.selected(); ok {
			keep = sel.Name
		}
		m.domains = msg.domains
		m.cursor = 0
		for i, d := range m.domains { // restore selection by name
			if d.Name == keep {
				m.cursor = i
				break
			}
		}
		if sel, ok := m.selected(); ok {
			m.detailName = sel.Name // also refresh the detail panel
			return m, detailCmd(sel.Name)
		}
		return m, nil

	case detailMsg:
		if msg.name != m.detailName {
			return m, nil // stale reply (cursor has moved on)
		}
		if msg.err != nil {
			m.detail = virsh.Detail{}
		} else {
			m.detail = msg.detail
		}
		return m, nil

	case opDoneMsg:
		m.logs = append(m.logs, logLine{when: time.Now(), text: opText(msg), err: msg.err != nil})
		if len(m.logs) > 200 {
			m.logs = m.logs[len(m.logs)-200:]
		}
		return m, refreshCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func opText(msg opDoneMsg) string {
	if msg.err != nil {
		return msg.desc + " → error: " + strings.TrimSpace(msg.err.Error())
	}
	if out := strings.TrimSpace(msg.output); out != "" {
		return msg.desc + " → " + firstLine(out)
	}
	return msg.desc + " → ok"
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// While a confirm dialog is open, only y/enter executes; anything else
	// closes it.
	if m.confirm != confirmNone {
		if key == "y" || key == "Y" || key == "enter" {
			kind, name := m.confirm, m.confirmName
			m.confirm, m.confirmName = confirmNone, ""
			switch kind {
			case confirmDestroy:
				return m, opCmd("destroy "+name, func() (string, error) { return virsh.Destroy(name) })
			case confirmUndefine:
				return m, opCmd("undefine "+name, func() (string, error) { return virsh.Undefine(name, false, false) })
			case confirmUndefineAll:
				return m, opCmd("undefine --nvram --remove-all-storage "+name,
					func() (string, error) { return virsh.Undefine(name, true, true) })
			}
			return m, nil
		}
		m.confirm, m.confirmName = confirmNone, ""
		return m, nil
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.domains)-1 {
			m.cursor++
			return m.selectChanged()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			return m.selectChanged()
		}

	case "s":
		if d, ok := m.selected(); ok {
			return m, opCmd("start "+d.Name, func() (string, error) { return virsh.Start(d.Name) })
		}
	case "S":
		if d, ok := m.selected(); ok {
			return m, opCmd("shutdown "+d.Name, func() (string, error) { return virsh.Shutdown(d.Name) })
		}
	case "r":
		if d, ok := m.selected(); ok {
			return m, opCmd("reboot "+d.Name, func() (string, error) { return virsh.Reboot(d.Name) })
		}
	case "x":
		// Forcing off a shut-off domain makes no sense; only ask when running.
		if d, ok := m.selected(); ok && d.State != "shut off" {
			m.confirm, m.confirmName = confirmDestroy, d.Name
		}
	case "a":
		if d, ok := m.selected(); ok {
			enable := d.Autostart != "enable"
			desc := "autostart --disable " + d.Name
			if enable {
				desc = "autostart " + d.Name
			}
			return m, opCmd(desc, func() (string, error) { return virsh.AutostartSet(d.Name, enable) })
		}
	case "d":
		if d, ok := m.selected(); ok {
			m.confirm, m.confirmName = confirmUndefine, d.Name
		}
	case "U":
		if d, ok := m.selected(); ok {
			m.confirm, m.confirmName = confirmUndefineAll, d.Name
		}
	case "R":
		return m, refreshCmd()
	}
	return m, nil
}

// selectChanged fetches the detail of the newly selected domain if it changed.
func (m Model) selectChanged() (tea.Model, tea.Cmd) {
	sel, ok := m.selected()
	if !ok || sel.Name == m.detailName {
		return m, nil
	}
	m.detailName = sel.Name
	m.detail = virsh.Detail{}
	return m, detailCmd(sel.Name)
}
