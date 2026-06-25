// ANS CLI — Agent Nervous System
// Commands: start, stop, status, verify, chain, agents, export, version, help
// SPDX-License-Identifier: Apache-2.0
package main

import (
	cryptoRand "crypto/rand"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/config"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/dashboard"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identity"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot"
	"golang.org/x/term"
)

var version = "dev"

func printUsage() {
	w := os.Stderr
	pretty.Banner(w)
	fmt.Fprintln(w)
	pretty.Header(w, "Usage")
	pretty.Code(w, "ans <command> [flags]")
	fmt.Fprintln(w)
	pretty.Header(w, "Commands")
	pretty.Item(w, "init", "Create config and data directory (--service to install system service)")
	pretty.Item(w, "start", "Start the ANS daemon in the background")
	pretty.Item(w, "stop", "Stop the running ANS daemon")
	pretty.Item(w, "status", "Show daemon status, chain stats, and uptime")
	pretty.Item(w, "doctor", "Show diagnostics (socket, PID, chain health)")
	pretty.Item(w, "verify", "Verify a receipt by ID (--chain for full chain)")
	pretty.Item(w, "chain", "Print the receipt chain (pretty tree)")
	pretty.Item(w, "agents", "List registered agents")
	pretty.Item(w, "register", "Register a new agent (--name --version)")
	pretty.Item(w, "export", "Export the chain (--format jsonl|csv|txt|pdf|parquet)")
	pretty.Item(w, "prune", "Prune receipts and create Merkle anchor (--up-to)")
	pretty.Item(w, "rotate", "Rotate an agent's keypair")
	pretty.Item(w, "snapshot", "Take/list/diff snapshots (--agent)")
	pretty.Item(w, "time-travel", "Restore state to a chain index or receipt hash")
	pretty.Item(w, "compensate", "Execute compensating actions for a chain index")
	pretty.Item(w, "policy", "Add/list/remove/eval policies")
	pretty.Item(w, "token", "Request/list/revoke ephemeral tokens")
	pretty.Item(w, "mcp", "Start/stop/status/log MCP security proxy")
	pretty.Item(w, "version", "Print version")
	pretty.Item(w, "update", "Update ANS to the latest version")
	pretty.Item(w, "uninstall", "Remove ANS binary, data, and config")
	pretty.Item(w, "dashboard", "Launch the full-screen terminal dashboard")
	fmt.Fprintln(w)
	pretty.Header(w, "Flags (start)")
	pretty.Item(w, "--ndjson", "Emit NDJSON receipt stream to stdout")
	pretty.Item(w, "--webhook", "Webhook URL for CloudEvents POST on each receipt")
	fmt.Fprintln(w)
	pretty.Header(w, "Flags (chain)")
	pretty.Item(w, "--n", "Receipts to show (default 20)")
	pretty.Item(w, "--agent", "Filter by agent ID")
	fmt.Fprintln(w)
	pretty.Header(w, "Flags (export)")
	pretty.Item(w, "--format", "jsonl | csv | txt | pdf | parquet (default jsonl)")
	pretty.Item(w, "--output", "Output file (default stdout)")
	fmt.Fprintln(w)
	pretty.Header(w, "Flags (verify)")
	pretty.Item(w, "--chain", "Verify entire chain integrity")
	fmt.Fprintln(w)
	pretty.Header(w, "Notes")
	pretty.Item(w, "NO_COLOR", "Set to 1 to disable ANSI color output")
	fmt.Fprintln(w)
}

