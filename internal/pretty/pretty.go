// Package pretty renders the ANS receipt chain as a colored tree.
// "git log for AI agents."
// Uses only stdlib ANSI codes. Respects NO_COLOR env var (https://no-color.org/).
// SPDX-License-Identifier: Apache-2.0
package pretty

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

// ansiRE matches ANSI escape sequences.
var ansiRE = regexp.MustCompile(`\033\[[0-9;]*[a-zA-Z]|\033][^\a]*(\a|\033\\)`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// Color palette — Daytona-inspired emerald-on-black.
// Single brand voltage: emerald green (#2ecc71) reserved for success, highlights, and emphasis.
// Uses 256-color / 24-bit ANSI codes for rich rendering.
const (
	Reset = "\033[0m"
	Bold  = "\033[1m"
	Dim   = "\033[2m"

	// Brand — single emerald voltage (#2ecc71)
	Emerald = "\033[38;2;46;204;113m"

	// Semantic colors
	Red    = "\033[38;2;231;76;60m"  // coral
	Yellow = "\033[38;2;241;196;15m" // amber
	Cyan   = "\033[38;2;52;152;219m" // cobalt blue

	// Grayscale
	Gray  = "\033[38;5;243m" // warm gray — labels, secondary text
	Muted = "\033[38;5;236m" // dim gray — borders, separators

	// Exported aliases for backward compatibility
	Green   = Emerald
	Primary = Emerald
	Success = Emerald
	Warning = Yellow
	Accent  = Cyan
	errClr  = Red
)

// Backward-compat aliases for internal use
var (
	reset   = Reset
	bold    = Bold
	dim     = Dim
	green   = Green
	red     = Red
	yellow  = Yellow
	cyan    = Cyan
	gray    = Gray
	muted   = Muted
	primary = Primary
	success = Success
	warning = Warning
	accent  = Accent
)

// Box dimensions
const boxW = 56 // total width of boxed elements including borders

const boxBorder = "  │ "

var (
	boxTop = "╭" + strings.Repeat("─", boxW-2) + "╮"
	boxBot = "╰" + strings.Repeat("─", boxW-2) + "╯"
)

func boxLine(content string) string {
	plain := stripANSI(content)
	pad := boxW - 4 - len(plain)
	if pad < 0 {
		pad = 0
	}
	return boxBorder + content + strings.Repeat(" ", pad) + " │"
}

func emptyLine() string {
	return boxLine("")
}

// HasColor returns true if the terminal supports color output.
func HasColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return true
}

// --- Styled output helpers — Daytona-inspired emerald theme ---

// Fprint styled text to w with the given ANSI codes and args.
func Fprint(w io.Writer, style string, args ...interface{}) {
	fmt.Fprint(w, style)
	fmt.Fprint(w, args...)
	fmt.Fprint(w, Reset)
}

// Fprintf styled formatted text to w.
func Fprintf(w io.Writer, style, format string, args ...interface{}) {
	fmt.Fprint(w, style)
	fmt.Fprintf(w, format, args...)
	fmt.Fprint(w, Reset)
}

// Fprintln styled text to w followed by newline.
func Fprintln(w io.Writer, style string, args ...interface{}) {
	fmt.Fprint(w, style)
	fmt.Fprint(w, args...)
	fmt.Fprintln(w, Reset)
}

// Banner prints the ANS branding header in a rounded box.
func Banner(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  "+muted+boxTop+Reset)
	fmt.Fprintln(w, boxLine("  "+Emerald+"✦"+Reset+"  "+bold+"Agent Nervous System"+Reset))
	fmt.Fprintln(w, boxLine("  "+dim+"Secure AI Agent Auditing"+Reset))
	fmt.Fprintln(w, "  "+muted+boxBot+Reset)
}

// Header prints a section title as a box divider with emerald accent.
func Header(w io.Writer, text string) {
	inner := " " + bold + Emerald + text + Reset + " "
	dashCount := boxW - 5 - len(text)
	if dashCount < 0 {
		dashCount = 0
	}
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "  %s╭─%s%s%s%s╮%s\n",
		muted, reset, inner, muted, strings.Repeat("─", dashCount), reset)
	fmt.Fprintln(w, emptyLine())
}

