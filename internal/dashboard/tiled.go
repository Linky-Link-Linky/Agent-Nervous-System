package dashboard

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/rivo/tview"
)

type panelID int

const (
	panelCPU panelID = iota
	panelMEM
	panelDISK
	panelNET
	panelGPU
	panelPROC
	panelCount
)

var panelNames = map[panelID]string{
	panelCPU:  "CPU",
	panelMEM:  "MEM",
	panelDISK: "DISK",
	panelNET:  "NET",
	panelGPU:  "GPU",
	panelPROC: "PROC",
}

type resourcePanel struct {
	*tview.TextView
	id panelID
}

func newResourcePanel(id panelID) *resourcePanel {
	tv := tview.NewTextView().SetDynamicColors(true).SetRegions(false)
	tv.SetBorder(true).SetBorderColor(borderColor).SetBackgroundColor(bgColor)
	tv.SetTitleColor(primaryColor)
	tv.SetTitleAlign(tview.AlignLeft)
	return &resourcePanel{TextView: tv, id: id}
}

type tiledLayout struct {
	*tview.Flex
	mu       sync.Mutex
	panels   map[panelID]*resourcePanel
	visible  map[panelID]bool
	sidebar  *tview.Flex
	sideOpen bool
	grid     *tview.Grid
	prov     providers.DashboardProvider
}

func newTiledLayout(prov providers.DashboardProvider) *tiledLayout {
	tl := &tiledLayout{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		panels:   make(map[panelID]*resourcePanel),
		visible:  make(map[panelID]bool),
		prov:     prov,
		sideOpen: true,
	}

	for id := panelCPU; id < panelCount; id++ {
		tl.panels[id] = newResourcePanel(id)
		tl.visible[id] = true
	}

	// Fixed 2-column grid: rows = (panelCount+1)/2 = 3
	tl.grid = tview.NewGrid().
		SetRows(0, 0, 0).
		SetColumns(0, 0).
		SetBorders(false).
		SetGap(1, 1)

	gridOrder := []panelID{panelCPU, panelMEM, panelDISK, panelNET, panelGPU, panelPROC}
	for i, id := range gridOrder {
		row := i / 2
		col := i % 2
		tl.grid.AddItem(tl.panels[id], row, col, 1, 1, 0, 0, false)
	}

	tl.sidebar = tl.buildSidebar()

	mainFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tl.grid, 0, 1, false).
		AddItem(tl.sidebar, 20, 0, false)

	tl.AddItem(mainFlex, 0, 1, false)
	return tl
}

func (tl *tiledLayout) buildSidebar() *tview.Flex {
	side := tview.NewFlex().SetDirection(tview.FlexRow)
	side.SetBorder(true).SetBorderColor(borderColor).SetBackgroundColor(bgColor)
	side.SetTitle("[#2ecc71]CONFIG[-]").SetTitleColor(primaryColor).SetTitleAlign(tview.AlignLeft)

	for id := panelCPU; id < panelCount; id++ {
		cid := id
		cb := tview.NewCheckbox()
		cb.SetLabel(fmt.Sprintf(" %s", panelNames[cid]))
		cb.SetChecked(tl.visible[cid])
		cb.SetBackgroundColor(bgColor)
		cb.SetLabelColor(foreground)
		cb.SetFieldTextColor(primaryColor)
		cb.SetCheckedString("[#2ecc71]●[-]")
		cb.SetUncheckedString("[#334155]○[-]")
		cb.SetChangedFunc(func(checked bool) {
			tl.visible[cid] = checked
			if checked {
				tl.grid.AddItem(tl.panels[cid], cidToRow(cid, 2), cidToCol(cid, 2), 1, 1, 0, 0, false)
			} else {
				tl.grid.RemoveItem(tl.panels[cid])
			}
		})
		side.AddItem(cb, 1, 0, false)
	}

	side.AddItem(tview.NewBox().SetBackgroundColor(bgColor), 0, 1, false)
	hint := tview.NewTextView().SetDynamicColors(true).
		SetText("[#94a3b8]i toggle\nsidebar[-]")
	hint.SetBackgroundColor(bgColor)
	side.AddItem(hint, 2, 0, false)
	return side
}

