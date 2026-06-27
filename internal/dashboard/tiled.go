package dashboard

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard/providers"
	"github.com/gdamore/tcell/v2"
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

const sparkWidth = 12

var panelNames = map[panelID]string{
	panelCPU:  "CPU",
	panelMEM:  "MEM",
	panelDISK: "DISK",
	panelNET:  "NET",
	panelGPU:  "GPU",
	panelPROC: "PROC",
}

var focusColor = tcell.NewRGBColor(0x2e, 0xcc, 0x71)

type procSortMode int

const (
	sortCPU procSortMode = iota
	sortMEM
	sortName
	sortPID
)

var sortLabels = map[procSortMode]string{
	sortCPU:  "CPU",
	sortMEM:  "MEM",
	sortName: "NAME",
	sortPID:  "PID",
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

	// History rings for sparklines
	cpuHist  *historyRing
	memHist  *historyRing
	diskHist *historyRing
	netHist  *historyRing

	// Process sort
	sortMode procSortMode
	sortAsc  bool

	// Panel focus
	focused panelID
}

func newTiledLayout(prov providers.DashboardProvider) *tiledLayout {
	tl := &tiledLayout{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		panels:   make(map[panelID]*resourcePanel),
		visible:  make(map[panelID]bool),
		prov:     prov,
		sideOpen: true,
		focused:  panelCPU,
		sortMode: sortCPU,
		sortAsc:  false,
		cpuHist:  &historyRing{},
		memHist:  &historyRing{},
		diskHist: &historyRing{},
		netHist:  &historyRing{},
	}

	for id := panelCPU; id < panelCount; id++ {
		tl.panels[id] = newResourcePanel(id)
		tl.visible[id] = true
	}

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
		cb.SetFieldBackgroundColor(bgColor)
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

func (tl *tiledLayout) focusNext() {
	ids := make([]panelID, 0, 6)
	for id := panelCPU; id < panelCount; id++ {
		if tl.visible[id] {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	for i, id := range ids {
		if id == tl.focused {
			tl.focused = ids[(i+1)%len(ids)]
			tl.updateFocusBorder()
			return
		}
	}
	tl.focused = ids[0]
	tl.updateFocusBorder()
}

func (tl *tiledLayout) focusPrev() {
	ids := make([]panelID, 0, 6)
	for id := panelCPU; id < panelCount; id++ {
		if tl.visible[id] {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	for i, id := range ids {
		if id == tl.focused {
			tl.focused = ids[(i-1+len(ids))%len(ids)]
			tl.updateFocusBorder()
			return
		}
	}
	tl.focused = ids[0]
	tl.updateFocusBorder()
}

func (tl *tiledLayout) updateFocusBorder() {
	for id := panelCPU; id < panelCount; id++ {
		if tl.panels[id] != nil {
			if id == tl.focused {
				tl.panels[id].SetBorderColor(focusColor)
			} else {
				tl.panels[id].SetBorderColor(borderColor)
			}
		}
	}
}

func (tl *tiledLayout) cycleSort() {
	tl.sortMode = (tl.sortMode + 1) % 4
}

func barMini(pct float64, w int) string {
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

func sparkShort(samples []float64) string {
	n := len(samples)
	if n == 0 {
		return "[#334155]" + strings.Repeat("─", sparkWidth) + "[-]"
	}
	min, max := samples[0], samples[0]
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

	step := float64(n) / float64(sparkWidth)
	var b strings.Builder
	for i := 0; i < sparkWidth; i++ {
		si := int(float64(i) * step)
		if si >= n {
			si = n - 1
		}
		norm := (samples[si] - min) / span
		clr := "#2ecc71"
		if norm > 0.8 {
			clr = "#e74c3c"
		} else if norm > 0.55 {
			clr = "#f59e0b"
		}
		ch := "⡀"
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
		case norm < 0.90:
			ch = "⣶"
		default:
			ch = "⣿"
		}
		b.WriteString(fmt.Sprintf("[%s]%s[-]", clr, ch))
	}
	return b.String()
}

func (tl *tiledLayout) updateAll(s providers.ComponentStats, evs []providers.AuditEvent, rules []providers.RuleEntry) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	// Push history samples
	tl.cpuHist.push(s.CPU.UsagePct)
	tl.memHist.push(s.Mem.Pct)
	if s.Disk.TotalGB > 0 {
		tl.diskHist.push(s.Disk.Pct)
	}
	tl.netHist.push(s.Net.SpeedInMBs)

	if tl.visible[panelCPU] && tl.panels[panelCPU] != nil {
		p := tl.panels[panelCPU]
		var b strings.Builder
		cpuModel := s.CPU.Model
		if cpuModel == "" {
			cpuModel = fmt.Sprintf("%d cores", s.CPU.Cores)
		}
		b.WriteString(fmt.Sprintf("  [#e2e8f0]%s[-]\n", trunc(cpuModel, 24)))
		b.WriteString(fmt.Sprintf("  %s  [#e2e8f0]%5.1f%%[-]\n", barMini(s.CPU.UsagePct, 18), s.CPU.UsagePct))
		ncores := len(s.CPU.PerCore)
		if ncores > 8 {
			ncores = 8
		}
		if ncores > 0 {
			perRow := 4
			for r := 0; r < ncores; r += perRow {
				line := "  "
				for c := r; c < r+perRow && c < ncores; c++ {
					line += fmt.Sprintf("[#94a3b8]C%d[-]%s [#e2e8f0]%2.0f%%[-]  ", c, barMini(s.CPU.PerCore[c], 5), s.CPU.PerCore[c])
				}
				b.WriteString(strings.TrimRight(line, " ") + "\n")
			}
		}
		// Sparkline
		sp := sparkShort(tl.cpuHist.samples())
		b.WriteString(fmt.Sprintf("  [#94a3b8]%s[-]\n", sp))
		p.SetTitle(fmt.Sprintf("[#2ecc71]CPU %5.1f%%[-]", s.CPU.UsagePct))
		_ = p.SetText(b.String())
	}

	if tl.visible[panelMEM] && tl.panels[panelMEM] != nil {
		p := tl.panels[panelMEM]
		mb := barMini(s.Mem.Pct, 18)
		sp := sparkShort(tl.memHist.samples())
		text := fmt.Sprintf("  %s\n  [#e2e8f0]%d/%d GB[-]  [#94a3b8](%5.1f%%)[-]\n  [#94a3b8]%s[-]\n", mb, s.Mem.UsedGB, s.Mem.TotalGB, s.Mem.Pct, sp)
		p.SetTitle(fmt.Sprintf("[#2ecc71]MEM %d/%d GB[-]", s.Mem.UsedGB, s.Mem.TotalGB))
		_ = p.SetText(text)
	}

	if tl.visible[panelDISK] && tl.panels[panelDISK] != nil && s.Disk.TotalGB > 0 {
		p := tl.panels[panelDISK]
		db := barMini(s.Disk.Pct, 18)
		sp := sparkShort(tl.diskHist.samples())
		text := fmt.Sprintf("  %s\n  [#e2e8f0]%d/%d GB[-]  [#94a3b8](%5.1f%%)[-]\n  [#94a3b8]R:%.0f  W:%.0f MB/s[-]\n  [#94a3b8]%s[-]\n", db, s.Disk.UsedGB, s.Disk.TotalGB, s.Disk.Pct, s.Disk.ReadSpeedMBs, s.Disk.WriteSpeedMBs, sp)
		p.SetTitle(fmt.Sprintf("[#2ecc71]DISK %d/%d GB[-]", s.Disk.UsedGB, s.Disk.TotalGB))
		_ = p.SetText(text)
	}

	if tl.visible[panelNET] && tl.panels[panelNET] != nil {
		p := tl.panels[panelNET]
		sp := sparkShort(tl.netHist.samples())
		text := fmt.Sprintf("\n  [#94a3b8]▼ %.0f MB/s[-]          [#94a3b8]▲ %.0f MB/s[-]\n\n  [#1a6b3a]%.0f ↓/%.0f ↑ MB[-]\n  [#94a3b8]%s[-]\n", s.Net.SpeedInMBs, s.Net.SpeedOutMBs, s.Net.BytesInMB, s.Net.BytesOutMB, sp)
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

		// Sort processes
		procs := make([]providers.ProcEntry, len(s.Procs))
		copy(procs, s.Procs)
		sort.Slice(procs, func(i, j int) bool {
			var less bool
			switch tl.sortMode {
			case sortCPU:
				less = procs[i].CPU < procs[j].CPU
			case sortMEM:
				less = procs[i].MemMB < procs[j].MemMB
			case sortName:
				less = procs[i].Name < procs[j].Name
			case sortPID:
				less = procs[i].PID < procs[j].PID
			}
			if tl.sortAsc {
				return less
			}
			return !less
		})

		for i, pr := range procs {
			if i >= 5 {
				break
			}
			cpub := barMini(pr.CPU, 8)
			mempct := float64(pr.MemMB) / 2000 * 100
			if mempct > 100 {
				mempct = 100
			}
			memb := barMini(mempct, 8)
			b.WriteString(fmt.Sprintf("  %s  [#e2e8f0]%5.1f%%[-]  %s  [#e2e8f0]%d MB[-]  [#94a3b8]%s[-]\n",
				cpub, pr.CPU, memb, pr.MemMB, trunc(pr.Name, 14)))
		}
		p.SetTitle(fmt.Sprintf("[#2ecc71]PROCESSES  [#94a3b8]sort:%s[-]", sortLabels[tl.sortMode]))
		_ = p.SetText(b.String())
	}
	tl.updateFocusBorder()
}

// --- Settings modal ---

func (a *App) showSettings() {
	form := tview.NewForm()
	form.SetBackgroundColor(bgColor)
	form.SetBorder(true).SetBorderColor(primaryColor)
	form.SetTitle("[#2ecc71] SETTINGS [-]").
		SetTitleColor(primaryColor).SetTitleAlign(tview.AlignLeft)
	form.SetFieldBackgroundColor(bgColor)
	form.SetButtonBackgroundColor(tcell.NewRGBColor(0x1E, 0x29, 0x30))
	form.SetButtonTextColor(primaryColor)

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
