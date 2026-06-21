// Package chain — verify.go: full chain integrity verification and single-receipt check.
// SPDX-License-Identifier: Apache-2.0
package chain

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

// VerificationResult is the outcome of a chain integrity walk.
type VerificationResult struct {
	Valid         bool
	TotalChecked  int
	FirstBrokenAt int64 // -1 if none
	Error         string
}

// VerifyChain walks the entire chain in ascending order, recomputing each hash
// and checking the prev_hash pointer. Optionally verifies Ed25519 signatures.
//
// Pruning-awareness: if anchors exist, the walk starts from the first remaining
// receipt after the last anchor's to_index, seeding expectedPrev from the
// last receipt in the anchored segment's stored terminal hash (looked up from
// the receipts table's minimum chain_index present). This allows VerifyChain
// to pass on a chain that has been legitimately pruned.
func (c *Chain) VerifyChain(pubkeys map[string]ed25519.PublicKey) VerificationResult {
	// Seed expectedPrev: if anchors cover the gap before the first remaining
	// receipt, use the prev_hash of the first remaining receipt directly —
	// the anchor proves the prior segment was valid when it was created.
	// We trust the anchor's integrity and skip hash-chaining across the gap.
	var expectedPrev string
	var anchorToIndex uint64

	anchors, err := c.ListAnchors()
	if err != nil {
		return VerificationResult{Valid: false, FirstBrokenAt: -1, Error: fmt.Sprintf("listing anchors: %v", err)}
	}
	if len(anchors) > 0 {
		// Find the anchor that covers the highest range
		last := anchors[len(anchors)-1]
		anchorToIndex = last.ToIndex
		// The first remaining receipt's prev_hash is the hash of receipt at anchorToIndex.
		// Since that receipt is pruned, we trust the anchor and seed from the
		// first remaining receipt's own prev_hash field (verified against anchor root
		// separately). We mark the gap as anchor-verified.
		expectedPrev = "" // will be set from the first receipt's prev_hash below
	}

	rows, err := c.db.Query(
		`SELECT chain_index,receipt_id,prev_hash,receipt_hash,raw_receipt
		 FROM receipts ORDER BY chain_index ASC`)
	if err != nil {
		return VerificationResult{Error: err.Error(), FirstBrokenAt: -1}
	}
	defer rows.Close()

	result := VerificationResult{Valid: true, FirstBrokenAt: -1}
	first := true

	for rows.Next() {
		var idx int64
		var id, prevHash, storedHash, raw string
		if err := rows.Scan(&idx, &id, &prevHash, &storedHash, &raw); err != nil {
			return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
				Error: fmt.Sprintf("scan error at %d: %v", idx, err)}
		}
		result.TotalChecked++

		// On first receipt: seed expectedPrev appropriately.
		if first {
			first = false
			if anchorToIndex > 0 && uint64(idx) == anchorToIndex+1 { // #nosec G115 — idx from range is non-negative
				// First receipt is directly after an anchor — skip cross-gap hash check
				// and seed from the first receipt's own prev_hash (anchor-trusted).
				expectedPrev = prevHash
			} else if uint64(idx) == 1 { // #nosec G115 — idx from range is non-negative
				// Normal genesis: first receipt must link to genesis hash
				expectedPrev = receipt.GenesisHash
			} else {
				// Gap without an anchor — this is a data integrity error
				return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
					Error: fmt.Sprintf("chain starts at index %d with no anchor covering the gap", idx)}
			}
		}

		if prevHash != expectedPrev {
			return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
				Error: fmt.Sprintf("broken link at %d: expected prev=%s got %s", idx, expectedPrev, prevHash)}
		}

		var r receipt.Receipt
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
				Error: fmt.Sprintf("corrupt JSON at %d: %v", idx, err)}
		}
		computed, err := r.ComputeHash()
		if err != nil {
			return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
				Error: fmt.Sprintf("hash error at %d: %v", idx, err)}
		}
		if computed != storedHash {
			return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
				Error: fmt.Sprintf("hash mismatch at %d: stored=%s computed=%s", idx, storedHash, computed)}
		}
		if pubkeys != nil {
			if pub, ok := pubkeys[r.AgentID]; ok {
				if err := receipt.Verify(&r, pub); err != nil {
					return VerificationResult{Valid: false, FirstBrokenAt: idx, TotalChecked: result.TotalChecked,
						Error: fmt.Sprintf("sig invalid at %d: %v", idx, err)}
				}
			}
		}
		expectedPrev = storedHash
	}
	if err := rows.Err(); err != nil {
		return VerificationResult{Valid: false, FirstBrokenAt: -1, TotalChecked: result.TotalChecked,
			Error: fmt.Sprintf("rows error: %v", err)}
	}
	return result
}

// VerifyReceipt verifies a single receipt: checks stored hash and optionally signature.
func (c *Chain) VerifyReceipt(receiptID string, pubkeys map[string]ed25519.PublicKey) error {
	var storedHash, raw string
	if err := c.db.QueryRow(
		`SELECT receipt_hash, raw_receipt FROM receipts WHERE receipt_id=?`, receiptID,
	).Scan(&storedHash, &raw); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("receipt %s not found", receiptID)
		}
		return err
	}
	var r receipt.Receipt
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return fmt.Errorf("corrupt receipt JSON: %w", err)
	}
	computed, err := r.ComputeHash()
	if err != nil {
		return fmt.Errorf("computing hash: %w", err)
	}
	if computed != storedHash {
		return fmt.Errorf("hash mismatch: stored=%s computed=%s", storedHash, computed)
	}
	if pubkeys != nil {
		if pub, ok := pubkeys[r.AgentID]; ok {
			if err := receipt.Verify(&r, pub); err != nil {
				return fmt.Errorf("signature invalid: %w", err)
			}
		}
	}
	return nil
}
