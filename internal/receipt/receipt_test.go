package receipt

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func TestPreActionReceipt(t *testing.T) {
	b := NewBuilder("ans_test123", GenesisHash, 1)
	payload := ActionPayload{Type: ActionFileWrite, Target: "/tmp/test.txt"}
	r := b.PreAction(payload, "write test file", PolicyAllow, "test context")

	if r.Phase != PhasePre {
		t.Errorf("Phase = %v, want %v", r.Phase, PhasePre)
	}
	if r.AgentID != "ans_test123" {
		t.Errorf("AgentID = %q, want %q", r.AgentID, "ans_test123")
	}
	if r.ActionType != ActionFileWrite {
		t.Errorf("ActionType = %v, want %v", r.ActionType, ActionFileWrite)
	}
	if r.PayloadHash == "" {
		t.Error("PayloadHash is empty")
	}
	if r.PrevReceiptHash != GenesisHash {
		t.Errorf("PrevReceiptHash = %q, want %q", r.PrevReceiptHash, GenesisHash)
	}
	if r.ChainIndex != 1 {
		t.Errorf("ChainIndex = %d, want 1", r.ChainIndex)
	}
}

func TestPostActionReceipt(t *testing.T) {
	b := NewBuilder("ans_test456", "prev123", 2)
	r := b.PostAction("pre_abc", ActionFileWrite, "hash123", "summary", OutcomeSuccess, "wrote 42 bytes", 15)

	if r.Phase != PhasePost {
		t.Errorf("Phase = %v, want %v", r.Phase, PhasePost)
	}
	if r.PreReceiptID != "pre_abc" {
		t.Errorf("PreReceiptID = %q, want %q", r.PreReceiptID, "pre_abc")
	}
	if r.Outcome != OutcomeSuccess {
		t.Errorf("Outcome = %v, want %v", r.Outcome, OutcomeSuccess)
	}
	if r.DurationMS != 15 {
		t.Errorf("DurationMS = %d, want 15", r.DurationMS)
	}
}

func TestSetReceiptIDDeterministic(t *testing.T) {
	b := NewBuilder("ans_det", GenesisHash, 1)
	payload := ActionPayload{Type: ActionCustom}
	r := b.PreAction(payload, "test", PolicyAllow, "")

	if err := r.SetReceiptID(); err != nil {
		t.Fatalf("SetReceiptID() failed: %v", err)
	}
	id1 := r.ReceiptID

	if err := r.SetReceiptID(); err != nil {
		t.Fatalf("Second SetReceiptID() failed: %v", err)
	}
	id2 := r.ReceiptID

	if id1 != id2 {
		t.Errorf("SetReceiptID() not deterministic: %q != %q", id1, id2)
	}
}

func TestSignAndVerify(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() failed: %v", err)
	}

	b := NewBuilder("ans_sign", GenesisHash, 1)
	payload := ActionPayload{Type: ActionCustom}
	r := b.PreAction(payload, "test", PolicyAllow, "")

	signer := NewSigner(priv)
	if err := signer.Sign(r); err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}

	if r.Signature == "" {
		t.Fatal("Signature is empty after Sign()")
	}

	if err := Verify(r, pub); err != nil {
		t.Errorf("Verify() failed: %v", err)
	}

	// Tamper with AgentID
	r.AgentID = "tampered"
	if err := Verify(r, pub); err == nil {
		t.Error("Verify() succeeded after tampering, want error")
	}
}

func TestSignableBytesStable(t *testing.T) {
	b := NewBuilder("ans_stable", GenesisHash, 1)
	payload := ActionPayload{Type: ActionCustom}
	r := b.PreAction(payload, "test", PolicyAllow, "")

	bytes1, err := r.SignableBytes()
	if err != nil {
		t.Fatalf("SignableBytes() failed: %v", err)
	}

	bytes2, err := r.SignableBytes()
	if err != nil {
		t.Fatalf("Second SignableBytes() failed: %v", err)
	}

	if string(bytes1) != string(bytes2) {
		t.Error("SignableBytes() not stable across calls")
	}
}

func TestPayloadHashHex(t *testing.T) {
	p1 := ActionPayload{Type: ActionFileWrite, Target: "/tmp/a.txt"}
	p2 := ActionPayload{Type: ActionFileWrite, Target: "/tmp/a.txt"}
	p3 := ActionPayload{Type: ActionFileWrite, Target: "/tmp/b.txt"}

	h1 := p1.HashHex()
	h2 := p2.HashHex()
	h3 := p3.HashHex()

	if h1 != h2 {
		t.Errorf("Same payload produced different hashes: %q != %q", h1, h2)
	}
	if h1 == h3 {
		t.Error("Different payloads produced same hash")
	}
	if len(h1) != 64 {
		t.Errorf("Hash length = %d, want 64 (256 bits hex)", len(h1))
	}
}

func TestTruncate(t *testing.T) {
	long := strings.Repeat("a", 100)
	result := truncate(long, 80)

	if len(result) != 80 {
		t.Errorf("truncate() length = %d, want 80", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("truncate() result = %q, want suffix '...'", result)
	}

	short := "short"
	result = truncate(short, 80)
	if result != short {
		t.Errorf("truncate() on short string = %q, want %q", result, short)
	}
}
