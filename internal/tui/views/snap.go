package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type SnapItem struct {
	ID, Time, Size, Agent string
	Index                 int
}

func (i SnapItem) Title() string       { return i.ID }
func (i SnapItem) Description() string { return fmt.Sprintf("idx:%d  %s  %s  %s", i.Index, i.Time, i.Size, i.Agent) }
func (i SnapItem) FilterValue() string { return i.ID + i.Agent }

type SnapModel struct {
	list        list.Model
	scrubber    int
	chainLen    int
	diffContent string
}

func NewSnap() SnapModel {
	items := []list.Item{
		SnapItem{ID: "snap-a3f2", Index: 142, Time: "12:30:01", Size: "1.2MB", Agent: "agent-7f3a"},
		SnapItem{ID: "snap-b81c", Index: 128, Time: "12:25:00", Size: "980KB", Agent: "agent-9c1b"},
		SnapItem{ID: "snap-c4e9", Index: 115, Time: "12:20:15", Size: "1.1MB", Agent: "agent-7f3a"},
	}
	l := list.New(items, list.NewDefaultDelegate(), 30, 15)
	l.Title = "Snapshots"
	return SnapModel{list: l, chainLen: 142, diffContent: "Select a snapshot to see diff"}
}

func (m SnapModel) Init() tea.Cmd { return nil }

func (m SnapModel) Update(msg tea.Msg) (SnapModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *SnapModel) ScrubLeft() {
	if m.scrubber > 0 {
		m.scrubber--
	}
}
func (m *SnapModel) ScrubRight() {
	if m.scrubber < m.chainLen {
		m.scrubber++
	}
}

func (m SnapModel) View(width, height int) string {
	t := styles.CurrentTheme
	leftW := width * 40 / 100

	// Scrubber bar
	scrubW := width - 4
	if scrubW < 10 {
		scrubW = 10
	}
	filled := m.scrubber * scrubW / m.chainLen
	if filled > scrubW {
		filled = scrubW
	}
	scrubber := strings.Repeat("▓", filled) + strings.Repeat("░", scrubW-filled)
	scrubLabel := fmt.Sprintf(" Index %d / %d ", m.scrubber, m.chainLen)
	scrubBar := lipgloss.NewStyle().Foreground(t.Accent).Render(scrubber) + "\n" +
		lipgloss.NewStyle().Foreground(t.FgMuted).Render(scrubLabel)

	diffView := t.BoxStyle().Width(width-leftW-2).Height(height - 4).Render(m.diffContent)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		t.BoxStyle().Width(leftW).Height(height-4).Render(m.list.View()),
		diffView,
	) + "\n" + scrubBar
}
