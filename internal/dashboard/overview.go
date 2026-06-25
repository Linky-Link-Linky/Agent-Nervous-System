package dashboard

import (
	"fmt"
	"strings"

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
		SetRegions(false).
		SetScrollable(false)
	tv.SetBorder(true).
		SetBorderColor(borderColor).
		SetTitle("[#2ecc71]RESOURCE MONITOR[-]").
		SetTitleColor(primaryColor).
		SetBackgroundColor(bgColor)
	tv.SetTitleAlign(tview.AlignLeft)

	p := &overviewPanel{TextView: tv, prov: prov}
	p.refresh()
	return p
}

func bar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct * float64(width) / 100)
	remain := width - filled

	var color string
	switch {
	case pct < 50:
		color = "#2ecc71"
	case pct < 80:
		color = "#f59e0b"
	default:
		color = "#ff6b6b"
	}

	fill := color
	blocks := ""
	for i := 0; i < filled; i++ {
		blocks += "█"
	}
	for i := 0; i < remain; i++ {
		blocks += "░"
	}
	return fmt.Sprintf("[%s]%s[-]", fill, blocks)
}

func (p *overviewPanel) refresh() {
	s := p.prov.Stats()

	barWidth := 20

	// CPU Header
	cpuModel := s.CPU.Model
	if cpuModel == "" {
		cpuModel = fmt.Sprintf("%d cores", s.CPU.Cores)
	}

	// CPU total usage bar
	cpuLabel := fmt.Sprintf("[#e2e8f0]CPU[-] [#94a3b8]%s[-]", cpuModel)
	cpuPct := s.CPU.UsagePct
	cpuBar := bar(cpuPct, barWidth)
	cpuLine := fmt.Sprintf("  %s\n  %s [#e2e8f0]%5.1f%%[-]\n", cpuLabel, cpuBar, cpuPct)

	// Per-core bars (show up to 8 in a compact 4-wide grid)
	coreLines := ""
	for i := 0; i < len(s.CPU.PerCore) && i < 8; i++ {
		if i%2 == 0 {
			coreLines += "  "
		} else {
			coreLines += "      "
		}
		corePct := s.CPU.PerCore[i]
		coreBar := bar(corePct, 10)
		coreLines += fmt.Sprintf("[#94a3b8]C%d[-] %s [#e2e8f0]%5.1f%%[-]", i, coreBar, corePct)
		if i%2 == 1 || i == len(s.CPU.PerCore)-1 || i == 7 {
			coreLines += "\n"
		} else {
			coreLines += "  "
		}
	}

	// Memory bar
	memPct := s.Mem.Pct
	memBar := bar(memPct, barWidth)
	memLine := fmt.Sprintf("  [#e2e8f0]MEM[-]\n  %s [#e2e8f0]%d/%d GB[-] [#94a3b8](%5.1f%%)[-]\n",
		memBar, s.Mem.UsedGB, s.Mem.TotalGB, memPct)

	// GPU section
	gpuLines := ""
	if s.GPU.Count > 0 {
		for i, m := range s.GPU.Models {
			gpuLines += fmt.Sprintf("  [#e2e8f0]GPU %d[-] [#94a3b8]%s[-]\n", i+1, m)
			if s.GPU.UsagePct > 0 {
				gpuBar := bar(s.GPU.UsagePct, barWidth)
				gpuLines += fmt.Sprintf("  %s [#e2e8f0]%5.1f%%[-]\n", gpuBar, s.GPU.UsagePct)
			}
			if s.GPU.MemTotalMB > 0 {
				gpuMemPct := float64(s.GPU.MemUsedMB) / float64(s.GPU.MemTotalMB) * 100
				gpuMemBar := bar(gpuMemPct, barWidth)
				gpuLines += fmt.Sprintf("  [#94a3b8]VRAM[-] %s [#e2e8f0]%d/%d MB[-]\n", gpuMemBar, s.GPU.MemUsedMB, s.GPU.MemTotalMB)
			}
			if s.GPU.TempC > 0 {
				gpuLines += fmt.Sprintf("  [#94a3b8]Temp[-] [#e2e8f0]%.0f°C[-]\n", s.GPU.TempC)
			}
		}
	} else {
		gpuLines = "  [#94a3b8]GPU: none detected[-]\n"
	}

	text := cpuLine + coreLines + memLine + gpuLines
	p.SetText(strings.TrimRight(text, "\n"))
}
