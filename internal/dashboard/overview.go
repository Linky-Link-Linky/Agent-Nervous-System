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
	gpuLine := ""
	if s.GPUCount > 0 {
		models := ""
		for i, m := range s.GPUModels {
			if i > 0 {
				models += ", "
			}
			models += m
		}
		gpuLine = fmt.Sprintf("  [#e2e8f0][ GPU[#94a3b8]:[#2ecc71] %dx[-] [#94a3b8]%s[-] ]\n", s.GPUCount, models)
	} else {
		gpuLine = "  [#e2e8f0][ GPU[#94a3b8]: [#94a3b8]none detected[-] ]\n"
	}
	cpuModel := s.CPUModel
	if cpuModel == "" {
		cpuModel = fmt.Sprintf("%d cores", s.CPUCores)
	}
	ramLine := fmt.Sprintf("[#e2e8f0]  [ RAM[#94a3b8]:[#2ecc71] %d GB[-] ]", s.TotalRAMGB)
	if s.UsedRAMGB > 0 {
		ramLine = fmt.Sprintf("[#e2e8f0]  [ RAM[#94a3b8]:[#2ecc71] %d/%d GB[-] ]", s.UsedRAMGB, s.TotalRAMGB)
	}
	text := fmt.Sprintf(
		"[#e2e8f0]  [ CPU[#94a3b8]:[#2ecc71] %s[-] ]  [ Usage[#94a3b8]: [#2ecc71]%.0f%%[-] ]\n"+
			ramLine+"\n"+
			gpuLine,
		cpuModel,
		s.CPUUsagePct,
	)
	p.SetText(text)
}
