package dashboard

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

type auditLogPanel struct {
	*tview.TextView
	prov   providers.DashboardProvider
	mu     sync.Mutex
	events []providers.AuditEvent
}

func newAuditLogPanel(prov providers.DashboardProvider) *auditLogPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetScrollable(true)
	tv.SetBorder(true).
		SetBorderColor(borderColor).
		SetTitle("[#2ecc71]AUDIT TRAIL[-]").
		SetTitleColor(primaryColor).
		SetBackgroundColor(bgColor)
	tv.SetTitleAlign(tview.AlignLeft)

	p := &auditLogPanel{TextView: tv, prov: prov}
	p.refresh()
	return p
}

func (p *auditLogPanel) refresh() {
	evs := p.prov.RecentEvents()
	p.mu.Lock()
	p.events = evs
	p.mu.Unlock()
	p.render()
}

func (p *auditLogPanel) injectEvent(ev providers.AuditEvent) {
	p.mu.Lock()
	p.events = append(p.events, ev)
	if len(p.events) > 200 {
		p.events = p.events[len(p.events)-200:]
	}
	p.mu.Unlock()
	p.render()
}

func (p *auditLogPanel) render() {
	p.mu.Lock()
	events := p.events
	p.mu.Unlock()

	var b strings.Builder
	start := 0
	if len(events) > 30 {
		start = len(events) - 30
	}
	for i := start; i < len(events); i++ {
		ev := events[i]
		ts := ev.Timestamp.Format("15:04:05")
		var color string
		switch ev.EventType {
		case providers.EventBlocked, providers.EventViolation, providers.EventExpired:
			color = "#ff6b6b"
		case providers.EventCommit, providers.EventSnapshot, providers.EventAllowed:
			color = "#2ecc71"
		default:
			color = "#94a3b8"
		}
		line := fmt.Sprintf("[#94a3b8]%s[-]  [#e2e8f0]%s[-]  [%s]%s[-]  [#1a6b3a]%s[-]\n",
			ts, ev.Component, color, ev.EventType, ev.Hash)
		b.WriteString(line)
	}
	p.SetText(b.String())
	p.ScrollToEnd()
}
