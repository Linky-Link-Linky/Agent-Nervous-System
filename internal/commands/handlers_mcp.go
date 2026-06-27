package commands

import (
	"flag"
	"fmt"
	"io"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
)

// --- mcp start ---

func cmdMCPStart(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("mcp start", flag.ContinueOnError)
	listen := fs.String("listen", ":8080", "Listen address")
	target := fs.String("target", "", "Target MCP server URL (required)")
	safetyDisable := fs.Bool("safety-disable", false, "Disable all safety features")
	rateLimit := fs.Int("rate-limit", 60, "Requests per minute per client IP (0 = unlimited)")
	tokenBudget := fs.Int("token-budget", 50000, "Estimated tokens per minute per agent (0 = unlimited)")
	piiRedact := fs.Bool("pii-redact", true, "Enable PII redaction on server responses")
	_ = fs.Parse(args)
	if *target == "" {
		return fmt.Errorf("usage: ans mcp start --listen :8080 --target http://mcp-server:8080")
	}
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
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
		return fmt.Errorf("mcp start failed: %v", err)
	}
	if resp.Success {
		pretty.Ok(w, fmt.Sprintf("MCP proxy started on %s -> %s", *listen, *target))
	} else {
		return fmt.Errorf("mcp start: %s", resp.Message)
	}
	return nil
}

// --- mcp stop ---

func cmdMCPStop(w io.Writer, args []string) error {
	_ = args
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgMCPStop, daemon.MCPStopReq{})
	var resp daemon.MCPStopResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("mcp stop failed: %v", err)
	}
	if resp.Success {
		pretty.Done(w, "MCP proxy stopped")
	} else {
		pretty.Warn(w, resp.Message)
	}
	return nil
}

// --- mcp status ---

func cmdMCPStatus(w io.Writer, args []string) error {
	_ = args
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgMCPStatus, nil)
	var resp daemon.MCPStatusResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("mcp status failed: %v", err)
	}
	if !resp.Running {
		pretty.Err(w, "MCP proxy not running")
		return nil
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
	return nil
}

// --- mcp log ---

func cmdMCPLog(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("mcp log", flag.ContinueOnError)
	limit := fs.Int("n", 20, "Number of entries")
	method := fs.String("method", "", "Filter by method")
	injOnly := fs.Bool("inj", false, "Show only injections")
	_ = fs.Parse(args)
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgMCPLog, daemon.MCPLogReq{Limit: *limit, Method: *method, InjOnly: *injOnly})
	var resp daemon.MCPLogResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("mcp log failed: %v", err)
	}
	if len(resp.Entries) == 0 {
		pretty.Warn(w, "No MCP log entries")
		return nil
	}
	pretty.Header(w, "MCP Audit Log")
	for _, e := range resp.Entries {
		method := e.Method
		if method == "" {
			method = "(response)"
		}
		content := e.Content
		if len(content) > 40 {
			content = content[:40] + "..."
		}
		inj := ""
		if e.Injection {
			inj = " " + pretty.Red + "INJ" + pretty.Reset
		}
		pretty.Item(w, fmt.Sprintf("#%d", e.ID), fmt.Sprintf("%s  %s  %d toks  %s%s", e.Direction, method, e.ToksEst, content, inj))
	}
	fmt.Fprintln(w)
	return nil
}