func main() {
	if len(os.Args) < 2 {
		if !isTerminal() || os.Getenv("ANS_TEST") != "" {
			printUsage()
			os.Exit(0)
		}
		cmdDashboard()
		return
	}

	// Internal re-exec subcommand — not shown in help.
	if os.Args[1] == "_daemon" {
		runDaemon()
		return
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
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
	case "doctor":
		cmdDoctor()
	case "update":
		cmdUpdate()
	case "uninstall":
		cmdUninstall()
	case "dashboard", "dash":
		cmdDashboard()
	case "version", "--version", "-v":
		pretty.Banner(os.Stderr)
		pretty.Item(os.Stderr, "Version", version)
		pretty.Item(os.Stderr, "Platform", runtime.GOOS+"/"+runtime.GOARCH)
		fmt.Fprintln(os.Stderr)
	case "help", "--help", "-h":
		printUsage()
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

	w := os.Stderr

	if conn, err := daemon.Dial(); err == nil {
		_ = conn.Close()
		pretty.Warn(w, "Daemon is already running")
		pretty.Item(w, "Socket", daemon.SocketPath())
		return
	}

	cfg, _ := config.Load()
	if !*ndjson {
		*ndjson = cfg.NDJSON
	}
	if *webhook == "" {
		*webhook = cfg.Webhook
	}

	pretty.Banner(w)
	pretty.Header(w, "Starting ANS Daemon")
	fmt.Fprintln(w)

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
	cmd := exec.Command(self, daemonArgs...) // #nosec G204 — self is binary path, args are flags
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fatalf("starting daemon: %v", err)
	}
	pretty.Done(w, "Daemon process launched")
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if conn, err := daemon.Dial(); err == nil {
			_ = conn.Close()
			pretty.Ok(w, "ANS Daemon is running!")
			pretty.Item(w, "Socket", daemon.SocketPath())
			fmt.Fprintln(w)
			pretty.Step(w, 1, "Register an agent:")
			pretty.Code(w, "ans register")
			fmt.Fprintln(w)
			pretty.Step(w, 2, "View the chain:")
			pretty.Code(w, "ans chain")
			fmt.Fprintln(w)
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
			_ = conn.Close()
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
				ag, loadErr := ks.Load(id)
				if loadErr == nil {
					pubkeys[ag.ID] = ag.PublicKey
				}
			}
		} else {
			fmt.Fprintf(os.Stderr, "ans: warning: keystore unavailable, signature verification skipped: %v\n", err)
		}
		result := c.VerifyChain(pubkeys)
		if result.Valid {
			fmt.Printf("\n"+pretty.Green+pretty.Bold+"✓ Chain integrity verified"+pretty.Reset+" — %d receipts checked\n\n",
				result.TotalChecked)
		} else {
			fmt.Fprintf(os.Stderr,
				"\n"+pretty.Red+pretty.Bold+"✗ Chain integrity FAILED"+pretty.Reset+" at index %d: %s\n\n",
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
	w := os.Stderr
	if len(ids) == 0 {
		pretty.Warn(w, "No agents registered yet")
		return
	}
	pretty.Header(w, "Registered Agents")
	for _, id := range ids {
		ag, err := ks.Load(id)
		if err != nil {
			pretty.Item(w, id, pretty.Red+"error: "+err.Error()+pretty.Reset)
			continue
		}
		pretty.Item(w, ag.ID, ag.Name+"  "+pretty.Dim+ag.Version+pretty.Reset)
	}
	fmt.Fprintln(w)
}

// cmdRegister registers a new agent with the daemon.
func cmdRegister(args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	name := fs.String("name", "", "Agent name (generated if empty)")
	version := fs.String("version", "1.0.0", "Agent version")
	owner := fs.String("owner", "", "Owner/creator of the agent")
	if err := fs.Parse(args); err != nil {
		fatalf("flag error: %v", err)
	}
	b := make([]byte, 4)
	cryptoRand.Read(b)
	rnd := hex.EncodeToString(b)
	if *name == "" {
		*name = "agent-" + rnd
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
	w := os.Stderr
	pretty.Done(w, "Agent registered")
	pretty.Item(w, "Agent ID", resp.AgentID)
	pretty.Item(w, "Name", *name)
	pretty.Item(w, "Version", *version)
	if *owner != "" {
		pretty.Item(w, "Owner", *owner)
	}
	fmt.Fprintln(w)
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
	if !filepath.IsAbs(clean) {
		abs, err := filepath.Abs(clean)
		if err != nil {
			fatalf("resolving output path: %v", err)
		}
		clean = abs
	}
	abs, err := filepath.Abs(clean)
		if err != nil {
			fatalf("resolving output path: %v", err)
		}
		f, err := os.Create(abs) // #nosec G304
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
		pretty.Done(os.Stderr, "Exported to "+*output)
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
	w := os.Stderr
	pretty.Done(w, fmt.Sprintf("Pruned %d receipts (index %d-%d)", anchor.Count, anchor.FromIndex, anchor.ToIndex))
	pretty.Item(w, "Merkle root", anchor.MerkleRoot)
	pretty.Item(w, "Anchor ID", fmt.Sprintf("%d", anchor.ID))
	fmt.Fprintln(w)
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
	w := os.Stderr
	pretty.Done(w, "Key rotated successfully")
	pretty.Item(w, "Old agent ID", agentID)
	pretty.Item(w, "New agent ID", newAgent.ID)
	pretty.Item(w, "New public key", fmt.Sprintf("%x", newAgent.PublicKey))
	pretty.Item(w, "Rotation record", fmt.Sprintf("old_sig=%s... new_sig=%s...", safeSig(rec.OldSignature), safeSig(rec.NewSignature)))
	pretty.Warn(w, "Update your SDK configuration to use the new agent ID")
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
	w := os.Stderr
	pretty.Done(w, "Snapshot taken")
	pretty.Item(w, "ID", resp.SnapshotID[:16])
	pretty.Item(w, "Index", fmt.Sprintf("%d", resp.ChainIndex))
	pretty.Item(w, "Size", fmt.Sprintf("%d bytes", resp.SizeBytes))
	pretty.Item(w, "Hash", fmt.Sprintf("%x…", resp.Hash[:16]))
	fmt.Fprintln(w)
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
		_ = daemon.WriteJSON(conn, daemon.MsgVerify, daemon.VerifyReq{ReceiptID: targetStr})
		var verifyResp daemon.VerifyResp
		if _, err := daemon.ReadJSON(conn, &verifyResp); err != nil {
			fatalf("resolving receipt %q: %v", targetStr, err)
		}
		if verifyResp.ChainIndex == 0 && !verifyResp.Valid {
			fatalf("receipt %q not found", targetStr)
		}
		targetIdx = verifyResp.ChainIndex
		pretty.Item(os.Stderr, "Resolved receipt", fmt.Sprintf("%s -> chain index %d", targetStr[:16], targetIdx))
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
		pretty.Ok(os.Stderr, fmt.Sprintf("State restored to chain index %d", targetIdx))
	} else {
		pretty.Err(os.Stderr, "Restore failed: "+resp.Message)
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
	_ = daemon.WriteJSON(conn, daemon.MsgSnapshotDiff, daemon.SnapshotDiffReq{
		AgentID: *agentID, SnapType: string(snapshot.SnapFileSystem),
	})
	var resp daemon.SnapshotDiffResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("snapshot diff: %v", err)
	}
	w := os.Stderr
	if resp.Message != "" {
		fmt.Fprintln(w, resp.Message)
		return
	}
	pretty.Subheader(w, "File-level diff")
	if len(resp.Added) > 0 {
		fmt.Fprintf(w, "  %sAdded:%s %d files\n", pretty.Green, pretty.Reset, len(resp.Added))
		for _, f := range resp.Added {
			fmt.Fprintf(w, "    %s+%s %s\n", pretty.Green, pretty.Reset, f)
		}
	}
	if len(resp.Modified) > 0 {
		fmt.Fprintf(w, "  %sModified:%s %d files\n", pretty.Yellow, pretty.Reset, len(resp.Modified))
		for _, f := range resp.Modified {
			fmt.Fprintf(w, "    %s~%s %s\n", pretty.Yellow, pretty.Reset, f)
		}
	}
	if len(resp.Deleted) > 0 {
		fmt.Fprintf(w, "  %sDeleted:%s %d files\n", pretty.Red, pretty.Reset, len(resp.Deleted))
		for _, f := range resp.Deleted {
			fmt.Fprintf(w, "    %s-%s %s\n", pretty.Red, pretty.Reset, f)
		}
	}
	if len(resp.Added)+len(resp.Modified)+len(resp.Deleted) == 0 {
		pretty.Item(w, "Result", "No changes (snapshots are identical)")
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
	w := os.Stderr
	if len(snaps) == 0 {
		pretty.Warn(w, "No snapshots found for agent "+*agentFilter)
		return
	}
	pretty.Header(w, fmt.Sprintf("Snapshots for %s", *agentFilter))
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
		pretty.Item(w, idShort, fmt.Sprintf("%s  index=%.0f  %s  %s", st, ci, sizeStr, tsTime.Format("15:04:05")))
	}
	pretty.Code(w, "ans time-travel <index> to restore")
	fmt.Fprintln(w)
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
	_ = daemon.WriteJSON(conn, daemon.MsgCompensate, daemon.CompensateReq{
		TargetIndex: targetIdx, DryRun: *dryRun,
	})
	var resp daemon.CompensateResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("compensation failed: %v", err)
	}
	w := os.Stderr
	for _, d := range resp.Details {
		pretty.Item(w, "  ", d)
	}
	if resp.Success {
		pretty.Ok(w, fmt.Sprintf("Compensation complete: %d run, %d failed", resp.ActionsRun, resp.ActionsFailed))
	} else {
		pretty.Err(w, fmt.Sprintf("Compensation had %d failures: %s", resp.ActionsFailed, resp.Message))
	}
}

