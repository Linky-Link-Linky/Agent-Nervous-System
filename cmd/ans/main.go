package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"ans/internal/cli"
	"ans/internal/client"
	"ans/internal/config"
	"ans/internal/poller"
	"ans/internal/ui"
)

func main() {
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		known := map[string]bool{
			"init": true, "doctor": true, "start": true, "stop": true,
			"status": true, "update": true, "uninstall": true, "version": true,
			"help": true, "chain": true, "verify": true, "agents": true,
			"register": true, "export": true, "prune": true, "rotate": true,
			"time-travel": true, "snapshot": true, "snapshots": true,
			"compensate": true, "policy": true, "token": true, "mcp": true,
		}
		if known[os.Args[1]] {
			cli.Run(os.Args[1], os.Args[2:], client.NewMockClient())
			return
		}
	}

	demo := flag.Bool("demo", false, "Run in demo mode with mock data (no daemon required)")
	sock := flag.String("sock", defaultSocketPath(), "Path to ANS daemon socket")
	refresh := flag.Int("refresh", 2000, "Base refresh interval in milliseconds")
	flag.Parse()

	var c client.Client
	if *demo {
		c = client.NewMockClient()
		fmt.Fprintln(os.Stderr, "ans: running in demo mode (--demo)")
	} else {
		c = client.NewSocketClient(*sock)
	}

	cfg := config.Load()
	baseMS := *refresh
	if baseMS < 100 {
		baseMS = 100
	}

	if cfg.RefreshIntervalMS != baseMS {
		cfg.RefreshIntervalMS = baseMS
		_ = cfg.Save()
	}

	p := poller.New(c, baseMS)
	app := ui.NewApp(p, *demo)

	p.Start()

	if err := app.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}

	p.Stop()
}

func defaultSocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ans", "daemon.sock")
}
