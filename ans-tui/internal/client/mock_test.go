package client

import (
	"testing"
)

func TestMockClient_DaemonStatus(t *testing.T) {
	m := NewMockClient()
	s, err := m.DaemonStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil status")
	}
	if !s.Running {
		t.Fatal("expected Running=true")
	}
	if s.ChainLength <= 0 {
		t.Fatal("expected positive ChainLength")
	}
}

func TestMockClient_ListReceipts(t *testing.T) {
	m := NewMockClient()
	r1, err := m.ListReceipts(10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r1) != 10 {
		t.Fatalf("expected 10 receipts, got %d", len(r1))
	}

	r2, err := m.ListReceipts(10, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r2) != 10 {
		t.Fatalf("expected 10 receipts, got %d", len(r2))
	}
}

func TestMockClient_ListTokens(t *testing.T) {
	m := NewMockClient()
	tokens, err := m.ListTokens()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) == 0 {
		t.Fatal("expected at least one token")
	}
	for _, tok := range tokens {
		if tok.TTLSeconds() <= 0 {
			t.Fatalf("expected positive TTL for token %s, got %d", tok.ID, tok.TTLSeconds())
		}
	}
}

func TestMockClient_TokenRevoke(t *testing.T) {
	m := NewMockClient()
	tokens, _ := m.ListTokens()
	if len(tokens) == 0 {
		t.Skip("no tokens to revoke")
	}
	id := tokens[0].ID

	if err := m.TokenRevoke(id); err != nil {
		t.Fatalf("revoke error: %v", err)
	}

	after, _ := m.ListTokens()
	for _, tok := range after {
		if tok.ID == id {
			t.Fatal("revoked token still appears in list")
		}
	}
}

func TestMockClient_ListPolicies(t *testing.T) {
	m := NewMockClient()
	policies, err := m.ListPolicies()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(policies) == 0 {
		t.Fatal("expected at least one policy")
	}
}
