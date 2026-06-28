package dashboard

import (
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func Sparkline(values []float64, width int, colour lipgloss.Color) string {
	if width <= 0 { return "" }
	if len(values) == 0 {
		return strings.Repeat(string(sparkChars[0]), width)
	}
    sampled := resample(values, width)
    mx := maxSlice(sampled)
    if mx == 0 { mx = 1 }
    style := lipgloss.NewStyle().Foreground(colour)
    var sb strings.Builder
    for _, v := range sampled {
        idx := int((v / mx) * float64(len(sparkChars)-1))
        if idx < 0 { idx = 0 }
        if idx >= len(sparkChars) { idx = len(sparkChars) - 1 }
        sb.WriteString(style.Render(string(sparkChars[idx])))
    }
    return sb.String()
}

func resample(in []float64, n int) []float64 {
	if n <= 1 {
		if len(in) == 0 { return make([]float64, 1) }
		return []float64{in[len(in)-1]}
	}
	if len(in) == n { return in }
	if len(in) == 0 { return make([]float64, n) }
	out := make([]float64, n)
	for i := range out {
		src := float64(i) * float64(len(in)-1) / float64(n-1)
        lo := int(src)
        if lo >= len(in)-1 { lo = len(in) - 2 }
        if lo < 0 { lo = 0 }
        hi := lo + 1
        if hi >= len(in) { hi = len(in) - 1 }
        out[i] = in[lo]*(1-src+float64(lo)) + in[hi]*(src-float64(lo))
    }
    return out
}

func maxSlice(s []float64) float64 {
    if len(s) == 0 { return 0 }
    m := s[0]
    for _, v := range s[1:] {
        if v > m { m = v }
    }
    return m
}

func TTLBar(remainingSecs, totalSecs int) string {
    if totalSecs <= 0 || remainingSecs <= 0 {
        return lipgloss.NewStyle().Foreground(theme.Failure).Render("░░░")
    }
    frac := float64(remainingSecs) / float64(totalSecs)
    filled := int(frac*3 + 0.5)
    if filled > 3 { filled = 3 }
    col := theme.Success
    switch {
    case remainingSecs <= 10: col = theme.Failure
    case remainingSecs <= 30: col = theme.Warn
    }
    s := lipgloss.NewStyle().Foreground(col)
    return s.Render(strings.Repeat("▓", filled)) +
        lipgloss.NewStyle().Foreground(theme.TextDim).Render(strings.Repeat("░", 3-filled))
}
