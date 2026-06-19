package chain

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

func TestAppendNew(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 10; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		r, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() %d failed: %v", i, err)
		}
		if r.ChainIndex != uint64(i) {
			t.Errorf("Receipt %d has ChainIndex=%d, want %d", i, r.ChainIndex, i)
		}
	}

	count, err := c.Count()
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 10 {
		t.Errorf("Count() = %d, want 10", count)
	}
}

func TestAppendNewIndexValidation(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	payload := receipt.ActionPayload{Type: receipt.ActionCustom}

	// First append
	r1, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
		b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
		return b.PreAction(payload, "first", receipt.PolicyAllow, ""), nil
	}, signer)
	if err != nil {
		t.Fatalf("First AppendNew() failed: %v", err)
	}
	if r1.ChainIndex != 1 {
		t.Errorf("First receipt ChainIndex = %d, want 1", r1.ChainIndex)
	}
	hash1, _ := r1.ComputeHash()

	// Second append
	r2, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
		b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
		return b.PreAction(payload, "second", receipt.PolicyAllow, ""), nil
	}, signer)
	if err != nil {
		t.Fatalf("Second AppendNew() failed: %v", err)
	}
	if r2.ChainIndex != 2 {
		t.Errorf("Second receipt ChainIndex = %d, want 2", r2.ChainIndex)
	}
	if r2.PrevReceiptHash != hash1 {
		t.Errorf("Second receipt PrevReceiptHash = %q, want %q", r2.PrevReceiptHash, hash1)
	}

	// Verify chain
	result := c.VerifyChain(nil)
	if !result.Valid {
		t.Errorf("VerifyChain() = invalid: %s", result.Error)
	}
	if result.TotalChecked != 2 {
		t.Errorf("VerifyChain() checked %d receipts, want 2", result.TotalChecked)
	}
}

func TestVerifyChainClean(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 10; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	result := c.VerifyChain(nil)
	if !result.Valid {
		t.Errorf("VerifyChain() = invalid: %s", result.Error)
	}
	if result.TotalChecked != 10 {
		t.Errorf("VerifyChain() checked %d receipts, want 10", result.TotalChecked)
	}
}

func TestVerifyChainTampered(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 5; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	// Tamper with receipt at index 3
	_, err = c.db.Exec(`UPDATE receipts SET raw_receipt = replace(raw_receipt, '"test"', '"tampered"') WHERE chain_index = 3`)
	if err != nil {
		t.Fatalf("Tampering failed: %v", err)
	}

	result := c.VerifyChain(nil)
	if result.Valid {
		t.Error("VerifyChain() = valid after tampering, want invalid")
	}
	if result.FirstBrokenAt != 3 {
		t.Errorf("VerifyChain() FirstBrokenAt = %d, want 3", result.FirstBrokenAt)
	}
}

func TestGetStatsEmpty(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	stats, err := c.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}
	if stats.TotalReceipts != 0 {
		t.Errorf("TotalReceipts = %d, want 0", stats.TotalReceipts)
	}
	if stats.TotalAgents != 0 {
		t.Errorf("TotalAgents = %d, want 0", stats.TotalAgents)
	}
}

