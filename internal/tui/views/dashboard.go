package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type refreshMsg time.Time

type DashboardModel struct {
	DaemonOK       bool
	ChainLen       int
	AgentCount     int
	DBSize         string
	Agents         []AgentInfo
	RecentReceipts []string
	ticker         int
	width, height  int
}

type AgentInfo struct {
	ID             string
	KeyFingerprint string
	LastSeen       string
}

func NewDashboard() DashboardModel {
	return DashboardModel{
		Agents: []AgentInfo{
			{ID: "agent-7f3a", KeyFingerprint: "a1:b2:c3…", LastSeen: "2s ago"},
			{ID: "agent-9c1b", KeyFingerprint: "d4:e5:f6…", LastSeen: "15s ago"},
		},
		RecentReceipts: make([]string, 0, 5),
		ChainLen:       142,
		AgentCount:     2,
		DBSize:         "4.2MB",
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg { return refreshMsg(t) })
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg.(type) {
	case refreshMsg:
		m.ticker++
		m.ChainLen += m.ticker % 3 // simulate growth
		return m, tea.Every(2*time.Second, func(t time.Time) tea.Msg { return refreshMsg(t) })
	}
	return m, nil
}

func (m DashboardModel) View(width, height int) string {
	t := styles.CurrentTheme
	// Stat cards row
	daemonCol := t.Success
	if !m.DaemonOK {
		daemonCol = t.Danger
	}
	daemonCard := t.BoxStyle().Width(24).Render(lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(t.FgMuted).Render("Daemon"),
		lipgloss.NewStyle().Foreground(daemonCol).Bold(true).Render("● Running"),
	))
	chainCard := t.BoxStyle().Width(24).Render(lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(t.FgMuted).Render("Chain Length"),
		t.TitleStyle().Render(fmt.Sprintf("%d", m.ChainLen)),
	))
	agentCard := t.BoxStyle().Width(24).Render(lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(t.FgMuted).Render("Active Agents"),
		t.TitleStyle().Render(fmt.Sprintf("%d", m.AgentCount)),
	))
	dbCard := t.BoxStyle().Width(24).Render(lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(t.FgMuted).Render("DB Size"),
		t.TitleStyle().Render(m.DBSize),
	))
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, daemonCard, chainCard, agentCard, dbCard)

	// Agent cards
	var agentCards []string
	for _, a := range m.Agents {
		card := t.BoxStyle().Width(30).Render(lipgloss.JoinVertical(lipgloss.Left,
			t.TitleStyle().Render(a.ID),
			lipgloss.NewStyle().Foreground(t.FgMuted).Render("Key: "+a.KeyFingerprint),
			lipgloss.NewStyle().Foreground(t.FgMuted).Render("Last: "+a.LastSeen),
		))
		agentCards = append(agentCards, card)
	}
	midRow := lipgloss.JoinHorizontal(lipgloss.Top, agentCards...)

	// Recent receipts ticker
	var recents []string
	for _, r := range m.RecentReceipts {
		recents = append(recents, lipgloss.NewStyle().Foreground(t.FgMuted).Render("  "+r))
	}
	if len(recents) == 0 {
		recents = append(recents, lipgloss.NewStyle().Foreground(t.FgMuted).Render("  No recent receipts"))
	}
	botBox := t.BoxStyle().Width(width-4).Render(lipgloss.JoinVertical(lipgloss.Left,
		append([]string{t.TitleStyle().Render(" Recent Receipts")}, recents...)...,
	))

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "\n", midRow, "\n", botBox)
}
