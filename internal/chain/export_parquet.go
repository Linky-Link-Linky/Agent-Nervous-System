// Package chain — export_parquet.go produces Apache Parquet files.
// SPDX-License-Identifier: Apache-2.0
package chain

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/parquet-go/parquet-go"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

type parquetRow struct {
	ChainIndex          int64  `parquet:"chain_index"`
	ReceiptID           string `parquet:"receipt_id"`
	Phase               string `parquet:"phase"`
	AgentID             string `parquet:"agent_id"`
	ParentAgentID       string `parquet:"parent_agent_id"`
	ActionType          string `parquet:"action_type"`
	PayloadHash         string `parquet:"payload_hash"`
	PayloadSummary      string `parquet:"payload_summary"`
	PolicyDecision      string `parquet:"policy_decision"`
	AuthorizingContext  string `parquet:"authorizing_context"`
	Outcome             string `parquet:"outcome"`
	OutcomeSummary      string `parquet:"outcome_summary"`
	DurationMS          int64  `parquet:"duration_ms"`
	PreReceiptID        string `parquet:"pre_receipt_id"`
	PrevReceiptHash     string `parquet:"prev_receipt_hash"`
	ReceiptHash         string `parquet:"receipt_hash"`
	TimestampNS         int64  `parquet:"timestamp_ns"`
	TimestampRFC3339    string `parquet:"timestamp"`
}

func rowFromReceipt(r *receipt.Receipt) (parquetRow, error) {
	ts := time.Unix(0, r.TimestampNS).UTC().Format(time.RFC3339Nano)
	hash, err := r.ComputeHash()
	if err != nil {
		return parquetRow{}, fmt.Errorf("compute hash: %w", err)
	}
	return parquetRow{
		ChainIndex:         int64(r.ChainIndex), // #nosec G115
		ReceiptID:          r.ReceiptID,
		Phase:              string(r.Phase),
		AgentID:            r.AgentID,
		ParentAgentID:      r.ParentAgentID,
		ActionType:         string(r.ActionType),
		PayloadHash:        r.PayloadHash,
		PayloadSummary:     r.PayloadSummary,
		PolicyDecision:     string(r.PolicyDecision),
		AuthorizingContext: r.AuthorizingContext,
		Outcome:            string(r.Outcome),
		OutcomeSummary:     r.OutcomeSummary,
		DurationMS:         r.DurationMS,
		PreReceiptID:       r.PreReceiptID,
		PrevReceiptHash:    r.PrevReceiptHash,
		ReceiptHash:        hash,
		TimestampNS:        r.TimestampNS,
		TimestampRFC3339:   ts,
	}, nil
}

func (c *Chain) ExportParquet(w io.Writer, opts QueryOptions) error {
	q, args := c.buildExportQuery(opts)
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var all []parquetRow
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return err
		}
		var r receipt.Receipt
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return fmt.Errorf("unmarshal receipt: %w", err)
		}
		pr, err := rowFromReceipt(&r)
		if err != nil {
			return fmt.Errorf("row at index %d: %w", r.ChainIndex, err)
		}
		all = append(all, pr)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(all) == 0 {
		return nil
	}

	writer := parquet.NewGenericWriter[parquetRow](w)
	for i := range all {
		if _, err := writer.Write([]parquetRow{all[i]}); err != nil {
			return fmt.Errorf("writing parquet row %d: %w", i, err)
		}
	}
	return writer.Close()
}

// buildExportQuery returns a SQL query and args that apply QueryOptions filters.
func (c *Chain) buildExportQuery(opts QueryOptions) (string, []interface{}) {
	q := `SELECT raw_receipt FROM receipts WHERE 1=1`
	var args []interface{}
	if opts.AgentID != "" {
		q += " AND agent_id=?"
		args = append(args, opts.AgentID)
	}
	if opts.ActionType != "" {
		q += " AND action_type=?"
		args = append(args, opts.ActionType)
	}
	if opts.Phase != "" {
		q += " AND phase=?"
		args = append(args, opts.Phase)
	}
	if !opts.Since.IsZero() {
		q += " AND timestamp_ns>=?"
		args = append(args, opts.Since.UnixNano())
	}
	q += " ORDER BY chain_index ASC"
	return q, args
}
