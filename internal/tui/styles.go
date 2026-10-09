package tui

import "github.com/charmbracelet/lipgloss"

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("236"))

	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	okStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("37"))

	cmdStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81"))

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("203")).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("203"))
)

// stateStyle colors by domain state: green=running, yellow=paused/suspended,
// gray=shut off, red=crashed, yellow for everything else.
func stateStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	case "shut off":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	case "crashed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	default: // paused / in shutdown / pmsuspended / saving ...
		return lipgloss.NewStyle().Foreground(lipgloss.Color("179"))
	}
}
