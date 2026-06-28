package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type tickMsg time.Time

type StatusBar struct {
	Clock      string
	DaemonOK   bool
	AgentCount int
	ActiveView string
	ViewNames  []string
	theme      styles.Theme
}

func NewStatusBar() StatusBar {
	return StatusBar{
		ViewNames: []string{"F1 DASH", "F2 LOG", "F3 STREAM", "F4 SNAP", "F5 POLICY", "F6 TOKEN", "F7 PROXY"},
	}
}

func (s StatusBar) Init() tea.Cmd {
	return s.tickCmd()
}

func (s StatusBar) tickCmd() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (s StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		s.Clock = time.Time(msg).Format("15:04:05")
		return s, s.tickCmd()
	}
	return s, nil
}

func (s StatusBar) View(width int) string {
	t := styles.CurrentTheme
	var tabs string
	for _, v := range s.ViewNames {
		col := t.FgMuted
		if v == s.ActiveView {
			col = t.Accent
		}
		tabs += lipgloss.NewStyle().Foreground(col).Padding(0, 1).Render(v)
	}
	dot := lipgloss.NewStyle().Foreground(t.Success).Render("●")
	if !s.DaemonOK {
		dot = lipgloss.NewStyle().Foreground(t.Danger).Render("○")
	}
	right := fmt.Sprintf("%s %d agents  %s", dot, s.AgentCount, s.Clock)
	bar := lipgloss.NewStyle().Background(t.Bg).Width(width).
		MaxHeight(1).Render(lipgloss.JoinHorizontal(lipgloss.Left, tabs, right))
	return bar
}