// Subheader prints a subsection with a small dot prefix.
func Subheader(w io.Writer, text string) {
	fmt.Fprintf(w, "  %s·%s  %s%s%s\n", Emerald, reset, bold, text, reset)
}

// Step prints a numbered step with emerald number.
func Step(w io.Writer, num int, text string) {
	fmt.Fprintf(w, "  %s%s%s %s%s%s\n", bold, Emerald, fmt.Sprintf("%d.", num), bold, text, reset)
}

// Done prints a completed item with emerald bullet.
func Done(w io.Writer, text string) {
	fmt.Fprintf(w, "  %s●%s  %s\n", Emerald, reset, text)
}

// Item prints a labeled value pair — clean two-column layout.
func Item(w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %s%s%s %s\n", Gray, label+":", reset, value)
}

// Code prints a command with a dim $ prompt.
func Code(w io.Writer, cmd string) {
	fmt.Fprintf(w, "    %s$%s %s\n", Gray, reset, cmd)
}

// Link prints a clickable resource reference.
func Link(w io.Writer, label, url string) {
	fmt.Fprintf(w, "  %s%s:%s %s\n", Gray, label, reset, url)
}

// Ok prints a success banner inside an emerald box.
func Ok(w io.Writer, text string) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  "+Emerald+"╭"+strings.Repeat("─", boxW-2)+"╮"+Reset)
	fmt.Fprintln(w, "  "+Emerald+"│"+Reset+boxPad("  "+Emerald+"✓"+Reset+"  "+bold+text+Reset, boxW-2)+Emerald+"│"+Reset)
	fmt.Fprintln(w, "  "+Emerald+"╰"+strings.Repeat("─", boxW-2)+"╯"+Reset)
}

// Warn prints a warning banner inside an amber box.
func Warn(w io.Writer, text string) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  "+Yellow+"╭"+strings.Repeat("─", boxW-2)+"╮"+Reset)
	fmt.Fprintln(w, "  "+Yellow+"│"+Reset+boxPad("  "+Yellow+"!"+Reset+"  "+bold+text+Reset, boxW-2)+Yellow+"│"+Reset)
	fmt.Fprintln(w, "  "+Yellow+"╰"+strings.Repeat("─", boxW-2)+"╯"+Reset)
}

// Err prints an error banner inside a coral box.
func Err(w io.Writer, text string) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  "+Red+"╭"+strings.Repeat("─", boxW-2)+"╮"+Reset)
	fmt.Fprintln(w, "  "+Red+"│"+Reset+boxPad("  "+Red+"✗"+Reset+"  "+bold+text+Reset, boxW-2)+Red+"│"+Reset)
	fmt.Fprintln(w, "  "+Red+"╰"+strings.Repeat("─", boxW-2)+"╯"+Reset)
}

