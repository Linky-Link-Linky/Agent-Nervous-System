package cli

import (
	"flag"
	"fmt"

	"ans/internal/client"
)

func runMCP(args []string, c client.Client) {
	if len(args) == 0 {
		Fail("Usage: ans mcp <start|stop|status|log> [flags]")
		exitErr(1)
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "start":
		runMCPStart(rest, c)
	case "stop":
		runMCPStop(c)
	case "status":
		runMCPStatus(c)
	case "log":
		runMCPLog(rest, c)
	default:
		Fail("unknown mcp subcommand: " + sub)
		exitErr(1)
	}
}

func runMCPStart(args []string, c client.Client) {
	fs := flag.NewFlagSet("mcp start", flag.ExitOnError)
	target := fs.String("target", "", "upstream MCP server URL (required)")
	listen := fs.String("listen", ":0", "listen address")
	inj := fs.Bool("safety-disable", false, "disable injection detection")
	rate := fs.Int("rate-limit", 100, "requests per minute limit")
	fs.Parse(args)

	if *target == "" {
		Fail("--target is required")
		exitErr(1)
	}

	info, err := c.MCPStart(*target, *listen, *inj, *rate)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("MCP proxy started on " + info.ListenAddr)
}

func runMCPStop(c client.Client) {
	if err := c.MCPStop(); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("MCP proxy stopped")
}

func runMCPStatus(c client.Client) {
	s, err := c.MCPStatus()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	statusDot := Green("● RUNNING")
	if !s.Running {
		statusDot = Red("● STOPPED")
	}
	fmt.Println(Bold("MCP PROXY STATUS"))
	fmt.Println(Dim("────────────────────────────────────"))
	fmt.Printf("  %-16s %s\n", "Status", statusDot)
	fmt.Printf("  %-16s %s\n", "Version", s.Version)
	fmt.Printf("  %-16s %d\n", "Injections Detected", s.Injections)
	fmt.Printf("  %-16s %d / %d req/min\n", "Rate Limit", s.RateLimited, s.RateLimited)
	fmt.Printf("  %-16s %s\n", "PID", Dim(s.ListenAddr))
}

func runMCPLog(args []string, c client.Client) {
	fs := flag.NewFlagSet("mcp log", flag.ExitOnError)
	inj := fs.Bool("inj", false, "show only injection events")
	method := fs.String("method", "", "filter by method")
	n := fs.Int("n", 20, "number of log lines")
	fs.Parse(args)

	logs, err := c.MCPLog(*inj, *method, *n)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if len(logs) == 0 {
		Warn("No log entries")
		return
	}
	fmt.Println(Bold("MCP LOG"))
	fmt.Println(Dim("────────────────────────────────────────────"))
	for _, l := range logs {
		ts := l.Timestamp.Format("15:04:05.000")
		lvl := l.Direction
		msg := l.ContentPreview
		coloredLvl := Dim(lvl)
		if lvl == "ERROR" || lvl == "WARN" {
			coloredLvl = Amber(lvl)
		}
		fmt.Printf("  %s %s %s\n", Dim(ts), coloredLvl, msg)
	}
}
