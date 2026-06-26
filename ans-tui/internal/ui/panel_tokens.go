package ui

import (
	"fmt"
	"sync"

	"ans-tui/internal/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TokenPanel struct {
	*tview.Flex
	mu     sync.Mutex
	table  *tview.Table
	tokens []*model.Token
}

func NewTokenPanel() *TokenPanel {
	table := tview.NewTable().SetFixed(1, 0)
	table.SetBackgroundColor(ColorPanelBG)
	table.SetBorderColor(ColorBorderNorm)
	table.SetTitleColor(ColorTextSecondary)
	table.SetTitleAlign(tview.AlignLeft)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.NewRGBColor(40, 30, 80)).
		Foreground(ColorTextPrimary))

	headers := []string{"CRED ID", "TYPE", "RESOURCE", "TTL", "AGENT"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary)
		table.SetCell(0, i, cell)
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetBorderColor(ColorBorderNorm).SetBackgroundColor(ColorPanelBG)
	flex.SetTitle(PanelTitle("⚡", "EPHEMERAL TOKENS")).SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	flex.AddItem(table, 0, 1, false)

	return &TokenPanel{
		Flex:  flex,
		table: table,
	}
}

var tokenTypeIcons = map[string]string{
	"aws-sts":  "☁",
	"vault":    "⌇",
	"gcp-iam":  "◆",
	"azure-ad": "▲",
	"oauth2":   "◎",
}

func (p *TokenPanel) Update(tokens []*model.Token) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tokens = tokens
	p.rebuildTable()
}

func (p *TokenPanel) rebuildTable() {
	p.table.Clear()
	headers := []string{"CRED ID", "TYPE", "RESOURCE", "TTL", "AGENT"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary)
		p.table.SetCell(0, i, cell)
	}

	for r, t := range p.tokens {
		row := r + 1
		ttl := t.TTLSeconds()
		icon, ok := tokenTypeIcons[t.Type]
		if !ok {
			icon = "◎"
		}
		ttlBar := TTLBar(ttl, 60)

		ttlColor := "[#64DC8C]"
		switch {
		case ttl < 10:
			ttlColor = "[#FF5064]"
		case ttl < 30:
			ttlColor = "[#FFBE3C]"
		}

		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[#50C8FF]%s[-]", Truncate(t.ID, 12))))
		p.table.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s %s[-]", icon, t.Type)))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(t.Resource, 28))))
		p.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("%s%ds %s[-]", ttlColor, ttl, ttlBar)))
		p.table.SetCell(row, 4, tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(t.AgentID, 14))))
	}
}

func (p *TokenPanel) SelectedToken() *model.Token {
	p.mu.Lock()
	defer p.mu.Unlock()
	row, _ := p.table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(p.tokens) {
		return nil
	}
	return p.tokens[idx]
}

func (p *TokenPanel) SetFocusColor(focused bool) {
	if focused {
		p.Flex.SetBorderColor(ColorBorderFoc)
	} else {
		p.Flex.SetBorderColor(ColorBorderNorm)
	}
}
