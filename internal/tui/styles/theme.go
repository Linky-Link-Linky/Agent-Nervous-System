package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
    Name         string
    Fg           lipgloss.Color
    FgMuted      lipgloss.Color
    Bg           lipgloss.Color
    BgAlt        lipgloss.Color
    Accent       lipgloss.Color
    Success      lipgloss.Color
    Warning      lipgloss.Color
    Danger       lipgloss.Color
    BorderNormal lipgloss.Color
    BorderActive lipgloss.Color
    CRTTint      [3]float64
}

var CurrentTheme Theme = Phosphor

var (
    Phosphor = Theme{
        Name: "Phosphor", Fg: "#39ff14", FgMuted: "#5a7a50",
        Bg: "#0a0f05", BgAlt: "#0f1a0a", Accent: "#39ff14",
        Success: "#39ff14", Warning: "#ffb000", Danger: "#ff3333",
        BorderNormal: "#1a3a10", BorderActive: "#39ff14",
        CRTTint: [3]float64{0.92, 1.0, 0.90},
    }
    Amber = Theme{
        Name: "Amber", Fg: "#ffb000", FgMuted: "#8a6a20",
        Bg: "#0d0900", BgAlt: "#141000", Accent: "#ffb000",
        Success: "#80d000", Warning: "#ff8000", Danger: "#ff3333",
        BorderNormal: "#3a2a00", BorderActive: "#ffb000",
        CRTTint: [3]float64{1.0, 0.94, 0.85},
    }
    Ice = Theme{
        Name: "Ice", Fg: "#7fefff", FgMuted: "#4a8a99",
        Bg: "#050a0f", BgAlt: "#0a1520", Accent: "#7fefff",
        Success: "#7fefff", Warning: "#ffb000", Danger: "#ff5555",
        BorderNormal: "#0a2a3a", BorderActive: "#7fefff",
        CRTTint: [3]float64{0.90, 0.95, 1.0},
    }
    Paper = Theme{
        Name: "Paper", Fg: "#1a1a1a", FgMuted: "#666666",
        Bg: "#f5f0e8", BgAlt: "#e8e0d0", Accent: "#8b7355",
        Success: "#2e7d32", Warning: "#bf6a00", Danger: "#c62828",
        BorderNormal: "#c0b8a8", BorderActive: "#8b7355",
        CRTTint: [3]float64{1.0, 1.0, 1.0},
    }
)

var Themes = []Theme{Phosphor, Amber, Ice, Paper}

func ThemeNames() []string {
    n := make([]string, len(Themes))
    for i, t := range Themes { n[i] = t.Name }
    return n
}

func (t Theme) BoxStyle() lipgloss.Style {
    return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
        BorderForeground(t.BorderNormal).Background(t.BgAlt).
        Padding(0, 1)
}

func (t Theme) ActiveBoxStyle() lipgloss.Style {
    return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
        BorderForeground(t.BorderActive).Background(t.BgAlt).
        Padding(0, 1)
}

func (t Theme) TitleStyle() lipgloss.Style {
    return lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
}

func (t Theme) StatCard(title, value string) string {
    s := t.BoxStyle().Width(20)
    return s.Render(lipgloss.JoinVertical(lipgloss.Center,
        lipgloss.NewStyle().Foreground(t.FgMuted).Render(title),
        t.TitleStyle().Render(value),
    ))
}

func (t Theme) Badge(text string, ok bool) string {
    col := t.Success
    if !ok { col = t.Danger }
    return lipgloss.NewStyle().Foreground(col).Bold(true).Render(" " + text + " ")
}

func (t Theme) PulsingBadge(text string, pulse bool) string {
    col := t.Accent
    if pulse { col = t.FgMuted }
    return lipgloss.NewStyle().Foreground(col).Render(text)
}
