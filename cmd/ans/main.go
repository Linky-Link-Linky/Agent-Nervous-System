// ANS CLI — Agent Nervous System
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/commands"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard"
	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		if !isTerminal() || os.Getenv("ANS_TEST") != "" {
			commands.PrintUsageTo(os.Stderr)
			os.Exit(0)
		}
		cmdDashboard(3)
		return
	}

	if os.Args[1] == "_daemon" {
		runDaemon()
		return
	}

	if os.Args[1] == "dashboard" || os.Args[1] == "dash" {
		refreshSec := 3
		if len(os.Args) > 2 {
			if os.Args[2] == "--refresh" && len(os.Args) > 3 {
				if s, err := strconv.Atoi(os.Args[3]); err == nil {
					refreshSec = s
				}
			}
		}
		cmdDashboard(refreshSec)
		return
	}
	if os.Getenv("ANS_DASHBOARD_REFRESH") != "" {
		if s, err := strconv.Atoi(os.Getenv("ANS_DASHBOARD_REFRESH")); err == nil {
			runDashboard(s)
			return
		}
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

func runDashboard(refreshSec int) {
	app := dashboard.NewApp(refreshSec)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ans: dashboard error: %v\n", err)
		os.Exit(1)
	}
}

func cmdDashboard(refreshSec int) {
	if conn, err := daemon.Dial(); err != nil {
		if self, exeErr := os.Executable(); exeErr == nil {
			cmd := exec.Command(self, "_daemon")
			cmd.Stdout = nil
			cmd.Stderr = nil
			_ = cmd.Start()
			for i := 0; i < 30; i++ {
				time.Sleep(100 * time.Millisecond)
				if conn, dialErr := daemon.Dial(); dialErr == nil {
					conn.Close()
					break
				}
			}
		}
	} else {
		conn.Close()
	}
	runDashboard(refreshSec)
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
