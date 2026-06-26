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

type colorTheme struct {
	bg          tcell.Color
	primary     tcell.Color
	secondary   tcell.Color
	dim         tcell.Color
	muted       tcell.Color
	fg          tcell.Color
	dimText     tcell.Color
	border      tcell.Color
	alert       tcell.Color
	success     tcell.Color
}

var themes = []colorTheme{
	// Dark (Default) — emerald on true black
	{
		bg:        tcell.NewRGBColor(0x0A, 0x0A, 0x0F),
		primary:   tcell.NewRGBColor(0x2E, 0xCC, 0x71),
		secondary: tcell.NewRGBColor(0x27, 0xAE, 0x60),
		dim:       tcell.NewRGBColor(0x1A, 0x6B, 0x3A),
		muted:     tcell.NewRGBColor(0x56, 0xD9, 0x8A),
		fg:        tcell.NewRGBColor(0xE2, 0xE8, 0xF0),
		dimText:   tcell.NewRGBColor(0x94, 0xA3, 0xB8),
		border:    tcell.NewRGBColor(0x1E, 0x29, 0x30),
		alert:     tcell.NewRGBColor(0xFF, 0x6B, 0x6B),
		success:   tcell.NewRGBColor(0x2E, 0xCC, 0x71),
	},
	// Light — slate on white
	{
		bg:        tcell.NewRGBColor(0xF1, 0xF5, 0xF9),
		primary:   tcell.NewRGBColor(0x16, 0xA3, 0x4A),
		secondary: tcell.NewRGBColor(0x1D, 0x8C, 0x3F),
		dim:       tcell.NewRGBColor(0xBB, 0xF7, 0xD0),
		muted:     tcell.NewRGBColor(0x22, 0xC5, 0x5E),
		fg:        tcell.NewRGBColor(0x1E, 0x29, 0x30),
		dimText:   tcell.NewRGBColor(0x64, 0x74, 0x8B),
		border:    tcell.NewRGBColor(0xCB, 0xD5, 0xE1),
		alert:     tcell.NewRGBColor(0xDC, 0x26, 0x26),
		success:   tcell.NewRGBColor(0x16, 0xA3, 0x4A),
	},
	// High contrast — bright emerald on black
	{
		bg:        tcell.NewRGBColor(0x00, 0x00, 0x00),
		primary:   tcell.NewRGBColor(0x00, 0xFF, 0x41),
		secondary: tcell.NewRGBColor(0x00, 0xE0, 0x38),
		dim:       tcell.NewRGBColor(0x00, 0x5C, 0x17),
		muted:     tcell.NewRGBColor(0x00, 0xFF, 0x88),
		fg:        tcell.NewRGBColor(0xFF, 0xFF, 0xFF),
		dimText:   tcell.NewRGBColor(0xAA, 0xCC, 0xEE),
		border:    tcell.NewRGBColor(0x33, 0x33, 0x33),
		alert:     tcell.NewRGBColor(0xFF, 0x33, 0x33),
		success:   tcell.NewRGBColor(0x00, 0xFF, 0x41),
	},
}

var currentTheme int

// Package-level vars used by all panels — updated on theme switch.
var (
	bgColor       tcell.Color
	primaryColor  tcell.Color
	secondaryColor tcell.Color
	dimColor      tcell.Color
	mutedColor    tcell.Color
	foreground    tcell.Color
	dimText       tcell.Color
	borderColor   tcell.Color
	alertColor    tcell.Color
	successColor  tcell.Color
)

func init() {
	applyTheme(0)
}

func applyTheme(idx int) {
	currentTheme = idx
	t := themes[idx]
	bgColor = t.bg
	primaryColor = t.primary
	secondaryColor = t.secondary
	dimColor = t.dim
	mutedColor = t.muted
	foreground = t.fg
	dimText = t.dimText
	borderColor = t.border
	alertColor = t.alert
	successColor = t.success
}

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
	alertBar  *tview.TextView
	alertTimer *time.Timer
	realProv  *providers.RealProvider
	tiled     *tiledLayout

	currentTab   int
	stopCh       chan struct{}
	refreshSec   int
}

func NewApp(refreshInterval int) *App {
	if refreshInterval < 1 {
		refreshInterval = 3
	}

	var realProv *providers.RealProvider
	prov := providers.DashboardProvider(providers.NewRealProvider())
	if rp, ok := prov.(*providers.RealProvider); ok {
		realProv = rp
	}
	if _, err := daemon.Dial(); err != nil {
		prov = providers.NewMockProvider()
	}

	overview := newOverviewPanel(prov)
	chart := newChartPanel(prov)
	tiled := newTiledLayout(prov)

	a := &App{
		tview:       tview.NewApplication(),
		provider:    prov,
		realProv:    realProv,
		pages:       tview.NewPages(),
		currentTab:  0,
		stopCh:      make(chan struct{}),
		refreshSec:  refreshInterval,
		overview:    overview,
		tiled:       tiled,
	}

	auditLog := newAuditLogPanel(prov, a.tview)
	policy := newPolicyPanel(prov)
	statusBar := newStatusBarPanel(prov)

	a.chart = chart
	a.auditLog = auditLog
	a.policy = policy
	a.statusBar = statusBar

	a.alertBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	a.alertBar.SetBackgroundColor(tcell.NewRGBColor(0x7F, 0x1F, 0x1F))
	a.alertBar.SetText("")

	a.tabPages = tview.NewPages()
	a.tabPages.AddPage("overview", a.tiled, true, true)
	a.tabPages.AddPage("chart", a.chart, true, false)
	a.tabPages.AddPage("audit", a.auditLog, true, false)
	a.tabPages.AddPage("policy", a.policy, true, false)
	a.tabBar = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	a.tabBar.SetBackgroundColor(bgColor)
	a.renderTabBar()

	a.cmdBar = newCommandBar(a.tview, prov)

	return a
}

