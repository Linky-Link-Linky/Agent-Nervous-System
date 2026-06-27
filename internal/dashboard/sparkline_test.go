package dashboard

import (
    "testing"
    "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/theme"
)

func TestSparkline_EmptyInput(t *testing.T) {
    s := Sparkline(nil, 10, theme.AccentB)
    if len([]rune(s)) != 10 { t.Fatalf("expected 10 chars, got %d", len([]rune(s))) }
    for _, r := range s { if r != '▁' { t.Fatalf("expected only ▁ chars, got %c", r) } }
}

func TestSparkline_Width(t *testing.T) {
    vals := []float64{1, 2, 3, 4, 5}
    for _, w := range []int{1, 3, 5, 10, 20} {
        s := Sparkline(vals, w, theme.AccentA)
        if len([]rune(s)) != w { t.Fatalf("width %d: got %d runes", w, len([]rune(s))) }
    }
}

func TestSparkline_Ascending(t *testing.T) {
    vals := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
    s := Sparkline(vals, 10, theme.AccentB)
    runes := []rune(s)
    for i := 1; i < len(runes); i++ {
        // each char should be >= previous in the sparkChars ordering
        if idxOf(runes[i]) < idxOf(runes[i-1]) { t.Fatalf("non-decreasing at position %d", i) }
    }
}

func idxOf(r rune) int {
    for i, c := range sparkChars { if c == r { return i } }
    return -1
}

func TestTTLBar_Full(t *testing.T) {
    s := TTLBar(60, 60)
    for _, r := range s { if r != '▓' && r != '\x1b' { break } }
    hasFilled := false
    for _, r := range s { if r == '▓' { hasFilled = true; break } }
    if !hasFilled { t.Fatal("expected filled bars") }
}

func TestTTLBar_Empty(t *testing.T) {
    s := TTLBar(0, 60)
    hasEmpty := false
    for _, r := range s { if r == '░' { hasEmpty = true; break } }
    if !hasEmpty { t.Fatal("expected empty bars") }
}

func TestTTLBar_Half(t *testing.T) {
    s := TTLBar(30, 60)
    hasFilled, hasEmpty := false, false
    for _, r := range s {
        if r == '▓' { hasFilled = true }
        if r == '░' { hasEmpty = true }
    }
    if !hasFilled || !hasEmpty { t.Fatal("expected mix of filled and empty") }
}
