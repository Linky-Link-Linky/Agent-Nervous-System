package cli

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

var exitErr = func(code int) { os.Exit(code) }

var useColour = term.IsTerminal(int(os.Stdout.Fd()))

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[38;2;100;220;140m"
	ansiRed    = "\033[38;2;255;80;100m"
	ansiAmber  = "\033[38;2;255;190;60m"
	ansiCyan   = "\033[38;2;80;200;255m"
	ansiPurple = "\033[38;2;180;80;255m"
	ansiDim    = "\033[38;2;70;60;110m"
	ansiBold   = "\033[1m"
)

func colorize(col, text string) string {
	if !useColour {
		return text
	}
	return col + text + ansiReset
}

func Green(s string) string  { return colorize(ansiGreen, s) }
func Red(s string) string    { return colorize(ansiRed, s) }
func Amber(s string) string  { return colorize(ansiAmber, s) }
func Cyan(s string) string   { return colorize(ansiCyan, s) }
func Purple(s string) string { return colorize(ansiPurple, s) }
func Dim(s string) string    { return colorize(ansiDim, s) }
func Bold(s string) string   { return colorize(ansiBold, s) }

func OK(msg string) {
	fmt.Println(Green("✓  ") + msg)
}

func Fail(msg string) {
	fmt.Fprintln(os.Stderr, Red("✗  ") + msg)
}

func Warn(msg string) {
	fmt.Fprintln(os.Stderr, Amber("⚠  ") + msg)
}
