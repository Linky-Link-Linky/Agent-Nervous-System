package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	path "path/filepath"

	"ans-tui/internal/client"
	"ans-tui/internal/config"
	"ans-tui/internal/poller"
	"ans-tui/internal/ui"
)

func main() {
	demo := flag.Bool("demo", false, "Run in demo mode with mock data (no daemon required)")
	sock := flag.String("sock", defaultSocketPath(), "Path to ANS daemon socket")
	refresh := flag.Int("refresh", 2000, "Base refresh interval in milliseconds")
	flag.Parse()

	var c client.Client
	if *demo {
		c = client.NewMockClient()
		fmt.Fprintln(os.Stderr, "ans-tui: running in demo mode (--demo)")
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
	return path.Join(home, ".ans", "daemon.sock")
}
