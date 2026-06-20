// ANS CLI — Agent Nervous System
// Commands: start, stop, status, verify, chain, agents, export, version, help
// SPDX-License-Identifier: MIT
package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identity"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot"
)

const version = "0.1.0"

const usageText = `ans — Agent Nervous System v` + version + `

USAGE
  ans <command> [flags]

COMMANDS
  start              Start the ANS daemon in the background
  stop               Stop the running ANS daemon
  status             Show daemon status, chain stats, and uptime
  verify [id]        Verify a receipt by ID (use --chain for full chain)
  chain              Print the receipt chain (pretty tree)
  agents             List registered agents
  register           Register a new agent (--name <name> --version <ver>)
  export             Export the chain (--format jsonl|csv|txt|pdf|parquet)
  prune              Prune old receipts and create Merkle anchor (--up-to <index>)
  rotate             Rotate an agent's keypair (ans rotate <agent-id>)
  snapshot take      Take a snapshot of agent workspace (--agent <id>)
  snapshot diff      Show file-level diff vs prior snapshot (--agent <id>)
  snapshot list      List agent snapshots (--agent <id>)  (alias: snapshots)
  time-travel <id>   Restore state to a chain index or receipt hash
  compensate <id>    Execute registered compensating actions for a chain index
  policy add <file>  Register a policy from JSON file
  policy list        List all policies
  policy remove <id> Remove a policy
  policy eval        Evaluate an action against policies (--action-type ...)
  token request      Provision ephemeral token (--resource <arn> --action <action>)
  token list         List active tokens
  token revoke <id>  Revoke a token
  mcp start          Start MCP security proxy (--listen :8080 --target http://...)
  mcp stop           Stop MCP proxy
  mcp status         Show proxy status and stats
  mcp log            Show recent MCP audit log
  version            Print version

FLAGS (start)
  --ndjson           Emit NDJSON receipt stream to stdout (capture with > file)
  --webhook  string  Webhook URL — POST CloudEvents-formatted payload for
                     each new receipt (e.g. --webhook https://hooks.example.com/ans)

FLAGS (chain)
  --n      int    Receipts to show (default 20)
  --agent  string Filter by agent ID

FLAGS (export)
  --format string  jsonl | csv | txt | pdf | parquet  (default jsonl)
  --output string  Output file (default stdout)

FLAGS (verify)
  --chain          Verify entire chain integrity

Set NO_COLOR=1 to disable ANSI color output.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usageText)
		os.Exit(0)
	}

	// Internal re-exec subcommand — not shown in help.
	if os.Args[1] == "_daemon" {
		runDaemon()
		return
	}

	switch os.Args[1] {
	case "start":
		cmdStart()
	case "stop":
		cmdStop()
	case "status":
		cmdStatus()
	case "verify":
		cmdVerify(os.Args[2:])
	case "chain":
		cmdChain(os.Args[2:])
	case "agents":
		cmdAgents()
	case "register":
		cmdRegister(os.Args[2:])
	case "export":
		cmdExport(os.Args[2:])
	case "prune":
		cmdPrune(os.Args[2:])
	case "rotate":
		cmdRotate(os.Args[2:])
	case "snapshot":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ans snapshot <take|list|diff> [flags]")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "take":
			cmdSnapshotTake(os.Args[3:])
		case "diff":
			cmdSnapshotDiff(os.Args[3:])
		case "list", "ls":
			cmdSnapshots(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "ans: unknown snapshot subcommand %q\n", os.Args[2])
			os.Exit(1)
		}
	case "policy":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ans policy <add|list|remove|eval> [flags]")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "add":
			cmdPolicyAdd(os.Args[3:])
		case "list", "ls":
			cmdPolicyList(os.Args[3:])
		case "remove", "rm", "delete", "del":
			cmdPolicyRemove(os.Args[3:])
		case "eval":
			cmdPolicyEval(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "ans: unknown policy subcommand %q\n", os.Args[2])
			os.Exit(1)
		}
	case "compensate":
		cmdCompensate(os.Args[2:])
	case "token":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ans token <request|list|revoke> [flags]")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "request":
			cmdTokenRequest(os.Args[3:])
		case "list", "ls":
			cmdTokenList(os.Args[3:])
		case "revoke", "rm":
			cmdTokenRevoke(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "ans: unknown token subcommand %q\n", os.Args[2])
			os.Exit(1)
		}
	case "mcp":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ans mcp <start|stop|status|log> [flags]")
			os.Exit(1)
		}
		switch os.Args[2] {
		case "start":
			cmdMCPStart(os.Args[3:])
		case "stop":
			cmdMCPStop()
		case "status":
			cmdMCPStatus()
		case "log":
			cmdMCPLog(os.Args[3:])
		default:
			fmt.Fprintf(os.Stderr, "ans: unknown mcp subcommand %q\n", os.Args[2])
			os.Exit(1)
		}
	case "time-travel", "restore", "rollback":
		cmdTimeTravel(os.Args[2:])
	case "snapshots":
		cmdSnapshots(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("ans version %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
	case "help", "--help", "-h":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "ans: unknown command %q\nRun 'ans help' for usage.\n", os.Args[1])
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

func cmdStart() {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	ndjson := fs.Bool("ndjson", false, "Emit NDJSON receipt stream to stdout")
	webhook := fs.String("webhook", "", "Webhook URL for CloudEvents POST on each receipt")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fatalf("flag error: %v", err)
	}

	if conn, err := daemon.Dial(); err == nil {
		conn.Close()
		fmt.Fprintln(os.Stderr, "ans: daemon is already running")
		return
	}
	self, err := os.Executable()
	if err != nil {
		fatalf("resolving executable: %v", err)
	}
	daemonArgs := []string{"_daemon"}
	if *ndjson {
		daemonArgs = append(daemonArgs, "--ndjson")
	}
	if *webhook != "" {
		daemonArgs = append(daemonArgs, "--webhook", *webhook)
	}
	cmd := exec.Command(self, daemonArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fatalf("starting daemon: %v", err)
	}
	// Poll until socket is ready (up to 3 seconds)
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if conn, err := daemon.Dial(); err == nil {
			conn.Close()
			fmt.Fprintf(os.Stderr, "ans: daemon started (socket: %s)\n", daemon.SocketPath())
			return
		}
	}
	fatalf("daemon did not become ready within 3 seconds")
}

func cmdStop() {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		fmt.Fprintln(os.Stderr, "ans: daemon is not running (no PID file)")
		os.Exit(1)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fatalf("invalid PID file: %v", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		fatalf("finding process %d: %v", pid, err)
	}
	// Verify the process is actually the ANS daemon, not a recycled PID.
	// On Unix we check /proc/<pid>/exe; on Windows we check the socket.
	if runtime.GOOS != "windows" {
		exe, _ := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		self, _ := os.Executable()
		if exe != "" && exe != self {
			fatalf("PID %d belongs to %s, not the ANS daemon", pid, exe)
		}
	} else {
		if conn, err := daemon.Dial(); err != nil {
			fmt.Fprintln(os.Stderr, "ans: daemon is not running")
			_ = os.Remove(pidFilePath())
			os.Exit(1)
		} else {
			conn.Close()
		}
	}
	if runtime.GOOS == "windows" {
		if err := proc.Kill(); err != nil {
			fatalf("killing process %d: %v", pid, err)
		}
	} else {
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			fatalf("sending SIGTERM to %d: %v", pid, err)
		}
	}
	_ = os.Remove(pidFilePath())
	fmt.Fprintln(os.Stderr, "ans: daemon stopped")
}

func cmdStatus() {
	conn := mustDial()
	defer conn.Close()
	if err := daemon.WriteFrame(conn, daemon.MsgStatus, nil); err != nil {
		fatalf("sending status request: %v", err)
	}
	var resp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("reading status: %v", err)
	}
	pretty.PrintStatus(os.Stdout, resp, noColor())
}

func cmdVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	fullChain := fs.Bool("chain", false, "Verify entire chain integrity")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}

	if *fullChain {
		c, err := chain.Open("")
		if err != nil {
			fatalf("opening chain: %v", err)
		}
		defer c.Close()
		pubkeys := make(map[string]ed25519.PublicKey)
		if ks, err := identity.NewKeystore(""); err == nil {
			ids, _ := ks.List()
			for _, id := range ids {
				if ag, err := ks.Load(id); err == nil {
					pubkeys[ag.ID] = ag.PublicKey
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "ans: warning: keystore unavailable, signature verification skipped: %v\n", err)
		}
		result := c.VerifyChain(pubkeys)
		if result.Valid {
			fmt.Printf("\n\033[32m\033[1m✓ Chain integrity verified\033[0m — %d receipts checked\n\n",
				result.TotalChecked)
		} else {
			fmt.Fprintf(os.Stderr,
				"\n\033[31m\033[1m✗ Chain integrity FAILED\033[0m at index %d: %s\n\n",
				result.FirstBrokenAt, result.Error)
			os.Exit(1)
		}
		return
	}

	receiptID := fs.Arg(0)
	if receiptID == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans verify <receipt_id>  or  ans verify --chain")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgVerify, daemon.VerifyReq{ReceiptID: receiptID}); err != nil {
		fatalf("sending verify: %v", err)
	}
	var resp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("reading verify response: %v", err)
	}
	pretty.PrintVerifyResult(os.Stdout, resp, noColor())
	if valid, _ := resp["valid"].(bool); !valid {
		os.Exit(1)
	}
}

func cmdChain(args []string) {
	fs := flag.NewFlagSet("chain", flag.ExitOnError)
	n := fs.Int("n", 20, "Number of receipts to show")
	agentFilter := fs.String("agent", "", "Filter by agent ID")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}

	c, err := chain.Open("")
	if err != nil {
		fatalf("opening chain: %v", err)
	}
	defer c.Close()

	limit := *n
	if limit < 0 {
		limit = 0
	}
	receipts, err := c.List(chain.QueryOptions{Limit: limit, AgentID: *agentFilter})
	if err != nil {
		fatalf("querying chain: %v", err)
	}
	if len(receipts) == 0 {
		fmt.Fprint(os.Stderr, "\nans: no receipts yet. Add @ans.trace to your agent tools to start recording.\n\n")
		return
	}
	// Reverse: List returns DESC, we want oldest-first for the tree
	for i, j := 0, len(receipts)-1; i < j; i, j = i+1, j-1 {
		receipts[i], receipts[j] = receipts[j], receipts[i]
	}
	pretty.PrintChain(os.Stdout, receipts, noColor())
}

func cmdAgents() {
	ks, err := identity.NewKeystore("")
	if err != nil {
		fatalf("opening keystore: %v", err)
	}
	ids, err := ks.List()
	if err != nil {
		fatalf("listing agents: %v", err)
	}
	if len(ids) == 0 {
		fmt.Println("No agents registered yet.")
		return
	}
	fmt.Printf("\n%-20s %-20s %-10s\n", "AGENT ID", "NAME", "VERSION")
	fmt.Println(strings.Repeat("─", 55))
	for _, id := range ids {
		ag, err := ks.Load(id)
		if err != nil {
			fmt.Printf("%-20s (error: %v)\n", id, err)
			continue
		}
		fmt.Printf("%-20s %-20s %-10s\n", ag.ID, ag.Name, ag.Version)
	}
	fmt.Print("\n")
}

// cmdRegister registers a new agent with the daemon.
func cmdRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	name := fs.String("name", "", "Agent name (required)")
	version := fs.String("version", "", "Agent version (required)")
	owner := fs.String("owner", "", "Owner/creator of the agent")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	if *name == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans register --name <name> --version <ver> [--owner <owner>]")
		os.Exit(1)
	}
	if *version == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans register --name <name> --version <ver> [--owner <owner>]")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgRegister, daemon.RegisterReq{
		Name: *name, Version: *version, Owner: *owner,
	}); err != nil {
		fatalf("sending register: %v", err)
	}
	var resp daemon.RegisterResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("register failed: %v", err)
	}
	fmt.Printf("\033[32m\u2713\033[0m Agent registered\n")
	fmt.Printf("  Agent ID: %s\n", resp.AgentID)
	fmt.Printf("  Name:     %s\n", *name)
	fmt.Printf("  Version:  %s\n", *version)
	if *owner != "" {
		fmt.Printf("  Owner:    %s\n", *owner)
	}
}

func cmdExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	format := fs.String("format", "jsonl", "Export format: jsonl, csv, txt, pdf")
	output := fs.String("output", "", "Output file (default stdout)")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}

	c, err := chain.Open("")
	if err != nil {
		fatalf("opening chain: %v", err)
	}
	defer c.Close()

	w := os.Stdout
	if *output != "" {
		clean := filepath.Clean(*output)
		if clean != *output {
			fatalf("invalid output path: %s", *output)
		}
		abs, err := filepath.Abs(clean)
		if err != nil {
			fatalf("resolving output path: %v", err)
		}
		f, err := os.Create(abs)
		if err != nil {
			fatalf("creating output file: %v", err)
		}
		defer f.Close()
		w = f
	}

	var exportErr error
	switch *format {
	case "jsonl":
		exportErr = c.ExportJSONL(w, chain.QueryOptions{})
	case "csv":
		exportErr = c.ExportCSV(w, chain.QueryOptions{})
	case "txt", "text":
		exportErr = c.ExportTextReport(w)
	case "pdf":
		exportErr = c.ExportPDF(w)
	case "parquet":
		exportErr = c.ExportParquet(w, chain.QueryOptions{})
	default:
		fatalf("unknown format %q — use jsonl, csv, txt, pdf, or parquet", *format)
	}
	if exportErr != nil {
		fatalf("export failed: %v", exportErr)
	}
	if *output != "" {
		fmt.Fprintf(os.Stderr, "ans: exported to %s\n", *output)
	}
}

// cmdPrune removes receipts up to a given chain index and records a Merkle anchor.
func cmdPrune(args []string) {
	fs := flag.NewFlagSet("prune", flag.ExitOnError)
	upTo := fs.Uint64("up-to", 0, "Prune receipts with chain_index <= this value (required)")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	if *upTo == 0 {
		fmt.Fprintln(os.Stderr, "Usage: ans prune --up-to <chain_index>")
		os.Exit(1)
	}
	c, err := chain.Open("")
	if err != nil {
		fatalf("opening chain: %v", err)
	}
	defer c.Close()
	anchor, err := c.Prune(*upTo)
	if err != nil {
		fatalf("pruning chain: %v", err)
	}
	fmt.Printf("Pruned %d receipts (index %d–%d)\n", anchor.Count, anchor.FromIndex, anchor.ToIndex)
	fmt.Printf("Merkle root: %s\n", anchor.MerkleRoot)
	fmt.Printf("Anchor ID:   %d\n", anchor.ID)
}

// cmdRotate generates a new keypair for an agent and prints the new agent ID.
func cmdRotate(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ans rotate <agent-id>")
		os.Exit(1)
	}
	agentID := args[0]
	ks, err := identity.NewKeystore("")
	if err != nil {
		fatalf("opening keystore: %v", err)
	}
	newAgent, rec, err := ks.Rotate(agentID)
	if err != nil {
		fatalf("rotating key: %v", err)
	}
	fmt.Printf("Key rotated successfully\n")
	fmt.Printf("Old agent ID: %s\n", agentID)
	fmt.Printf("New agent ID: %s\n", newAgent.ID)
	fmt.Printf("New public key: %x\n", newAgent.PublicKey)
	fmt.Printf("Rotation record: old_sig=%s... new_sig=%s...\n",
		rec.OldSignature[:16], rec.NewSignature[:16])
	fmt.Println("Update your SDK configuration to use the new agent ID.")
}

// cmdSnapshotTake takes a snapshot of an agent's workspace.
func cmdSnapshotTake(args []string) {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	agentID := fs.String("agent", "", "Agent ID to snapshot")
	snapType := fs.String("type", "filesystem", "Snapshot type: filesystem, memory, database")
	paths := fs.String("paths", "", "Comma-separated paths to snapshot (empty = full workspace)")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	if *agentID == "" {
		*agentID = fs.Arg(0)
	}
	if *agentID == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans snapshot take --agent <id> [--type filesystem] [--paths a,b]")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgSnapshot, daemon.SnapshotReq{
		AgentID: *agentID, SnapType: *snapType, Paths: *paths,
	}); err != nil {
		fatalf("sending snapshot request: %v", err)
	}
	var resp daemon.SnapshotResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("snapshot failed: %v", err)
	}
	fmt.Printf("\x1b[32m\u2713\x1b[0m Snapshot taken: id=%s  index=%d  size=%d  hash=%s\n",
		resp.SnapshotID[:16], resp.ChainIndex, resp.SizeBytes, resp.Hash[:16])
}

// cmdTimeTravel restores agent state to a given chain index or receipt hash.
func cmdTimeTravel(args []string) {
	fs := flag.NewFlagSet("time-travel", flag.ExitOnError)
	snapType := fs.String("type", "filesystem", "Snapshot type: filesystem, memory, database")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	targetStr := fs.Arg(0)
	if targetStr == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans time-travel <chain-index-or-hash> [--type filesystem]")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()

	// Auto-detect: 64-char hex = receipt hash, else numeric index
	var targetIdx uint64
	if len(targetStr) == 64 && isHex(targetStr) {
		daemon.WriteJSON(conn, daemon.MsgVerify, daemon.VerifyReq{ReceiptID: targetStr})
		var verifyResp daemon.VerifyResp
		if _, err := daemon.ReadJSON(conn, &verifyResp); err != nil {
			fatalf("resolving receipt %q: %v", targetStr, err)
		}
		if verifyResp.ChainIndex == 0 {
			fatalf("receipt %q not found at chain index 0", targetStr)
		}
		targetIdx = verifyResp.ChainIndex
		fmt.Printf("Resolved receipt %s to chain index %d\n", targetStr[:16], targetIdx)
	} else {
		var err error
		targetIdx, err = strconv.ParseUint(targetStr, 10, 64)
		if err != nil {
			fatalf("invalid chain index or receipt hash: %v", err)
		}
	}

	if err := daemon.WriteJSON(conn, daemon.MsgRestore, daemon.RestoreReq{
		TargetIndex: targetIdx, SnapType: *snapType,
	}); err != nil {
		fatalf("sending restore request: %v", err)
	}
	var resp daemon.RestoreResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("restore failed: %v", err)
	}
	if resp.Success {
		fmt.Printf("\033[32m\u2713\033[0m State restored to chain index %d\n", targetIdx)
	} else {
		fmt.Fprintf(os.Stderr, "\033[31m\u2717\033[0m Restore failed: %s\n", resp.Message)
		os.Exit(1)
	}
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// cmdSnapshotDiff shows file-level diff vs prior snapshot.
func cmdSnapshotDiff(args []string) {
	fs := flag.NewFlagSet("snapshot diff", flag.ExitOnError)
	agentID := fs.String("agent", "", "Agent ID")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	if *agentID == "" {
		*agentID = fs.Arg(0)
	}
	if *agentID == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans snapshot diff --agent <id>")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgSnapshotDiff, daemon.SnapshotDiffReq{
		AgentID: *agentID, SnapType: string(snapshot.SnapFileSystem),
	})
	var resp daemon.SnapshotDiffResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("snapshot diff: %v", err)
	}
	if resp.Message != "" {
		fmt.Println(resp.Message)
		return
	}
	fmt.Printf("File-level diff:\n")
	if len(resp.Added) > 0 {
		fmt.Printf("  Added:    %d files\n", len(resp.Added))
		for _, f := range resp.Added {
			fmt.Printf("    + %s\n", f)
		}
	}
	if len(resp.Modified) > 0 {
		fmt.Printf("  Modified: %d files\n", len(resp.Modified))
		for _, f := range resp.Modified {
			fmt.Printf("    ~ %s\n", f)
		}
	}
	if len(resp.Deleted) > 0 {
		fmt.Printf("  Deleted:  %d files\n", len(resp.Deleted))
		for _, f := range resp.Deleted {
			fmt.Printf("    - %s\n", f)
		}
	}
	if len(resp.Added)+len(resp.Modified)+len(resp.Deleted) == 0 {
		fmt.Println("  No changes (snapshots are identical)")
	}
}

// cmdSnapshots lists snapshots for an agent.
func cmdSnapshots(args []string) {
	fs := flag.NewFlagSet("snapshots", flag.ExitOnError)
	agentFilter := fs.String("agent", "", "Filter by agent ID")
	snapType := fs.String("type", "filesystem", "Snapshot type")
	n := fs.Int("n", 20, "Number of snapshots to show")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	if *agentFilter == "" {
		// Use first arg as agent ID if --agent not given
		if arg := fs.Arg(0); arg != "" {
			agentFilter = &arg
		}
	}
	if *agentFilter == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans snapshots --agent <id> [--type filesystem] [--n 20]")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgSnapshotList, daemon.SnapshotListReq{
		AgentID: *agentFilter, SnapType: *snapType, Limit: *n,
	}); err != nil {
		fatalf("sending snapshot list request: %v", err)
	}
	var resp map[string]interface{}
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("reading snapshot list: %v", err)
	}
	snaps, _ := resp["snapshots"].([]interface{})
	if len(snaps) == 0 {
		fmt.Println("No snapshots found for agent", *agentFilter)
		return
	}
	fmt.Printf("\n%-20s %-8s %-10s %-10s %-16s\n", "SNAPSHOT ID", "TYPE", "INDEX", "SIZE", "TIMESTAMP")
	fmt.Println(strings.Repeat("\u2500", 70))
	for _, s := range snaps {
		snap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		sid, _ := snap["snapshot_id"].(string)
		st, _ := snap["snap_type"].(string)
		ci, _ := snap["chain_index"].(float64)
		sz, _ := snap["size_bytes"].(float64)
		ts, _ := snap["timestamp_ns"].(float64)
		tsTime := time.Unix(0, int64(ts))
		sizeStr := fmt.Sprintf("%.1f KB", sz/1024)
		if sz < 1024 {
			sizeStr = fmt.Sprintf("%.0f B", sz)
		}
		idShort := sid
		if len(idShort) > 16 {
			idShort = idShort[:16]
		}
		fmt.Printf("%-20s %-8s %-10d %-10s %-16s\n", idShort, st, int64(ci), sizeStr, tsTime.Format("15:04:05"))
	}
	fmt.Print("\nTo restore: ans time-travel <index>\n\n")
}

// cmdCompensate executes registered compensating actions for a chain index.
func cmdCompensate(args []string) {
	fs := flag.NewFlagSet("compensate", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Show what would be executed without running")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	targetStr := fs.Arg(0)
	if targetStr == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans compensate <chain-index> [--dry-run]")
		os.Exit(1)
	}
	targetIdx, err := strconv.ParseUint(targetStr, 10, 64)
	if err != nil {
		fatalf("invalid chain index: %v", err)
	}

	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgCompensate, daemon.CompensateReq{
		TargetIndex: targetIdx, DryRun: *dryRun,
	})
	var resp daemon.CompensateResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("compensation failed: %v", err)
	}
	for _, d := range resp.Details {
		fmt.Println("  ", d)
	}
	if resp.Success {
		fmt.Printf("\033[32m\u2713\033[0m Compensation complete: %d run, %d failed\n", resp.ActionsRun, resp.ActionsFailed)
	} else {
		fmt.Printf("\033[31m\u2717\033[0m Compensation had %d failures: %s\n", resp.ActionsFailed, resp.Message)
	}
}

// cmdPolicyAdd registers a new policy from a JSON file.
func cmdPolicyAdd(args []string) {
	if len(args) < 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans policy add <file.json>")
		os.Exit(1)
	}
	path := filepath.Clean(args[0])
	if strings.Contains(path, "..") || path == "/" || path == "." {
		fatalf("invalid policy file path: %s", args[0])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fatalf("reading policy file: %v", err)
	}
	var pol struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		Enabled     bool   `json:"enabled"`
		Priority    int    `json:"priority"`
		Severity    string `json:"severity,omitempty"`
		Conditions  interface{} `json:"conditions"`
		Action      interface{} `json:"action"`
	}
	if err := json.Unmarshal(data, &pol); err != nil {
		fatalf("parsing policy JSON: %v", err)
	}
	condJSON, _ := json.Marshal(pol.Conditions)
	actJSON, _ := json.Marshal(pol.Action)

	conn := mustDial()
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgPolicyRegister, daemon.PolicyRegisterReq{
		ID: pol.ID, Name: pol.Name, Description: pol.Description,
		Enabled: pol.Enabled, Priority: pol.Priority, Severity: pol.Severity,
		Conditions: string(condJSON), Action: string(actJSON),
	}); err != nil {
		fatalf("sending policy register: %v", err)
	}
	var resp daemon.PolicyRegisterResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy register failed: %v", err)
	}
	if !resp.Success {
		fatalf("policy rejected: %s", resp.Message)
	}
	fmt.Printf("\033[32m\u2713\033[0m Policy %q registered\n", pol.ID)
}

// cmdPolicyList lists all registered policies.
func cmdPolicyList(args []string) {
	fs := flag.NewFlagSet("policy list", flag.ExitOnError)
	enabledOnly := fs.Bool("enabled", false, "Show only enabled policies")
	fs.Parse(args)

	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgPolicyList, daemon.PolicyListReq{EnabledOnly: *enabledOnly})
	var resp daemon.PolicyListResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy list failed: %v", err)
	}
	if len(resp.Policies) == 0 {
		fmt.Println("No policies registered")
		return
	}
	fmt.Printf("\n%-24s %-20s %-8s %-6s %-6s\n", "ID", "NAME", "ENABLED", "PRIORITY", "SEVERITY")
	fmt.Println(strings.Repeat("\u2500", 70))
	for _, p := range resp.Policies {
		en := "\033[31mno\033[0m"
		if p.Enabled {
			en = "\033[32myes\033[0m"
		}
		shortID := p.ID
		if len(shortID) > 22 {
			shortID = shortID[:22] + "…"
		}
		shortName := p.Name
		if len(shortName) > 18 {
			shortName = shortName[:18] + "…"
		}
		fmt.Printf("%-24s %-20s %-8s %-6d %-6s\n", shortID, shortName, en, p.Priority, p.Severity)
	}
}

// cmdPolicyRemove deletes a policy by ID.
func cmdPolicyRemove(args []string) {
	if len(args) < 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans policy remove <id>")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgPolicyDelete, daemon.PolicyDeleteReq{ID: args[0]})
	var resp daemon.PolicyDeleteResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy delete failed: %v", err)
	}
	if resp.Success {
		fmt.Printf("\033[32m\u2713\033[0m Policy %q removed\n", args[0])
	} else {
		fatalf("removing policy: %s", resp.Message)
	}
}

// cmdPolicyEval evaluates an action against policies.
func cmdPolicyEval(args []string) {
	fs := flag.NewFlagSet("policy eval", flag.ExitOnError)
	actionType := fs.String("action-type", "", "Action type (required)")
	payloadSummary := fs.String("payload-summary", "", "Payload summary text")
	fs.Parse(args)
	if *actionType == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans policy eval --action-type <type> [--payload-summary <text>]")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgPolicyEvaluate, daemon.PolicyEvaluateReq{
		AgentID: "_cli", ActionType: *actionType, Phase: "pre",
		PayloadSummary: *payloadSummary,
	})
	var resp daemon.PolicyEvaluateResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy eval failed: %v", err)
	}
	if resp.Denied {
		fmt.Printf("\033[31m\u2717\033[0m DENIED")
		if resp.Nociception != nil {
			fmt.Printf(" — %s", resp.Nociception.Message)
		}
		fmt.Println()
	} else if resp.Allowed {
		fmt.Printf("\033[32m\u2713\033[0m ALLOWED\n")
	}
	if len(resp.PolicyResults) > 0 {
		fmt.Println()
		for _, pr := range resp.PolicyResults {
			icon := "\033[32m\u2713\033[0m"
			if pr.Matched && pr.Effect == "deny" {
				icon = "\033[31m\u2717\033[0m"
			}
			matched := "no"
			if pr.Matched {
				matched = "yes"
			}
			fmt.Printf("  %s %s (%s, matched: %s)\n", icon, pr.PolicyName, pr.Effect, matched)
			if pr.ErrorMessage != "" {
				fmt.Printf("    %s\n", pr.ErrorMessage)
			}
		}
	}
}

// cmdTokenRequest provisions an ephemeral token via the broker.
func cmdTokenRequest(args []string) {
	fs := flag.NewFlagSet("token request", flag.ExitOnError)
	resource := fs.String("resource", "", "Resource ARN or path (required)")
	action := fs.String("action", "read", "Action (read, write, etc.)")
	ttl := fs.Int("ttl", 60, "Token TTL in seconds (max 60)")
	fs.Parse(args)
	if *resource == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans token request --resource <arn> [--action read] [--ttl 60]")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgTokenRequest, daemon.TokenRequestReq{
		AgentID: "_cli", Resource: *resource, Action: *action, TTLSeconds: *ttl, SingleUse: true,
	})
	var resp daemon.TokenRequestResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("token request failed: %v", err)
	}
	if !resp.Success {
		fatalf("token provisioning failed: %s", resp.Message)
	}
	fmt.Printf("\033[32m\u2713\033[0m Token provisioned\n")
	fmt.Printf("  Token ID:  %s\n", resp.TokenID)
	fmt.Printf("  Type:      %s\n", resp.TokenType)
	if resp.AccessKey != "" {
		fmt.Printf("  Access Key: %s\n", resp.AccessKey)
	}
	if resp.SecretKey != "" {
		fmt.Printf("  Secret Key: %s\n", resp.SecretKey)
	}
	if resp.BearerToken != "" {
		fmt.Printf("  Bearer:    %s\n", resp.BearerToken)
	}
	fmt.Printf("  Resource:  %s\n", resp.Resource)
	fmt.Printf("  Expires:   %d ns\n", resp.ExpiresNS)
}

// cmdTokenList lists active tokens.
func cmdTokenList(args []string) {
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgTokenList, daemon.TokenListReq{})
	var resp daemon.TokenListResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("token list failed: %v", err)
	}
	if len(resp.Tokens) == 0 {
		fmt.Println("No active tokens")
		return
	}
	fmt.Printf("\n%-24s %-10s %-12s %-22s %-8s\n", "TOKEN ID", "PROVIDER", "TYPE", "RESOURCE", "STATE")
	fmt.Println(strings.Repeat("\u2500", 80))
	for _, t := range resp.Tokens {
		shortID := t.TokenID
		if len(shortID) > 22 {
			shortID = shortID[:22]
		}
		shortRes := t.Resource
		if len(shortRes) > 20 {
			shortRes = shortRes[:20] + "…"
		}
		fmt.Printf("%-24s %-10s %-12s %-22s %-8s\n", shortID, t.Provider, t.TokenType, shortRes, t.State)
	}
}

// cmdTokenRevoke revokes a token by ID.
func cmdTokenRevoke(args []string) {
	if len(args) < 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans token revoke <token-id>")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgTokenRevoke, daemon.TokenRevokeReq{TokenID: args[0]})
	var resp daemon.TokenRevokeResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("token revoke failed: %v", err)
	}
	if resp.Success {
		fmt.Printf("\033[32m\u2713\033[0m Token %q revoked\n", args[0])
	} else {
		fatalf("revoking token: %s", resp.Message)
	}
}

// cmdMCPStart starts the MCP security proxy.
func cmdMCPStart(args []string) {
	fs := flag.NewFlagSet("mcp start", flag.ExitOnError)
	listen := fs.String("listen", ":8080", "Listen address")
	target := fs.String("target", "", "Target MCP server URL (required)")
	safetyDisable := fs.Bool("safety-disable", false, "Disable all safety features (PII redaction, rate limiting, etc.)")
	rateLimit := fs.Int("rate-limit", 60, "Requests per minute per client IP (0 = unlimited)")
	tokenBudget := fs.Int("token-budget", 50000, "Estimated tokens per minute per agent (0 = unlimited)")
	piiRedact := fs.Bool("pii-redact", true, "Enable PII redaction on server responses")
	fs.Parse(args)
	if *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans mcp start --listen :8080 --target http://mcp-server:8080")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	rl := *rateLimit
	tb := *tokenBudget
	pr := *piiRedact
	daemon.WriteJSON(conn, daemon.MsgMCPStart, daemon.MCPStartReq{
		ListenAddr:    *listen,
		TargetURL:     *target,
		SafetyDisable: *safetyDisable,
		RedactPII:     &pr,
		RateLimit:     &rl,
		TokenBudget:   &tb,
	})
	var resp daemon.MCPStartResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp start failed: %v", err)
	}
	if resp.Success {
		fmt.Printf("\033[32m\u2713\033[0m MCP proxy started on %s -> %s\n", *listen, *target)
	} else {
		fatalf("mcp start: %s", resp.Message)
	}
}

// cmdMCPStop stops the MCP proxy.
func cmdMCPStop() {
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgMCPStop, daemon.MCPStopReq{})
	var resp daemon.MCPStopResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp stop failed: %v", err)
	}
	if resp.Success {
		fmt.Printf("\033[32m\u2713\033[0m MCP proxy stopped\n")
	} else {
		fmt.Printf("\033[33m!\033[0m %s\n", resp.Message)
	}
}

// cmdMCPStatus shows MCP proxy status and stats.
func cmdMCPStatus() {
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgMCPStatus, nil)
	var resp daemon.MCPStatusResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp status failed: %v", err)
	}
	if !resp.Running {
		fmt.Println("MCP proxy: \033[31mnot running\033[0m")
		return
	}
	fmt.Printf("MCP proxy: \033[32mrunning\033[0m\n")
	fmt.Printf("  Uptime:      %ds\n", resp.UptimeSecs)
	fmt.Printf("  Messages:    %d\n", resp.TotalMsgs)
	fmt.Printf("  Total Toks:  %d\n", resp.TotalToks)
	fmt.Printf("  Burn Rate:   %.1f toks/s\n", resp.BurnRate)
	fmt.Printf("  Injections:  %d\n", resp.InjCount)
	fmt.Printf("  Pruned:      %d msgs (%.0f KB)\n", resp.PrunedCount, float64(resp.PrunedBytes)/1024)
}

// cmdMCPLog shows recent MCP audit log entries.
func cmdMCPLog(args []string) {
	fs := flag.NewFlagSet("mcp log", flag.ExitOnError)
	limit := fs.Int("n", 20, "Number of entries")
	method := fs.String("method", "", "Filter by method")
	injOnly := fs.Bool("inj", false, "Show only injections")
	fs.Parse(args)
	conn := mustDial()
	defer conn.Close()
	daemon.WriteJSON(conn, daemon.MsgMCPLog, daemon.MCPLogReq{Limit: *limit, Method: *method, InjOnly: *injOnly})
	var resp daemon.MCPLogResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp log failed: %v", err)
	}
	if len(resp.Entries) == 0 {
		fmt.Println("No MCP log entries")
		return
	}
	fmt.Printf("\n%-6s %-5s %-28s %-7s %-6s %s\n", "ID", "DIR", "METHOD", "TOKS", "INJ", "CONTENT")
	fmt.Println(strings.Repeat("\u2500", 90))
	for _, e := range resp.Entries {
		inj := ""
		if e.Injection {
			inj = "\033[31mINJ\033[0m"
		}
		method := e.Method
		if method == "" {
			method = "(response)"
		}
		if len(method) > 26 {
			method = method[:26]
		}
		content := e.Content
		if len(content) > 35 {
			content = content[:35] + "…"
		}
		fmt.Printf("%-6d %-5s %-28s %-7d %-6s %s\n", e.ID, e.Direction, method, e.ToksEst, inj, content)
	}
}

// --- helpers ---

// mustDial connects to the daemon or exits. Returns a valid net.Conn.
func mustDial() net.Conn {
	conn, err := daemon.Dial()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return conn
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ans: "+format+"\n", args...)
	os.Exit(1)
}

func noColor() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("ANS_NO_COLOR") != ""
}

func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ans", "daemon.pid")
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
