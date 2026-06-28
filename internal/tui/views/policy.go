package views

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type PolicyItem struct {
	ID, Name string
	Enabled  bool
}

func (i PolicyItem) Title() string { return i.Name }
func (i PolicyItem) Description() string {
	badge := styles.CurrentTheme.Badge("ON", true)
	if !i.Enabled {
		badge = styles.CurrentTheme.Badge("OFF", false)
	}
	return i.ID + "  " + badge
}
func (i PolicyItem) FilterValue() string { return i.Name + i.ID }

type PolicyModel struct {
	list       list.Model
	detail     string
	detailVP   viewport.Model
	adding     bool
	fileInput  textinput.Model
	testMode   bool
	testAction textinput.Model
	testPayload textinput.Model
	testResult string
}

func NewPolicy() PolicyModel {
	items := []list.Item{
		PolicyItem{ID: "pol-001", Name: "block-chat-pii", Enabled: true},
		PolicyItem{ID: "pol-002", Name: "allow-read-only", Enabled: true},
		PolicyItem{ID: "pol-003", Name: "rate-limit-admin", Enabled: false},
	}
	l := list.New(items, list.NewDefaultDelegate(), 30, 10)
	l.Title = "Policies"
	ti := textinput.New()
	ti.Placeholder = "File path to policy YAML…"
	ta := textinput.New()
	ta.Placeholder = "action-type (e.g. chat)"
	tp := textinput.New()
	tp.Placeholder = "payload-summary"
	return PolicyModel{
		list: l, detail: "Select a policy to view details",
		fileInput: ti, testAction: ta, testPayload: tp,
	}
}

func (m PolicyModel) Init() tea.Cmd { return nil }

func (m PolicyModel) Update(msg tea.Msg) (PolicyModel, tea.Cmd) { return m, nil }

func (m PolicyModel) View(width, height int) string {
	t := styles.CurrentTheme
	listH := height * 50 / 100
	detailH := height - listH - 2

	// Test panel
	testPanel := ""
	if m.testMode {
		testPanel = t.BoxStyle().Width(width - 4).Render(lipgloss.JoinVertical(lipgloss.Left,
			t.TitleStyle().Render(" Policy Eval Test"),
			m.testAction.View(),
			m.testPayload.View(),
			lipgloss.NewStyle().Foreground(t.Fg).Render(m.testResult),
		))
	}

	m.detailVP.Width = width - 4
	m.detailVP.Height = detailH
	m.detailVP.SetContent(m.detail)
	return lipgloss.JoinVertical(lipgloss.Left,
		t.BoxStyle().Width(width).Height(listH).Render(m.list.View()),
		t.BoxStyle().Width(width).Height(detailH).Render(m.detailVP.View()),
		testPanel,
	)
}
