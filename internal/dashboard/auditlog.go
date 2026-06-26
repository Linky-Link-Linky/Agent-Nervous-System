package dashboard

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

type auditLogPanel struct {
	*tview.Flex
	prov       providers.DashboardProvider
	mu         sync.Mutex
	events     []providers.AuditEvent
	filter     string
	filterInp  *tview.InputField
	eventView  *tview.TextView
}

func newAuditLogPanel(prov providers.DashboardProvider, app *tview.Application) *auditLogPanel {
	evView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false).
		SetScrollable(true)
	evView.SetBackgroundColor(bgColor)

	filterInp := tview.NewInputField().
		SetLabel("[#94a3b8]filter[-] ").
		SetPlaceholder("Search by component, event type, or hash...").
		SetFieldWidth(0)
	filterInp.SetBackgroundColor(bgColor)
	filterInp.SetPlaceholderTextColor(dimText)
	filterInp.SetFieldTextColor(foreground)
	filterInp.SetLabelColor(dimText)

	p := &auditLogPanel{
		Flex:      tview.NewFlex().SetDirection(tview.FlexRow),
		prov:      prov,
		eventView: evView,
		filterInp: filterInp,
	}

	filterInp.SetChangedFunc(func(text string) {
		p.mu.Lock()
		p.filter = strings.ToLower(text)
		p.mu.Unlock()
		p.render()
	})

	p.AddItem(filterInp, 1, 0, false)
	p.AddItem(evView, 0, 1, false)
	p.SetBorder(true).
		SetBorderColor(borderColor).
		SetTitle("[#2ecc71]AUDIT TRAIL[-]").
		SetTitleColor(primaryColor).
		SetBackgroundColor(bgColor)
	p.SetTitleAlign(tview.AlignLeft)

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
	filter := p.filter
	p.mu.Unlock()

	var b strings.Builder
	start := 0
	if len(events) > 200 {
		start = len(events) - 200
	}
	count := 0
	for i := start; i < len(events); i++ {
		ev := events[i]
		if filter != "" {
			lower := strings.ToLower(string(ev.Component) + " " + string(ev.EventType) + " " + ev.Hash)
			if !strings.Contains(lower, filter) {
				continue
			}
		}
		count++
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
	if filter != "" {
		header := fmt.Sprintf("[#94a3b8]Filter: \"%s\" — %d matches[-]\n\n", filter, count)
		p.eventView.SetText(header + b.String())
	} else {
		p.eventView.SetText(b.String())
	}
	p.eventView.ScrollToEnd()
}
