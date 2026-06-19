package pretty

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

func TestPrintChainColor(t *testing.T) {
	b := receipt.NewBuilder("ans_test", receipt.GenesisHash, 1)
	payload := receipt.ActionPayload{Type: receipt.ActionFileWrite}
	r1 := b.PreAction(payload, "write file", receipt.PolicyAllow, "")
	r1.SetReceiptID()

	b2 := receipt.NewBuilder("ans_test", "hash1", 2)
	r2 := b2.PostAction(r1.ReceiptID, receipt.ActionFileWrite, payload.HashHex(), "write file", receipt.OutcomeSuccess, "wrote 42 bytes", 10)
	r2.SetReceiptID()

	receipts := []*receipt.Receipt{r1, r2}

	var buf bytes.Buffer
	PrintChain(&buf, receipts, false)
	output := buf.String()

	if !strings.Contains(output, r1.ReceiptID[:8]) {
		t.Errorf("Output missing receipt ID %s", r1.ReceiptID[:8])
	}
	if !strings.Contains(output, "file.write") {
		t.Error("Output missing action type")
	}
}

func TestPrintChainNoColor(t *testing.T) {
	b := receipt.NewBuilder("ans_test", receipt.GenesisHash, 1)
	payload := receipt.ActionPayload{Type: receipt.ActionCustom}
	r := b.PreAction(payload, "test", receipt.PolicyAllow, "")
	r.SetReceiptID()

	var buf bytes.Buffer
	PrintChain(&buf, []*receipt.Receipt{r}, true)
	output := buf.String()

	if strings.Contains(output, "\033[") {
		t.Error("No-color output contains ANSI escape sequences")
	}
	if !strings.Contains(output, "ANS Receipt Chain") {
		t.Error("No-color output missing header")
	}
}

func TestPrintChainOrphan(t *testing.T) {
	b := receipt.NewBuilder("ans_test", receipt.GenesisHash, 1)
	payload := receipt.ActionPayload{Type: receipt.ActionCustom}
	r := b.PreAction(payload, "orphan", receipt.PolicyAllow, "")
	r.SetReceiptID()

	var buf bytes.Buffer
	PrintChain(&buf, []*receipt.Receipt{r}, false)
	output := buf.String()

	if !strings.Contains(output, "(pending)") {
		t.Error("Orphan receipt missing (pending) indicator")
	}
}

func TestPrintChainEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintChain(&buf, []*receipt.Receipt{}, false)
	// Should not panic
	if buf.Len() < 10 {
		t.Error("Empty chain output too short")
	}
}

func TestPrintStatusColor(t *testing.T) {
	status := map[string]interface{}{
		"uptime":         "1h23m",
		"chain_length":   uint64(100),
		"total_receipts": int64(50),
		"total_agents":   int64(3),
		"db_size_bytes":  int64(1024000),
		"started_at":     "2025-01-01T00:00:00Z",
	}

	var buf bytes.Buffer
	PrintStatus(&buf, status, false)
	output := buf.String()

	if !strings.Contains(output, "uptime") {
		t.Error("Status missing uptime")
	}
	if !strings.Contains(output, "chain") {
		t.Error("Status missing chain info")
	}
}

func TestPrintVerifyValid(t *testing.T) {
	resp := map[string]interface{}{
		"valid":      true,
		"receipt_id": "abcd1234",
		"agent_id":   "ans_test123",
		"agent_name": "TestAgent",
	}

	var buf bytes.Buffer
	PrintVerifyResult(&buf, resp, false)
	output := buf.String()

	if !strings.Contains(output, "verified") {
		t.Error("Verify output missing 'verified'")
	}
	if !strings.Contains(output, "abcd1234") {
		t.Error("Verify output missing receipt ID")
	}
}

func TestPrintVerifyInvalid(t *testing.T) {
	resp := map[string]interface{}{
		"valid":      false,
		"receipt_id": "bad1234",
		"error":      "signature mismatch",
	}

	var buf bytes.Buffer
	PrintVerifyResult(&buf, resp, false)
	output := buf.String()

	if !strings.Contains(output, "INVALID") {
		t.Error("Verify output missing 'INVALID'")
	}
	if !strings.Contains(output, "signature mismatch") {
		t.Error("Verify output missing error message")
	}
}

func TestSafeID(t *testing.T) {
	long := "0123456789abcdef"
	if safeID(long) != "01234567" {
		t.Errorf("safeID(%q) = %q, want %q", long, safeID(long), "01234567")
	}

	short := "abc"
	if safeID(short) != "abc" {
		t.Errorf("safeID(%q) = %q, want %q", short, safeID(short), "abc")
	}
}