func (a *App) showAlert(msg string) {
	a.alertBar.SetText(fmt.Sprintf("[#ffffff] %s [-]", msg))
	if a.alertTimer != nil {
		a.alertTimer.Stop()
	}
	a.alertTimer = time.AfterFunc(8*time.Second, func() {
		a.tview.QueueUpdateDraw(func() {
			a.alertBar.SetText("")
		})
	})
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
			case 't', 'T':
				a.cycleTheme()
				return nil
			case 'o', 'O':
				a.showSettings()
				return nil
			case 'i', 'I':
				a.tiled.toggleSidebar()
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

	return a.tview.SetRoot(a.pages, true).EnableMouse(true).Run()
}

func (a *App) cycleTheme() {
	next := (currentTheme + 1) % len(themes)
	applyTheme(next)

	a.tview.QueueUpdateDraw(func() {
		a.overview.SetBackgroundColor(bgColor)
		a.overview.SetBorderColor(borderColor)
		a.overview.SetTitleColor(primaryColor)
		a.chart.SetBackgroundColor(bgColor)
		a.chart.SetBorderColor(borderColor)
		a.chart.SetTitleColor(primaryColor)
		a.auditLog.SetBackgroundColor(bgColor)
		a.auditLog.SetBorderColor(borderColor)
		a.auditLog.SetTitleColor(primaryColor)
		a.policy.SetBackgroundColor(bgColor)
		a.policy.SetBorderColor(borderColor)
		a.policy.SetTitleColor(primaryColor)
		a.tabBar.SetBackgroundColor(bgColor)
		a.statusBar.SetBackgroundColor(tcell.NewRGBColor(0x1E, 0x29, 0x30))
		a.cmdBar.input.SetBackgroundColor(bgColor)
		a.cmdBar.input.SetFieldTextColor(foreground)
		a.cmdBar.output.SetBackgroundColor(bgColor)
		a.alertBar.SetBackgroundColor(alertColor)

		themeNames := []string{"dark", "light", "high-contrast"}
		a.showAlert(fmt.Sprintf("Theme switched to %s", themeNames[next]))

		a.renderTabBar()
		a.overview.refresh()
		a.chart.refresh()
		a.auditLog.refresh()
		a.policy.refresh()
		a.statusBar.refresh()
		s := a.provider.Stats()
		ev := a.provider.RecentEvents()
		rules := a.provider.ActiveRules()
		a.tiled.updateAll(s, ev, rules)
	})
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

	// Alert bar is hidden by default (0 height) unless a message is set
	alertFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	alertFlex.AddItem(a.alertBar, 0, 0, false)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tabBar, 1, 0, false).
		AddItem(a.tabPages, 0, 1, false).
		AddItem(alertFlex, 0, 0, false).
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
		SetText("[#334155]Tab/← → switch │ o settings │ i sidebar │ t theme │ q quit[-]")
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
	tick := time.NewTicker(time.Duration(a.refreshSec) * time.Second)
	defer tick.Stop()
	reconnTick := time.NewTicker(10 * time.Second)
	defer reconnTick.Stop()
	for {
		select {
		case <-tick.C:
			a.provider.RefreshHardware()
			a.overview.sample()

			// Fetch blocking data in background before QueueUpdateDraw
			events := a.provider.RecentEvents()
			s := a.provider.Stats()
			rules := a.provider.ActiveRules()

			a.tview.QueueUpdateDraw(func() {
				a.overview.refresh()
				a.chart.refresh()
				a.auditLog.setEvents(events)
				a.policy.setData(s, rules)
				a.statusBar.refresh()
				a.tiled.updateAll(s, events, rules)
				a.checkAlerts()
			})
		case <-reconnTick.C:
			if a.realProv != nil {
				if _, ok := a.provider.(*providers.MockProvider); ok {
					if a.realProv.TryReconnect() {
						a.provider = providers.DashboardProvider(a.realProv)
						a.provider.RefreshHardware()
						a.overview.sample()

						ev := a.provider.RecentEvents()
						st := a.provider.Stats()
						rl := a.provider.ActiveRules()

						a.tview.QueueUpdateDraw(func() {
							a.showAlert("Reconnected to daemon")
							a.overview.refresh()
							a.chart.refresh()
							a.auditLog.setEvents(ev)
							a.policy.setData(st, rl)
							a.statusBar.refresh()
						})
					}
				}
			}
		case <-a.stopCh:
			return
		}
	}
}

func (a *App) checkAlerts() {
	s := a.provider.Stats() // Stats() is a fast mutex read
	if s.Violations24h > 20 {
		a.alertBar.SetBackgroundColor(tcell.NewRGBColor(0x7F, 0x1F, 0x1F))
		a.showAlert(fmt.Sprintf("ALERT: %d policy violations in the last 24 hours", s.Violations24h))
	} else if s.MCPStatus == "DEGRADED" || s.MCPStatus == "DOWN" {
		a.alertBar.SetBackgroundColor(tcell.NewRGBColor(0x7F, 0x4F, 0x0F))
		a.showAlert(fmt.Sprintf("WARNING: MCP proxy is %s", s.MCPStatus))
	}
}

func fmtDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", h, m)
}
