package tui

import "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"

type Keybinding struct {
	Key  string
	Desc string
}

type KeymapGroup struct {
	Name  string
	Binds []Keybinding
}

var GlobalKeymap = []Keybinding{
	{"Tab / Ctrl+P", "Toggle command palette"},
	{"?", "Toggle help overlay"},
	{"F1–F7", "Switch view"},
	{"Ctrl+T", "Cycle theme"},
	{"Ctrl+C / q", "Quit (with confirmation)"},
	{"Ctrl+R", "Reverse history search"},
}

var ViewKeymaps = map[string][]Keybinding{
	"dashboard": {
		{"j/k or ↑/↓", "Scroll"},
		{"Enter", "Select / expand"},
		{"w", "Toggle watch mode"},
	},
	"log": {
		{"j/k", "Scroll up/down"},
		{"gg", "Jump to top"},
		{"G", "Jump to bottom"},
		{"/", "Search"},
		{"v", "Verify selected receipt"},
		{"Enter", "Expand receipt"},
	},
	"stream": {
		{"p", "Pause stream"},
		{"c", "Clear buffer"},
		{"j/k", "Scroll"},
		{"w", "Toggle auto-refresh"},
	},
	"snap": {
		{"←/→", "Move time-travel scrubber"},
		{"Enter", "Restore selected snapshot"},
		{"j/k", "Navigate list"},
	},
	"policy": {
		{"n", "Add policy from file"},
		{"d", "Delete selected policy"},
		{"t", "Test/eval policy"},
		{"Enter", "View policy JSON"},
	},
	"token": {
		{"n", "New token"},
		{"r", "Revoke selected"},
		{"Enter", "View token details"},
	},
	"proxy": {
		{"o", "Proxy on"},
		{"f", "Proxy off"},
		{"l", "Load full log"},
		{"w", "Toggle watch"},
	},
}

var CommandList = styles.CommandList
