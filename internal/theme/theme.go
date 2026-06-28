package theme

import (
    "fmt"
    "strings"
    "time"
    "unicode/utf8"

    "github.com/charmbracelet/lipgloss"
)

// Backgrounds
var (
    BG       = lipgloss.Color("#0A0814")
    PanelBG  = lipgloss.Color("#120E26")
    Overlay  = lipgloss.Color("#1A1535")
)

// Borders
var (
    BorderNorm  = lipgloss.Color("#5032A0")
    BorderFocus = lipgloss.Color("#A064FF")
    BorderDim   = lipgloss.Color("#281E50")
)

// Text
var (
    TextPrimary   = lipgloss.Color("#DCD2FF")
    TextSecondary = lipgloss.Color("#826EBE")
    TextDim       = lipgloss.Color("#463C6E")
    TextInverse   = lipgloss.Color("#0A0814")
)

// Semantic
var (
    Success = lipgloss.Color("#64DC8C")
    Failure = lipgloss.Color("#FF5064")
    Warn    = lipgloss.Color("#FFBE3C")
    Partial = lipgloss.Color("#64A0FF")
    Denied  = lipgloss.Color("#FF3C50")
    Allowed = lipgloss.Color("#50DCA0")
    Pending = lipgloss.Color("#C8A0FF")
)

// Accent
var (
    AccentA = lipgloss.Color("#B450FF")
    AccentB = lipgloss.Color("#50C8FF")
    AccentC = lipgloss.Color("#FF78B4")
)

func ActionColor(actionType string) lipgloss.Color {
    switch {
    case len(actionType) >= 5 && actionType[:5] == "file.":   return AccentB
    case len(actionType) >= 5 && actionType[:5] == "http.":   return Warn
    case len(actionType) >= 6 && actionType[:6] == "shell.":  return Failure
    case len(actionType) >= 6 && actionType[:6] == "agent.":  return AccentA
    case len(actionType) >= 3 && actionType[:3] == "db.":     return AccentA
    default:                                                   return TextPrimary
    }
}

func OutcomeStyle(outcome string) lipgloss.Style {
    switch outcome {
    case "success": return lipgloss.NewStyle().Foreground(Success)
    case "failure": return lipgloss.NewStyle().Foreground(Failure)
    case "partial": return lipgloss.NewStyle().Foreground(Partial)
    case "denied":  return lipgloss.NewStyle().Foreground(Denied)
    default:        return lipgloss.NewStyle().Foreground(TextDim)
    }
}

func OutcomeGlyph(outcome string) string {
    switch outcome {
    case "success": return "✓ ok"
    case "failure": return "✗ fail"
    case "partial": return "◐ part"
    case "denied":  return "⊘ deny"
    default:        return "— —"
    }
}

func OutcomeShort(outcome string) string {
    switch outcome {
    case "success": return "ok"
    case "failure": return "fail"
    case "partial": return "part"
    case "denied":  return "deny"
    default:        return "—"
    }
}

func PanelStyle(focused bool) lipgloss.Style {
    border := BorderNorm
    if focused { border = BorderFocus }
    return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1)
}

func TitleStyle(focused bool) lipgloss.Style {
    fg := TextSecondary
    if focused { fg = TextPrimary }
    return lipgloss.NewStyle().Foreground(fg).Bold(true)
}

var (
    TableHeader = lipgloss.NewStyle().Foreground(TextSecondary).Bold(true).
        BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(BorderDim)
    TableRowSelected = lipgloss.NewStyle().Background(lipgloss.Color("#2A1F55")).Foreground(TextPrimary).Bold(true)
    TableRowNormal   = lipgloss.NewStyle().Foreground(TextPrimary)
    TableRowDim      = lipgloss.NewStyle().Foreground(TextSecondary)
)

var (
    HeaderBrand    = lipgloss.NewStyle().Foreground(AccentA).Bold(true)
    HeaderStat     = lipgloss.NewStyle().Foreground(TextSecondary)
    HeaderStatVal  = lipgloss.NewStyle().Foreground(TextPrimary)
    StatusBarStyle = lipgloss.NewStyle().Foreground(TextDim).Background(lipgloss.Color("#0D0A1E")).PaddingLeft(1)
    KeyHint        = lipgloss.NewStyle().Foreground(AccentA).Bold(true)
    KeyDesc        = lipgloss.NewStyle().Foreground(TextDim)
)

var (
    ModalBox     = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(AccentA).Background(Overlay).Padding(1, 3)
    ModalTitle   = lipgloss.NewStyle().Foreground(AccentA).Bold(true)
    ModalLabel   = lipgloss.NewStyle().Foreground(TextSecondary).Width(18)
    ModalValue   = lipgloss.NewStyle().Foreground(TextPrimary)
    ModalButton  = lipgloss.NewStyle().Foreground(TextInverse).Background(AccentA).Padding(0, 2).MarginLeft(1)
    ModalButtonAlt = lipgloss.NewStyle().Foreground(TextPrimary).Border(lipgloss.RoundedBorder()).BorderForeground(BorderNorm).Padding(0, 2).MarginLeft(1)
)

var (
    CLISuccess = lipgloss.NewStyle().Foreground(Success).Bold(true)
    CLIFailure = lipgloss.NewStyle().Foreground(Failure).Bold(true)
    CLIWarn    = lipgloss.NewStyle().Foreground(Warn).Bold(true)
    CLILabel   = lipgloss.NewStyle().Foreground(TextSecondary).Width(18)
    CLIValue   = lipgloss.NewStyle().Foreground(TextPrimary)
    CLIHash    = lipgloss.NewStyle().Foreground(AccentB)
    CLIAgentID = lipgloss.NewStyle().Foreground(AccentA)
    CLISep     = lipgloss.NewStyle().Foreground(BorderDim)
    CLITitle   = lipgloss.NewStyle().Foreground(AccentA).Bold(true).MarginBottom(1)
    CLIDim     = lipgloss.NewStyle().Foreground(TextDim)
)

func FormatDuration(ms int64) string {
    if ms < 1000 { return fmt.Sprintf("%dms", ms) }
    return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}

func FormatBytes(b int64) string {
	if b < 0 { return "0B" }
	switch {
	case b < 1024:            return fmt.Sprintf("%dB", b)
	case b < 1024*1024:       return fmt.Sprintf("%.1fKB", float64(b)/1024)
	default:                  return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	}
}

func FormatUptime(d time.Duration) string {
    h := int(d.Hours())
    m := int(d.Minutes()) % 60
    s := int(d.Seconds()) % 60
    if h > 0 { return fmt.Sprintf("%dh %dm %ds", h, m, s) }
    if m > 0 { return fmt.Sprintf("%dm %ds", m, s) }
    return fmt.Sprintf("%ds", s)
}

func Trunc(s string, n int) string {
	if n <= 0 { return "" }
	r := []rune(s)
	if utf8.RuneCountInString(s) <= n { return s }
	return string(r[:n-1]) + "…"
}

func PadRight(s string, n int) string {
	if n <= 0 { return "" }
	l := utf8.RuneCountInString(s)
	if l >= n { return Trunc(s, n) }
	return s + strings.Repeat(" ", n-l)
}

func Sep(w int) string {
    return lipgloss.NewStyle().Foreground(BorderDim).Render(strings.Repeat("─", w))
}
