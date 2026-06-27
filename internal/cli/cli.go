package cli

import (
	"fmt"
	"os"

	"ans/internal/client"
)

func Run(subcmd string, args []string, c client.Client) {
	switch subcmd {
	case "init":
		runInit(c)
	case "doctor":
		runDoctor(c)
	case "start":
		runStart(args, c)
	case "stop":
		runStop(c)
	case "status":
		runStatus(c)
	case "update":
		runUpdate(c)
	case "uninstall":
		runUninstall(c)
	case "version":
		runVersion()
	case "help":
		runHelp()
	case "chain":
		runChain(args, c)
	case "verify":
		runVerify(args, c)
	case "agents":
		runAgents(c)
	case "register":
		runRegister(args, c)
	case "export":
		runExport(args, c)
	case "prune":
		runPrune(args, c)
	case "rotate":
		runRotate(args, c)
	case "time-travel":
		runTimeTravel(args, c)
	case "snapshot":
		runSnapshot(args, c)
	case "snapshots":
		runSnapshots(args, c)
	case "compensate":
		runCompensate(args, c)
	case "policy":
		runPolicy(args, c)
	case "token":
		runToken(args, c)
	case "mcp":
		runMCP(args, c)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\nRun 'ans help' for usage.\n", subcmd)
		exitErr(1)
	}
}

func runHelp() {
	fmt.Print(`Usage: ans <subcommand> [flags]

Setup & Maintenance:
  init                   Create ~/.ans/ config and chain db
  doctor                 Run system diagnostics
  start [--ndjson] [--webhook <url>]
  stop                   Stop the daemon
  status                 Show daemon status
  update                 Self-update to latest release
  uninstall              Remove ANS and all data
  version                Print version

Chain & Receipts:
  chain [--n N] [--agent <id>]
  verify <id> [--chain]
  agents                 List registered agents
  register --name <name> [--version <ver>] [--owner <org>]
  export --format <fmt> [--output <path>]
  prune --up-to <index>
  rotate <agent-id>

Snapshots & Time-Travel:
  time-travel <index> [--type <type>]
  snapshot take [--agent <id>] [--type <type>] [--paths <a,b>]
  snapshot diff [--agent <id>]
  snapshots [--agent <id>] [--n <int>]

Compensation:
  compensate <index> [--dry-run]

Policies:
  policy add <file.json>
  policy list [--enabled]
  policy remove <id>
  policy eval --action-type <type> [--payload-summary <text>]

Tokens:
  token request --resource <arn> [--action <act>] [--ttl <sec>]
  token list
  token revoke <id>

MCP Security Proxy:
  mcp start --target <url> [--listen <:port>] [--safety-disable] [--rate-limit <n>]
  mcp stop
  mcp status
  mcp log [--inj] [--method <name>] [-n]

Run 'ans <subcommand> --help' for flags.
`)
}
