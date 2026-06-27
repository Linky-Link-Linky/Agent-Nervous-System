package cli

import (
	"flag"
	"fmt"

	"ans/internal/client"
)

func runInit(c client.Client) {
	if err := c.Init(); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Initialised. Run 'ans start' to begin.")
}

func runDoctor(c client.Client) {
	report, err := c.Doctor()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	fmt.Println(Bold("ANS DIAGNOSTICS"))
	fmt.Println(Dim("────────────────────────────────────"))
	for _, chk := range report.Checks {
		status := Green("✓")
		if chk.Status == "warn" {
			status = Amber("⚠")
		} else if chk.Status == "fail" {
			status = Red("✗")
		}
		fmt.Printf("  %-16s %-26s %s  %s\n", chk.Name, chk.Value, status, chk.Detail)
	}
	if report.AllOK {
		OK("All checks passed")
	} else {
		Warn("Some checks failed")
	}
}

func runStart(args []string, c client.Client) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	ndjson := fs.Bool("ndjson", false, "stream NDJSON output")
	webhook := fs.String("webhook", "", "webhook URL")
	fs.Parse(args)

	if err := c.StartDaemon(*ndjson, *webhook); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Daemon started")
}

func runStop(c client.Client) {
	if err := c.StopDaemon(); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Daemon stopped")
}

func runStatus(c client.Client) {
	s, err := c.DaemonStatus()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	statusDot := Green("● RUNNING")
	if !s.Running {
		statusDot = Red("● STOPPED")
	}
	verified := Green("✓")
	if !s.ChainVerified {
		verified = Red("✗")
	}
	fmt.Println(Bold("ANS DAEMON STATUS"))
	fmt.Println(Dim("────────────────────────────────────"))
	fmt.Printf("  %-16s %s\n", "Status", statusDot)
	fmt.Printf("  %-16s %s\n", "Uptime", s.Uptime)
	fmt.Printf("  %-16s %d receipts\n", "Chain Length", s.ChainLength)
	fmt.Printf("  %-16s %d\n", "Agents", s.AgentCount)
	fmt.Printf("  %-16s %.1f MB\n", "DB Size", s.DBSizeMB)
	fmt.Printf("  %-16s %s\n", "Chain", verified)
	fmt.Printf("  %-16s %s\n", "Version", s.Version)
}

func runUpdate(c client.Client) {
	ver, err := c.Update()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Updated to " + ver)
}

func runUninstall(c client.Client) {
	if err := c.Uninstall(); err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("ANS uninstalled")
}

func runVersion() {
	fmt.Println("ans v0.1.0 (Go 1.22)")
}
