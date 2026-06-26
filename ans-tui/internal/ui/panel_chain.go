package ui

import (
	"fmt"
	"sync"
	"time"

	"ans-tui/internal/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChainPanel struct {
	*tview.Flex
	mu         sync.Mutex
	table      *tview.Table
	sparkView  *tview.TextView
	receipts   []*model.Receipt
	rateHist   []float64
	userScroll bool
	lastTop    int
}

func NewChainPanel() *ChainPanel {
	table := tview.NewTable().SetFixed(1, 0)
	table.SetBackgroundColor(ColorPanelBG)
	table.SetBorderColor(ColorBorderNorm)
	table.SetTitleColor(ColorTextSecondary)
	table.SetTitleAlign(tview.AlignLeft)

	table.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.NewRGBColor(40, 30, 80)).
		Foreground(ColorTextPrimary))

	headers := []string{"IDX", "HASH", "TIMESTAMP", "ACTION TYPE", "AGENT", "OUTCOME", "DURATION", "SIG"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary).
			SetAlign(tview.AlignLeft)
		if i == 0 {
			cell.SetExpansion(0).SetAlign(tview.AlignRight)
		}
		table.SetCell(0, i, cell)
	}

	sparkView := tview.NewTextView().SetDynamicColors(true)
	sparkView.SetBackgroundColor(ColorPanelBG)
	sparkView.SetText("[#826EBE]RATE  waiting for data...[-]")

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true).SetBorderColor(ColorBorderNorm).SetBackgroundColor(ColorPanelBG)
	flex.SetTitle(PanelTitle("◈", "RECEIPT CHAIN")).SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	flex.AddItem(table, 0, 1, false)
	flex.AddItem(sparkView, 1, 0, false)

	return &ChainPanel{
		Flex:     flex,
		table:    table,
		sparkView: sparkView,
	}
}

func (p *ChainPanel) Update(receipts []*model.Receipt) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.receipts = receipts

	// Track if user has scrolled away from top
	row, _ := p.table.GetOffset()
	if row > 1 {
		p.userScroll = true
	} else {
		p.userScroll = false
	}

	p.rebuildTable()

	// Update sparkline
	p.rateHist = append(p.rateHist, float64(len(receipts)))
	if len(p.rateHist) > 60 {
		p.rateHist = p.rateHist[len(p.rateHist)-60:]
	}
	spark := RenderSparkline(p.rateHist, 20)

	lastIdx := 0
	if len(receipts) > 0 {
		lastIdx = receipts[0].Index
	}
	p.sparkView.SetText(fmt.Sprintf("[#826EBE]RATE  %s  [#DCD2FF]%d[-][#826EBE] rcpt/min   chain idx [#DCD2FF]%d[-]",
		spark, len(receipts), lastIdx))
}

func (p *ChainPanel) rebuildTable() {
	if len(p.receipts) == 0 {
		return
	}

	// Preserve offset for auto-scroll behavior
	offsetRow, _ := p.table.GetOffset()

	p.table.Clear()
	headers := []string{"IDX", "HASH", "TIMESTAMP", "ACTION TYPE", "AGENT", "OUTCOME", "DURATION", "SIG"}
	for i, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", h)).
			SetSelectable(false).
			SetExpansion(1).
			SetTextColor(ColorTextSecondary).
			SetAlign(tview.AlignLeft)
		if i == 0 {
			cell.SetExpansion(0).SetAlign(tview.AlignRight)
		}
		p.table.SetCell(0, i, cell)
	}

	for r, rc := range p.receipts {
		row := r + 1

		idxCell := tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%d[-]", rc.Index)).
			SetAlign(tview.AlignRight)

		hashCell := tview.NewTableCell(fmt.Sprintf("[#50C8FF]%s[-]", Truncate(rc.ID, 12)))

		tsCell := tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", rc.Timestamp.Format("15:04:05")))

		actionColor := ActionTypeColor(rc.ActionType)
		actionCell := tview.NewTableCell(fmt.Sprintf("%s%s[-]", actionColor, rc.ActionType))

		agentCell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(rc.AgentID, 14)))

		outcomeCell := tview.NewTableCell(OutcomeGlyph(rc.Outcome))

		durCell := tview.NewTableCell(fmt.Sprintf("[#DCD2FF]%s[-]", FormatDuration(rc.DurationMS)))

		sigCell := tview.NewTableCell(fmt.Sprintf("[#826EBE]%s[-]", Truncate(rc.Signature, 12)))

		p.table.SetCell(row, 0, idxCell)
		p.table.SetCell(row, 1, hashCell)
		p.table.SetCell(row, 2, tsCell)
		p.table.SetCell(row, 3, actionCell)
		p.table.SetCell(row, 4, agentCell)
		p.table.SetCell(row, 5, outcomeCell)
		p.table.SetCell(row, 6, durCell)
		p.table.SetCell(row, 7, sigCell)
	}

	// Auto-scroll: if user hasn't manually scrolled, jump to top
	if !p.userScroll && offsetRow > 0 {
		p.table.ScrollToBeginning()
	}
}

