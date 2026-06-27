package dashboard

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderStatusBar() string {
	prefix := ""
	if m.paused {
		prefix = lipgloss.NewStyle().Foreground(theme.Warn).Render("⏸ PAUSED  ")
	}
	line1 := prefix + theme.KeyHint.Render("[TAB]") + " " + theme.KeyDesc.Render("focus") +
		"  " + theme.KeyHint.Render("[A-E]") + " " + theme.KeyDesc.Render("jump") +
		"  " + theme.KeyHint.Render("[↑↓/jk]") + " " + theme.KeyDesc.Render("nav") +
		"  " + theme.KeyHint.Render("[ENTER]") + " " + theme.KeyDesc.Render("detail") +
		"  " + theme.KeyHint.Render("[R]") + " " + theme.KeyDesc.Render("refresh") +
		"  " + theme.KeyHint.Render("[P]") + " " + theme.KeyDesc.Render("pause") +
		"  " + theme.KeyHint.Render("[Q]") + " " + theme.KeyDesc.Render("quit") +
		"  " + theme.KeyHint.Render("[?]") + " " + theme.KeyDesc.Render("help")
	line2 := theme.KeyHint.Render("[V]") + " " + theme.KeyDesc.Render("verify chain (A)") +
		"  " + theme.KeyHint.Render("[T]") + " " + theme.KeyDesc.Render("toggle policy (C)") +
		"  " + theme.KeyHint.Render("[X]") + " " + theme.KeyDesc.Render("revoke token (D)")
	return theme.StatusBarStyle.Render(line1 + "\n" + line2)
}