func TestGetStatsFilled(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 3; i++ {
		agentID := "ans_agent1"
		if i > 2 {
			agentID = "ans_agent2"
		}
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder(agentID, prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	stats, err := c.GetStats()
	if err != nil {
		t.Fatalf("GetStats() failed: %v", err)
	}
	if stats.TotalReceipts != 3 {
		t.Errorf("TotalReceipts = %d, want 3", stats.TotalReceipts)
	}
	if stats.TotalAgents != 2 {
		t.Errorf("TotalAgents = %d, want 2", stats.TotalAgents)
	}
}

func TestListFilter(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	agents := []string{"ans_agentA", "ans_agentB", "ans_agentA"}
	for _, agentID := range agents {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder(agentID, prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	receipts, err := c.List(QueryOptions{AgentID: "ans_agentA"})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(receipts) != 2 {
		t.Errorf("List(agentA) returned %d receipts, want 2", len(receipts))
	}
	for _, r := range receipts {
		if r.AgentID != "ans_agentA" {
			t.Errorf("List(agentA) returned receipt with AgentID=%s", r.AgentID)
		}
	}
}

func TestExportJSONLRoundTrip(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 5; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	var buf bytes.Buffer
	if err := c.ExportJSONL(&buf, QueryOptions{}); err != nil {
		t.Fatalf("ExportJSONL() failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 5 {
		t.Errorf("ExportJSONL() produced %d lines, want 5", len(lines))
	}
}

func TestExportCSV(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 3; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	var buf bytes.Buffer
	if err := c.ExportCSV(&buf, QueryOptions{}); err != nil {
		t.Fatalf("ExportCSV() failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 { // header + 3 rows
		t.Errorf("ExportCSV() produced %d lines, want 4", len(lines))
	}
}

func TestConcurrentAppendNew(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	var wg sync.WaitGroup
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				payload := receipt.ActionPayload{Type: receipt.ActionCustom}
				_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
					b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
					return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
				}, signer)
				if err != nil {
					t.Errorf("AppendNew() failed: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	count, err := c.Count()
	if err != nil {
		t.Fatalf("Count() failed: %v", err)
	}
	if count != 100 {
		t.Errorf("Count() = %d, want 100", count)
	}

	result := c.VerifyChain(nil)
	if !result.Valid {
		t.Errorf("VerifyChain() after concurrent writes: %s", result.Error)
	}
}

func TestExportPDF(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 5; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	var buf bytes.Buffer
	if err := c.ExportPDF(&buf); err != nil {
		t.Fatalf("ExportPDF() failed: %v", err)
	}

	pdf := buf.String()
	if !strings.HasPrefix(pdf, "%PDF-1.4") {
		t.Error("PDF does not start with %PDF-1.4")
	}
	if !strings.Contains(pdf, "%"+"%EOF") {
		t.Error("PDF does not contain EOF marker")
	}
	if len(pdf) < 1000 {
		t.Errorf("PDF length = %d, suspiciously short", len(pdf))
	}
}

func TestPruneBasic(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 20; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	anchor, err := c.Prune(10)
	if err != nil {
		t.Fatalf("Prune() failed: %v", err)
	}
	if anchor.Count != 10 {
		t.Errorf("Anchor.Count = %d, want 10", anchor.Count)
	}
	if anchor.MerkleRoot == "" {
		t.Error("Anchor.MerkleRoot is empty")
	}

	count, _ := c.Count()
	if count != 10 {
		t.Errorf("Count() after prune = %d, want 10", count)
	}

	anchors, err := c.ListAnchors()
	if err != nil {
		t.Fatalf("ListAnchors() failed: %v", err)
	}
	if len(anchors) != 1 {
		t.Errorf("ListAnchors() returned %d anchors, want 1", len(anchors))
	}

	result := c.VerifyChain(nil)
	if !result.Valid {
		t.Errorf("VerifyChain() after prune: %s", result.Error)
	}
}

func TestPruneRejectsSmallWindow(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	for i := 1; i <= 15; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_test", prevHash, nextIdx)
			return b.PreAction(payload, "test", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	_, err = c.Prune(12) // only 3 receipts remain
	if err == nil {
		t.Error("Prune() succeeded with small window, want error")
	}
}

func TestMerkleRoot(t *testing.T) {
	leaves1 := [][]byte{[]byte("a")}
	root1 := merkleRoot(leaves1)
	if root1 == "" {
		t.Error("merkleRoot([1 leaf]) is empty")
	}

	leaves2 := [][]byte{[]byte("a"), []byte("b")}
	root2 := merkleRoot(leaves2)
	if root2 == "" {
		t.Error("merkleRoot([2 leaves]) is empty")
	}
	if root1 == root2 {
		t.Error("merkleRoot([1 leaf]) == merkleRoot([2 leaves])")
	}

	// Deterministic
	root2Again := merkleRoot(leaves2)
	if root2 != root2Again {
		t.Error("merkleRoot() not deterministic")
	}

	// Changed leaf
	leaves2[0] = []byte("c")
	root3 := merkleRoot(leaves2)
	if root2 == root3 {
		t.Error("merkleRoot() same after changing leaf")
	}
}

func TestMergeChainsTwoAgents(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signer := receipt.NewSigner(priv)

	// Agent A: 3 receipts
	for i := 0; i < 3; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_agentA", prevHash, nextIdx)
			return b.PreAction(payload, "testA", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	// Agent B: 3 receipts
	for i := 0; i < 3; i++ {
		payload := receipt.ActionPayload{Type: receipt.ActionCustom}
		_, err := c.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
			b := receipt.NewBuilder("ans_agentB", prevHash, nextIdx)
			return b.PreAction(payload, "testB", receipt.PolicyAllow, ""), nil
		}, signer)
		if err != nil {
			t.Fatalf("AppendNew() failed: %v", err)
		}
	}

	merged, err := c.MergeChains([]string{"ans_agentA", "ans_agentB"})
	if err != nil {
		t.Fatalf("MergeChains() failed: %v", err)
	}
	if len(merged) != 6 {
		t.Errorf("MergeChains() returned %d receipts, want 6", len(merged))
	}
}

func TestMergeChainsEmpty(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	c, err := Open(chainPath)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	defer c.Close()

	merged, err := c.MergeChains([]string{"ans_noexist"})
	if err != nil {
		t.Fatalf("MergeChains() failed: %v", err)
	}
	if merged != nil && len(merged) != 0 {
		t.Errorf("MergeChains() returned non-empty result, want nil or empty")
	}
}
