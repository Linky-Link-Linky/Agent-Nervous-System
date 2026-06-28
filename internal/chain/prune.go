// Package chain — prune.go implements chain pruning with a cryptographic root anchor.
//
// Pruning allows long-running deployments to remove old receipts while preserving
// the ability to prove that the removed receipts existed and were valid.
//
// Mechanism:
//   1. Compute a Merkle root over all receipts being pruned (SHA-256 binary tree).
//   2. Insert a single "anchor receipt" into a separate `anchors` table, storing:
//      the Merkle root, the range of pruned chain indices, and the count.
//   3. Delete the pruned rows from the `receipts` table.
//   4. Future VerifyChain calls check anchors and treat them as verified segments.
//
// A Merkle proof for any individual pruned receipt can be computed from its
// original raw JSON — callers who retained the raw receipt JSON can prove
// inclusion against the stored root.
// SPDX-License-Identifier: Apache-2.0
package chain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

const anchorSchema = `
CREATE TABLE IF NOT EXISTS anchors (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	from_index   INTEGER NOT NULL,
	to_index     INTEGER NOT NULL,
	count        INTEGER NOT NULL,
	merkle_root  TEXT    NOT NULL,
	merkle_tree  TEXT    NOT NULL DEFAULT '',
	pruned_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_anchor_range ON anchors(from_index, to_index);
`

// Anchor represents a cryptographic summary of a pruned chain segment.
type Anchor struct {
	ID         int64
	FromIndex  uint64
	ToIndex    uint64
	Count      int
	MerkleRoot string
	MerkleTree [][]string // full tree, stored as JSON; level 0 = leaves, last level = root
	PrunedAt   time.Time
}

// anchorSchema is applied by chain.Open() immediately after schema.
// It must be defined in this package so chain.go can reference it.

// Prune removes receipts with chain_index <= upToIndex, replacing them with a
// Merkle-rooted anchor. upToIndex must be at least 10 less than the current tip
// to preserve a meaningful verification window.
//
// Returns the Anchor that was created.
func (c *Chain) Prune(upToIndex uint64) (*Anchor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if upToIndex >= c.lastIdx {
		return nil, fmt.Errorf("upToIndex (%d) must be less than chain tip (%d)", upToIndex, c.lastIdx)
	}
	if c.lastIdx-upToIndex < 10 {
		return nil, fmt.Errorf("must preserve at least 10 receipts before tip; tip=%d upTo=%d", c.lastIdx, upToIndex)
	}

	// Load all receipts in the range to be pruned, ordered ascending
	rows, err := c.db.Query(
		`SELECT raw_receipt FROM receipts WHERE chain_index <= ? ORDER BY chain_index ASC`,
		upToIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("querying receipts to prune: %w", err)
	}
	var raws [][]byte
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			_ = rows.Close()
			return nil, err
		}
		raws = append(raws, []byte(raw))
	}
	_ = rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, fmt.Errorf("no receipts found up to index %d", upToIndex)
	}

	// Compute Merkle root and full tree
	tree := buildMerkleTree(raws)
	root := tree[len(tree)-1][0] // root is the sole entry at the top level
	treeJSON, _ := json.Marshal(tree)

	// Determine the actual minimum index being pruned (not always 1 on subsequent prunes)
	var fromIndex uint64
	if err = c.db.QueryRow(
		`SELECT MIN(chain_index) FROM receipts WHERE chain_index <= ?`, upToIndex,
	).Scan(&fromIndex); err != nil || fromIndex == 0 {
		fromIndex = 1 // fallback: receipts start at 1
	}

	// Insert anchor
	now := time.Now().UnixNano()
	res, err := c.db.Exec(
		`INSERT INTO anchors (from_index, to_index, count, merkle_root, merkle_tree, pruned_at) VALUES (?,?,?,?,?,?)`,
		fromIndex, upToIndex, len(raws), root, string(treeJSON), now,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting anchor: %w", err)
	}
	anchorID, _ := res.LastInsertId()

	// Delete pruned receipts
	if _, err := c.db.Exec(`DELETE FROM receipts WHERE chain_index <= ?`, upToIndex); err != nil {
		return nil, fmt.Errorf("deleting pruned receipts: %w", err)
	}

	return &Anchor{
		ID:         anchorID,
		FromIndex:  fromIndex,
		ToIndex:    upToIndex,
		Count:      len(raws),
		MerkleRoot: root,
		MerkleTree: tree,
		PrunedAt:   time.Unix(0, now),
	}, nil
}

