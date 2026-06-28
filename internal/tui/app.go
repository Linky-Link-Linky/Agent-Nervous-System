package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/components"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/views"
)

type View int

const (
	ViewDashboard View = iota
	ViewLog
	ViewStream
	ViewSnap
	ViewPolicy
	ViewToken
	ViewProxy
)

type App struct {
	current   View
	dashboard views.DashboardModel
	log       views.LogModel
	stream    views.StreamModel
	snap      views.SnapModel
	policy    views.PolicyModel
	token     views.TokenModel
	proxy     views.ProxyModel
	statusbar components.StatusBar
	repl      components.REPLModel
	palette   components.PaletteModel
	toast     components.ToastModel
	help      components.HelpModel
	vp        viewport.Model

	paletteOpen bool
	helpOpen    bool
	themeIdx    int
	width       int
	height      int
	replActive  bool
}

func NewApp() *App {
	return &App{
		current:    ViewDashboard,
		dashboard:  views.NewDashboard(),
		log:        views.NewLog(),
		stream:     views.NewStream(),
		snap:       views.NewSnap(),
		policy:     views.NewPolicy(),
		token:      views.NewToken(),
		proxy:      views.NewProxy(),
		statusbar:  components.NewStatusBar(),
		repl:       components.NewREPL(),
		palette:    components.NewPalette(),
		toast:      components.NewToast(),
		help:       components.NewHelp(),
		vp:         viewport.New(80, 24),
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.dashboard.Init(),
		a.stream.Init(),
		a.statusbar.Init(),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.vp.Width = msg.Width
		a.vp.Height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if a.paletteOpen {
				a.palette.Close()
				a.paletteOpen = false
				return a, nil
			}
			a.repl.AcceptGhost()
			return a, nil

		case "ctrl+p":
			if a.helpOpen {
				a.help.Close()
				a.helpOpen = false
			}
			a.paletteOpen = !a.paletteOpen
			if a.paletteOpen {
				a.palette.Open()
			} else {
				a.palette.Close()
			}
			return a, nil

		case "?":
			if a.paletteOpen {
				a.palette.Close()
				a.paletteOpen = false
			}
			a.helpOpen = !a.helpOpen
			if a.helpOpen {
				a.help.Open(a.width, a.height)
			} else {
				a.help.Close()
			}
			return a, nil

		case "esc":
			if a.paletteOpen {
				a.palette.Close()
				a.paletteOpen = false
				return a, nil
			}
			if a.helpOpen {
				a.help.Close()
				a.helpOpen = false
				return a, nil
			}

		case "ctrl+t":
			a.themeIdx = (a.themeIdx + 1) % len(styles.Themes)
			styles.CurrentTheme = styles.Themes[a.themeIdx]
			styles.SaveConfig(styles.Config{ThemeIndex: a.themeIdx})
			return a, nil

		case "f1":
			a.current = ViewDashboard
			a.statusbar.ActiveView = "F1 DASH"
		case "f2":
			a.current = ViewLog
			a.statusbar.ActiveView = "F2 LOG"
		case "f3":
			a.current = ViewStream
			a.statusbar.ActiveView = "F3 STREAM"
		case "f4":
			a.current = ViewSnap
			a.statusbar.ActiveView = "F4 SNAP"
		case "f5":
			a.current = ViewPolicy
			a.statusbar.ActiveView = "F5 POLICY"
		case "f6":
			a.current = ViewToken
			a.statusbar.ActiveView = "F6 TOKEN"
		case "f7":
			a.current = ViewProxy
			a.statusbar.ActiveView = "F7 PROXY"

		case "ctrl+c", "q":
			if a.current != ViewDashboard {
				a.current = ViewDashboard
				return a, nil
			}
			return a, tea.Quit

		case "enter":
			if a.paletteOpen {
				if sel, ok := a.palette.Selected(); ok {
					a.palette.Close()
					a.paletteOpen = false
					a.switchToViewByName(sel.Name)
					a.statusbar.ActiveView = sel.Name
				}
				return a, nil
			}
			// Submit REPL
			_ = a.repl.Submit()

		case "up":
			if a.paletteOpen {
				a.palette.CursorUp()
				return a, nil
			}
			a.repl.HistoryUp()

		case "down":
			if a.paletteOpen {
				a.palette.CursorDown()
				return a, nil
			}
			a.repl.HistoryDown()

		}

	case components.PaletteModel:
		a.palette = msg

	case components.StatusBar:
		a.statusbar = msg

	case components.ToastModel:
		a.toast = msg
	}

	// Fan-out to all sub-models
	var cmd tea.Cmd
	a.dashboard, cmd = a.dashboard.Update(msg)
	cmds = append(cmds, cmd)
	a.log, cmd = a.log.Update(msg)
	cmds = append(cmds, cmd)
	a.stream, cmd = a.stream.Update(msg)
	cmds = append(cmds, cmd)
	a.snap, cmd = a.snap.Update(msg)
	cmds = append(cmds, cmd)
	a.policy, cmd = a.policy.Update(msg)
	cmds = append(cmds, cmd)
	a.token, cmd = a.token.Update(msg)
	cmds = append(cmds, cmd)
	a.proxy, cmd = a.proxy.Update(msg)
	cmds = append(cmds, cmd)
	a.statusbar, cmd = a.statusbar.Update(msg)
	cmds = append(cmds, cmd)
	a.repl, cmd = a.repl.Update(msg)
	cmds = append(cmds, cmd)
	a.palette, cmd = a.palette.Update(msg)
	cmds = append(cmds, cmd)
	a.toast, cmd = a.toast.Update(msg)
	cmds = append(cmds, cmd)
	a.help, cmd = a.help.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a *App) switchToViewByName(name string) {
	switch name {
	case "dashboard":
		a.current = ViewDashboard
		a.statusbar.ActiveView = "F1 DASH"
	case "chain":
		a.current = ViewLog
		a.statusbar.ActiveView = "F2 LOG"
	case "stream":
		a.current = ViewStream
		a.statusbar.ActiveView = "F3 STREAM"
	case "snap":
		a.current = ViewSnap
		a.statusbar.ActiveView = "F4 SNAP"
	case "policy":
		a.current = ViewPolicy
		a.statusbar.ActiveView = "F5 POLICY"
	case "token":
		a.current = ViewToken
		a.statusbar.ActiveView = "F6 TOKEN"
	case "proxy":
		a.current = ViewProxy
		a.statusbar.ActiveView = "F7 PROXY"
	}
}

func (a *App) View() string {
	t := styles.CurrentTheme
	w := a.width
	h := a.height
	if w == 0 {
		w = 120
	}
	if h == 0 {
		h = 36
	}

	statusH := 1
	replH := 1
	viewH := h - statusH - replH

	var viewContent string
	switch a.current {
	case ViewDashboard:
		viewContent = a.dashboard.View(w, viewH)
	case ViewLog:
		viewContent = a.log.View(w, viewH)
	case ViewStream:
		viewContent = a.stream.View(w, viewH)
	case ViewSnap:
		viewContent = a.snap.View(w, viewH)
	case ViewPolicy:
		viewContent = a.policy.View(w, viewH)
	case ViewToken:
		viewContent = a.token.View(w, viewH)
	case ViewProxy:
		viewContent = a.proxy.View(w, viewH)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Background(t.Bg).Width(w).Render(viewContent),
		a.repl.View(w),
		a.statusbar.View(w),
	)

	if a.paletteOpen {
		body = a.palette.View(w, h)
	}
	if a.helpOpen {
		body = a.help.View(w, h)
	}

	toastView := a.toast.View(w)
	if toastView != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, toastView)
	}

	return body
}
