package ui

import (
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/poller"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	tv       *tview.Application
	pages    *tview.Pages
	root     *tview.Flex

	header     *Header
	statusbar  *StatusBar
	panelChain     *ChainPanel
	panelSnaps     *SnapshotPanel
	panelPolicy    *PolicyPanel
	panelTokens    *TokenPanel
	panelMCP       *MCPPanel

	poller *poller.Poller
	focus  int
	paused bool
	demo   bool
	stopOnce sync.Once
}

func NewApp(p *poller.Poller, demo bool) *App {
	a := &App{
		tv:      tview.NewApplication(),
		pages:   tview.NewPages(),
		poller:  p,
		focus:   0,
		paused:  false,
		demo:    demo,
	}

	a.header = NewHeader()
	a.statusbar = NewStatusBar()
	a.panelChain = NewChainPanel()
	a.panelSnaps = NewSnapshotPanel()
	a.panelPolicy = NewPolicyPanel()
	a.panelTokens = NewTokenPanel()
	a.panelMCP = NewMCPPanel()

	// Middle row: Panel A + Panel B
	middleRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.panelChain.Flex, 0, 55, false).
		AddItem(a.panelSnaps.Flex, 0, 45, false)

	// Bottom row: Panel C + D + E
	bottomRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.panelPolicy.Flex, 0, 33, false).
		AddItem(a.panelTokens.Flex, 0, 34, false).
		AddItem(a.panelMCP.Flex, 0, 33, false)

	a.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.header.Flex, 3, 0, false).
		AddItem(middleRow, 0, 40, false).
		AddItem(bottomRow, 0, 35, false).
		AddItem(a.statusbar, 1, 0, false)

	a.root.SetBackgroundColor(ColorBG)

	a.pages.AddPage("main", a.root, true, true)

	a.tv.SetInputCapture(a.globalKeys)
	a.focusPanel(0)

	return a
}

func (a *App) focusPanel(idx int) {
	if idx < 0 || idx > 4 {
		return
	}
	a.focus = idx
	panels := []struct {
		panel interface{ SetFocusColor(bool) }
	}{
		{a.panelChain},
		{a.panelSnaps},
		{a.panelPolicy},
		{a.panelTokens},
		{a.panelMCP},
	}
	for i, p := range panels {
		p.panel.SetFocusColor(i == idx)
	}
}

func (a *App) focusPanelByRune(r rune) {
	m := map[rune]int{'1': 0, '2': 1, '3': 2, '4': 3, '5': 4}
	if idx, ok := m[r]; ok {
		a.focusPanel(idx)
	}
}

func (a *App) togglePause() {
	a.paused = !a.paused
	if a.paused {
		a.poller.Pause()
	} else {
		a.poller.Resume()
	}
	a.statusbar.SetPaused(a.paused)
}

func (a *App) triggerRefresh() {
	a.poller.ForceRefresh()
}

func (a *App) modalOpen() bool {
	name, _ := a.pages.GetFrontPage()
	return name != "main"
}

func (a *App) globalKeys(event *tcell.EventKey) *tcell.EventKey {
	if a.modalOpen() {
		switch event.Key() {
		case tcell.KeyEscape:
			a.pages.RemovePage("help")
			a.pages.RemovePage("detail")
			a.pages.SwitchToPage("main")
			a.tv.SetFocus(a.root)
			return nil
		}
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			a.pages.RemovePage("help")
			a.pages.RemovePage("detail")
			a.pages.SwitchToPage("main")
			a.tv.SetFocus(a.root)
			return nil
		}
		return event
	}

	switch event.Rune() {
	case 'q', 'Q':
		a.header.Stop()
		a.poller.Stop()
		a.stopOnce.Do(func() { a.tv.Stop() })
		return nil
	case '?':
		a.showHelp()
		return nil
	case 'r', 'R':
		a.triggerRefresh()
		return nil
	case 'p', 'P':
		a.togglePause()
		return nil
	case 'v', 'V':
		a.triggerRefresh()
		return nil
	case '1', '2', '3', '4', '5':
		a.focusPanelByRune(event.Rune())
		return nil
	case 't', 'T':
		if a.focus == 2 {
			pl := a.panelPolicy.SelectedPolicy()
			if pl != nil {
				a.poller.ForceRefresh()
			}
		}
		return nil
	case 'x', 'X':
		if a.focus == 3 {
			tk := a.panelTokens.SelectedToken()
			if tk != nil {
				a.poller.ForceRefresh()
			}
		}
		return nil
	case 'd', 'D':
		if a.focus == 1 {
			snap := a.panelSnaps.SelectedSnapshot()
			if snap != nil {
				a.poller.ForceRefresh()
			}
		}
		return nil
	}

	switch event.Key() {
	case tcell.KeyEnter:
		if a.focus == 0 {
			a.showReceiptDetail()
			return nil
		}
	case tcell.KeyTab:
		a.focusPanel((a.focus + 1) % 5)
		return nil
	case tcell.KeyBacktab:
		a.focusPanel((a.focus + 4) % 5)
		return nil
	}

	return event
}

func (a *App) Run() error {
	go a.listenPollerChannels()
	return a.tv.SetRoot(a.pages, true).SetFocus(a.root).Run()
}

func (a *App) listenPollerChannels() {
	for {
		select {
		case receipts := <-a.poller.ChainCh:
			a.tv.QueueUpdateDraw(func() { a.panelChain.Update(receipts) })
		case snaps := <-a.poller.SnapshotCh:
			a.tv.QueueUpdateDraw(func() { a.panelSnaps.Update(snaps) })
		case policies := <-a.poller.PolicyCh:
			a.tv.QueueUpdateDraw(func() { a.panelPolicy.Update(policies) })
		case tokens := <-a.poller.TokenCh:
			a.tv.QueueUpdateDraw(func() { a.panelTokens.Update(tokens) })
		case mcp := <-a.poller.MCPCh:
			a.tv.QueueUpdateDraw(func() { a.panelMCP.Update(mcp) })
		case status := <-a.poller.DaemonCh:
			a.tv.QueueUpdateDraw(func() { a.header.Update(status) })
		}
	}
}
