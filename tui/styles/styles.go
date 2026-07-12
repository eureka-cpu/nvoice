package styles

import "github.com/charmbracelet/lipgloss"

var (
	Selected  = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("15"))
	Header    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	Label     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(20)
	Value     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	Auto      = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	Cursor    = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	Footer    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	Status    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	ErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	Dirty     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("●")
	Separator = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("  │  ")
)
