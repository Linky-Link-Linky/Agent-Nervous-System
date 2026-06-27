package ui

import (
	"fmt"
	"sync"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type PolicyPanel struct {
	*tview.Flex
	mu       sync.Mutex
	table    *tview.Table
	policies []*model.Policy
}

func NewPolicyPanel() *PolicyPanel {
	table := tview.NewTable().SetFixed(1, 0)
	table.SetBackgroundColor(ColorPanelBG)
	table.SetBorderColor(ColorBorderNorm)
	table.SetTitleColor(ColorTextSecondary)
	table.SetTitleAlign(tview.AlignLeft)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.NewRGBColor(40, 30, 80)).
		Foreground(ColorTextPrimary))

	headers := []string{"ID", "ENABLED", "PRIORITY", "SEVERITY", "EFFECT", "NAME"}
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

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetBorderColor(ColorBorderNorm).SetBackgroundColor(ColorPanelBG)
	flex.SetTitle(PanelTitle("⚙", "POLICIES")).SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	flex.AddItem(table, 0, 1, false)

	return &PolicyPanel{
		Flex:  flex,
		table: table,
	}
}

func (p *PolicyPanel) Update(policies []*model.Policy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.policies = policies
	p.rebuildTable()
}

func (p *PolicyPanel) rebuildTable() {
	p.table.Clear()
	headers := []string{"ID", "ENABLED", "PRIORITY", "SEVERITY", "EFFECT", "NAME"}
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

	enabledCount := 0
	for r, pl := range p.policies {
		row := r + 1
		if pl.Enabled {
			enabledCount++
		}

		enableGlyph := "[#826EBE]○[-]"
		if pl.Enabled {
			enableGlyph = "[#64DC8C]●[-]"
		}

		var severityColor string
		switch pl.Severity {
		case "high", "critical":
			severityColor = "[#FF5064]"
		case "medium":
			severityColor = "[#FFBE3C]"
		default:
			severityColor = "[#826EBE]"
		}

		var effectColor string
		switch pl.Effect {
		case "deny":
			effectColor = "[#FF5064]"
		case "warn":
			effectColor = "[#FFBE3C]"
		default:
			effectColor = "[#50C8FF]"
		}

		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(pl.ID, 20))))
		p.table.SetCell(row, 1, tview.NewTableCell(enableGlyph))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%d[-]", pl.Priority)).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%s%s[-]", severityColor, pl.Severity)))
		p.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("%s%s[-]", effectColor, pl.Effect)))
		p.table.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", pl.Name)))
	}

	// Footer
	_ = enabledCount
}

func (p *PolicyPanel) SelectedPolicy() *model.Policy {
	p.mu.Lock()
	defer p.mu.Unlock()
	row, _ := p.table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(p.policies) {
		return nil
	}
	return p.policies[idx]
}

func (p *PolicyPanel) SetFocusColor(focused bool) {
	if focused {
		p.Flex.SetBorderColor(ColorBorderFoc)
	} else {
		p.Flex.SetBorderColor(ColorBorderNorm)
	}
}
