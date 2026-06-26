package dashboard

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

const sparklineLen = 60

type historyRing struct {
	buf  [sparklineLen]float64
	idx  int
	full bool
}

func (r *historyRing) push(v float64) {
	r.buf[r.idx] = v
	r.idx = (r.idx + 1) % sparklineLen
	if r.idx == 0 {
		r.full = true
	}
}

func (r *historyRing) samples() []float64 {
	n := r.idx
	if r.full {
		n = sparklineLen
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		pos := (r.idx - n + i) % sparklineLen
		if pos < 0 {
			pos += sparklineLen
		}
		out[i] = r.buf[pos]
	}
	return out
}

// spark renders a 6-char mini trend line from samples, normalized to [0,1].
func spark(samples []float64) string {
	if len(samples) < 2 {
		return "[#94a3b8]······[-]"
	}
	min, max := math.MaxFloat64, -math.MaxFloat64
	for _, v := range samples {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	if span < 1 {
		span = 1
	}

	chars := 6
	step := float64(len(samples)) / float64(chars)
	var b strings.Builder
	for i := 0; i < chars; i++ {
		si := int(float64(i) * step)
		if si >= len(samples) {
			si = len(samples) - 1
		}
		norm := (samples[si] - min) / span
		clr := "#2ecc71"
		if norm > 0.8 {
			clr = "#e74c3c"
		} else if norm > 0.55 {
			clr = "#f59e0b"
		}
		var ch string
		switch {
		case norm < 0.15:
			ch = "⡀"
		case norm < 0.30:
			ch = "⢀"
		case norm < 0.45:
			ch = "⠠"
		case norm < 0.60:
			ch = "⣀"
		case norm < 0.75:
			ch = "⣄"
		default:
			ch = "⣿"
		}
		b.WriteString(fmt.Sprintf("[%s]%s[-]", clr, ch))
	}
	return b.String()
}

type overviewPanel struct {
	*tview.TextView
	prov      providers.DashboardProvider
	mu        sync.Mutex
	last      providers.ComponentStats
	cpuHist   historyRing
	memHist   historyRing
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
	p.sample()
	p.render()
	return p
}

// sample fetches fresh hardware data and pushes to history rings.
func (p *overviewPanel) sample() {
	s := p.prov.Stats()
	p.mu.Lock()
	p.last = s
	p.cpuHist.push(s.CPU.UsagePct)
	p.memHist.push(s.Mem.Pct)
	p.mu.Unlock()
}

func (p *overviewPanel) refresh() {
	p.render()
}

func bar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct * float64(width) / 100)

	var b strings.Builder
	for i := 0; i < width; i++ {
		pos := float64(i) / float64(width) * 100
		var color string
		switch {
		case pos < 40:
			color = "#2ecc71"
		case pos < 60:
			color = "#74d14c"
		case pos < 70:
			color = "#a8c83a"
		case pos < 80:
			color = "#d4b82b"
		case pos < 85:
			color = "#e8a220"
		case pos < 90:
			color = "#f08718"
		case pos < 94:
			color = "#f06814"
		default:
			color = "#e74c3c"
		}
		if i < filled {
			b.WriteString(fmt.Sprintf("[%s]█[-]", color))
		} else {
			b.WriteString("[#1e2930]░[-]")
		}
	}
	return b.String()
}

func (p *overviewPanel) render() {
	p.mu.Lock()
	s := p.last
	p.mu.Unlock()

	barWidth := 20

	cpuModel := s.CPU.Model
	if cpuModel == "" {
		cpuModel = fmt.Sprintf("%d cores", s.CPU.Cores)
	}

	cpuLabel := fmt.Sprintf("[#e2e8f0]CPU[-] [#94a3b8]%s[-]", cpuModel)
	cpuPct := s.CPU.UsagePct
	cpuBar := bar(cpuPct, barWidth)
	cpuSpark := spark(p.cpuHist.samples())
	cpuLine := fmt.Sprintf("  %s  %s\n  %s [#e2e8f0]%5.1f%%[-]\n", cpuLabel, cpuSpark, cpuBar, cpuPct)

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

	memPct := s.Mem.Pct
	memBar := bar(memPct, barWidth)
	memSpark := spark(p.memHist.samples())
	memLine := fmt.Sprintf("  [#e2e8f0]MEM[-]  %s\n  %s [#e2e8f0]%d/%d GB[-] [#94a3b8](%5.1f%%)[-]\n",
		memSpark, memBar, s.Mem.UsedGB, s.Mem.TotalGB, memPct)

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

	diskLine := ""
	if s.Disk.TotalGB > 0 {
		diskBar := bar(s.Disk.Pct, barWidth)
		diskLine = fmt.Sprintf("\n  [#e2e8f0]DISK[-]\n  %s [#e2e8f0]%d/%d GB[-] [#94a3b8](%5.1f%%)[-]  [#94a3b8]R:%s W:%s[-]\n",
			diskBar, s.Disk.UsedGB, s.Disk.TotalGB, s.Disk.Pct,
			fmt.Sprintf("%.1fMB/s", s.Disk.ReadSpeedMBs),
			fmt.Sprintf("%.1fMB/s", s.Disk.WriteSpeedMBs))
	}

	netLine := fmt.Sprintf("\n  [#e2e8f0]NET[-]\n  [#94a3b8]▼ %.1f MB/s[-]  [#94a3b8]▲ %.1f MB/s[-]  [#1a6b3a]%s[-]\n",
		s.Net.SpeedInMBs, s.Net.SpeedOutMBs,
		fmt.Sprintf("total: %.0f↓/%.0f↑ MB", s.Net.BytesInMB, s.Net.BytesOutMB))

	procLines := ""
	if len(s.Procs) > 0 {
		procLines += "\n  [#e2e8f0]TOP PROCESSES[-]\n"
		for i, p := range s.Procs {
			if i >= 5 {
				break
			}
			procBar := bar(p.CPU, 10)
			procLines += fmt.Sprintf("  [#94a3b8]%-16s[-] %s [#e2e8f0]%5.1f%%[-] [#94a3b8]%d MB[-]\n",
				trunc(p.Name, 16), procBar, p.CPU, p.MemMB)
		}
	}

	text := cpuLine + coreLines + memLine + diskLine + netLine + procLines + gpuLines
	p.SetText(strings.TrimRight(text, "\n"))
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
