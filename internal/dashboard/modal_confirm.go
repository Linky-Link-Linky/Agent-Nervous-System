package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderConfirmModal() string {
	var actionDesc string
	switch m.modal {
	case modalConfirmTimeTravel:
		actionDesc = "Time-travel to snapshot"
	case modalConfirmRevoke:
		actionDesc = "Revoke credential immediately"
	case modalConfirmToggle:
		actionDesc = "Toggle policy enabled state"
	default:
		actionDesc = "Confirm action"
	}

	content := strings.Join([]string{
		theme.ModalTitle.Render("CONFIRM"),
		"",
		"  " + actionDesc,
		"  " + lipgloss.NewStyle().Foreground(theme.Warn).Render(m.confirmTarget),
		"",
		"  " + theme.ModalButton.Render("YES — PROCEED") + "  " + theme.ModalButtonAlt.Render("CANCEL"),
	}, "\n")
	return theme.ModalBox.Width(50).Render(content)
}