func cidToRow(id panelID, cols int) int {
	return int(id) / cols
}
func cidToCol(id panelID, cols int) int {
	return int(id) % cols
}

func (tl *tiledLayout) toggleSidebar() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if tl.Flex.GetItemCount() == 0 {
		return
	}
	mainFlex, ok := tl.Flex.GetItem(0).(*tview.Flex)
	if !ok {
		return
	}
	if tl.sideOpen {
		mainFlex.RemoveItem(tl.sidebar)
	} else {
		mainFlex.AddItem(tl.sidebar, 20, 0, false)
	}
	tl.sideOpen = !tl.sideOpen
}

func barMini(pct float64, w int) string {
	// Compact bar using block chars
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct * float64(w) / 100)
	var b strings.Builder
	for i := 0; i < w; i++ {
		pos := float64(i) / float64(w) * 100
		var color string
		switch {
		case pos < 50:
			color = "#2ecc71"
		case pos < 75:
			color = "#d4b82b"
		default:
			color = "#e74c3c"
		}
		if i < filled {
			b.WriteString(fmt.Sprintf("[%s]▉[-]", color))
		} else {
			b.WriteString("[#1e2930]░[-]")
		}
	}
	return b.String()
}

func (tl *tiledLayout) updateAll(s providers.ComponentStats, evs []providers.AuditEvent, rules []providers.RuleEntry) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	if tl.visible[panelCPU] && tl.panels[panelCPU] != nil {
		p := tl.panels[panelCPU]
		var b strings.Builder
		cpuModel := s.CPU.Model
		if cpuModel == "" {
			cpuModel = fmt.Sprintf("%d cores", s.CPU.Cores)
		}
		b.WriteString(fmt.Sprintf("  [#e2e8f0]%s[-]\n", cpuModel))
		b.WriteString(fmt.Sprintf("  %s  [#e2e8f0]%5.1f%%[-]\n", barMini(s.CPU.UsagePct, 18), s.CPU.UsagePct))
		ncores := len(s.CPU.PerCore)
		if ncores > 8 {
			ncores = 8
		}
		if ncores > 0 {
			line := "  "
			for i := 0; i < ncores; i++ {
				line += fmt.Sprintf("[#94a3b8]C%d[-]%s ", i, barMini(s.CPU.PerCore[i], 3))
			}
			b.WriteString(line + "\n")
		}
		p.SetTitle(fmt.Sprintf("[#2ecc71]CPU %5.1f%%[-]", s.CPU.UsagePct))
		_ = p.SetText(b.String())
	}

	if tl.visible[panelMEM] && tl.panels[panelMEM] != nil {
		p := tl.panels[panelMEM]
		mb := barMini(s.Mem.Pct, 18)
		text := fmt.Sprintf("  %s\n  [#e2e8f0]%d/%d GB[-]  [#94a3b8](%5.1f%%)[-]\n", mb, s.Mem.UsedGB, s.Mem.TotalGB, s.Mem.Pct)
		p.SetTitle(fmt.Sprintf("[#2ecc71]MEM %d/%d GB[-]", s.Mem.UsedGB, s.Mem.TotalGB))
		_ = p.SetText(text)
	}

	if tl.visible[panelDISK] && tl.panels[panelDISK] != nil && s.Disk.TotalGB > 0 {
		p := tl.panels[panelDISK]
		db := barMini(s.Disk.Pct, 18)
		text := fmt.Sprintf("  %s\n  [#e2e8f0]%d/%d GB[-]  [#94a3b8](%5.1f%%)[-]\n", db, s.Disk.UsedGB, s.Disk.TotalGB, s.Disk.Pct)
		text += fmt.Sprintf("  [#94a3b8]R:%.0f  W:%.0f MB/s[-]\n", s.Disk.ReadSpeedMBs, s.Disk.WriteSpeedMBs)
		p.SetTitle(fmt.Sprintf("[#2ecc71]DISK %d/%d GB[-]", s.Disk.UsedGB, s.Disk.TotalGB))
		_ = p.SetText(text)
	}

	if tl.visible[panelNET] && tl.panels[panelNET] != nil {
		p := tl.panels[panelNET]
		text := fmt.Sprintf("\n  [#94a3b8]▼ %.0f MB/s[-]          [#94a3b8]▲ %.0f MB/s[-]\n\n", s.Net.SpeedInMBs, s.Net.SpeedOutMBs)
		text += fmt.Sprintf("  [#1a6b3a]%.0f ↓ / %.0f ↑ MB total[-]\n", s.Net.BytesInMB, s.Net.BytesOutMB)
		p.SetTitle("[#2ecc71]NET[-]")
		_ = p.SetText(text)
	}

	if tl.visible[panelGPU] && tl.panels[panelGPU] != nil && s.GPU.Count > 0 {
		p := tl.panels[panelGPU]
		var b strings.Builder
		for i, m := range s.GPU.Models {
			line := fmt.Sprintf("  [#e2e8f0]GPU%d[-] [#94a3b8]%s[-]", i+1, trunc(m, 18))
			b.WriteString(line + "\n")
			if s.GPU.UsagePct > 0 {
				b.WriteString(fmt.Sprintf("  %s  [#e2e8f0]%.0f%%[-]\n", barMini(s.GPU.UsagePct, 10), s.GPU.UsagePct))
			}
			if s.GPU.MemTotalMB > 0 {
				gpuMemPct := float64(s.GPU.MemUsedMB) / float64(s.GPU.MemTotalMB) * 100
				b.WriteString(fmt.Sprintf("  [#94a3b8]VRAM[-] %s  [#e2e8f0]%d/%d MB[-]",
					barMini(gpuMemPct, 8), s.GPU.MemUsedMB, s.GPU.MemTotalMB))
			}
			if s.GPU.TempC > 0 {
				b.WriteString(fmt.Sprintf("  [#94a3b8]%.0f°C[-]", s.GPU.TempC))
			}
			b.WriteString("\n")
		}
		p.SetTitle("[#2ecc71]GPU[-]")
		_ = p.SetText(b.String())
	}

	if tl.visible[panelPROC] && tl.panels[panelPROC] != nil && len(s.Procs) > 0 {
		p := tl.panels[panelPROC]
		var b strings.Builder
		for i, pr := range s.Procs {
			if i >= 5 {
				break
			}
			b.WriteString(fmt.Sprintf("  [#94a3b8]%-14s[-] %s  [#e2e8f0]%5.1f%%[-]  [#94a3b8]%d MB[-]\n",
				trunc(pr.Name, 14), barMini(pr.CPU, 8), pr.CPU, pr.MemMB))
		}
		p.SetTitle("[#2ecc71]PROCESSES[-]")
		_ = p.SetText(b.String())
	}
}

