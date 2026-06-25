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
	primaryColor  = tcell.NewRGBColor(0x2E, 0xCC, 0x71)
	secondaryColor = tcell.NewRGBColor(0x27, 0xAE, 0x60)
	dimColor      = tcell.NewRGBColor(0x1A, 0x6B, 0x3A)
	mutedColor    = tcell.NewRGBColor(0x56, 0xD9, 0x8A)
	foreground    = tcell.NewRGBColor(0xE2, 0xE8, 0xF0)
	dimText       = tcell.NewRGBColor(0x94, 0xA3, 0xB8)
	borderColor   = tcell.NewRGBColor(0x1E, 0x29, 0x30)
	alertColor    = tcell.NewRGBColor(0xFF, 0x6B, 0x6B)
	successColor  = tcell.NewRGBColor(0x2E, 0xCC, 0x71)
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
		if ev.Key() == tcell.KeyCtrlC {
			a.tview.Stop()
			return nil
		}
		// Hotkeys only fire when the input field is empty
		if a.cmdBar.input.GetText() == "" {
			switch ev.Rune() {
			case 'q', 'Q':
				a.tview.Stop()
				return nil
			case 's', 'S':
				a.cmdBar.execute("snapshot take")
				return nil
			case '1':
				a.cmdBar.execute("status")
				return nil
			case '2':
				a.cmdBar.execute("chain --n 5")
				return nil
			case '3':
				a.cmdBar.execute("agents")
				return nil
			case '4':
				a.cmdBar.execute("verify --chain")
				return nil
			case 'h', 'H':
				a.cmdBar.showLocalHelp()
				return nil
			case 'c', 'C':
				a.cmdBar.outputs = nil
				a.cmdBar.output.SetText("")
				return nil
			}
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
			a.tview.SetFocus(a.cmdBar.input)
		})
	case <-a.stopCh:
	}
}

func (a *App) buildSplash() tview.Primitive {
	logo := `[#2ecc71]  █████╗ ███╗   ██╗███████╗
  ██╔══██╗████╗  ██║██╔════╝
  ███████║██╔██╗ ██║███████╗
  ██╔══██║██║╚██╗██║╚════██║
  ██║  ██║██║ ╚████║███████║
  ╚═╝  ╚═╝╚═╝  ╚═══╝╚══════╝[-]

[#94a3b8]Agent Nervous System v0.8.0[-]

[#2ecc71]Initializing components...[#94a3b8]
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
		dots += "[#2ecc71]■[-]  "
	}

	text := fmt.Sprintf("[#94a3b8]v0.8.0[-]  %s[#2ecc71]AGENT NERVOUS SYSTEM[-]", dots)

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(text)
	tv.SetBackgroundColor(bgColor)
	return tv
}

func (a *App) dataLoop() {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			a.provider.RefreshHardware()
			a.overview.sample()
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