// cmdPolicyAdd registers a new policy from a JSON file.
func cmdPolicyAdd(args []string) {
	if len(args) < 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans policy add <file.json>")
		os.Exit(1)
	}
	path := filepath.Clean(args[0])
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			fatalf("resolving policy path: %v", err)
		}
		path = abs
	}
	if path == "/" {
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
	pretty.Done(os.Stderr, fmt.Sprintf("Policy %q registered", pol.ID))
}

// cmdPolicyList lists all registered policies.
func cmdPolicyList(args []string) {
	fs := flag.NewFlagSet("policy list", flag.ExitOnError)
	enabledOnly := fs.Bool("enabled", false, "Show only enabled policies")
	_ = fs.Parse(args)

	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgPolicyList, daemon.PolicyListReq{EnabledOnly: *enabledOnly})
	var resp daemon.PolicyListResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy list failed: %v", err)
	}
	w := os.Stderr
	if len(resp.Policies) == 0 {
		pretty.Warn(w, "No policies registered")
		return
	}
	pretty.Header(w, "Policies")
	for _, p := range resp.Policies {
		en := pretty.Red + "no" + pretty.Reset
		if p.Enabled {
			en = pretty.Green + "yes" + pretty.Reset
		}
		pretty.Item(w, p.ID, fmt.Sprintf("%s  priority=%d  severity=%s  enabled=%s", p.Name, p.Priority, p.Severity, en))
	}
	fmt.Fprintln(w)
}