// boxPad returns s padded with trailing spaces to reach width w (ANSI-aware).
func boxPad(s string, w int) string {
	plain := stripANSI(s)
	pad := w - len(plain)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

// safeID returns the first 8 chars of id, or the full id if shorter, with ANSI stripped.
func safeID(id string) string {
	id = stripANSI(id)
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// PrintChain renders receipts as a pretty tree to w.
func PrintChain(w io.Writer, receipts []*receipt.Receipt, noColor bool) {
	if noColor {
		printChainPlain(w, receipts)
		return
	}
	printChainColor(w, receipts)
}

func printChainColor(w io.Writer, receipts []*receipt.Receipt) {
	fmt.Fprintf(w, "\n%s%s    ANS — Receipt Chain%s\n", bold, Emerald, reset)
	fmt.Fprintf(w, "  %s%s%s\n\n", muted, strings.Repeat("━", boxW-4), reset)

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
	fmt.Fprintf(w, "  %s╭─%s %s%s%s  %s%s %s%s  %s%s%s  %s%s%s\n",
		muted, reset,
		Emerald, safeID(pre.ReceiptID), reset,
		dim, ts.Format("2006-01-02"), ts.Format("15:04:05.000"), reset,
		bold, stripANSI(string(pre.ActionType)), reset,
		dim, stripANSI(pre.AgentID), reset,
	)
	if pre.PayloadSummary != "" {
		fmt.Fprintf(w, "  %s│%s  %s%s%s\n", muted, reset, dim, stripANSI(pre.PayloadSummary), reset)
	}
	policyColor := Emerald
	if pre.PolicyDecision == receipt.PolicyDeny {
		policyColor = Red
	} else if pre.PolicyDecision == receipt.PolicyAllowWithConditions {
		policyColor = Yellow
	}
	policyStr := stripANSI(string(pre.PolicyDecision))
	if policyStr == "" {
		policyStr = "allow"
	}
	fmt.Fprintf(w, "  %s│%s  policy: %s%s%s\n", muted, reset, policyColor, policyStr, reset)

	if post != nil {
		icon := Emerald + "✓" + Reset
		if post.Outcome == receipt.OutcomeFailure {
			icon = Red + "✗" + Reset
		} else if post.Outcome == receipt.OutcomePartial {
			icon = Yellow + "◐" + Reset
		}
		dur := ""
		if post.DurationMS > 0 {
			dur = fmt.Sprintf("  %s%dms%s", dim, post.DurationMS, reset)
		}
		fmt.Fprintf(w, "  %s╰─%s %s %s%s%s%s\n",
			muted, reset, icon, dim, stripANSI(post.OutcomeSummary), reset, dur)
		if len(post.Signature) >= 16 {
			fmt.Fprintf(w, "     %ssig: %s…%s\n", Gray, post.Signature[:16], reset)
		}
	} else {
		fmt.Fprintf(w, "  %s╰─%s %s(pending)%s\n", muted, reset, dim, reset)
	}
	fmt.Fprintln(w)
}

func printOrphan(w io.Writer, r *receipt.Receipt) {
	ts := time.Unix(0, r.TimestampNS).UTC()
	fmt.Fprintf(w, "  %s○%s %s%s%s  %s%s%s  %s\n",
		muted, reset,
		Emerald, safeID(r.ReceiptID), reset,
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
	fmt.Fprintf(w, "\n%s%s    ANS Daemon Status%s\n", bold, Emerald, reset)
	fmt.Fprintf(w, "  %s%s%s\n\n", muted, strings.Repeat("━", boxW-4), reset)
	for _, kv := range []struct{ k, vk string }{
		{"uptime", "uptime"}, {"chain length", "chain_length"},
		{"total receipts", "total_receipts"}, {"total agents", "total_agents"},
		{"db size", "db_size_bytes"}, {"started at", "started_at"},
	} {
		fmt.Fprintf(w, "  %s%-16s%s %s%v%s\n", Gray, kv.k, reset, bold, status[kv.vk], reset)
	}
	fmt.Fprintln(w, "  "+muted+strings.Repeat("━", boxW-4)+reset)
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
		fmt.Fprint(w, "\n  "+Emerald+"╭"+strings.Repeat("─", boxW-2)+"╮"+Reset+"\n")
		fmt.Fprint(w, "  "+Emerald+"│"+Reset+boxPad("  "+bold+"✓ Receipt verified"+Reset, boxW-2)+Emerald+"│"+Reset+"\n")
		fmt.Fprint(w, "  "+Emerald+"╰"+strings.Repeat("─", boxW-2)+"╯"+Reset+"\n")
	} else {
		fmt.Fprint(w, "\n  "+Red+"╭"+strings.Repeat("─", boxW-2)+"╮"+Reset+"\n")
		fmt.Fprint(w, "  "+Red+"│"+Reset+boxPad("  "+bold+"✗ Receipt INVALID"+Reset, boxW-2)+Red+"│"+Reset+"\n")
		fmt.Fprint(w, "  "+Red+"╰"+strings.Repeat("─", boxW-2)+"╯"+Reset+"\n")
	}
	for _, key := range []string{"receipt_id", "agent_id", "agent_name", "action_type",
		"phase", "policy_decision", "outcome", "chain_index"} {
		if v, ok := resp[key]; ok && v != nil && v != "" {
			fmt.Fprintf(w, "  %s%-18s%s %v\n", Gray, key, reset, stripANSI(fmt.Sprint(v)))
		}
	}
	if errMsg, ok := resp["error"].(string); ok && errMsg != "" {
		fmt.Fprintf(w, "\n  %s%s%s\n", Red, errMsg, reset)
	}
	fmt.Fprintln(w)
}
