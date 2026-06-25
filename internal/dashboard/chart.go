package dashboard

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

var (
	compColors = map[providers.Component]string{
		providers.AuditTrail:     "#2ecc71",
		providers.SnapshotEngine: "#3498db",
		providers.MCPProxy:       "#f59e0b",
		providers.PolicyEngine:   "#ff6b6b",
		providers.IdentityBroker: "#9b59b6",
	}
	compOrder = []providers.Component{
		providers.IdentityBroker,
		providers.PolicyEngine,
		providers.SnapshotEngine,
		providers.AuditTrail,
		providers.MCPProxy,
	}
	shades = []string{" ", "░", "▒", "▓", "█"}
)

type chartPanel struct {
	*tview.TextView
	prov      providers.DashboardProvider
	mu        sync.Mutex
	animRatio float64
	timeRange int
	compType  int
}

func newChartPanel(prov providers.DashboardProvider) *chartPanel {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(false)
	tv.SetBorder(true).
		SetBorderColor(borderColor).
		SetTitle("[#2ecc71]BREAKDOWN[-]").
		SetTitleColor(primaryColor).
		SetBackgroundColor(bgColor)
	tv.SetTitleAlign(tview.AlignLeft)

	p := &chartPanel{
		TextView:  tv,
		prov:      prov,
		animRatio: 1.0,
		timeRange: 2,
		compType:  0,
	}
	p.render()
	return p
}

func (p *chartPanel) refresh() {
	p.render()
}

func (p *chartPanel) render() {
	p.mu.Lock()
	ratio := p.animRatio
	tr := p.timeRange
	p.mu.Unlock()

	data := p.prov.ChartData()

	ranges := []string{"1 hour", "6 hours", "1 day", "7 days"}
	timeLabel := ranges[tr%len(ranges)]

	compLabels := []string{"Type: all"}
	for _, c := range compOrder {
		compLabels = append(compLabels, fmt.Sprintf("Type: %s", c))
	}
	compLabel := compLabels[p.compType%len(compLabels)]

	selectors := fmt.Sprintf("[#94a3b8][ [#e2e8f0]%s[-] ▾ ]  [ [#94a3b8]%s[-] ▾ ][-]\n\n",
		timeLabel,
		compLabel,
	)

	legend := ""
	for _, c := range compOrder {
		legend += fmt.Sprintf("[%s]■[-] [#94a3b8]%s[-]  ", compColors[c], c)
	}

	barChart := p.buildChart(data, ratio)

	text := selectors + legend + "\n" + barChart
	p.SetText(text)
}

func (p *chartPanel) buildChart(data []providers.ChartDataPoint, ratio float64) string {
	if len(data) == 0 {
		return "[#94a3b8]  no data[-]"
	}

	maxVal := 0.0
	for _, pt := range data {
		total := 0.0
		for _, v := range pt.Values {
			total += v
		}
		if total > maxVal {
			maxVal = total
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	availH := 8
	scale := float64(availH) / maxVal * ratio

	ncols := 24
	if len(data) < ncols {
		ncols = len(data)
	}

	var b strings.Builder
	for row := availH - 1; row >= 0; row-- {
		threshold := float64(row+1) / scale
		b.WriteString("  ")
		for col := len(data) - ncols; col < len(data); col++ {
			pt := data[col]
			total := 0.0
			for _, c := range compOrder {
				total += pt.Values[c]
			}
			if total < 0.5 {
				b.WriteString("[#1a1a2e]░[-]")
				continue
			}
			accum := 0.0
			drawn := false
			for i := len(compOrder) - 1; i >= 0; i-- {
				c := compOrder[i]
				v := pt.Values[c]
				if v < 0.5 {
					continue
				}
				accum += v
				if accum >= threshold {
					shadeI := 4
					if v < 5 {
						shadeI = 1
					} else if v < 10 {
						shadeI = 2
					} else if v < 20 {
						shadeI = 3
					}
					s := shades[shadeI]
					clr := compColors[c]
					b.WriteString(fmt.Sprintf("[%s]%s[-]", clr, s))
					drawn = true
					break
				}
			}
			if !drawn {
				b.WriteString("[#1a1a2e]░[-]")
			}
		}
		if row > 0 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (p *chartPanel) animateBars(app *tview.Application) {
	tick := time.NewTicker(30 * time.Millisecond)
	defer tick.Stop()
	step := 0
	for step < 20 {
		select {
		case <-tick.C:
			step++
			p.mu.Lock()
			p.animRatio = float64(step) / 20.0
			if p.animRatio > 1.0 {
				p.animRatio = 1.0
			}
			p.mu.Unlock()
			app.QueueUpdateDraw(func() {
				p.render()
			})
		}
	}
	p.mu.Lock()
	p.animRatio = 1.0
	p.mu.Unlock()
}
