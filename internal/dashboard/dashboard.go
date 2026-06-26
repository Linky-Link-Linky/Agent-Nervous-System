package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
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
	tview    *tview.Application
	provider  providers.DashboardProvider
	pages    *tview.Pages
	tabPages *tview.Pages
	tabBar   *tview.TextView

	overview  *overviewPanel
	chart     *chartPanel
	auditLog  *auditLogPanel
	policy    *policyPanel
	statusBar *statusBarPanel
	cmdBar    *commandBar

	currentTab int
	stopCh     chan struct{}
}

func NewApp() *App {
	prov := providers.DashboardProvider(providers.NewRealProvider())
	// If daemon is not running, fall back to mock provider for demonstration
	if _, err := daemon.Dial(); err != nil {
		prov = providers.NewMockProvider()
	}

	overview := newOverviewPanel(prov)
	chart := newChartPanel(prov)
	auditLog := newAuditLogPanel(prov)
	policy := newPolicyPanel(prov)
	statusBar := newStatusBarPanel(prov)

	a := &App{
		tview:      tview.NewApplication(),
		provider:   prov,
		pages:      tview.NewPages(),
		currentTab: 0,
		stopCh:     make(chan struct{}),
		overview:   overview,
		chart:      chart,
		auditLog:   auditLog,
		policy:     policy,
		statusBar:  statusBar,
	}

	a.tabPages = tview.NewPages()
	a.tabPages.AddPage("overview", a.overview, true, true)
	a.tabPages.AddPage("chart", a.chart, true, false)
	a.tabPages.AddPage("audit", a.auditLog, true, false)
	a.tabPages.AddPage("policy", a.policy, true, false)
	a.tabBar = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	a.tabBar.SetBackgroundColor(bgColor)
	a.renderTabBar()

	a.cmdBar = newCommandBar(a.tview, prov)

	return a
}

func (a *App) Run() error {
	a.tview.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyCtrlC {
			a.tview.Stop()
			return nil
		}
		if a.cmdBar.input.GetText() == "" {
			switch ev.Key() {
			case tcell.KeyTab, tcell.KeyBacktab, tcell.KeyLeft, tcell.KeyRight:
				a.handleTabNav(ev.Key())
				return nil
			}
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
	tabBar := tview.NewFlex().
		AddItem(a.tabBar, 0, 1, false).
		AddItem(a.buildTabHint(), 0, 1, false)
	tabBar.SetBackgroundColor(bgColor)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tabBar, 1, 0, false).
		AddItem(a.tabPages, 0, 1, false).
		AddItem(a.cmdBar.flex, 6, 0, false).
		AddItem(a.statusBar, 1, 0, false)

	mainFlex.SetBorder(false)
	return mainFlex
}

var tabNames = []string{"overview", "chart", "audit", "policy"}

func (a *App) renderTabBar() {
	parts := make([]string, len(tabNames))
	for i, name := range tabNames {
		if i == a.currentTab {
			parts[i] = fmt.Sprintf("[#2ecc71]● %s[-]", name)
		} else {
			parts[i] = fmt.Sprintf("[#334155]○ %s[-]", name)
		}
	}
	a.tabBar.SetText(strings.Join(parts, "  [#334155]│[-]  "))
}

func (a *App) buildTabHint() tview.Primitive {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText("[#334155]Tab/← → switch │ Esc clear │ q quit[-]")
	tv.SetBackgroundColor(bgColor)
	return tv
}

func (a *App) switchToTab(idx int) {
	if idx < 0 {
		idx = len(tabNames) - 1
	}
	if idx >= len(tabNames) {
		idx = 0
	}
	a.currentTab = idx
	a.renderTabBar()
	a.tabPages.SwitchToPage(tabNames[idx])
}

func (a *App) handleTabNav(key tcell.Key) {
	if a.cmdBar.input.GetText() != "" {
		return
	}
	switch key {
	case tcell.KeyTab:
		a.switchToTab(a.currentTab + 1)
	case tcell.KeyBacktab:
		a.switchToTab(a.currentTab - 1)
	case tcell.KeyLeft:
		a.switchToTab(a.currentTab - 1)
	case tcell.KeyRight:
		a.switchToTab(a.currentTab + 1)
	}
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

func fmtDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
