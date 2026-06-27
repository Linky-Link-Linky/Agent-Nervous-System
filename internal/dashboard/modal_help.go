package dashboard

import (
	"strings"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func (m DashboardModel) renderHelpModal() string {
	content := strings.Join([]string{
		theme.ModalTitle.Render("ANS — KEYBINDINGS"),
		"",
		"  NAVIGATION",
		"  " + theme.KeyHint.Render("TAB / Shift-TAB") + "    " + theme.KeyDesc.Render("Cycle panel focus"),
		"  " + theme.KeyHint.Render("A  B  C  D  E") + "    " + theme.KeyDesc.Render("Jump directly to panel"),
		"  " + theme.KeyHint.Render("↑ ↓  /  j k") + "      " + theme.KeyDesc.Render("Navigate rows"),
		"",
		"  ACTIONS",
		"  " + theme.KeyHint.Render("ENTER") + "             " + theme.KeyDesc.Render("Open detail / confirm"),
		"  " + theme.KeyHint.Render("R") + "               " + theme.KeyDesc.Render("Force refresh all panels"),
		"  " + theme.KeyHint.Render("P") + "               " + theme.KeyDesc.Render("Pause / resume live updates"),
		"  " + theme.KeyHint.Render("V") + "               " + theme.KeyDesc.Render("Verify full chain (Panel A)"),
		"  " + theme.KeyHint.Render("T") + "               " + theme.KeyDesc.Render("Toggle policy enabled (Panel C)"),
		"  " + theme.KeyHint.Render("X") + "               " + theme.KeyDesc.Render("Revoke token immediately (Panel D)"),
		"",
		"  MODALS",
		"  " + theme.KeyHint.Render("ESC / Q") + "          " + theme.KeyDesc.Render("Close modal"),
		"  " + theme.KeyHint.Render("?") + "               " + theme.KeyDesc.Render("This help screen"),
		"  " + theme.KeyHint.Render("Q (no modal)") + "     " + theme.KeyDesc.Render("Quit dashboard"),
	}, "\n")
	return theme.ModalBox.Width(52).Render(content)
}
