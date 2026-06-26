package ui

import (
	"testing"
	"unicode/utf8"
)

func TestRenderSparkline_Empty(t *testing.T) {
	result := RenderSparkline(nil, 10)
	if n := utf8.RuneCountInString(result); n != 10 {
		t.Fatalf("expected 10 chars, got %d", n)
	}
	for _, r := range result {
		if r != sparkChars[0] {
			t.Fatalf("expected all '▁', got %c", r)
		}
	}
}

func TestRenderSparkline_Constant(t *testing.T) {
	input := []float64{50, 50, 50, 50, 50}
	result := RenderSparkline(input, 5)
	if n := utf8.RuneCountInString(result); n != 5 {
		t.Fatalf("expected 5 chars, got %d", n)
	}
	for _, r := range result {
		if r == sparkChars[0] {
			t.Fatalf("constant input should not produce minimum char")
		}
	}
}

func TestRenderSparkline_Ascending(t *testing.T) {
	input := []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	result := RenderSparkline(input, 11)
	if n := utf8.RuneCountInString(result); n != 11 {
		t.Fatalf("expected 11 chars, got %d", n)
	}
	// Every value should be >= the previous
	var prevIdx int
	for i, r := range result {
		cur := indexOf(r)
		if i > 0 && cur < prevIdx {
			t.Fatalf("decreasing sparkline at position %d: %c -> %c", i, prevIdx, cur)
		}
		prevIdx = cur
	}
}

func TestRenderSparkline_Width(t *testing.T) {
	input := []float64{1, 2, 3, 4, 5}
	for _, w := range []int{3, 5, 10, 20} {
		result := RenderSparkline(input, w)
		if n := utf8.RuneCountInString(result); n != w {
			t.Fatalf("expected width %d, got %d", w, n)
		}
	}
}

func indexOf(r rune) int {
	for i, c := range sparkChars {
		if c == r {
			return i
		}
	}
	return -1
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   int64
		want string
	}{
		{0, "0ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{2500, "2.5s"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.ms)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.ms, got, tt.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		b    int64
		want string
	}{
		{0, "0B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1048576, "1.0MB"},
	}
	for _, tt := range tests {
		got := FormatBytes(tt.b)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.b, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Errorf("short string: got %q", got)
	}
	if got := Truncate("hello world this is long", 10); len([]rune(got)) != 10 {
		t.Errorf("long string: got %q (len %d)", got, len([]rune(got)))
	}
}

func TestTTLBar(t *testing.T) {
	if got := TTLBar(60, 60); got == "[#FF5064]░░░[-]" {
		t.Errorf("full TTL should not be all empty: %q", got)
	}
	if got := TTLBar(0, 60); got != "[#FF5064]░░░[-]" {
		t.Errorf("zero TTL should be all empty: %q", got)
	}
}