// cmdPolicyRemove deletes a policy by ID.
func cmdPolicyRemove(args []string) {
	if len(args) < 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans policy remove <id>")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgPolicyDelete, daemon.PolicyDeleteReq{ID: args[0]})
	var resp daemon.PolicyDeleteResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy delete failed: %v", err)
	}
	if resp.Success {
		pretty.Done(os.Stderr, fmt.Sprintf("Policy %q removed", args[0]))
	} else {
		fatalf("removing policy: %s", resp.Message)
	}
}

// cmdPolicyEval evaluates an action against policies.
func cmdPolicyEval(args []string) {
	fs := flag.NewFlagSet("policy eval", flag.ExitOnError)
	actionType := fs.String("action-type", "", "Action type (required)")
	payloadSummary := fs.String("payload-summary", "", "Payload summary text")
	_ = fs.Parse(args)
	if *actionType == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans policy eval --action-type <type> [--payload-summary <text>]")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgPolicyEvaluate, daemon.PolicyEvaluateReq{
		AgentID: "_cli", ActionType: *actionType, Phase: "pre",
		PayloadSummary: *payloadSummary,
	})
	var resp daemon.PolicyEvaluateResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("policy eval failed: %v", err)
	}
	w := os.Stderr
	if resp.Denied {
		pretty.Err(w, "DENIED")
		if resp.Nociception != nil {
			pretty.Item(w, "Reason", resp.Nociception.Message)
		}
	} else if resp.Allowed {
		pretty.Ok(w, "ALLOWED")
	}
	if len(resp.PolicyResults) > 0 {
		fmt.Fprintln(w)
		for _, pr := range resp.PolicyResults {
			icon := pretty.Green + pretty.Bold + "\u2713" + pretty.Reset
			if pr.Matched && pr.Effect == "deny" {
				icon = pretty.Red + pretty.Bold + "\u2717" + pretty.Reset
			}
			matched := "no"
			if pr.Matched {
				matched = "yes"
			}
			fmt.Fprintf(w, "  %s %s (%s, matched: %s)\n", icon, pr.PolicyName, pr.Effect, matched)
			if pr.ErrorMessage != "" {
				pretty.Item(w, "  Error", pr.ErrorMessage)
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
	_ = fs.Parse(args)
	if *resource == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans token request --resource <arn> [--action read] [--ttl 60]")
		os.Exit(1)
	}

	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgTokenRequest, daemon.TokenRequestReq{
		AgentID: "_cli", Resource: *resource, Action: *action, TTLSeconds: *ttl, SingleUse: true,
	})
	var resp daemon.TokenRequestResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("token request failed: %v", err)
	}
	if !resp.Success {
		fatalf("token provisioning failed: %s", resp.Message)
	}
	w := os.Stderr
	pretty.Done(w, "Token provisioned")
	pretty.Item(w, "Token ID", resp.TokenID)
	pretty.Item(w, "Type", resp.TokenType)
	pretty.Item(w, "Access Key", maskSecret(resp.AccessKey))
	pretty.Item(w, "Secret Key", maskSecret(resp.SecretKey))
	pretty.Item(w, "Bearer", maskSecret(resp.BearerToken))
	pretty.Item(w, "Resource", resp.Resource)
	pretty.Item(w, "Expires", fmt.Sprintf("%d ns", resp.ExpiresNS))
	fmt.Fprintln(w)
}

// cmdTokenList lists active tokens.
func cmdTokenList(args []string) {
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgTokenList, daemon.TokenListReq{})
	var resp daemon.TokenListResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("token list failed: %v", err)
	}
	w := os.Stderr
	if len(resp.Tokens) == 0 {
		pretty.Warn(w, "No active tokens")
		return
	}
	pretty.Header(w, "Active Tokens")
	for _, t := range resp.Tokens {
		stateClr := pretty.Green
		if t.State == "revoked" || t.State == "expired" {
			stateClr = pretty.Red
		}
		pretty.Item(w, t.TokenID, fmt.Sprintf("%s  type=%s  resource=%s  state=%s%s%s", t.Provider, t.TokenType, t.Resource, stateClr, t.State, pretty.Reset))
	}
	fmt.Fprintln(w)
}

