package dashboard

import (
	"fmt"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type statusBarPanel struct {
	*tview.TextView
	prov providers.DashboardProvider
}

func newStatusBarPanel(prov providers.DashboardProvider) *statusBarPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	tv.SetBackgroundColor(tcell.NewRGBColor(0x1E, 0x29, 0x30))

	p := &statusBarPanel{TextView: tv, prov: prov}
	p.refresh()
	return p
}

func (p *statusBarPanel) refresh() {
	s := p.prov.Stats()

	mcpClr := "#2ecc71"
	if s.MCPStatus == "DEGRADED" {
		mcpClr = "#ff6b6b"
	}
	brokerClr := "#94a3b8"
	switch s.BrokerStatus {
	case "ACTIVE":
		brokerClr = "#2ecc71"
	case "EXPIRED":
		brokerClr = "#ff6b6b"
	}

	text := fmt.Sprintf(
		"[#e2e8f0]UPTIME[-]: [#94a3b8]%s[-]  [#e2e8f0]|[-]  [#e2e8f0]ACTIVE AGENTS[-]: [#94a3b8]%d[-]  [#e2e8f0]|[-]  [#e2e8f0]LAST SNAPSHOT[-]: [#94a3b8]%s[-]  [#e2e8f0]|[-]  [#e2e8f0]MCP PROXY[-]: [%s]%s[-]  [#e2e8f0]|[-]  [#e2e8f0]IDENTITY BROKER[-]: [%s]%s[-]",
		fmtDuration(s.Uptime),
		s.ActiveAgents,
		s.LastSnapshot.Format("15:04:05"),
		mcpClr, s.MCPStatus,
		brokerClr, s.BrokerStatus,
	)
	p.SetText(text)
}
