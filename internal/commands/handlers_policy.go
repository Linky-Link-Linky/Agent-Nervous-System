package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	path "path/filepath"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/pretty"
)

// --- policy add ---

func cmdPolicyAdd(w io.Writer, args []string) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("usage: ans policy add <file.json>")
	}
	p := path.Clean(args[0])
	if !path.IsAbs(p) {
		abs, err := path.Abs(p)
		if err != nil {
			return fmt.Errorf("resolving policy path: %v", err)
		}
		p = abs
	}
	if p == "/" {
		return fmt.Errorf("invalid policy file path: %s", args[0])
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("reading policy file: %v", err)
	}
	var pol struct {
		ID          string      `json:"id"`
		Name        string      `json:"name"`
		Description string      `json:"description,omitempty"`
		Enabled     bool        `json:"enabled"`
		Priority    int         `json:"priority"`
		Severity    string      `json:"severity,omitempty"`
		Conditions  interface{} `json:"conditions"`
		Action      interface{} `json:"action"`
	}
	if err := json.Unmarshal(data, &pol); err != nil {
		return fmt.Errorf("parsing policy JSON: %v", err)
	}
	condJSON, _ := json.Marshal(pol.Conditions)
	actJSON, _ := json.Marshal(pol.Action)

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	if err := daemon.WriteJSON(conn, daemon.MsgPolicyRegister, daemon.PolicyRegisterReq{
		ID: pol.ID, Name: pol.Name, Description: pol.Description,
		Enabled: pol.Enabled, Priority: pol.Priority, Severity: pol.Severity,
		Conditions: string(condJSON), Action: string(actJSON),
	}); err != nil {
		return fmt.Errorf("sending policy register: %v", err)
	}
	var resp daemon.PolicyRegisterResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("policy register failed: %v", err)
	}
	if !resp.Success {
		return fmt.Errorf("policy rejected: %s", resp.Message)
	}
	pretty.Done(w, fmt.Sprintf("Policy %q registered", pol.ID))
	return nil
}

// --- policy list ---

func cmdPolicyList(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("policy list", flag.ContinueOnError)
	enabledOnly := fs.Bool("enabled", false, "Show only enabled policies")
	_ = fs.Parse(args)

	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgPolicyList, daemon.PolicyListReq{EnabledOnly: *enabledOnly})
	var resp daemon.PolicyListResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("policy list failed: %v", err)
	}
	if len(resp.Policies) == 0 {
		pretty.Warn(w, "No policies registered")
		return nil
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
	return nil
}

// --- policy remove ---

func cmdPolicyRemove(w io.Writer, args []string) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("usage: ans policy remove <id>")
	}
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgPolicyDelete, daemon.PolicyDeleteReq{ID: args[0]})
	var resp daemon.PolicyDeleteResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("policy delete failed: %v", err)
	}
	if resp.Success {
		pretty.Done(w, fmt.Sprintf("Policy %q removed", args[0]))
	} else {
		return fmt.Errorf("removing policy: %s", resp.Message)
	}
	return nil
}

// --- policy eval ---

func cmdPolicyEval(w io.Writer, args []string) error {
	fs := flag.NewFlagSet("policy eval", flag.ContinueOnError)
	actionType := fs.String("action-type", "", "Action type (required)")
	payloadSummary := fs.String("payload-summary", "", "Payload summary text")
	_ = fs.Parse(args)
	if *actionType == "" {
		return fmt.Errorf("usage: ans policy eval --action-type <type> [--payload-summary <text>]")
	}
	conn, err := daemon.Dial()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer conn.Close()
	_ = daemon.WriteJSON(conn, daemon.MsgPolicyEvaluate, daemon.PolicyEvaluateReq{
		AgentID: "_cli", ActionType: *actionType, Phase: "pre",
		PayloadSummary: *payloadSummary,
	})
	var resp daemon.PolicyEvaluateResp
	if _, err := daemon.ReadJSON(conn, &resp); err != nil {
		return fmt.Errorf("policy eval failed: %v", err)
	}
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
	return nil
}
