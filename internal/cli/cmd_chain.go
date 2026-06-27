package cli

import (
	"flag"
	"fmt"
	"strings"

	"ans/internal/client"
)

func runChain(args []string, c client.Client) {
	fs := flag.NewFlagSet("chain", flag.ExitOnError)
	n := fs.Int("n", 20, "number of receipts")
	agent := fs.String("agent", "", "filter by agent ID")
	fs.Parse(args)

	receipts, err := c.ListReceipts(*n, *agent)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if len(receipts) == 0 {
		Warn("No receipts found")
		return
	}

	fmt.Println(Bold("ANS — Agent Nervous System"))
	fmt.Println(Dim("──────────────────────────────────────────────────"))
	fmt.Println()

	for _, r := range receipts {
		hash := Cyan(trunc(r.ID, 12))
		timeStr := r.Timestamp.Format("2006-01-02 15:04:05")
		actionColored := actionColor(r.ActionType) + r.ActionType + Dim("")
		outcomeGlyph := outcomeStr(r.Outcome)
		decision := Dim("")
		if r.PolicyDecision == "deny" {
			decision = Red("deny")
		} else {
			decision = Green("allow")
		}

		fmt.Printf("%s  %s  %s  %s  %s\n",
			Dim("┌─")+hash,
			Dim(timeStr),
			actionColored,
			r.AgentID,
			Dim(r.PayloadSummary))
		fmt.Printf("  %s %s  %s  %dms\n",
			Dim("│  policy"), decision, outcomeGlyph, r.DurationMS)
		fmt.Printf("  %s sig %s\n",
			Dim("└─"), Dim(trunc(r.Signature, 16)+"…"))
		fmt.Println()
	}
}

func runVerify(args []string, c client.Client) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	fullChain := fs.Bool("chain", false, "verify entire chain")
	fs.Parse(args)

	if *fullChain {
		fmt.Print("  Verifying receipts ")
		done := make(chan struct{})
		defer close(done)
		spinner := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
		go func() {
			for i := 0; ; i++ {
				select {
				case <-done:
					fmt.Print("\r                             \r")
					return
				default:
					fmt.Printf("\r  Verifying receipts %s ", spinner[i%len(spinner)])
				}
			}
		}()

		verified, count, err := c.VerifyChain()
		if err != nil {
			Fail(err.Error())
			exitErr(1)
		}
		if verified {
			OK(fmt.Sprintf("Chain integrity verified — %d receipts checked (all hashes, all signatures)", count))
		} else {
			Fail(fmt.Sprintf("Chain BROKEN — %d receipts checked", count))
			exitErr(1)
		}
		return
	}

	if len(fs.Args()) == 0 {
		Fail("Usage: ans verify <receipt-id-or-hash> [--chain]")
		exitErr(1)
	}

	id := fs.Args()[0]
	valid, err := c.VerifyReceipt(id)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	if valid {
		OK("Receipt " + trunc(id, 16) + " — VALID")
	} else {
		Fail("Receipt " + trunc(id, 16) + " — INVALID")
		exitErr(1)
	}
}

func runAgents(c client.Client) {
	agents, err := c.ListAgents()
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	fmt.Println(Bold("REGISTERED AGENTS"))
	fmt.Println(Dim("────────────────────────────────────────────────"))
	fmt.Printf("%-20s %-14s %-10s %-18s %s\n", "ID", "NAME", "VERSION", "OWNER", "PUBKEY")
	fmt.Println(Dim(strings.Repeat("─", 90)))
	for _, a := range agents {
		fmt.Printf("%-20s %-14s %-10s %-18s %s\n",
			Purple(trunc(a.ID, 16)),
			a.Name,
			a.Version,
			a.Owner,
			Dim(trunc(a.PublicKey, 16)))
	}
}

func runRegister(args []string, c client.Client) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	name := fs.String("name", "", "agent name (required)")
	version := fs.String("version", "1.0.0", "agent version")
	owner := fs.String("owner", "", "owner/org name")
	fs.Parse(args)

	if *name == "" {
		Fail("--name is required")
		exitErr(1)
	}

	agent, err := c.RegisterAgent(*name, *version, *owner)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Agent registered")
	fmt.Printf("  ID:      %s\n", Purple(agent.ID))
	fmt.Printf("  Name:    %s\n", agent.Name)
	fmt.Printf("  Version: %s\n", agent.Version)
	fmt.Printf("  Owner:   %s\n", agent.Owner)
	fmt.Printf("  PubKey:  %s\n", Dim(agent.PublicKey))
}

func runExport(args []string, c client.Client) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	format := fs.String("format", "jsonl", "output format (jsonl/csv/txt/pdf/parquet)")
	output := fs.String("output", "", "output file path")
	fs.Parse(args)

	if *format == "pdf" || *format == "parquet" {
		if *output == "" {
			Fail("--output is required for " + *format + " format")
			exitErr(1)
		}
	}

	bytesWritten, err := c.Export(*format, *output)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK(fmt.Sprintf("Exported %d bytes → %s", bytesWritten, *output))
}

func runPrune(args []string, c client.Client) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	upTo := fs.Int("up-to", 0, "compact receipts up to this index (required)")
	fs.Parse(args)

	if *upTo <= 0 {
		Fail("--up-to <index> is required")
		exitErr(1)
	}

	merkleRoot, err := c.Prune(*upTo)
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Pruned. Merkle root: " + merkleRoot)
}

func runRotate(args []string, c client.Client) {
	fs := flag.NewFlagSet("rotate", flag.ExitOnError)
	fs.Parse(args)

	if len(fs.Args()) == 0 {
		Fail("Usage: ans rotate <agent-id>")
		exitErr(1)
	}

	agent, err := c.RotateKey(fs.Args()[0])
	if err != nil {
		Fail(err.Error())
		exitErr(1)
	}
	OK("Key rotated")
	fmt.Printf("  New PubKey: %s\n", Dim(agent.PublicKey))
}

func trunc(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

func actionColor(t string) string {
	switch {
	case strings.HasPrefix(t, "file."):
		return Cyan("")
	case strings.HasPrefix(t, "http."):
		return Purple("")
	case strings.HasPrefix(t, "shell."):
		return Red("")
	case strings.HasPrefix(t, "agent."), strings.HasPrefix(t, "db."):
		return Amber("")
	default:
		return Dim("")
	}
}

func outcomeStr(o string) string {
	switch o {
	case "success":
		return Green("✓ ok")
	case "failure":
		return Red("✗ fail")
	default:
		return Dim("? " + o)
	}
}
