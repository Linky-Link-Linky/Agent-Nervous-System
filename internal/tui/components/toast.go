package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type toastItem struct {
	Icon    string
	Message string
	IsErr   bool
	Added   time.Time
}

type dismissMsg int

type ToastModel struct {
	queue []toastItem
	idSeq int
}

func NewToast() ToastModel { return ToastModel{} }

func (t ToastModel) Init() tea.Cmd { return nil }

func (t ToastModel) Update(msg tea.Msg) (ToastModel, tea.Cmd) {
	switch msg.(type) {
	case dismissMsg:
		if len(t.queue) > 0 {
			t.queue = t.queue[1:]
		}
	}
	return t, nil
}

func (t *ToastModel) Push(icon, message string, isErr bool) {
	t.idSeq++
	t.queue = append(t.queue, toastItem{Icon: icon, Message: message, IsErr: isErr, Added: time.Now()})
	if len(t.queue) > 5 {
		t.queue = t.queue[1:]
	}
	if !isErr {
		go func() {
			time.Sleep(4 * time.Second)
		}()
	}
}

func (t ToastModel) View(width int) string {
	if len(t.queue) == 0 {
		return ""
	}
	t2 := styles.CurrentTheme
	var items []string
	for _, ti := range t.queue {
		s := lipgloss.NewStyle().Foreground(t2.Success)
		if ti.IsErr {
			s = lipgloss.NewStyle().Foreground(t2.Danger).Border(lipgloss.RoundedBorder()).BorderForeground(t2.Danger)
		}
		items = append(items, s.Render(fmt.Sprintf("%s %s", ti.Icon, ti.Message)))
	}
	joined := lipgloss.JoinVertical(lipgloss.Right, items...)
	return lipgloss.Place(width-1, 0, lipgloss.Right, lipgloss.Top, joined)
}
