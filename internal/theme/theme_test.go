package theme

import (
    "testing"
)

func TestFormatDuration(t *testing.T) {
    cases := []struct { in int64; want string }{
        {0, "0ms"},
        {999, "999ms"},
        {1000, "1.0s"},
        {2500, "2.5s"},
    }
    for _, c := range cases {
        got := FormatDuration(c.in)
        if got != c.want { t.Errorf("FormatDuration(%d) = %q, want %q", c.in, got, c.want) }
    }
}

func TestFormatBytes(t *testing.T) {
    cases := []struct { in int64; want string }{
        {0, "0B"},
        {1023, "1023B"},
        {1024, "1.0KB"},
        {1048576, "1.0MB"},
    }
    for _, c := range cases {
        got := FormatBytes(c.in)
        if got != c.want { t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want) }
    }
}

func TestTrunc(t *testing.T) {
    if s := Trunc("hello", 10); s != "hello" { t.Fatalf("short string changed: %q", s) }
    s := Trunc("hello world", 5)
    if len([]rune(s)) != 5 { t.Fatalf("expected 5 runes, got %d", len([]rune(s))) }
    if string([]rune(s)[4:]) != "…" { t.Fatalf("expected … at end, got %q", s) }
}

func TestPadRight(t *testing.T) {
    if s := PadRight("hi", 5); len([]rune(s)) != 5 { t.Fatalf("expected 5 runes, got %d", len([]rune(s))) }
    s2 := PadRight("hello world", 5)
    if len([]rune(s2)) != 5 { t.Fatalf("expected truncated to 5, got %d", len([]rune(s2))) }
}
