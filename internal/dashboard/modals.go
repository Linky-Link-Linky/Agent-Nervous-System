package dashboard

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) overlayModal(body string) string {
	var modalContent string
	switch m.modal {
	case modalHelp:
		modalContent = m.renderHelpModal()
	case modalReceiptDetail:
		modalContent = m.renderReceiptDetailModal()
	case modalConfirmTimeTravel, modalConfirmRevoke, modalConfirmToggle:
		modalContent = m.renderConfirmModal()
	}
	bg := theme.BG
	dimmed := lipgloss.NewStyle().Background(bg).Render(body)
	overlay := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalContent)
	return lipgloss.JoinVertical(lipgloss.Left, dimmed, overlay)
}
