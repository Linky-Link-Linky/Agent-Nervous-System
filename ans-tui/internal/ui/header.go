package ui

import (
	"fmt"
	"sync"
	"time"

	"ans-tui/internal/model"
	"github.com/rivo/tview"
)

type Header struct {
	*tview.Flex
	mu         sync.Mutex
	textRow1   *tview.TextView
	textRow2   *tview.TextView
	separator  *tview.TextView
	status     *model.DaemonStatus
	pulseOn    bool
	pulseTick  *time.Ticker
}

func NewHeader() *Header {
	row1 := tview.NewTextView().SetDynamicColors(true)
	row1.SetBackgroundColor(ColorBG)
	row1.SetText(fmt.Sprintf("  [#B450FF]◉[-] [#DCD2FF]ANS — AGENT NERVOUS SYSTEM[-]  [#826EBE]v0.1.0[-]          [#826EBE]DAEMON[-] [#FFBE3C]○ CONNECTING...[-]"))

	row2 := tview.NewTextView().SetDynamicColors(true)
	row2.SetBackgroundColor(ColorBG)
	row2.SetText(fmt.Sprintf("  [#826EBE]AGENT[-] [#DCD2FF]--[-]  │  [#826EBE]DB[-] [#DCD2FF]--[-]  │  [#826EBE]UPTIME[-] [#DCD2FF]--[-]  │  [#826EBE]VERIFIED[-] [#DCD2FF]--[-]"))

	sep := tview.NewTextView().SetDynamicColors(true)
	sep.SetBackgroundColor(ColorBG)
	sep.SetText(fmt.Sprintf("  [#403080]%s[-]", "═══════════════════════════════════════════════════════════════════════════════"))

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(ColorBG)
	flex.AddItem(row1, 1, 0, false)
	flex.AddItem(row2, 1, 0, false)
	flex.AddItem(sep, 1, 0, false)

	h := &Header{
		Flex:    flex,
		textRow1: row1,
		textRow2: row2,
		separator: sep,
		status: &model.DaemonStatus{Uptime: "0s"},
	}

	h.pulseTick = time.NewTicker(1 * time.Second)
	go func() {
		for range h.pulseTick.C {
			h.pulseOn = !h.pulseOn
			h.mu.Lock()
			if h.status != nil {
				h.render()
			}
			h.mu.Unlock()
		}
	}()

	return h
}

func (h *Header) Stop() {
	h.pulseTick.Stop()
}

func (h *Header) Update(status *model.DaemonStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
	h.render()
}

func (h *Header) render() {
	if h.status == nil {
		return
	}

	var daemonDot string
	var daemonLabel string
	if h.status.Running {
		if h.pulseOn {
			daemonDot = "[#B450FF]●[-]"
		} else {
			daemonDot = "[#50C8FF]●[-]"
		}
		daemonLabel = "[#64DC8C]RUNNING[-]"
	} else {
		daemonDot = "[#FF5064]●[-]"
		daemonLabel = "[#FF5064]STOPPED[-]"
	}

	h.textRow1.SetText(fmt.Sprintf("  [#B450FF]◉[-] [#DCD2FF]ANS — AGENT NERVOUS SYSTEM[-]  [#826EBE]v%s[-]          [#826EBE]DAEMON[-] %s %s  [#826EBE]chain[-] [#DCD2FF]%d[-] [#826EBE]receipts[-]",
		h.status.Version, daemonDot, daemonLabel, h.status.ChainLength))

	verifiedGlyph := "[#64DC8C]✓[-]"
	if !h.status.ChainVerified {
		verifiedGlyph = "[#FF5064]✗[-]"
	}

	h.textRow2.SetText(fmt.Sprintf("  [#826EBE]AGENT[-] [#DCD2FF]%d[-]  │  [#826EBE]DB[-] [#DCD2FF]%.1f MB[-]  │  [#826EBE]UPTIME[-] [#DCD2FF]%s[-]  │  [#826EBE]VERIFIED[-] %s",
		h.status.AgentCount, h.status.DBSizeMB, h.status.Uptime, verifiedGlyph))
}