// --- Settings modal ---

func (a *App) showSettings() {
	form := tview.NewForm()
	form.SetBackgroundColor(bgColor)
	form.SetBorder(true).SetBorderColor(primaryColor)
	form.SetTitle("[#2ecc71] SETTINGS [-]").
		SetTitleColor(primaryColor).SetTitleAlign(tview.AlignLeft)

	for id := panelCPU; id < panelCount; id++ {
		cid := id
		cbLabel := fmt.Sprintf("Show %s", panelNames[id])
		form.AddCheckbox(cbLabel, a.tiled.visible[id], func(checked bool) {
			a.tiled.visible[cid] = checked
			if checked {
				a.tiled.grid.AddItem(a.tiled.panels[cid], cidToRow(cid, 2), cidToCol(cid, 2), 1, 1, 0, 0, false)
			} else {
				a.tiled.grid.RemoveItem(a.tiled.panels[cid])
			}
		})
	}

	form.AddButton("Close", func() {
		a.pages.RemovePage("settings")
		a.pages.SwitchToPage("main")
		a.tview.SetFocus(a.cmdBar.input)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox().SetBackgroundColor(bgColor), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox().SetBackgroundColor(bgColor), 0, 1, false).
			AddItem(form, 36, 0, true).
			AddItem(tview.NewBox().SetBackgroundColor(bgColor), 0, 1, false),
			0, 1, false).
		AddItem(tview.NewBox().SetBackgroundColor(bgColor), 0, 1, false)

	a.pages.AddPage("settings", flex, true, false)
	a.pages.SwitchToPage("settings")
	a.tview.SetFocus(form)
}
