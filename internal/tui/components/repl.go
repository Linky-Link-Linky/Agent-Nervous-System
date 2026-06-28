package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type REPLModel struct {
	input    textinput.Model
	history  []string
	histPos  int
	ghostText string
	active   bool
}

func NewREPL() REPLModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command…"
	ti.CharLimit = 200
	ti.Width = 60
	return REPLModel{input: ti, history: make([]string, 0, 50)}
}

func (r REPLModel) Init() tea.Cmd { return nil }

func (r REPLModel) Update(msg tea.Msg) (REPLModel, tea.Cmd) {
	var cmd tea.Cmd
	r.input, cmd = r.input.Update(msg)
	r.updateGhost()
	return r, cmd
}

func (r *REPLModel) updateGhost() {
	input := r.input.Value()
	if input == "" {
		r.ghostText = ""
		return
	}
	for _, c := range styles.CommandList {
		if strings.HasPrefix(c.Name, input) && c.Name != input {
			r.ghostText = c.Name[len(input):]
			return
		}
	}
	r.ghostText = ""
}

func (r *REPLModel) Submit() string {
	v := r.input.Value()
	if v == "" {
		return ""
	}
	r.history = append(r.history, v)
	if len(r.history) > 50 {
		r.history = r.history[1:]
	}
	r.histPos = len(r.history)
	r.input.SetValue("")
	r.ghostText = ""
	return v
}

func (r *REPLModel) HistoryUp() {
	if len(r.history) == 0 {
		return
	}
	if r.histPos > 0 {
		r.histPos--
	}
	r.input.SetValue(r.history[r.histPos])
	r.input.SetCursor(len(r.history[r.histPos]))
}

func (r *REPLModel) HistoryDown() {
	if r.histPos >= len(r.history)-1 {
		r.input.SetValue("")
		r.histPos = len(r.history)
		return
	}
	r.histPos++
	r.input.SetValue(r.history[r.histPos])
	r.input.SetCursor(len(r.history[r.histPos]))
}

func (r REPLModel) AcceptGhost() {
	r.input.SetValue(r.input.Value() + r.ghostText)
	r.ghostText = ""
}

func (r REPLModel) View(width int) string {
	t := styles.CurrentTheme
	ghost := lipgloss.NewStyle().Foreground(t.FgMuted).Render(r.ghostText)
	prompt := lipgloss.NewStyle().Foreground(t.Accent).Render("> ")
	input := r.input.View()
	return lipgloss.NewStyle().Background(t.Bg).Width(width).Render(prompt + input + ghost)
}
