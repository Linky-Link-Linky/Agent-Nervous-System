package chain

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ans-project/ans/internal/receipt"
)

// TestConcurrentOrdering proves chain safety under concurrent appends:
//
//   Safety property: After N concurrent writers each append K receipts,
//   the chain contains exactly N*K receipts, chain_index values are
//   1..N*K with no gaps, and every adjacent pair is hash-linked.
//
//   This tests the internal mutex in Chain.AppendNew: without it,
//   concurrent goroutines would race on nextIdx and produce interleaved
//   chain_index values that violate the hash chain.
func TestConcurrentOrdering(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "concurrent_chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()

	const numWriters = 10
	const receiptsPerWriter = 25
	const totalReceipts = numWriters * receiptsPerWriter

	var wg sync.WaitGroup
	errs := make(chan error, numWriters*receiptsPerWriter)

	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			_, priv, _ := ed25519.GenerateKey(rand.Reader)
			signer := receipt.NewSigner(priv)

			for i := 0; i < receiptsPerWriter; i++ {
				payload := receipt.ActionPayload{Type: receipt.ActionCustom}
				_, err := c.AppendNew(func(prev string, nextIdx uint64) (*receipt.Receipt, error) {
					b := receipt.NewBuilder("ans_conc", prev, nextIdx)
					return b.PreAction(payload, "concurrent test", receipt.PolicyAllow, ""), nil
				}, signer)
				errs <- err
			}
		}(w)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("AppendNew failed: %v", err)
		}
	}

	count, err := c.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != int64(totalReceipts) {
		t.Fatalf("Count = %d, want %d", count, totalReceipts)
	}

	// Verify chain integrity: each receipt's chain_index must be monotonic
	// and the hash chain must be fully linked.
	var prevHash string
	for idx := uint64(1); idx <= uint64(count); idx++ {
		r, err := c.GetByIndex(idx)
		if err != nil {
			t.Fatalf("GetByIndex(%d): %v", idx, err)
		}
		if r.ChainIndex != idx {
			t.Fatalf("ChainIndex = %d at position %d", r.ChainIndex, idx)
		}
		if idx == 1 && r.PrevReceiptHash != receipt.GenesisHash {
			t.Fatalf("genesis prev_hash = %q", r.PrevReceiptHash)
		}
		if idx > 1 && r.PrevReceiptHash != prevHash {
			t.Fatalf("receipt %d prev_hash links broken: %q != %q", idx, r.PrevReceiptHash, prevHash)
		}
		h, err := r.ComputeHash()
		if err != nil {
			t.Fatalf("ComputeHash(%d): %v", idx, err)
		}
		prevHash = h
	}

	t.Logf("Concurrent safety verified: %d writers x %d receipts = %d total, hash chain intact",
		numWriters, receiptsPerWriter, totalReceipts)
}
