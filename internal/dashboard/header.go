package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderHeader() string {
	pulseCol := theme.AccentA
	if m.daemonPulse {
		pulseCol = theme.AccentB
	}
	pulse := lipgloss.NewStyle().Foreground(pulseCol).Render("◉")
	brand := theme.HeaderBrand.Render(fmt.Sprintf(" ANS — AGENT NERVOUS SYSTEM  %s", m.version))

	daemonStatus := lipgloss.NewStyle().Foreground(theme.Failure).Render("○ STOPPED")
	if m.daemon != nil && m.daemon.Running {
		daemonStatus = lipgloss.NewStyle().Foreground(theme.Success).Render("● RUNNING")
	}
	if m.daemon == nil {
		daemonStatus = lipgloss.NewStyle().Foreground(theme.Warn).Render("○ CONNECTING…")
	}

	chainLen := "?"
	if m.daemon != nil {
		chainLen = fmt.Sprintf("%d receipts", m.daemon.ChainLength)
	}
	statusRight := fmt.Sprintf("DAEMON %s   chain %s", daemonStatus, chainLen)

	fillWidth := max(0, m.width-50-len(m.version))
	line1 := lipgloss.JoinHorizontal(lipgloss.Center,
		pulse+"  ", brand,
		lipgloss.NewStyle().Foreground(theme.TextDim).Render(strings.Repeat(" ", fillWidth)),
		theme.HeaderStat.Render(statusRight))

	agent := "—"
	uptime := "—"
	db := "—"
	verified := "—"
	if m.daemon != nil {
		agent = m.daemon.Version
		uptime = theme.FormatUptime(time.Duration(m.daemon.UptimeSeconds) * time.Second)
		db = fmt.Sprintf("%.1f MB", m.daemon.DBSizeMB)
		if m.daemon.ChainVerified {
			verified = lipgloss.NewStyle().Foreground(theme.Success).Render("✓")
		} else {
			verified = lipgloss.NewStyle().Foreground(theme.Failure).Render("✗")
		}
	}
	line2 := fmt.Sprintf("  AGENT %s  │  DB %s  │  UPTIME %s  │  VERIFIED %s",
		theme.CLIAgentID.Render(agent), db, uptime, verified)

	sep := theme.Sep(m.width)
	return lipgloss.JoinVertical(lipgloss.Left, line1, line2, sep)
}
