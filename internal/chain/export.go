// Package chain — export.go: JSONL, CSV, and plain-text audit report.
// SPDX-License-Identifier: MIT
package chain

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ans-project/ans/internal/receipt"
)

func (c *Chain) ExportJSONL(w io.Writer, opts QueryOptions) error {
	q, args := c.buildExportQuery(opts)
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		if err := enc.Encode(json.RawMessage(raw)); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (c *Chain) ExportCSV(w io.Writer, opts QueryOptions) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"chain_index", "receipt_id", "phase", "agent_id", "action_type",
		"policy_decision", "outcome", "duration_ms", "timestamp",
		"payload_summary", "outcome_summary", "prev_receipt_hash",
	}); err != nil {
		return err
	}
	q, args := c.buildExportQuery(opts)
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var r receipt.Receipt
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return fmt.Errorf("corrupt receipt at chain_index: %w", err)
		}
		ts := time.Unix(0, r.TimestampNS).UTC().Format(time.RFC3339Nano)
		if err := cw.Write([]string{
			fmt.Sprintf("%d", r.ChainIndex), r.ReceiptID, string(r.Phase),
			r.AgentID, string(r.ActionType), string(r.PolicyDecision),
			string(r.Outcome), fmt.Sprintf("%d", r.DurationMS), ts,
			r.PayloadSummary, r.OutcomeSummary, r.PrevReceiptHash,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return rows.Err()
}

func (c *Chain) ExportTextReport(w io.Writer) error {
	stats, err := c.GetStats()
	if err != nil {
		return err
	}
	result := c.VerifyChain(nil)
	fmt.Fprintf(w, "=================================================================\n")
	fmt.Fprintf(w, "  ANS AUDIT REPORT\n")
	fmt.Fprintf(w, "=================================================================\n\n")
	fmt.Fprintf(w, "Generated:      %s\n\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(w, "Total receipts: %d\n", stats.TotalReceipts)
	fmt.Fprintf(w, "Total agents:   %d\n", stats.TotalAgents)
	fmt.Fprintf(w, "Chain length:   %d\n", stats.ChainLength)
	if stats.OldestReceiptNS > 0 {
		fmt.Fprintf(w, "Oldest:         %s\n", time.Unix(0, stats.OldestReceiptNS).UTC().Format(time.RFC3339))
		fmt.Fprintf(w, "Newest:         %s\n", time.Unix(0, stats.NewestReceiptNS).UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(w, "\nINTEGRITY\n-----------------------------------------------------------------\n")
	if result.Valid {
		fmt.Fprintf(w, "PASS - %d receipts verified\n\n", result.TotalChecked)
	} else {
		fmt.Fprintf(w, "FAIL at index %d: %s\n\n", result.FirstBrokenAt, result.Error)
	}
	fmt.Fprintf(w, "RECEIPTS (most recent 50)\n-----------------------------------------------------------------\n")
	receipts, err := c.List(QueryOptions{Limit: 50})
	if err != nil {
		return err
	}
	for _, r := range receipts {
		ts := time.Unix(0, r.TimestampNS).UTC().Format("2006-01-02 15:04:05")
		status := "->"
		if r.Phase == receipt.PhasePost {
			if r.Outcome == receipt.OutcomeSuccess {
				status = "OK"
			} else if r.Outcome == receipt.OutcomeFailure {
				status = "FAIL"
			}
		}
		fmt.Fprintf(w, "[%s] %-4s %s %-20s %s\n",
			ts, status, safeID(r.ReceiptID), string(r.ActionType), r.PayloadSummary)
	}
	return nil
}

func safeID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}
