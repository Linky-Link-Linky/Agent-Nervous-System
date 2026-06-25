package dashboard

import (
	"fmt"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	bgColor       = tcell.NewRGBColor(0x0A, 0x0A, 0x0F)
	primaryColor  = tcell.NewRGBColor(0xA8, 0x55, 0xF7)
	secondaryColor = tcell.NewRGBColor(0x7C, 0x3A, 0xED)
	dimColor      = tcell.NewRGBColor(0x4C, 0x1D, 0x95)
	mutedColor    = tcell.NewRGBColor(0xC0, 0x84, 0xFC)
	foreground    = tcell.NewRGBColor(0xE2, 0xE8, 0xF0)
	dimText       = tcell.NewRGBColor(0x94, 0xA3, 0xB8)
	borderColor   = tcell.NewRGBColor(0x3B, 0x07, 0x64)
	alertColor    = tcell.NewRGBColor(0xF4, 0x72, 0xB6)
	successColor  = tcell.NewRGBColor(0x86, 0xEF, 0xAC)
)


type App struct {
	tview   *tview.Application
	provider providers.DashboardProvider
	pages   *tview.Pages

	overview  *overviewPanel
	chart     *chartPanel
	auditLog  *auditLogPanel
	policy    *policyPanel
	statusBar *statusBarPanel
	cmdBar    *commandBar

	stopCh chan struct{}
}

func NewApp() *App {
	prov := providers.NewMockProvider()

	a := &App{
		tview:    tview.NewApplication(),
		provider: prov,
		pages:    tview.NewPages(),
		stopCh:   make(chan struct{}),
	}

	a.overview = newOverviewPanel(prov)
	a.chart = newChartPanel(prov)
	a.auditLog = newAuditLogPanel(prov)
	a.policy = newPolicyPanel(prov)
	a.statusBar = newStatusBarPanel(prov)
	a.cmdBar = newCommandBar(a.tview, prov)

	return a
}

func (a *App) Run() error {
	a.tview.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyCtrlC || ev.Key() == tcell.KeyEscape {
			if a.cmdBar.inputMode {
				a.cmdBar.deactivate()
				return nil
			}
			a.tview.Stop()
			return nil
		}
		if ev.Rune() == 'q' || ev.Rune() == 'Q' {
			if a.cmdBar.inputMode {
				return ev
			}
			a.tview.Stop()
			return nil
		}
		if ev.Rune() == 's' || ev.Rune() == 'S' {
			if !a.cmdBar.inputMode {
				a.triggerSnapshot()
			}
			return nil
		}
		if ev.Rune() == ':' && !a.cmdBar.inputMode {
			a.cmdBar.activate()
			return nil
		}
		return ev
	})

	splash := a.buildSplash()
	mainUI := a.buildMainUI()

	a.pages.AddPage("splash", splash, true, true)
	a.pages.AddPage("main", mainUI, true, false)
	a.pages.SwitchToPage("splash")

	go a.splashCountdown()
	go a.dataLoop()
	go a.chart.animateBars(a.tview)

	return a.tview.SetRoot(a.pages, true).EnableMouse(false).Run()
}

func (a *App) Stop() {
	close(a.stopCh)
}

func (a *App) splashCountdown() {
	t := time.NewTimer(1500 * time.Millisecond)
	defer t.Stop()
	select {
	case <-t.C:
		a.tview.QueueUpdateDraw(func() {
			a.pages.SwitchToPage("main")
		})
	case <-a.stopCh:
	}
}

func (a *App) buildSplash() tview.Primitive {
	logo := `[#a855f7]  █████╗ ███╗   ██╗███████╗
  ██╔══██╗████╗  ██║██╔════╝
  ███████║██╔██╗ ██║███████╗
  ██╔══██║██║╚██╗██║╚════██║
  ██║  ██║██║ ╚████║███████║
  ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝[-]

[#94a3b8]Agent Nervous System v0.8.0[-]

[#a855f7]Initializing components...[#94a3b8]
  audit-trail      [  OK  ]
  snapshot-engine  [  OK  ]
  mcp-proxy        [  OK  ]
  policy-engine    [  OK  ]
  identity-broker  [  OK  ][-]`

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText(logo), 0, 2, false).
		AddItem(tview.NewBox(), 0, 1, false)

	return flex
}

func (a *App) buildMainUI() tview.Primitive {
	header := a.buildHeader()

	resFlex := tview.NewFlex().
		AddItem(a.overview, 0, 1, false)

	midFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(resFlex, 12, 0, false).
		AddItem(a.chart, 0, 1, false)

	bottomFlex := tview.NewFlex().
		AddItem(a.auditLog, 0, 1, false).
		AddItem(a.policy, 0, 1, false)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 3, 0, false).
		AddItem(midFlex, 0, 3, false).
		AddItem(bottomFlex, 0, 3, false).
		AddItem(a.cmdBar.flex, 6, 0, false).
		AddItem(a.statusBar, 1, 0, false)

	mainFlex.SetBorder(false)
	return mainFlex
}

func (a *App) buildHeader() tview.Primitive {
	dots := ""
	for i := 0; i < 16; i++ {
		dots += "[#a855f7]■[-]  "
	}

	text := fmt.Sprintf("[#94a3b8]v0.8.0[-]  %s[#a855f7]AGENT NERVOUS SYSTEM[-]", dots)

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(text)
	tv.SetBackgroundColor(bgColor)
	return tv
}

func (a *App) dataLoop() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			a.tview.QueueUpdateDraw(func() {
				a.overview.refresh()
				a.chart.refresh()
				a.auditLog.refresh()
				a.policy.refresh()
				a.statusBar.refresh()
			})
		case <-a.stopCh:
			return
		}
	}
}

func (a *App) triggerSnapshot() {
	ev := providers.AuditEvent{
		Timestamp: time.Now(),
		Component: providers.SnapshotEngine,
		EventType: providers.EventSnapshot,
		Hash:      fmt.Sprintf("%x", time.Now().UnixNano())[:12],
	}
	a.auditLog.injectEvent(ev)
}

func fmtDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
