package dashboard

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderMCPPanel() string {
	w := m.panel3Width()
	h := row2Height(m.height, m.width)
	style := theme.PanelStyle(m.focus == focusMCP).Width(w).Height(h)
	title := theme.TitleStyle(m.focus == focusMCP).Render(" ⬡ MCP SECURITY PROXY ")

	if m.mcpStatus == nil {
		return style.Render(title + "\n\n  No MCP status.")
	}

	var b string
	b += title + "\n"

	// Status row
	statusDot := lipgloss.NewStyle().Foreground(theme.Failure).Render("○ STOPPED")
	if m.mcpStatus.Running {
		statusDot = lipgloss.NewStyle().Foreground(theme.Success).Render("● RUNNING")
	}
	b += fmt.Sprintf("STATUS  %s   UPTIME %ds   LISTEN %s   TARGET %s\n",
		statusDot, m.mcpStatus.UptimeSeconds, m.mcpStatus.ListenAddr, m.mcpStatus.TargetURL)
	b += theme.CLISep.Render("─") + "\n"
	b += fmt.Sprintf("MSGS %d   TOKENS %d   BURN %.0f/s   INJ %d   PRUNED %d\n",
		m.mcpStatus.TotalMessages, m.mcpStatus.TotalTokens, m.mcpStatus.BurnRate,
		m.mcpStatus.Injections, m.mcpStatus.Pruned)
	b += fmt.Sprintf("RATELIM %d   BUDGET %d   POLICY %d   TOOLS %d\n",
		m.mcpStatus.RateLimited, m.mcpStatus.BudgetExceeded, m.mcpStatus.PolicyDenied, m.mcpStatus.ToolsDenied)

	// Sparklines
	reqSpark := Sparkline(m.mcpStatus.ReqHistory, 20, theme.AccentB)
	tokSpark := Sparkline(m.mcpStatus.TokHistory, 20, theme.AccentA)
	b += fmt.Sprintf("REQ/s   %s  peak %.0f/s\n", reqSpark, maxFloat(m.mcpStatus.ReqHistory))
	b += fmt.Sprintf("TOK/s   %s  peak %.0f/s\n", tokSpark, maxFloat(m.mcpStatus.TokHistory))

	// Audit log
	b += "\n"
	n := len(m.mcpLog)
	if n > 8 {
		n = 8
	}
	for i, e := range m.mcpLog[:n] {
		selected := i == m.mcpLogCursor && m.focus == focusMCP
		rowStyle := theme.TableRowNormal
		if selected {
			rowStyle = theme.TableRowSelected
		}

		dirCol := theme.AccentB
		if e.Direction == "response" {
			dirCol = theme.AccentA
		}
		dir := lipgloss.NewStyle().Foreground(dirCol).Render(map[string]string{"request": "→", "response": "←"}[e.Direction])

		inj := "✗"
		if e.InjDetected {
			inj = lipgloss.NewStyle().Foreground(theme.AccentC).Render("✓")
		}
		pii := "✗"
		if e.PIIFound {
			pii = lipgloss.NewStyle().Foreground(theme.Warn).Render("✓")
		}

		pol := lipgloss.NewStyle().Foreground(theme.Success).Render("✓ ok")
		if e.PolicyResult == "deny" {
			pol = lipgloss.NewStyle().Foreground(theme.Failure).Render("✗ deny")
		}

		line := fmt.Sprintf("%s  %s  %-18s  %6d  %s  %s  %s",
			e.TimeStr(), dir, e.ShortMethod(), e.TokenEstimate, inj, pii, pol)
		b += rowStyle.Render(line) + "\n"
	}

	return style.Render(b)
}

func maxFloat(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
