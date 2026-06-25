package dashboard

import (
	"fmt"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

type overviewPanel struct {
	*tview.TextView
	prov providers.DashboardProvider
}

func newOverviewPanel(prov providers.DashboardProvider) *overviewPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false)
	tv.SetBorder(true).
		SetBorderColor(borderColor).
		SetTitle("[#a855f7]RESOURCE OVERVIEW[-]").
		SetTitleColor(primaryColor).
		SetBackgroundColor(bgColor)
	tv.SetTitleAlign(tview.AlignLeft)

	p := &overviewPanel{TextView: tv, prov: prov}
	p.refresh()
	return p
}

func (p *overviewPanel) refresh() {
	s := p.prov.Stats()
	text := fmt.Sprintf(
		"[#e2e8f0]  [ GPU's: [#a855f7]%d[-] ]  [ [#94a3b8]Units[-]: [#a855f7]%d[-] ]\n"+
			"  [ [#94a3b8]A100s[-]: [#a855f7]%d[-] ]  [ [#94a3b8]H100s[-]: [#a855f7]%d[-] ]",
		s.GPUCount,
		s.Units,
		s.A100Count,
		s.H100Count,
	)
	p.SetText(text)
}
