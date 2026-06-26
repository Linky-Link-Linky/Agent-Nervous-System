package ui

import (
	"github.com/rivo/tview"
)

type StatusBar struct {
	*tview.TextView
}

func NewStatusBar() *StatusBar {
	tv := tview.NewTextView().SetDynamicColors(true)
	tv.SetBackgroundColor(ColorBG)
	tv.SetText("[#463C6E]  [TAB] focus  [1-5] jump  [ENTER] detail  [R] refresh  [P] pause  [V] verify  [?] help  [Q] quit[-]")
	return &StatusBar{TextView: tv}
}

func (s *StatusBar) SetPaused(paused bool) {
	if paused {
		s.SetText("[#FFBE3C]  ⏸ PAUSED  [-][#463C6E][TAB] focus  [1-5] jump  [ENTER] detail  [R] refresh  [P] resume  [V] verify  [?] help  [Q] quit[-]")
	} else {
		s.SetText("[#463C6E]  [TAB] focus  [1-5] jump  [ENTER] detail  [R] refresh  [P] pause  [V] verify  [?] help  [Q] quit[-]")
	}
}