func (p *ChainPanel) SelectedReceipt() *model.Receipt {
	p.mu.Lock()
	defer p.mu.Unlock()
	row, _ := p.table.GetSelection()
	idx := row - 1
	if idx < 0 || idx >= len(p.receipts) {
		return nil
	}
	return p.receipts[idx]
}

func (p *ChainPanel) SetFocusColor(focused bool) {
	if focused {
		p.Flex.SetBorderColor(ColorBorderFoc)
	} else {
		p.Flex.SetBorderColor(ColorBorderNorm)
	}
}

func (a *App) showReceiptDetail() {
	rc := a.panelChain.SelectedReceipt()
	if rc == nil {
		return
	}

	form := tview.NewForm()
	form.SetBackgroundColor(ColorPanelBG)
	form.SetBorder(true).SetBorderColor(ColorBorderFoc)
	form.SetTitle(fmt.Sprintf(" ◈ RECEIPT DETAIL  %s ", Truncate(rc.ID, 20))).
		SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	form.SetFieldBackgroundColor(ColorPanelBG)
	form.SetButtonBackgroundColor(tcell.NewRGBColor(30, 20, 60))
	form.SetButtonTextColor(ColorTextPrimary)

	lines := []string{
		fmt.Sprintf("[#826EBE]INDEX        [-][#DCD2FF]%d[-]", rc.Index),
		fmt.Sprintf("[#826EBE]HASH         [-][#50C8FF]%s[-]", rc.ID),
		fmt.Sprintf("[#826EBE]PREV HASH    [-][#50C8FF]%s[-]", Truncate(rc.PrevHash, 32)),
		fmt.Sprintf("[#826EBE]AGENT        [-][#DCD2FF]%s[-]", rc.AgentID),
		fmt.Sprintf("[#826EBE]ACTION TYPE  [-][#DCD2FF]%s[-]", rc.ActionType),
		fmt.Sprintf("[#826EBE]PHASE        [-][#DCD2FF]%s[-]", rc.Phase),
		fmt.Sprintf("[#826EBE]OUTCOME      [-][#DCD2FF]%s[-] %s", rc.Outcome, OutcomeGlyph(rc.Outcome)),
		fmt.Sprintf("[#826EBE]DURATION     [-][#DCD2FF]%s[-]", FormatDuration(rc.DurationMS)),
		fmt.Sprintf("[#826EBE]TIMESTAMP    [-][#DCD2FF]%s[-]", rc.Timestamp.Format(time.RFC3339Nano)),
		fmt.Sprintf("[#826EBE]POLICY       [-][#DCD2FF]%s[-]", rc.PolicyDecision),
		fmt.Sprintf("[#826EBE]SIGNATURE    [-][#826EBE]%s[-]", Truncate(rc.Signature, 32)),
		"",
		fmt.Sprintf("[#826EBE]PAYLOAD SUMMARY[-]"),
		fmt.Sprintf("[#DCD2FF]%s[-]", rc.PayloadSummary),
	}

	for _, line := range lines {
		form.AddTextView("", line, 0, 1, false, false)
	}

	closeFn := func() {
		a.tv.SetRoot(a.root, true)
		a.tv.SetFocus(a.panelChain.table)
	}

	if rc.SnapshotID != "" {
		form.AddButton("TIME-TRAVEL TO THIS", func() {
			// TODO: implement time-travel from receipt detail
			closeFn()
		})
	}
	form.AddButton("CLOSE", closeFn)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false).
			AddItem(form, 0, 1, true).
			AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false),
			0, 1, false).
		AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false)

	a.pages.AddPage("detail", flex, true, true)
	a.pages.SwitchToPage("detail")
	a.tv.SetFocus(form)
}
