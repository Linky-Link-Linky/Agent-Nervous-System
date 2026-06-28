// ANS CLI — Agent Nervous System
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/assets"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/commands"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tuiengine"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/tui/styles"
	"golang.org/x/term"
)

func main() {
	cfg := styles.LoadConfig()
	if cfg.ThemeIndex >= 0 && cfg.ThemeIndex < len(styles.Themes) {
		styles.CurrentTheme = styles.Themes[cfg.ThemeIndex]
	}

	if len(os.Args) < 2 {
		if !isTerminal() || os.Getenv("ANS_TEST") != "" {
			commands.PrintUsageTo(os.Stderr)
			os.Exit(0)
		}
		runTerminalTUI()
		return
	}

	if os.Args[1] == "_daemon" {
		runDaemon()
		return
	}

	if os.Args[1] == "dashboard" || os.Args[1] == "dash" || os.Args[1] == "tui" {
		runEbitenTUI()
		return
	}

	if err := commands.Dispatch(os.Args[1:]); err != nil {
		if err.Error() != "" {
			fmt.Fprintf(os.Stderr, "ans: %v\n", err)
		}
		os.Exit(1)
	}
}

// runDaemon is the actual daemon entry point, invoked via re-exec.
func runDaemon() {
	fs := flag.NewFlagSet("_daemon", flag.ContinueOnError)
	ndjson := fs.Bool("ndjson", false, "")
	webhook := fs.String("webhook", "", "")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "ans: flag error: %v\n", err)
		os.Exit(1)
	}

	writePID()
	defer func() {
		if err := os.Remove(pidFilePath()); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "ans: warning: removing PID file: %v\n", err)
		}
	}()
	d, err := daemon.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ans daemon: init failed: %v\n", err)
		os.Exit(1)
	}
	if *ndjson {
		d.NDJSONWriter = os.Stdout
	}
	if *webhook != "" {
		d.WebhookURL = *webhook
	}
	if err := d.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ans daemon: %v\n", err)
		os.Exit(1)
	}
}

const Version = "v0.9.0"

func runEbitenTUI() {
	if os.Getenv("ANSTUI_NO_EBITEN") != "" {
		runTerminalTUI()
		return
	}

	tuiOutR, tuiOutW := io.Pipe()
	tuiInR, tuiInW := io.Pipe()

	btApp := tui.NewApp()
	p := tea.NewProgram(btApp, tea.WithOutput(tuiOutW), tea.WithInput(tuiInR))
	go func() {
		if _, err := p.Run(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}()

	fontData, err := assets.FS.ReadFile("JetBrainsMono-Regular.ttf")
	if err != nil {
		fontData, err = os.ReadFile(filepath.Join(assetDir(), "JetBrainsMono-Regular.ttf"))
		if err != nil {
			runTerminalTUI()
			return
		}
	}

	shaderData, err := assets.FS.ReadFile("crt.kage")
	if err != nil {
		shaderData, _ = os.ReadFile(filepath.Join(assetDir(), "crt.kage"))
	}

	bridge, err := tuiengine.NewBridge(tuiOutR, fontData)
	if err != nil {
		runTerminalTUI()
		return
	}

	game, err := tuiengine.NewGame(bridge, tuiInW, shaderData)
	if err != nil {
		runTerminalTUI()
		return
	}

	ebiten.SetWindowSize(1280, 800)
	ebiten.SetWindowTitle("ANS — Agent Notification System")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	playBootSound()

	if err := ebiten.RunGame(game); err != nil {
		os.Exit(1)
	}
}

func runTerminalTUI() {
	p := tea.NewProgram(tui.NewApp(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		os.Exit(1)
	}
}

func playBootSound() {
	data, _ := assets.FS.ReadFile("boot.wav")
	if data != nil {
		tuiengine.PlayBootSoundData(data)
	}
}

func assetDir() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "assets")
}

// --- helpers ---

func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ans", "daemon.pid")
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func writePID() {
	p := pidFilePath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "ans: warning: creating PID dir: %v\n", err)
		return
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			fmt.Fprintf(os.Stderr, "ans: daemon already running (PID file exists: %s)\n", p)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "ans: warning: creating PID file: %v\n", err)
		return
	}
	fmt.Fprintf(f, "%d\n", os.Getpid())
	f.Close()
}
