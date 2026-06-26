package ui

import (
	"fmt"
	"sync"
	"time"

	"ans-tui/internal/model"
	"ans-tui/internal/poller"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MCPPanel struct {
	*tview.Flex
	mu        sync.Mutex
	table     *tview.Table
	statusBar *tview.TextView
	sparkView *tview.TextView
	status    *model.MCPStatus
	log       []*model.MCPLogEntry
}

func NewMCPPanel() *MCPPanel {
	statusBar := tview.NewTextView().SetDynamicColors(true)
	statusBar.SetBackgroundColor(ColorPanelBG)
	statusBar.SetText("[#826EBE]STATUS  waiting...[-]")

	sparkView := tview.NewTextView().SetDynamicColors(true)
	sparkView.SetBackgroundColor(ColorPanelBG)
	sparkView.SetText("[#826EBE]REQ/s  waiting for data...[-]")

	table := tview.NewTable().SetFixed(1, 0)
	table.SetBackgroundColor(ColorPanelBG)
	table.SetBorderColor(ColorBorderNorm)
	table.SetTitleColor(ColorTextSecondary)
	table.SetTitleAlign(tview.AlignLeft)
	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.NewRGBColor(40, 30, 80)).
		Foreground(ColorTextPrimary))

	headers := []string{"TIME", "DIR", "METHOD", "TOKENS", "INJ", "PII", "POLICY", "PREVIEW"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary)
		if i == 3 {
			cell.SetExpansion(0).SetAlign(tview.AlignRight)
		}
		table.SetCell(0, i, cell)
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetBorderColor(ColorBorderNorm).SetBackgroundColor(ColorPanelBG)
	flex.SetTitle(PanelTitle("⬡", "MCP SECURITY PROXY")).SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	flex.AddItem(statusBar, 2, 0, false)
	flex.AddItem(sparkView, 2, 0, false)
	flex.AddItem(table, 0, 1, false)

	return &MCPPanel{
		Flex:      flex,
		table:     table,
		statusBar: statusBar,
		sparkView: sparkView,
	}
}

func (p *MCPPanel) Update(mcp *poller.MCPState) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status = mcp.Status
	p.log = mcp.Log

	// Status bar
	running := StatusDot(p.status.Running)
	uptime := time.Duration(p.status.UptimeSeconds) * time.Second
	uptimeStr := FormatDuration(int64(uptime.Seconds() * 1000))
	p.statusBar.SetText(fmt.Sprintf(
		"%s [#826EBE]STATUS[-] %s [#826EBE]UPTIME[-] [#DCD2FF]%s[-] [#826EBE]MSGS[-] [#DCD2FF]%d[-] [#826EBE]TOKENS[-] [#DCD2FF]%d[-] [#826EBE]BURN[-] [#DCD2FF]%.0f/s[-]",
		running,
		StatusDot(true),
		uptimeStr,
		p.status.TotalMessages,
		p.status.TotalTokens,
		p.status.BurnRate,
	))

	// Sparklines
	reqSpark := RenderSparkline(p.status.ReqHistory, 20)
	tokSpark := RenderSparkline(p.status.TokHistory, 20)
	peakReq := 0.0
	for _, v := range p.status.ReqHistory {
		if v > peakReq {
			peakReq = v
		}
	}
	peakTok := 0.0
	for _, v := range p.status.TokHistory {
		if v > peakTok {
			peakTok = v
		}
	}
	p.sparkView.SetText(fmt.Sprintf(
		"[#826EBE]REQ/s  [#DCD2FF]%s[-][#826EBE]  peak %.0f/s[-]\n[#826EBE]TOK/s  [#DCD2FF]%s[-][#826EBE]  peak %.0f/s[-]",
		reqSpark, peakReq, tokSpark, peakTok))

	// Log table
	p.rebuildLog()
}

func (p *MCPPanel) rebuildLog() {
	p.table.Clear()
	headers := []string{"TIME", "DIR", "METHOD", "TOKENS", "INJ", "PII", "POLICY", "PREVIEW"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary)
		if i == 3 {
			cell.SetExpansion(0).SetAlign(tview.AlignRight)
		}
		p.table.SetCell(0, i, cell)
	}

	for r, entry := range p.log {
		row := r + 1

		dirColor := "[#50C8FF]"
		dirGlyph := "→"
		if entry.Direction == "response" {
			dirColor = "[#B450FF]"
			dirGlyph = "←"
		}

		injGlyph := "[#64DC8C]✗[-]"
		if entry.InjDetected {
			injGlyph = "[#FF5064]✓[-]"
		}
		piiGlyph := "[#64DC8C]✗[-]"
		if entry.PIIFound {
			piiGlyph = "[#FFBE3C]✓[-]"
		}

		policyColor := "[#64DC8C]"
		policyLabel := "ok"
		if entry.PolicyResult == "deny" {
			policyColor = "[#FF5064]"
			policyLabel = "deny"
		}

		p.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", entry.Timestamp.Format("15:04:05"))))
		p.table.SetCell(row, 1, tview.NewTableCell(fmt.Sprintf("%s%s[-]", dirColor, dirGlyph)))
		p.table.SetCell(row, 2, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", entry.Method)))
		p.table.SetCell(row, 3, tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%d[-]", entry.TokenEstimate)).SetAlign(tview.AlignRight))
		p.table.SetCell(row, 4, tview.NewTableCell(injGlyph))
		p.table.SetCell(row, 5, tview.NewTableCell(piiGlyph))
		p.table.SetCell(row, 6, tview.NewTableCell(fmt.Sprintf("%s✓ %s[-]", policyColor, policyLabel)))
		p.table.SetCell(row, 7, tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(entry.ContentPreview, 36))))
	}
}

func (p *MCPPanel) SetFocusColor(focused bool) {
	if focused {
		p.Flex.SetBorderColor(ColorBorderFoc)
	} else {
		p.Flex.SetBorderColor(ColorBorderNorm)
	}
}
