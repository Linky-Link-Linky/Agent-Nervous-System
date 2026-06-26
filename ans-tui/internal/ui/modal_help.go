package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (a *App) showHelp() {
	form := tview.NewForm()
	form.SetBackgroundColor(ColorPanelBG)
	form.SetBorder(true).SetBorderColor(ColorBorderFoc)
	form.SetTitle(" ? KEYBINDINGS ").SetTitleColor(ColorTextSecondary).SetTitleAlign(tview.AlignLeft)
	form.SetFieldBackgroundColor(ColorPanelBG)
	form.SetButtonBackgroundColor(tcell.NewRGBColor(30, 20, 60))
	form.SetButtonTextColor(ColorTextPrimary)

	keys := []struct{ key, desc string }{
		{"TAB / Shift-TAB", "Cycle panel focus"},
		{"A B C D E", "Jump directly to panel"},
		{"↑ ↓", "Navigate rows"},
		{"ENTER", "Open detail"},
		{"R", "Force refresh all panels"},
		{"P", "Pause / resume polling"},
		{"V", "Verify full chain"},
		{"T", "Toggle policy (Panel C)"},
		{"X", "Revoke token (Panel D)"},
		{"D", "Snapshot diff (Panel B)"},
		{"ESC / Q", "Close modal / quit"},
		{"?", "This help screen"},
	}

	for _, kv := range keys {
		form.AddTextView(kv.key, kv.desc, 0, 1, false, false)
	}

	form.AddButton("CLOSE", func() {
		a.pages.RemovePage("help")
		a.pages.SwitchToPage("main")
		a.tv.SetFocus(a.root)
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false).
			AddItem(form, 50, 0, true).
			AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false),
			0, 1, false).
		AddItem(tview.NewBox().SetBackgroundColor(ColorBG), 0, 1, false)

	a.pages.AddPage("help", flex, true, true)
	a.pages.SwitchToPage("help")
	a.tv.SetFocus(form)
}
