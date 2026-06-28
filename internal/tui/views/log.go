package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type LogEntry struct {
	Index     int
	AgentID   string
	Hash      string
	Timestamp string
	Verified  bool
}

type LogModel struct {
	entries  []LogEntry
	vp       viewport.Model
	ready    bool
	search   string
	filtered []LogEntry
}

func NewLog() LogModel {
	entries := make([]LogEntry, 100)
	for i := range entries {
		entries[i] = LogEntry{
			Index: 100 - i, AgentID: fmt.Sprintf("agent-%04x", i*13%65535),
			Hash: fmt.Sprintf("%x…", i*0xabcdef), Timestamp: fmt.Sprintf("%dm ago", i*3),
			Verified: i%5 != 0,
		}
	}
	return LogModel{entries: entries, filtered: entries}
}

func (m LogModel) Init() tea.Cmd { return nil }

func (m LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *LogModel) SetSearch(q string) {
	m.search = q
	if q == "" {
		m.filtered = m.entries
		return
	}
	var f []LogEntry
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(fmt.Sprintf("%d %s %s", e.Index, e.AgentID, e.Hash)), strings.ToLower(q)) {
			f = append(f, e)
		}
	}
	m.filtered = f
}

func (m LogModel) View(width, height int) string {
	t := styles.CurrentTheme
	if !m.ready {
		m.vp = viewport.New(width-2, height-2)
		m.ready = true
	}
	m.vp.Width = width - 2
	m.vp.Height = height - 2

	var lines []string
	colors := []lipgloss.Color{t.Accent, t.Success, t.Warning, t.Fg}
	for i, e := range m.filtered {
		agentColor := colors[i%len(colors)]
		verifiedMark := lipgloss.NewStyle().Foreground(t.Success).Render("✓")
		if !e.Verified {
			verifiedMark = lipgloss.NewStyle().Foreground(t.Danger).Render("✗")
		}
		line := fmt.Sprintf("  ┌── [#%03d] %s  %s  %s  %s",
			e.Index,
			lipgloss.NewStyle().Foreground(agentColor).Render(e.AgentID),
			lipgloss.NewStyle().Foreground(t.FgMuted).Render(e.Hash),
			lipgloss.NewStyle().Foreground(t.FgMuted).Render(e.Timestamp),
			verifiedMark,
		)
		lines = append(lines, line)
	}
	m.vp.SetContent(strings.Join(lines, "\n"))
	return t.BoxStyle().Width(width).Height(height).Render(m.vp.View())
}
