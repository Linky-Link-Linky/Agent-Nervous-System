package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type TokenItem struct {
	ID, Resource, Action string
	ExpiresIn            int
	Revoked              bool
}

func (i TokenItem) Title() string { return i.ID }
func (i TokenItem) Description() string {
	t := styles.CurrentTheme
	expires := lipgloss.NewStyle().Foreground(t.FgMuted).Render(fmt.Sprintf("%ds left", i.ExpiresIn))
	if i.Revoked {
		expires = t.Badge("REVOKED", false)
	}
	return fmt.Sprintf("%s %s  %s", i.Resource, i.Action, expires)
}
func (i TokenItem) FilterValue() string { return i.ID + i.Resource }

type TokenModel struct {
	list       list.Model
	formOpen   bool
	resourceIn textinput.Model
	actionIn   textinput.Model
	ttlIn      textinput.Model
}

func NewToken() TokenModel {
	items := []list.Item{
		TokenItem{ID: "tok-a1b2", Resource: "arn:ans:chain:r", Action: "read", ExpiresIn: 3420, Revoked: false},
		TokenItem{ID: "tok-c3d4", Resource: "arn:ans:agent:*", Action: "write", ExpiresIn: 120, Revoked: false},
		TokenItem{ID: "tok-e5f6", Resource: "arn:ans:policy", Action: "admin", ExpiresIn: 0, Revoked: true},
	}
	l := list.New(items, list.NewDefaultDelegate(), 30, 15)
	l.Title = "Active Tokens"
	ti := textinput.New()
	ti.Placeholder = "Resource ARN…"
	ai := textinput.New()
	ai.Placeholder = "Action…"
	tt := textinput.New()
	tt.Placeholder = "TTL seconds…"
	return TokenModel{
		list: l, resourceIn: ti, actionIn: ai, ttlIn: tt,
	}
}

func (m TokenModel) Init() tea.Cmd { return nil }
func (m TokenModel) Update(msg tea.Msg) (TokenModel, tea.Cmd) { return m, nil }

func (m TokenModel) View(width, height int) string {
	t := styles.CurrentTheme
	view := t.BoxStyle().Width(width).Height(height).Render(m.list.View())
	if m.formOpen {
		form := t.BoxStyle().Width(40).Render(lipgloss.JoinVertical(lipgloss.Left,
			t.TitleStyle().Render(" New Token"),
			m.resourceIn.View(),
			m.actionIn.View(),
			m.ttlIn.View(),
		))
		view = lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, form)
	}
	return view
}
