package commands

import (
	"fmt"
	"io"
	"os"
	path "path/filepath"
	"runtime"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
)

var version = "0.8.0"

// DispatchTo executes a command with args (format: os.Args[1:]) and writes
// output to w. Returns an error if the command fails.
func DispatchTo(w io.Writer, args []string) error {
	if len(args) == 0 {
		PrintUsageTo(w)
		return nil
	}

	switch args[0] {
	case "init":
		return cmdInit(w, args[1:])
	case "start":
		return cmdStart(w, args[1:])
	case "stop":
		return cmdStop(w, args[1:])
	case "status":
		return cmdStatus(w, args[1:])
	case "verify":
		return cmdVerify(w, args[1:])
	case "chain":
		return cmdChain(w, args[1:])
	case "agents":
		return cmdAgents(w, args[1:])
	case "register":
		return cmdRegister(w, args[1:])
	case "export":
		return cmdExport(w, args[1:])
	case "prune":
		return cmdPrune(w, args[1:])
	case "rotate":
		return cmdRotate(w, args[1:])
	case "snapshot":
		if len(args) < 2 {
			return fmt.Errorf("usage: ans snapshot <take|diff|list> [flags]")
		}
		switch args[1] {
		case "take":
			return cmdSnapshotTake(w, args[2:])
		case "diff":
			return cmdSnapshotDiff(w, args[2:])
		case "list", "ls":
			return cmdSnapshots(w, args[2:])
		default:
			return fmt.Errorf("ans: unknown snapshot subcommand %q", args[1])
		}
	case "policy":
		if len(args) < 2 {
			return fmt.Errorf("usage: ans policy <add|list|remove|eval> [flags]")
		}
		switch args[1] {
		case "add":
			return cmdPolicyAdd(w, args[2:])
		case "list", "ls":
			return cmdPolicyList(w, args[2:])
		case "remove", "rm", "delete", "del":
			return cmdPolicyRemove(w, args[2:])
		case "eval":
			return cmdPolicyEval(w, args[2:])
		default:
			return fmt.Errorf("ans: unknown policy subcommand %q", args[1])
		}
	case "compensate":
		return cmdCompensate(w, args[1:])
	case "token":
		if len(args) < 2 {
			return fmt.Errorf("usage: ans token <request|list|revoke> [flags]")
		}
		switch args[1] {
		case "request":
			return cmdTokenRequest(w, args[2:])
		case "list", "ls":
			return cmdTokenList(w, args[2:])
		case "revoke", "rm":
			return cmdTokenRevoke(w, args[2:])
		default:
			return fmt.Errorf("ans: unknown token subcommand %q", args[1])
		}
	case "mcp":
		if len(args) < 2 {
			return fmt.Errorf("usage: ans mcp <start|stop|status|log> [flags]")
		}
		switch args[1] {
		case "start":
			return cmdMCPStart(w, args[2:])
		case "stop":
			return cmdMCPStop(w, args[2:])
		case "status":
			return cmdMCPStatus(w, args[2:])
		case "log":
			return cmdMCPLog(w, args[2:])
		default:
			return fmt.Errorf("ans: unknown mcp subcommand %q", args[1])
		}
	case "time-travel", "restore", "rollback":
		return cmdTimeTravel(w, args[1:])
	case "snapshots":
		return cmdSnapshots(w, args[1:])
	case "doctor":
		return cmdDoctor(w, args[1:])
	case "update":
		return cmdUpdate(w, args[1:])
	case "uninstall":
		return cmdUninstall(w, args[1:])
	case "version", "--version", "-v":
		printVersion(w)
		return nil
	case "help", "--help", "-h":
		PrintUsageTo(w)
		return nil
	default:
		return fmt.Errorf("ans: unknown command %q\nRun 'ans help' for usage.", args[0])
	}
}

// Dispatch calls DispatchTo with os.Stderr as the output writer.
func Dispatch(args []string) error {
	return DispatchTo(os.Stderr, args)
}

func printVersion(w io.Writer) {
	pretty.Banner(w)
	pretty.Item(w, "Version", version)
	pretty.Item(w, "Platform", runtime.GOOS+"/"+runtime.GOARCH)
	fmt.Fprintln(w)
}

// PrintUsageTo writes the full usage text to w.
func PrintUsageTo(w io.Writer) {
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

// helpers used by multiple handler files
func noColor() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("ANS_NO_COLOR") != ""
}

// NoColor is the exported equivalent of noColor, for external callers.
func NoColor() bool { return noColor() }

func safeSig(sig string) string {
	if len(sig) < 16 {
		return "[short]"
	}
	return sig[:16]
}

func maskSecret(s string) string {
	if s == "" {
		return "[REDACTED]"
	}
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// IsHex is the exported equivalent of isHex, for external callers.
func IsHex(s string) bool { return isHex(s) }

func pidFilePath() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".ans", "daemon.pid")
}

// PidFilePath is the exported equivalent of pidFilePath, for external callers.
func PidFilePath() string { return pidFilePath() }
