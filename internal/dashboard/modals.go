package dashboard

import (
	"github.com/charmbracelet/lipgloss"
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
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modalContent)
}
