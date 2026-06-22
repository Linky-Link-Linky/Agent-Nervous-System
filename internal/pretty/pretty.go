// Package pretty renders the ANS receipt chain as a colored tree.
// "git log for AI agents."
// Uses only stdlib ANSI codes. Respects NO_COLOR env var (https://no-color.org/).
// SPDX-License-Identifier: Apache-2.0
package pretty

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

// ansiRE matches ANSI escape sequences.
var ansiRE = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]|\033][^\a]*(\a|\033\\)`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
)

// safeID returns the first 8 chars of id, or the full id if shorter, with ANSI stripped.
func safeID(id string) string {
	id = stripANSI(id)
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// PrintChain renders receipts as a pretty tree to w.
// noColor strips all ANSI codes.
func PrintChain(w io.Writer, receipts []*receipt.Receipt, noColor bool) {
	if noColor {
		printChainPlain(w, receipts)
		return
	}
	printChainColor(w, receipts)
}

func printChainColor(w io.Writer, receipts []*receipt.Receipt) {
	fmt.Fprintf(w, "\n%s%sANS — Agent Nervous System%s\n", bold, cyan, reset)
	fmt.Fprintf(w, "%s%s%s\n\n", gray, strings.Repeat("─", 50), reset)

	seen := make(map[string]bool, len(receipts))
	for _, r := range receipts {
		if r == nil || seen[r.ReceiptID] || r.Phase != receipt.PhasePre {
			continue
		}
		seen[r.ReceiptID] = true
		var post *receipt.Receipt
		for _, r2 := range receipts {
			if r2 == nil {
				continue
			}
			if r2.Phase == receipt.PhasePost && r2.PreReceiptID == r.ReceiptID {
				post = r2
				seen[r2.ReceiptID] = true
				break
			}
		}
		printPair(w, r, post)
	}
	// Orphan receipts (no paired pre found)
	for _, r := range receipts {
		if r == nil || seen[r.ReceiptID] {
			continue
		}
		printOrphan(w, r)
		seen[r.ReceiptID] = true
	}
}

func printPair(w io.Writer, pre *receipt.Receipt, post *receipt.Receipt) {
	ts := time.Unix(0, pre.TimestampNS).UTC()
	fmt.Fprintf(w, "%s┌─%s %s%s%s  %s%s %s%s  %s%s%s  %s%s%s\n",
		gray, reset,
		yellow, safeID(pre.ReceiptID), reset,
		dim, ts.Format("2006-01-02"), ts.Format("15:04:05.000"), reset,
		bold, stripANSI(string(pre.ActionType)), reset,
		dim, stripANSI(pre.AgentID), reset,
	)
	if pre.PayloadSummary != "" {
		fmt.Fprintf(w, "%s│  %s%s%s\n", gray, dim, stripANSI(pre.PayloadSummary), reset)
	}
	policyColor := green
	if pre.PolicyDecision == receipt.PolicyDeny {
		policyColor = red
	} else if pre.PolicyDecision == receipt.PolicyAllowWithConditions {
		policyColor = yellow
	}
	policyStr := stripANSI(string(pre.PolicyDecision))
	if policyStr == "" {
		policyStr = "allow"
	}
	fmt.Fprintf(w, "%s│%s  policy %s%s%s\n", gray, reset, policyColor, policyStr, reset)

	if post != nil {
		icon := green + "✓" + reset
		if post.Outcome == receipt.OutcomeFailure {
			icon = red + "✗" + reset
		} else if post.Outcome == receipt.OutcomePartial {
			icon = yellow + "◐" + reset
		}
		dur := ""
		if post.DurationMS > 0 {
			dur = fmt.Sprintf("  %s%dms%s", dim, post.DurationMS, reset)
		}
		fmt.Fprintf(w, "%s└─%s %s %s%s%s%s\n",
			gray, reset, icon, dim, stripANSI(post.OutcomeSummary), reset, dur)
		if len(post.Signature) >= 16 {
			fmt.Fprintf(w, "   %ssig %s…%s\n", gray, post.Signature[:16], reset)
		}
	} else {
		fmt.Fprintf(w, "%s└─%s %s(pending)%s\n", gray, reset, dim, reset)
	}
	fmt.Fprintln(w)
}

func printOrphan(w io.Writer, r *receipt.Receipt) {
	ts := time.Unix(0, r.TimestampNS).UTC()
	fmt.Fprintf(w, "%s○%s %s%s%s  %s%s%s  %s\n",
		gray, reset,
		yellow, safeID(r.ReceiptID), reset,
		dim, ts.Format("15:04:05"), reset,
		stripANSI(string(r.ActionType)),
	)
	fmt.Fprintln(w)
}

func printChainPlain(w io.Writer, receipts []*receipt.Receipt) {
	fmt.Fprintln(w, "ANS Receipt Chain")
	fmt.Fprintln(w, strings.Repeat("-", 60))
	for _, r := range receipts {
		if r == nil {
			continue
		}
		ts := time.Unix(0, r.TimestampNS).UTC().Format(time.RFC3339)
		fmt.Fprintf(w, "[%s] %s %s %s %s\n",
			ts, safeID(r.ReceiptID), stripANSI(string(r.Phase)), stripANSI(string(r.ActionType)), stripANSI(r.PayloadSummary))
	}
}

// PrintStatus renders a daemon status dashboard to w.
func PrintStatus(w io.Writer, status map[string]interface{}, noColor bool) {
	if noColor {
		fmt.Fprintln(w, "ANS daemon status")
		for _, kv := range []string{"uptime", "chain_length", "total_receipts", "total_agents", "db_size_bytes"} {
			fmt.Fprintf(w, "  %s: %v\n", kv, status[kv])
		}
		return
	}
	fmt.Fprintf(w, "\n%s%sANS daemon status%s\n\n", bold, cyan, reset)
	for _, kv := range []struct{ k, vk string }{
		{"uptime", "uptime"}, {"chain length", "chain_length"},
		{"total receipts", "total_receipts"}, {"total agents", "total_agents"},
		{"db size", "db_size_bytes"}, {"started at", "started_at"},
	} {
		fmt.Fprintf(w, "  %s%-16s%s %s%v%s\n", gray, kv.k, reset, bold, status[kv.vk], reset)
	}
	fmt.Fprintln(w)
}

// PrintVerifyResult renders a receipt verification result to w.
func PrintVerifyResult(w io.Writer, resp map[string]interface{}, noColor bool) {
	valid, _ := resp["valid"].(bool)
	receiptID, _ := resp["receipt_id"].(string)
	agentID, _ := resp["agent_id"].(string)

	if noColor {
		if valid {
			fmt.Fprintf(w, "VALID %s (agent: %s)\n", receiptID, agentID)
		} else {
			fmt.Fprintf(w, "INVALID %s: %v\n", receiptID, resp["error"])
		}
		return
	}
	if valid {
		fmt.Fprintf(w, "\n%s%s✓ Receipt verified%s\n\n", bold, green, reset)
	} else {
		fmt.Fprintf(w, "\n%s%s✗ Receipt INVALID%s\n\n", bold, red, reset)
	}
	for _, key := range []string{"receipt_id", "agent_id", "agent_name", "action_type",
		"phase", "policy_decision", "outcome", "chain_index"} {
		if v, ok := resp[key]; ok && v != nil && v != "" {
			fmt.Fprintf(w, "  %s%-18s%s %v\n", gray, key, reset, stripANSI(fmt.Sprint(v)))
		}
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		fmt.Fprintf(w, "\n  %s%s%s\n", red, errMsg, reset)
	}
	fmt.Fprintln(w)
}
