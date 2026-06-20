// Package chain — merge.go implements multi-agent chain merge with causal ordering.
// When two or more agents produce separate chains on the same machine, Merge
// interleaves their receipts in causal order: receipts linked by AgentDelegateAction
// (pre-receipt → post-receipt → delegated sub-agent receipts) are sorted causally;
// otherwise receipts are sorted by TimestampNS ascending.
// SPDX-License-Identifier: MIT
package chain

import (
	"encoding/json"
	"log"
	"sort"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

// MergedReceipt is a receipt annotated with its source agent ID for display.
type MergedReceipt struct {
	*receipt.Receipt
	SourceAgentID string
}

// MergeChains loads all receipts from the given chain and returns them
// interleaved in causal order.
//
// Causal ordering rules (applied in priority order):
//  1. A post-action receipt always follows its paired pre-action receipt.
//  2. A sub-agent receipt (ParentAgentID set) follows the delegation receipt
//     that spawned it (the pre-receipt whose AgentID == sub-agent's ParentAgentID
//     and ActionType == agent.delegate).
//  3. All remaining receipts are ordered by TimestampNS ascending.
//
// This produces a single timeline that reflects what actually happened causally,
// even when agents ran concurrently and their wall-clock timestamps overlap.
func (c *Chain) MergeChains(agentIDs []string) ([]*MergedReceipt, error) {
	// Collect all receipts for the given agents
	var all []*receipt.Receipt
	for _, aid := range agentIDs {
		receipts, err := c.List(QueryOptions{AgentID: aid})
		if err != nil {
			return nil, err
		}
		all = append(all, receipts...)
	}
	if len(all) == 0 {
		return nil, nil
	}

	// Index receipts by ReceiptID for O(1) lookup
	byID := make(map[string]*receipt.Receipt, len(all))
	for _, r := range all {
		byID[r.ReceiptID] = r
	}

	// Build causal dependency graph:
	// deps[id] = list of receipt IDs that must come BEFORE id
	deps := make(map[string][]string, len(all))
	for _, r := range all {
		var before []string
		// Post-action depends on its pre-action
		if r.Phase == receipt.PhasePost && r.PreReceiptID != "" {
			if _, ok := byID[r.PreReceiptID]; ok {
				before = append(before, r.PreReceiptID)
			}
		}
		// Sub-agent receipts depend on the delegation receipt.
		// If the parent agent's delegation receipt isn't in our filtered set,
		// query the chain directly to find it.
		if r.ParentAgentID != "" {
			found := false
			for _, other := range all {
				if other.AgentID == r.ParentAgentID &&
					other.ActionType == receipt.ActionAgentDelegate &&
					other.Phase == receipt.PhasePre {
					before = append(before, other.ReceiptID)
					found = true
					break
				}
			}
			if !found {
				// Cross-agent reference: query the chain for the delegation receipt
				allReceipts, err := c.List(QueryOptions{
					AgentID:    r.ParentAgentID,
					ActionType: string(receipt.ActionAgentDelegate),
					Phase:      string(receipt.PhasePre),
				})
			if err == nil && len(allReceipts) > 0 {
				before = append(before, allReceipts[0].ReceiptID)
			} else if err != nil {
				log.Printf("merge: cross-agent query for %s: %v", r.ParentAgentID, err)
			}
			}
		}
		deps[r.ReceiptID] = before
	}

	// Topological sort with timestamp as tiebreaker
	sorted := topoSort(all, deps, byID)

	result := make([]*MergedReceipt, len(sorted))
	for i, r := range sorted {
		result[i] = &MergedReceipt{Receipt: r, SourceAgentID: r.AgentID}
	}
	return result, nil
}

// topoSort performs a stable topological sort over receipts using Kahn's algorithm.
// Receipts with no unresolved dependencies are sorted by TimestampNS ascending
// before being appended, preserving chronological order within the same causal level.
// byID is the receipt index built by MergeChains — passed in to avoid rebuilding it.
func topoSort(receipts []*receipt.Receipt, deps map[string][]string, byID map[string]*receipt.Receipt) []*receipt.Receipt {
	// Count in-degrees
	inDegree := make(map[string]int, len(receipts))
	revDeps := make(map[string][]string, len(receipts)) // who depends on me
	for _, r := range receipts {
		if _, ok := inDegree[r.ReceiptID]; !ok {
			inDegree[r.ReceiptID] = 0
		}
		for _, dep := range deps[r.ReceiptID] {
			inDegree[r.ReceiptID]++
			revDeps[dep] = append(revDeps[dep], r.ReceiptID)
		}
	}

	// Seed queue with zero-dependency receipts, sorted by timestamp
	var queue []*receipt.Receipt
	for _, r := range receipts {
		if inDegree[r.ReceiptID] == 0 {
			queue = append(queue, r)
		}
	}
	sort.Slice(queue, func(i, j int) bool {
		return queue[i].TimestampNS < queue[j].TimestampNS
	})

	var result []*receipt.Receipt
	for len(queue) > 0 {
		// Pop front
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Reduce in-degrees of dependents; collect newly free ones
		var freed []*receipt.Receipt
		for _, depID := range revDeps[node.ReceiptID] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				if r, ok := byID[depID]; ok {
					freed = append(freed, r)
				}
			}
		}
		sort.Slice(freed, func(i, j int) bool {
			return freed[i].TimestampNS < freed[j].TimestampNS
		})
		queue = append(freed, queue...)
		sort.SliceStable(queue, func(i, j int) bool {
			return queue[i].TimestampNS < queue[j].TimestampNS
		})
	}

	// Append any remaining (cycle-broken) receipts sorted by timestamp
	if len(result) < len(receipts) {
		inResult := make(map[string]bool, len(result))
		for _, r := range result {
			inResult[r.ReceiptID] = true
		}
		var remainder []*receipt.Receipt
		for _, r := range receipts {
			if !inResult[r.ReceiptID] {
				remainder = append(remainder, r)
			}
		}
		sort.Slice(remainder, func(i, j int) bool {
			return remainder[i].TimestampNS < remainder[j].TimestampNS
		})
		result = append(result, remainder...)
	}
	return result
}

// MarshalMergedChain serialises a merged chain as JSONL bytes.
// Each line is a JSON-encoded MergedReceipt followed by a newline.
func MarshalMergedChain(merged []*MergedReceipt) ([]byte, error) {
	var lines []byte
	for _, m := range merged {
		b, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		lines = append(lines, b...)
		lines = append(lines, '\n')
	}
	return lines, nil
}