// cmdTokenRevoke revokes a token by ID.
func cmdTokenRevoke(args []string) {
	if len(args) < 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans token revoke <token-id>")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgTokenRevoke, daemon.TokenRevokeReq{TokenID: args[0]})
	var resp daemon.TokenRevokeResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("token revoke failed: %v", err)
	}
	if resp.Success {
		pretty.Done(os.Stderr, fmt.Sprintf("Token %q revoked", args[0]))
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
	_ = fs.Parse(args)
	if *target == "" {
		fmt.Fprintln(os.Stderr, "Usage: ans mcp start --listen :8080 --target http://mcp-server:8080")
		os.Exit(1)
	}
	conn := mustDial()
	defer conn.Close()
	rl := *rateLimit
	tb := *tokenBudget
	pr := *piiRedact
	_ = daemon.WriteJSON(conn, daemon.MsgMCPStart, daemon.MCPStartReq{
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
		pretty.Ok(os.Stderr, fmt.Sprintf("MCP proxy started on %s -> %s", *listen, *target))
	} else {
		fatalf("mcp start: %s", resp.Message)
	}
}

// cmdMCPStop stops the MCP proxy.
func cmdMCPStop() {
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgMCPStop, daemon.MCPStopReq{})
	var resp daemon.MCPStopResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp stop failed: %v", err)
	}
	if resp.Success {
		pretty.Done(os.Stderr, "MCP proxy stopped")
	} else {
		pretty.Warn(os.Stderr, resp.Message)
	}
}

// cmdMCPStatus shows MCP proxy status and stats.
func cmdMCPStatus() {
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgMCPStatus, nil)
	var resp daemon.MCPStatusResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp status failed: %v", err)
	}
	w := os.Stderr
	if !resp.Running {
		pretty.Err(w, "MCP proxy not running")
		return
	}
	pretty.Header(w, "MCP Proxy Status")
	pretty.Item(w, "Status", pretty.Green+pretty.Bold+"running"+pretty.Reset)
	pretty.Item(w, "Uptime", fmt.Sprintf("%ds", resp.UptimeSecs))
	pretty.Item(w, "Messages", fmt.Sprintf("%d", resp.TotalMsgs))
	pretty.Item(w, "Total Toks", fmt.Sprintf("%d", resp.TotalToks))
	pretty.Item(w, "Burn Rate", fmt.Sprintf("%.1f toks/s", resp.BurnRate))
	pretty.Item(w, "Injections", fmt.Sprintf("%d", resp.InjCount))
	pretty.Item(w, "Pruned", fmt.Sprintf("%d msgs (%.0f KB)", resp.PrunedCount, float64(resp.PrunedBytes)/1024))
	fmt.Fprintln(w)
}

// cmdMCPLog shows recent MCP audit log entries.
func cmdMCPLog(args []string) {
	fs := flag.NewFlagSet("mcp log", flag.ExitOnError)
	limit := fs.Int("n", 20, "Number of entries")
	method := fs.String("method", "", "Filter by method")
	injOnly := fs.Bool("inj", false, "Show only injections")
	_ = fs.Parse(args)
	conn := mustDial()
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgMCPLog, daemon.MCPLogReq{Limit: *limit, Method: *method, InjOnly: *injOnly})
	var resp daemon.MCPLogResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		fatalf("mcp log failed: %v", err)
	}
	w := os.Stderr
	if len(resp.Entries) == 0 {
		pretty.Warn(w, "No MCP log entries")
		return
	}
	pretty.Header(w, "MCP Audit Log")
	for _, e := range resp.Entries {
		method := e.Method
		if method == "" {
			method = "(response)"
		}
		content := e.Content
		if len(content) > 40 {
			content = content[:40] + "…"
		}
		inj := ""
		if e.Injection {
			inj = " " + pretty.Red + "INJ" + pretty.Reset
		}
		pretty.Item(w, fmt.Sprintf("#%d", e.ID), fmt.Sprintf("%s  %s  %d toks  %s%s", e.Direction, method, e.ToksEst, content, inj))
	}
	fmt.Fprintln(w)
}

// --- init ---

func cmdInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	svc := fs.Bool("service", false, "Install system service (systemd/launchd)")
	webhook := fs.String("webhook", "", "Default webhook URL")
	ndjson := fs.Bool("ndjson", false, "Default NDJSON output")
	_ = fs.Parse(os.Args[2:])

	w := os.Stderr
	pretty.Banner(w)
	pretty.Header(w, "Initializing ANS")
	fmt.Fprintln(w)

	dir, err := config.EnsureDir()
	if err != nil {
		fatalf("creating data directory: %v", err)
	}
	pretty.Done(w, "Data directory ready: "+dir)

	cfg, err := config.Load()
	if err != nil {
		pretty.Warn(w, "Loading config: "+err.Error())
		cfg = config.DefaultConfig()
	}
	if *webhook != "" {
		cfg.Webhook = *webhook
	}
	if *ndjson {
		cfg.NDJSON = true
	}
	if err := config.Save(cfg); err != nil {
		fatalf("saving config: %v", err)
	}
	pretty.Done(w, "Configuration written")

	if *svc {
		installService()
	}

	pretty.Ok(w, "ANS is ready!")
	pretty.Step(w, 1, "Start the daemon:")
	pretty.Code(w, "ans start")
	fmt.Fprintln(w)
	pretty.Step(w, 2, "Register an agent:")
	pretty.Code(w, "ans register")
	fmt.Fprintln(w)
	pretty.Step(w, 3, "View the receipt chain:")
	pretty.Code(w, "ans chain")
	fmt.Fprintln(w)
}

