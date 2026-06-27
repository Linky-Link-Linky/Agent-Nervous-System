package dashboard

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderChainPanel() string {
	w := m.chainPanelWidth()
	h := m.chainPanelHeight()
	style := theme.PanelStyle(m.focus == focusChain).Width(w).Height(h)
	title := theme.TitleStyle(m.focus == focusChain).Render(" ◈ RECEIPT CHAIN ")
	if len(m.chain) == 0 {
		return style.Render(title + "\n\n  No receipts yet.")
	}

	var b strings.Builder
	b.WriteString(title + "\n")

	// table header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		theme.TableHeader.Render(theme.PadRight("IDX", 5)),
		theme.TableHeader.Render(theme.PadRight("HASH", 12)),
		theme.TableHeader.Render(theme.PadRight("TIME", 8)),
		theme.TableHeader.Render(theme.PadRight("ACTION", 16)),
		theme.TableHeader.Render(theme.PadRight("AGENT", 13)),
		theme.TableHeader.Render(theme.PadRight("OUTCOME", 9)),
		theme.TableHeader.Render(theme.PadRight("DUR", 6)),
		theme.TableHeader.Render(theme.PadRight("SIG", 16)),
	)
	b.WriteString(header + "\n")
	b.WriteString(theme.TableHeader.Render(strings.Repeat("─", w-4)) + "\n")

	n := len(m.chain)
	if n > 50 {
		n = 50
	}
	rows := m.chain[:n]

	for i, r := range rows {
		selected := i == m.chainCursor && m.focus == focusChain
		rowStyle := theme.TableRowNormal
		if selected {
			rowStyle = theme.TableRowSelected
		}

		idx := theme.PadRight(strconv.Itoa(r.Index), 5)
		hash := theme.CLIHash.Render(theme.PadRight(r.ShortID(), 12))
		tm := r.Timestamp.Format("15:04:05")
		timeS := theme.PadRight(tm, 8)
		act := theme.ActionColor(r.ActionType)
		actStr := theme.PadRight(r.ActionType, 16)
		action := lipgloss.NewStyle().Foreground(act).Render(actStr)
		agent := theme.CLIAgentID.Render(theme.PadRight(r.ShortAgent(), 13))
		outcome := theme.OutcomeStyle(r.Outcome).Render(theme.PadRight(theme.OutcomeShort(r.Outcome), 9))
		dur := theme.PadRight(theme.FormatDuration(r.DurationMS), 6)
		sig := theme.CLIHash.Render(theme.PadRight(r.ShortSignature(), 16))

		line := rowStyle.Render(idx + " " + hash + " " + timeS + " " + action + " " + agent + " " + outcome + " " + dur + " " + sig)
		b.WriteString(line + "\n")
	}

	// sparkline footer
	rate := ""
	if len(m.chainReqRate) > 0 {
		rate = "  " + lipgloss.NewStyle().Foreground(theme.TextDim).Render(fmt.Sprintf("%.0f rcpt/min", m.chainReqRate[len(m.chainReqRate)-1]))
	}
	spark := Sparkline(m.chainReqRate, w-30, theme.AccentB)
	verifyStatus := lipgloss.NewStyle().Foreground(theme.Success).Render("verified ✓")
	if m.chainVerify == "checking" {
		verifyStatus = lipgloss.NewStyle().Foreground(theme.Warn).Render("checking…")
	} else if m.chainVerify == "broken" {
		verifyStatus = lipgloss.NewStyle().Foreground(theme.Failure).Render("CHAIN BROKEN ✗")
	}
	footer := lipgloss.NewStyle().Foreground(theme.TextDim).Render(spark) + rate + "    chain: " + verifyStatus
	b.WriteString("\n" + footer)

	return style.Render(b.String())
}
