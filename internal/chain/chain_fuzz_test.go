package chain

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"

	"github.com/ans-project/ans/internal/receipt"
)

// FuzzHashChain ensures that for any sequence of valid receipts,
// the hash chain invariants hold:
//   - prev_receipt_hash of receipt[i] == hash(receipt[i-1])
//   - receipt[0].prev_receipt_hash == GenesisHash
//   - chain_index sequence is 1, 2, 3, ... (no gaps)
func FuzzHashChain(f *testing.F) {
	f.Add(uint64(1))
	f.Add(uint64(5))
	f.Add(uint64(10))
	f.Add(uint64(50))

	f.Fuzz(func(t *testing.T, n uint64) {
		if n == 0 || n > 100 {
			return
		}
		chainPath := filepath.Join(t.TempDir(), "fuzz_chain.db")
		c, err := Open(chainPath)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer c.Close()

		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		signer := receipt.NewSigner(priv)

		var prevHash string
		for i := uint64(1); i <= n; i++ {
			payload := receipt.ActionPayload{Type: receipt.ActionCustom}
			r, err := c.AppendNew(func(prev string, nextIdx uint64) (*receipt.Receipt, error) {
				b := receipt.NewBuilder("ans_fuzz", prev, nextIdx)
				return b.PreAction(payload, "fuzz test", receipt.PolicyAllow, ""), nil
			}, signer)
			if err != nil {
				t.Fatalf("AppendNew %d: %v", i, err)
			}

			// Invariant: chain_index must match position
			if r.ChainIndex != i {
				t.Errorf("receipt %d has ChainIndex=%d", i, r.ChainIndex)
			}

			// Invariant: prev_receipt_hash must link
			if i == 1 {
				if r.PrevReceiptHash != receipt.GenesisHash {
					t.Errorf("genesis prev_hash=%q, want %q", r.PrevReceiptHash, receipt.GenesisHash)
				}
			} else {
				if r.PrevReceiptHash != prevHash {
					t.Errorf("receipt %d prev_hash=%q, want %q", i, r.PrevReceiptHash, prevHash)
				}
			}

			h, err := r.ComputeHash()
			if err != nil {
				t.Fatalf("ComputeHash %d: %v", i, err)
			}
			prevHash = h
		}

		// Verify the whole chain
		result := c.VerifyChain(nil)
		if !result.Valid {
			t.Errorf("VerifyChain: %s", result.Error)
		}
	})
}
