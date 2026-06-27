// ANS CLI — Agent Nervous System
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/client"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/commands"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/poller"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		if !isTerminal() || os.Getenv("ANS_TEST") != "" {
			commands.PrintUsageTo(os.Stderr)
			os.Exit(0)
		}
		runTUI(false)
		return
	}

	if os.Args[1] == "_daemon" {
		runDaemon()
		return
	}

	if os.Args[1] == "dashboard" || os.Args[1] == "dash" || os.Args[1] == "tui" {
		runTUI(false)
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

func runTUI(demo bool) {
	var c client.Client
	if demo {
		c = client.NewMock()
	} else {
		c = client.NewSocket(client.DefaultSockPath())
	}

	p := poller.New(c)
	p.Start()
	defer p.Stop()

	m := dashboard.New(p, c, demo, Version)
	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
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
	if err := os.WriteFile(p, []byte(strconv.Itoa(os.Getpid())), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "ans: warning: writing PID file: %v\n", err)
	}
}