func installService() {
	switch runtime.GOOS {
	case "linux":
		installSystemd()
	case "darwin":
		installLaunchd()
	case "windows":
		installWinService()
	default:
		fmt.Fprintf(os.Stderr, "ans: unsupported OS for service: %s\n", runtime.GOOS)
		os.Exit(1)
	}
}

func installSystemd() {
	self, err := os.Executable()
	if err != nil {
		fatalf("resolving executable: %v", err)
	}
	unit := fmt.Sprintf(`[Unit]
Description=Agent Nervous System Daemon
After=network.target

[Service]
Type=simple
ExecStart=%s start
Restart=on-failure
RestartSec=5
Environment=ANS_SERVICE=1

[Install]
WantedBy=multi-user.target
`, self)
	paths := []string{
		"/etc/systemd/system/ans.service",
		filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user", "ans.service"),
	}
	installed := false
	for _, p := range paths {
		dir := filepath.Dir(p)
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}
		if err := os.WriteFile(p, []byte(unit), 0644); err != nil {
			continue
		}
		fmt.Fprintf(os.Stderr, "ans: systemd unit written: %s\n", p)
		installed = true
		user := strings.Contains(p, "HOME")
		if user {
			_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
			_ = exec.Command("systemctl", "--user", "enable", "ans").Run()
		} else {
			_ = exec.Command("systemctl", "daemon-reload").Run()
			_ = exec.Command("systemctl", "enable", "ans").Run()
		}
		fmt.Fprintf(os.Stderr, "ans: systemd service enabled. Start with: systemctl %s start ans\n",
			map[bool]string{true: "--user", false: ""}[user])
	}
	if !installed {
		fmt.Fprintf(os.Stderr, "ans: warning: could not write systemd unit. Try running as root.\n")
		fmt.Fprintf(os.Stderr, "ans: unit content:\n%s\n", unit)
	}
}

