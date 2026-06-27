package dashboard

import (
	"fmt"
	"strings"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

type policyPanel struct {
	*tview.TextView
	prov       providers.DashboardProvider
	cached     providers.ComponentStats
	cachedRules []providers.RuleEntry
}

func newPolicyPanel(prov providers.DashboardProvider) *policyPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false)
	tv.SetBorder(true).
		SetBorderColor(borderColor).
		SetTitle("[#2ecc71]POLICY ENGINE[-]").
		SetTitleColor(primaryColor).
		SetBackgroundColor(bgColor)
	tv.SetTitleAlign(tview.AlignLeft)

	p := &policyPanel{TextView: tv, prov: prov}
	p.refresh()
	return p
}

// refresh fetches from provider then renders (can block — call from background).
func (p *policyPanel) refresh() {
	s := p.prov.Stats()
	rules := p.prov.ActiveRules()
	p.cached = s
	p.cachedRules = rules
	p.renderWith(s, rules)
}

// setData stores pre-fetched data and renders (safe for QueueUpdateDraw).
func (p *policyPanel) setData(s providers.ComponentStats, rules []providers.RuleEntry) {
	p.cached = s
	p.cachedRules = rules
	p.renderWith(s, rules)
}

func (p *policyPanel) renderWith(s providers.ComponentStats, rules []providers.RuleEntry) {
	var b strings.Builder
	if s.ActiveRules == 0 && len(rules) == 0 {
		b.WriteString("[#94a3b8]Policy engine idle. Start the daemon with 'ans start' to enable policy enforcement.[-]")
		p.SetText(b.String())
		return
	}
	b.WriteString(fmt.Sprintf("[#94a3b8]Active Rules[-]: [#e2e8f0]%d[-]\n", s.ActiveRules))
	b.WriteString(fmt.Sprintf("[#94a3b8]Violations (24h)[-]: [#ff6b6b]%d[-]\n", s.Violations24h))
	b.WriteString(fmt.Sprintf("[#94a3b8]Last Enforcement[-]: [#e2e8f0]%s[-]\n\n", s.LastEnforcement.Format("15:04:05")))

	for i, r := range rules {
		if i >= 6 {
			break
		}
		if r.Verdict == "DENY" {
			b.WriteString(fmt.Sprintf("[#ff6b6b][ DENY  ][-]  [#94a3b8]%s[-]\n", r.Rule))
		} else {
			b.WriteString(fmt.Sprintf("[#2ecc71][ ALLOW ][-]  [#94a3b8]%s[-]\n", r.Rule))
		}
	}

	p.SetText(b.String())
}
