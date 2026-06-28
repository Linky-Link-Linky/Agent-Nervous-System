package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type StreamEntry struct {
	JSON string
	Time time.Time
}

type StreamModel struct {
	entries   []StreamEntry
	paused    bool
	rate      float64
	count     int
	startTime time.Time
}

func NewStream() StreamModel {
	return StreamModel{startTime: time.Now()}
}

func (m StreamModel) Init() tea.Cmd {
	return tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg {
		return streamMsg(t)
	})
}

type streamMsg time.Time

func (m StreamModel) Update(msg tea.Msg) (StreamModel, tea.Cmd) {
	switch msg.(type) {
	case streamMsg:
		if !m.paused {
			m.entries = append(m.entries, StreamEntry{
				JSON: fmt.Sprintf(`{"receipt_id":"r-%04x","agent":"agent-%x","action":"chat","timestamp":%d}`,
					m.count, m.count*7, time.Now().UnixNano()),
				Time: time.Now(),
			})
			m.count++
			if len(m.entries) > 200 {
				m.entries = m.entries[50:]
			}
			elapsed := time.Since(m.startTime).Seconds()
			if elapsed > 0 {
				m.rate = float64(m.count) / elapsed
			}
		}
		return m, tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg { return streamMsg(t) })
	}
	return m, nil
}

func (m *StreamModel) TogglePause() { m.paused = !m.paused }
func (m *StreamModel) Clear() {
	m.entries = nil
	m.count = 0
	m.startTime = time.Now()
	m.rate = 0
}

func (m StreamModel) View(width, height int) string {
	t := styles.CurrentTheme
	rateStr := fmt.Sprintf("%.1f/s", m.rate)
	pauseStr := ""
	if m.paused {
		pauseStr = lipgloss.NewStyle().Foreground(t.Warning).Render(" ⏸ PAUSED")
	}
	title := t.TitleStyle().Render(" LIVE STREAM ") + lipgloss.NewStyle().Foreground(t.FgMuted).Render(rateStr) + pauseStr

	var lines []string
	for _, e := range m.entries {
		lines = append(lines, lipgloss.NewStyle().Foreground(t.Fg).Render("  "+e.JSON))
	}
	if len(lines) > height-3 {
		lines = lines[len(lines)-height+3:]
	}

	return t.BoxStyle().Width(width).Height(height).Render(lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title}, lines...)...,
	))
}
