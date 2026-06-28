package components

import (
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type PaletteModel struct {
	open    bool
	input   textinput.Model
	results []styles.CommandDef
	cursor  int
}

func NewPalette() PaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Search commands…"
	ti.CharLimit = 50
	ti.Width = 40
	return PaletteModel{input: ti}
}

func (p PaletteModel) Init() tea.Cmd { return nil }

func (p PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd) {
	if !p.open {
		return p, nil
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.filter()
	return p, cmd
}

func (p *PaletteModel) Open() {
	p.open = true
	p.input.SetValue("")
	p.input.Focus()
	p.results = styles.CommandList
	p.cursor = 0
}

func (p *PaletteModel) Close() {
	p.open = false
	p.input.Blur()
}

func (p *PaletteModel) filter() {
	q := p.input.Value()
	if q == "" {
		p.results = styles.CommandList
		return
	}
	matches := fuzzy.Find(q, commandNames())
	p.results = nil
	for _, m := range matches {
		p.results = append(p.results, styles.CommandList[m.Index])
	}
}

func commandNames() []string {
	n := make([]string, len(styles.CommandList))
	for i, c := range styles.CommandList {
		n[i] = c.Name
	}
	return n
}

func (p *PaletteModel) CursorUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *PaletteModel) CursorDown() {
	if p.cursor < len(p.results)-1 {
		p.cursor++
	}
}

func (p *PaletteModel) Selected() (styles.CommandDef, bool) {
	if len(p.results) == 0 {
		return styles.CommandDef{}, false
	}
	return p.results[p.cursor], true
}

func (p PaletteModel) View(w, h int) string {
	t := styles.CurrentTheme
	overlayW := w * 60 / 100
	overlayH := h * 50 / 100
	var b strings.Builder
	b.WriteString(t.BoxStyle().Width(overlayW-4).Render(" Palette ") + "\n")
	b.WriteString(p.input.View() + "\n")
	for i, r := range p.results {
		if i >= overlayH-3 {
			break
		}
		line := r.Name
		if r.Desc != "" {
			line += "  —  " + r.Desc
		}
		if r.Shortcut != "" {
			line += "  [" + r.Shortcut + "]"
		}
		style := lipgloss.NewStyle().Foreground(t.Fg)
		if i == p.cursor {
			style = lipgloss.NewStyle().Foreground(t.Bg).Background(t.Accent)
		}
		b.WriteString(style.Width(overlayW - 6).Render(line) + "\n")
	}
	content := t.BoxStyle().Width(overlayW).Height(overlayH).Render(b.String())
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
}
