package ui

import (
	"fmt"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SnapshotPanel struct {
	*tview.Flex
	mu       sync.Mutex
	table    *tview.Table
	stats    *tview.TextView
	snaps    []*model.Snapshot
}

func NewSnapshotPanel() *SnapshotPanel {
	table := tview.NewTable().SetFixed(1, 0)
	table.SetBackgroundColor(ColorPanelBG)
	table.SetBorderColor(ColorBorderNorm)
	table.SetTitleColor(ColorTextSecondary)
	table.SetTitleAlign(tview.AlignLeft)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.NewRGBColor(40, 30, 80)).
		Foreground(ColorTextPrimary))

	headers := []string{"SNAP ID", "TYPE", "IDX", "SIZE", "TIMESTAMP", "AGENT"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary)
		if i == 2 {
			cell.SetExpansion(0).SetAlign(tview.AlignRight)
		}
		table.SetCell(0, i, cell)
	}

	stats := tview.NewTextView().SetDynamicColors(true)
	stats.SetBackgroundColor(ColorPanelBG)
	stats.SetText("[#826EBE]TOTAL 0     SIZE 0     OLDEST --[-]")

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetBorderColor(ColorBorderNorm).SetBackgroundColor(ColorPanelBG)
	flex.SetTitle(PanelTitle("⏱", "SNAPSHOTS & TIME-TRAVEL")).SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	flex.AddItem(table, 0, 1, false)
	flex.AddItem(stats, 3, 0, false)

	return &SnapshotPanel{
		Flex:  flex,
		table: table,
		stats: stats,
	}
}

func (p *SnapshotPanel) Update(snaps []*model.Snapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.snaps = snaps
	p.rebuildTable()
	p.updateStats()
}

func (p *SnapshotPanel) rebuildTable() {
	p.table.Clear()
	headers := []string{"SNAP ID", "TYPE", "IDX", "SIZE", "TIMESTAMP", "AGENT"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary)
		if i == 2 {
			cell.SetExpansion(0).SetAlign(tview.AlignRight)
		}
		p.table.SetCell(0, i, cell)
	}

	var totalBytes int64
	var oldest, newest string
	for r, s := range p.snaps {
		row := r + 1
		totalBytes += s.SizeBytes
		if r == 0 {
			newest = s.Timestamp.Format("15:04:05")
		}
		oldest = s.Timestamp.Format("15:04:05")

		snapID := Truncate(s.ID, 12)
		typeLabel := s.Type
		if s.IsDiff {
			typeLabel = "diff"
		}
		sizeStr := FormatBytes(s.SizeBytes)

		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[#50C8FF]%s[-]", snapID)))
		p.table.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", typeLabel)))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%d[-]", s.ChainIndex)).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", sizeStr)))
		p.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", s.Timestamp.Format("15:04:05"))))
		p.table.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(s.AgentID, 14))))
	}

	_ = oldest
	_ = newest
}

func (p *SnapshotPanel) updateStats() {
	if len(p.snaps) == 0 {
		p.stats.SetText("[#826EBE]TOTAL 0     SIZE 0     OLDEST --     NEWEST --[-]")
		return
	}
	var totalBytes int64
	diffs := 0
	for _, s := range p.snaps {
		totalBytes += s.SizeBytes
		if s.IsDiff {
			diffs++
		}
	}
	oldest := p.snaps[len(p.snaps)-1].Timestamp.Format("2006-01-02 15:04")
	newest := p.snaps[0].Timestamp.Format("2006-01-02 15:04")

	p.stats.SetText(fmt.Sprintf("[#826EBE]TOTAL %-4d     SIZE %-8s     DIFF %-4d     OLDEST %s     NEWEST %s[-]",
		len(p.snaps), FormatBytes(totalBytes), diffs, oldest, newest))
}

func (p *SnapshotPanel) SelectedSnapshot() *model.Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	row, _ := p.table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(p.snaps) {
		return nil
	}
	return p.snaps[idx]
}

func (p *SnapshotPanel) SetFocusColor(focused bool) {
	if focused {
		p.Flex.SetBorderColor(ColorBorderFoc)
	} else {
		p.Flex.SetBorderColor(ColorBorderNorm)
	}
}