func installLaunchd() {
	self, err := os.Executable()
	if err != nil {
		fatalf("resolving executable: %v", err)
	}
	home, _ := os.UserHomeDir()
	label := "com.ans.daemon"
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>start</string>
    </array>
    <key>KeepAlive</key>
    <true/>
    <key>RunAtLoad</key>
    <true/>
    <key>EnvironmentVariables</key>
    <dict>
        <key>ANS_SERVICE</key>
        <string>1</string>
    </dict>
    <key>StandardOutPath</key>
    <string>/tmp/ans-daemon.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/ans-daemon.log</string>
</dict>
</plist>
`, label, self)
	plistPath := filepath.Join(home, "Library", "LaunchAgents", label+".plist")
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		fatalf("creating LaunchAgents dir: %v", err)
	}
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		fatalf("writing launchd plist: %v", err)
	}
	fmt.Fprintf(os.Stderr, "ans: launchd plist written: %s\n", plistPath)
	_ = exec.Command("launchctl", "load", plistPath).Run()
	fmt.Fprintf(os.Stderr, "ans: launchd service loaded. Manage with: launchctl %s\n", label)
}

func installWinService() {
	self, err := os.Executable()
	if err != nil {
		fatalf("resolving executable: %v", err)
	}
	script := fmt.Sprintf(`@echo off
:: ANS Daemon startup — generated by ans init --service
start /B "" "%s" start
`, self)
	startupDir := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	scriptPath := filepath.Join(startupDir, "ans-daemon.bat")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		fatalf("creating Startup dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		fatalf("writing startup script: %v", err)
	}
	fmt.Fprintf(os.Stderr, "ans: startup script written: %s\n", scriptPath)
	fmt.Fprintln(os.Stderr, "ans: ANS will start automatically on next login.")
}

// --- doctor ---

func cmdDoctor() {
	w := os.Stderr
	pretty.Banner(w)
	pretty.Header(w, "ANS Diagnostics")
	fmt.Fprintln(w)

	status := func(ok bool, label string) string {
		if ok {
			return pretty.Green + pretty.Bold + "OK" + pretty.Reset
		}
		return pretty.Yellow + pretty.Bold + "MISSING" + pretty.Reset
	}

	socketPath := daemon.SocketPath()
	daemonOK := false
	if conn, err := daemon.Dial(); err == nil {
		_ = conn.Close()
		daemonOK = true
	}
	pretty.Item(w, "Daemon", status(daemonOK, ""))
	pretty.Item(w, "  Socket", socketPath)

	pidPath := pidFilePath()
	pidOK := false
	pid := ""
	if data, err := os.ReadFile(pidPath); err == nil {
		pid = strings.TrimSpace(string(data))
		pidOK = true
	}
	pretty.Item(w, "  PID file", pidPath)
	if pidOK {
		pretty.Item(w, "  PID", pid)
	}

	cfgPath, _ := config.Path()
	cfgOK := false
	if _, err := os.Stat(cfgPath); err == nil {
		cfgOK = true
	}
	pretty.Item(w, "Config", status(cfgOK, ""))
	if !cfgOK {
		pretty.Item(w, "  Path", cfgPath)
	}

	dataDir, _ := config.Dir()
	dirOK := false
	items := 0
	if entries, err := os.ReadDir(dataDir); err == nil {
		dirOK = true
		items = len(entries)
	}
	pretty.Item(w, "Data dir", status(dirOK, ""))
	pretty.Item(w, "  Path", dataDir)
	if dirOK {
		pretty.Item(w, "  Items", fmt.Sprintf("%d", items))
	}

	chainPath := filepath.Join(dataDir, "chain.db")
	chainOK := false
	if _, err := os.Stat(chainPath); err == nil {
		chainOK = true
	}
	pretty.Item(w, "Chain DB", status(chainOK, ""))
	if !chainOK {
		pretty.Item(w, "  Note", "Created on first start")
	}

	pretty.Item(w, "Version", version+" ("+runtime.GOOS+"/"+runtime.GOARCH+")")

	fmt.Fprintln(w)
	if !cfgOK || !dirOK {
		pretty.Warn(w, "Not fully set up yet")
		pretty.Step(w, 1, "Run first-time setup:")
		pretty.Code(w, "ans init")
		fmt.Fprintln(w)
	}
	if !daemonOK {
		pretty.Step(w, 2, "Start the daemon:")
		pretty.Code(w, "ans start")
		fmt.Fprintln(w)
	}
	if daemonOK {
		pretty.Ok(w, "Everything looks good!")
	}
}

// --- update ---

func cmdUpdate() {
	w := os.Stderr
	pretty.Banner(w)
	pretty.Header(w, "Updating ANS")
	fmt.Fprintln(w)

	repo := "Linky-Link-Linky/Agent-Nervous-System"
	arch := runtime.GOARCH
	if arch == "x86_64" {
		arch = "amd64"
	}
	asset := fmt.Sprintf("ans_%s_%s", runtime.GOOS, arch)
	if runtime.GOOS == "windows" {
		asset += ".exe"
	}

	self, err := os.Executable()
	if err != nil {
		pretty.Err(w, "Cannot find current binary: "+err.Error())
		return
	}

	base := "https://github.com/" + repo + "/releases/latest/download"
	url := base + "/" + asset
	chkURL := base + "/checksums.txt"

	pretty.Step(w, 1, "Downloading "+asset)
	resp, err := http.Get(url)
	if err != nil {
		pretty.Err(w, "Download failed: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		pretty.Err(w, fmt.Sprintf("Download failed: HTTP %d", resp.StatusCode))
		return
	}
	tmp, err := os.CreateTemp("", "ans-*"+filepath.Ext(asset))
	if err != nil {
		pretty.Err(w, "Creating temp file: "+err.Error())
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	hasher := sha256.New()
	multi := io.MultiWriter(tmp, hasher)
	if _, err := io.Copy(multi, resp.Body); err != nil {
		tmp.Close()
		pretty.Err(w, "Download incomplete: "+err.Error())
		return
	}
	tmp.Close()
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	pretty.Done(w, "Downloaded")

	pretty.Step(w, 2, "Verifying checksum")
	chkResp, chkErr := http.Get(chkURL)
	if chkErr == nil && chkResp.StatusCode == 200 {
		defer chkResp.Body.Close()
		chkBody, _ := io.ReadAll(chkResp.Body)
		for _, line := range strings.Split(string(chkBody), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && (parts[1] == asset || strings.HasSuffix(parts[1], "/"+asset)) {
				if parts[0] != actualHash {
					pretty.Err(w, "Checksum mismatch")
					return
				}
				break
			}
		}
		pretty.Done(w, "Checksum verified")
	} else {
		pretty.Warn(w, "Checksum file not available — skipped")
	}

	pretty.Step(w, 3, "Installing update")
	if err := os.Rename(tmpName, self); err == nil {
		pretty.Done(w, "Updated to the latest version")
		fmt.Fprintln(w)
		pretty.Step(w, 4, "Verify installation:")
		pretty.Code(w, "ans version")
		return
	}

	// Rename failed — likely the binary is running (Windows) or we lack permissions.
	// Try placing a .new file alongside the binary.
	pretty.Warn(w, "Could not replace running binary directly")
	updated := self + ".new"
	copyOK := false
	if src, err := os.Open(tmpName); err == nil {
		defer src.Close()
		if dst, err := os.Create(updated); err == nil {
			defer dst.Close()
			if _, err := io.Copy(dst, src); err == nil {
				copyOK = true
			}
		}
	}

	if copyOK {
		pretty.Done(w, "Staged to "+filepath.Base(updated))
		fmt.Fprintln(w)
		pretty.Step(w, 4, "Complete the update in a new terminal:")
		if runtime.GOOS == "windows" {
			pretty.Code(w, fmt.Sprintf(`powershell -Command "Move-Item '%s' '%s' -Force"`, updated, self))
		} else {
			pretty.Code(w, fmt.Sprintf(`cp "%s" "%s" && rm "%s"`, updated, self, updated))
		}
		return
	}

	// Could not write alongside the binary (e.g. protected directory).
	// Fall back to a writable temp location.
	tmpUpdated := filepath.Join(os.TempDir(), filepath.Base(self)+".new")
	copyOK = false
	if src, err := os.Open(tmpName); err == nil {
		defer src.Close()
		if dst, err := os.Create(tmpUpdated); err == nil {
			defer dst.Close()
			if _, err := io.Copy(dst, src); err == nil {
				copyOK = true
			}
		}
	}

	if copyOK {
		pretty.Done(w, "Staged to "+tmpUpdated)
		fmt.Fprintln(w)
		pretty.Step(w, 4, "Complete the update manually:")
		if runtime.GOOS == "windows" {
			pretty.Code(w, fmt.Sprintf(`copy "%s" "%s"`, tmpUpdated, self))
		} else {
			pretty.Code(w, fmt.Sprintf(`cp "%s" "%s"`, tmpUpdated, self))
		}
		pretty.Item(w, "Tip", "Close all ANS processes first, then run the command above")
		return
	}

	// Absolute last resort: temp file
	pretty.Err(w, "Could not write update file — permission denied")
	fmt.Fprintln(w)
	pretty.Step(w, 4, "Install manually from temp:")
	pretty.Code(w, fmt.Sprintf(`copy "%s" "%s"`, tmpName, self))
	pretty.Item(w, "Hint", "Run from an elevated (Admin) terminal if permissions are restricted")
}

func cmdUninstall() {
	w := os.Stderr
	pretty.Banner(w)
	pretty.Header(w, "Uninstalling ANS")
	fmt.Fprintln(w)

	// 1. Stop the daemon if running.
	pretty.Step(w, 1, "Stopping daemon")
	if pidData, err := os.ReadFile(pidFilePath()); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				if runtime.GOOS == "windows" {
					proc.Kill()
				} else {
					proc.Signal(syscall.SIGTERM)
				}
				os.Remove(pidFilePath())
			}
		}
	}
	pretty.Done(w, "Daemon stopped")

	// 2. Remove data directory.
	pretty.Step(w, 2, "Removing data directory")
	dataDir, dataErr := config.Dir()
	removed := false
	if dataErr == nil {
		if err := os.RemoveAll(dataDir); err == nil {
			pretty.Done(w, "Deleted " + dataDir)
			removed = true
		} else {
			pretty.Warn(w, "Could not delete " + dataDir + ": " + err.Error())
		}
	}
	if !removed {
		// Fallback: remove known paths individually.
		home, _ := os.UserHomeDir()
		known := []string{
			filepath.Join(home, ".ans"),
		}
		for _, p := range known {
			if err := os.RemoveAll(p); err == nil {
				pretty.Done(w, "Deleted " + p)
				break
			}
		}
	}

	// 3. Clean PATH on Windows.
	if runtime.GOOS == "windows" {
		pretty.Step(w, 3, "Cleaning PATH")
		binDir := filepath.Join(dataDir, "bin")
		userPath := os.Getenv("Path")
		parts := strings.Split(userPath, ";")
		filtered := make([]string, 0, len(parts))
		for _, p := range parts {
			if strings.EqualFold(p, binDir) || strings.EqualFold(p, binDir+`\`) {
				continue
			}
			filtered = append(filtered, p)
		}
		newPath := strings.Join(filtered, ";")
		if newPath != userPath {
			if err := os.Setenv("Path", newPath); err == nil {
				pretty.Done(w, "Removed " + binDir + " from PATH")
			}
			// Also persist for future terminals.
			_ = exec.Command("powershell", "-NoProfile",
				"-Command",
				fmt.Sprintf(`[Environment]::SetEnvironmentVariable("Path", "%s", "User")`,
					strings.ReplaceAll(newPath, `"`, `""`)),
			).Run()
		} else {
			pretty.Done(w, "PATH already clean")
		}
	}

	fmt.Fprintln(w)
	pretty.Ok(w, "ANS has been uninstalled")
	pretty.Item(w, "Note", "Close and reopen your terminal to refresh PATH")
	fmt.Fprintln(w)
	pretty.Step(w, 4, "Reinstall anytime with")
	pretty.Code(w, `irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex`)
}

func cmdDashboard() {
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

	app := dashboard.NewApp()
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ans: dashboard error: %v\n", err)
		os.Exit(1)
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

// maskSecret masks all but the last 4 characters of a secret, or shows "[REDACTED]" if empty.
func maskSecret(s string) string {
	if s == "" {
		return "[REDACTED]"
	}
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ans", "daemon.pid")
}

// safeSig returns the first 16 hex chars of a signature, or "[short]" if shorter.
func safeSig(sig string) string {
	if len(sig) < 16 {
		return "[short]"
	}
	return sig[:16]
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