// ListAnchors returns all stored anchors, ordered by from_index ascending.
func (c *Chain) ListAnchors() ([]Anchor, error) {
	rows, err := c.db.Query(
		`SELECT id, from_index, to_index, count, merkle_root, merkle_tree, pruned_at FROM anchors ORDER BY from_index ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var anchors []Anchor
	for rows.Next() {
		var a Anchor
		var prunedNS int64
		var treeStr string
		if err := rows.Scan(&a.ID, &a.FromIndex, &a.ToIndex, &a.Count, &a.MerkleRoot, &treeStr, &prunedNS); err != nil {
			return nil, err
		}
		if treeStr != "" {
			_ = json.Unmarshal([]byte(treeStr), &a.MerkleTree)
		}
		a.PrunedAt = time.Unix(0, prunedNS)
		anchors = append(anchors, a)
	}
	return anchors, rows.Err()
}

// VerifyInclusion checks whether rawReceipt's SHA-256 hash appears as a leaf
// in the anchor's Merkle tree by walking the stored full tree.
//
// For single-receipt anchors, leaf hash must equal the root.
// For multi-receipt anchors, the full tree must be available in MerkleTree.
func VerifyInclusion(anchor *Anchor, rawReceipt []byte) error {
	if anchor.MerkleRoot == "" {
		return fmt.Errorf("anchor has no Merkle root")
	}
	if anchor.Count == 1 {
		h := sha256.Sum256(rawReceipt)
		leafHex := fmt.Sprintf("%x", h)
		if leafHex != anchor.MerkleRoot {
			return fmt.Errorf("receipt not found in anchor: leaf=%s root=%s", leafHex, anchor.MerkleRoot)
		}
		return nil
	}
	if len(anchor.MerkleTree) == 0 {
		return fmt.Errorf("anchor covers %d receipts but no full Merkle tree stored", anchor.Count)
	}

	// Find the leaf hash in level 0
	leafHash := fmt.Sprintf("%x", sha256.Sum256(rawReceipt))
	leafIdx := -1
	for i, h := range anchor.MerkleTree[0] {
		if h == leafHash {
			leafIdx = i
			break
		}
	}
	if leafIdx == -1 {
		return fmt.Errorf("receipt not found in anchor Merkle tree")
	}

	// Enforce maximum proof depth (DoS protection)
	const maxProofDepth = 64
	treeDepth := len(anchor.MerkleTree) - 1
	if treeDepth > maxProofDepth {
		return fmt.Errorf("merkle tree depth %d exceeds maximum %d", treeDepth, maxProofDepth)
	}

	// Walk up the tree recomputing parent hashes
	idx := leafIdx
	for level := 0; level < treeDepth; level++ {
		nodes := anchor.MerkleTree[level]
		siblingIdx := idx
		if idx%2 == 0 {
			siblingIdx = idx + 1
		} else {
			siblingIdx = idx - 1
		}
		// If sibling is out of bounds (odd-length level), duplicate self
		var siblingHash string
		if siblingIdx < 0 || siblingIdx >= len(nodes) {
			siblingHash = nodes[idx]
		} else {
			siblingHash = nodes[siblingIdx]
		}

		left := nodes[idx]
		right := siblingHash
		if idx%2 == 1 {
			left = siblingHash
			right = nodes[idx]
		}

		var combined [64]byte
		leftDec := hexDecode(left)
		if len(leftDec) != 32 {
			return fmt.Errorf("invalid left hash at level %d: %s", level, left)
		}
		rightDec := hexDecode(right)
		if len(rightDec) != 32 {
			return fmt.Errorf("invalid right hash at level %d: %s", level, right)
		}
		copy(combined[:32], leftDec)
		copy(combined[32:], rightDec)
		parentHash := fmt.Sprintf("%x", sha256.Sum256(combined[:]))

		// Verify against stored parent
		if parentHash != anchor.MerkleTree[level+1][idx/2] {
			return fmt.Errorf("merkle proof failed at level %d: computed=%s stored=%s", level, parentHash, anchor.MerkleTree[level+1][idx/2])
		}
		idx = idx / 2
	}

	return nil
}

func hexDecode(s string) []byte {
	if len(s)%2 != 0 {
		return nil
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		var hi, lo byte
		if s[i] >= 'a' && s[i] <= 'f' {
			hi = s[i] - 'a' + 10
		} else if s[i] >= 'A' && s[i] <= 'F' {
			hi = s[i] - 'A' + 10
		} else {
			hi = s[i] - '0'
		}
		if s[i+1] >= 'a' && s[i+1] <= 'f' {
			lo = s[i+1] - 'a' + 10
		} else if s[i+1] >= 'A' && s[i+1] <= 'F' {
			lo = s[i+1] - 'A' + 10
		} else {
			lo = s[i+1] - '0'
		}
		b[i/2] = hi<<4 | lo
	}
	return b
}

// buildMerkleTree computes the full SHA-256 Merkle tree from leaf data.
// Returns all levels: level 0 = leaves, last level = root (single entry).
// Odd-length levels duplicate the last element (standard Bitcoin Merkle convention).
func buildMerkleTree(leaves [][]byte) [][]string {
	if len(leaves) == 0 {
		root := fmt.Sprintf("%x", sha256.Sum256(nil))
		return [][]string{{root}}
	}
	tree := [][]string{}
	current := make([]string, len(leaves))
	for i, leaf := range leaves {
		current[i] = fmt.Sprintf("%x", sha256.Sum256(leaf))
	}
	tree = append(tree, current)
	for len(current) > 1 {
		var next []string
		for i := 0; i < len(current); i += 2 {
			left := current[i]
			right := left
			if i+1 < len(current) {
				right = current[i+1]
			}
			combined := hexDecode(left)
			combined = append(combined, hexDecode(right)...)
			next = append(next, fmt.Sprintf("%x", sha256.Sum256(combined)))
		}
		current = next
		tree = append(tree, current)
	}
	return tree
}

// merkleRoot is a convenience wrapper returning just the root.
func merkleRoot(leaves [][]byte) string {
	tree := buildMerkleTree(leaves)
	return tree[len(tree)-1][0]
}
