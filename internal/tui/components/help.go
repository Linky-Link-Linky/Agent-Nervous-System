package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
)

type HelpModel struct {
	open  bool
	vp    viewport.Model
	ready bool
}

func NewHelp() HelpModel { return HelpModel{} }

func (m *HelpModel) IsOpen() bool { return m.open }

func (m HelpModel) Init() tea.Cmd { return nil }

func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	if !m.open {
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *HelpModel) Open(pw, ph int) {
	m.open = true
	if !m.ready {
		m.vp = viewport.New(pw-4, ph-4)
		m.vp.SetContent(helpMarkdown)
		m.ready = true
	} else {
		m.vp.Width = pw - 4
		m.vp.Height = ph - 4
	}
}

func (m *HelpModel) Close() { m.open = false }

func (m HelpModel) View(pw, ph int) string {
	t := styles.CurrentTheme
	r, _ := glamour.NewTermRenderer(glamour.WithStylePath("dark"),
		glamour.WithWordWrap(pw-10))
	rendered, _ := r.Render(helpMarkdown)
	content := t.BoxStyle().Width(pw - 4).Height(ph - 4).Render(rendered)
	return lipgloss.Place(pw, ph, lipgloss.Center, lipgloss.Center, content)
}

var helpMarkdown = "# ANS TUI — Keyboard Shortcuts\n\n" +
    "## Global\n| Key | Action |\n|-----|--------|\n" +
    "| `Tab` / `Ctrl+P` | Toggle command palette |\n" +
    "| `?` | Toggle help overlay |\n" +
    "| `F1`–`F7` | Switch views |\n" +
    "| `Ctrl+T` | Cycle theme |\n" +
    "| `Ctrl+C` / `q` | Quit |\n" +
    "| `Ctrl+R` | Reverse history search |\n\n" +
    "## Dashboard (F1)\n| Key | Action |\n|-----|--------|\n" +
    "| `j`/`k` | Scroll |\n" +
    "| `Enter` | Select |\n" +
    "| `w` | Toggle watch mode |\n\n" +
    "## Log (F2)\n| Key | Action |\n|-----|--------|\n" +
    "| `j`/`k` | Scroll |\n" +
    "| `gg`/`G` | Jump top/bottom |\n" +
    "| `/` | Search |\n" +
    "| `v` | Verify receipt |\n" +
    "| `Enter` | Expand receipt |\n\n" +
    "## Stream (F3)\n| Key | Action |\n|-----|--------|\n" +
    "| `p` | Pause/resume |\n" +
    "| `c` | Clear buffer |\n" +
    "| `j`/`k` | Scroll |\n\n" +
    "## Snap (F4)\n| Key | Action |\n|-----|--------|\n" +
    "| `←`/`→` | Scrub time |\n" +
    "| `Enter` | Restore |\n" +
    "| `j`/`k` | Navigate |\n\n" +
    "## Policy (F5)\n| Key | Action |\n|-----|--------|\n" +
    "| `n` | Add policy |\n" +
    "| `d` | Delete |\n" +
    "| `t` | Test/eval |\n\n" +
    "## Token (F6)\n| Key | Action |\n|-----|--------|\n" +
    "| `n` | New token |\n" +
    "| `r` | Revoke |\n\n" +
    "## Proxy (F7)\n| Key | Action |\n|-----|--------|\n" +
    "| `o` | Proxy on |\n" +
    "| `f` | Proxy off |\n" +
    "| `l` | Load full log |\n"
