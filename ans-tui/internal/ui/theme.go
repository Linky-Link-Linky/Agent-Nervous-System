package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var (
	ColorBG         = tcell.NewRGBColor(10, 8, 20)
	ColorPanelBG    = tcell.NewRGBColor(18, 14, 38)
	ColorBorderNorm = tcell.NewRGBColor(80, 50, 160)
	ColorBorderFoc  = tcell.NewRGBColor(160, 100, 255)
	ColorBorderDim  = tcell.NewRGBColor(40, 30, 80)

	ColorTextPrimary   = tcell.NewRGBColor(220, 210, 255)
	ColorTextSecondary = tcell.NewRGBColor(130, 110, 190)
	ColorTextDim       = tcell.NewRGBColor(70, 60, 110)

	ColorSuccess = tcell.NewRGBColor(100, 220, 140)
	ColorFailure = tcell.NewRGBColor(255, 80, 100)
	ColorWarn    = tcell.NewRGBColor(255, 190, 60)
	ColorPartial = tcell.NewRGBColor(100, 160, 255)
	ColorDenied  = tcell.NewRGBColor(255, 60, 80)
	ColorAllowed = tcell.NewRGBColor(80, 220, 160)
	ColorPending = tcell.NewRGBColor(200, 160, 255)

	ColorAccentA = tcell.NewRGBColor(180, 80, 255)
	ColorAccentB = tcell.NewRGBColor(80, 200, 255)
	ColorAccentC = tcell.NewRGBColor(255, 120, 180)

	SparkColors = []tcell.Color{
		tcell.NewRGBColor(60, 40, 100),
		tcell.NewRGBColor(100, 60, 180),
		tcell.NewRGBColor(140, 80, 220),
		tcell.NewRGBColor(180, 100, 255),
		tcell.NewRGBColor(220, 150, 255),
	}
)

func FormatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func FormatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%dB", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1fKB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
}

var panelGlyphs = map[string]string{
	"chain":     "◈",
	"snapshots": "⏱",
	"policy":    "⚙",
	"tokens":    "⚡",
	"mcp":       "⬡",
}

func PanelTitle(glyph, name string) string {
	return fmt.Sprintf(" %s %s ", glyph, name)
}

func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func OutcomeGlyph(outcome string) string {
	switch outcome {
	case "success":
		return "[#64DC8C]✓ ok[-]"
	case "failure":
		return "[#FF5064]✗ fail[-]"
	case "partial":
		return "[#64A0FF]◐ part[-]"
	case "denied":
		return "[#FF3C50]⊘ deny[-]"
	default:
		return "[#826EBE]— —[-]"
	}
}

func ActionTypeColor(actionType string) string {
	switch {
	case strings.HasPrefix(actionType, "file."):
		return "[#50C8FF]"
	case strings.HasPrefix(actionType, "http."):
		return "[#FFBE3C]"
	case strings.HasPrefix(actionType, "shell."):
		return "[#FF5064]"
	case strings.HasPrefix(actionType, "agent."), strings.HasPrefix(actionType, "db."):
		return "[#B450FF]"
	default:
		return "[#DCD2FF]"
	}
}

func TTLBar(remaining, total int) string {
	if total <= 0 || remaining <= 0 {
		return "[#FF5064]░░░[-]"
	}
	frac := float64(remaining) / float64(total)
	filled := int(frac*3 + 0.5)
	if filled > 3 {
		filled = 3
	}
	bar := strings.Repeat("▓", filled) + strings.Repeat("░", 3-filled)
	var color string
	switch {
	case remaining > 30:
		color = "[#64DC8C]"
	case remaining > 10:
		color = "[#FFBE3C]"
	default:
		color = "[#FF5064]"
	}
	return color + bar + "[-]"
}

func StatusDot(running bool) string {
	if running {
		return "[#64DC8C]●[-]"
	}
	return "[#FFBE3C]○[-]"
}
