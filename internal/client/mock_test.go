package client

import (
    "testing"
)

func TestMock_DaemonStatus(t *testing.T) {
    m := NewMock()
    s, err := m.DaemonStatus()
    if err != nil { t.Fatal(err) }
    if s == nil { t.Fatal("expected non-nil status") }
    if !s.Running { t.Fatal("expected Running == true") }
}

func TestMock_ListReceipts_Growing(t *testing.T) {
	m := NewMock()
	r1, _ := m.ListReceipts(0, "") // 0 = no limit
	r2, _ := m.ListReceipts(0, "")
	if len(r2) <= len(r1) { t.Fatalf("second call should return more receipts: %d <= %d", len(r2), len(r1)) }
}

func TestMock_TokenRevoke(t *testing.T) {
    m := NewMock()
    tokens, _ := m.ListTokens()
    if len(tokens) == 0 { t.Fatal("expected tokens") }
    id := tokens[0].ID
	m.TokenRevoke(id)
	after, _ := m.ListTokens()
	for _, tok := range after {
		if tok.ID == id { t.Fatal("revoked token still present") }
	}
}

func TestMock_PolicyToggle(t *testing.T) {
    m := NewMock()
    policies, _ := m.ListPolicies()
    if len(policies) == 0 { t.Fatal("expected policies") }
    p := policies[0]
    orig := p.Enabled
	m.PolicyToggle(p.ID, !orig)
	after2, _ := m.ListPolicies()
	for _, a := range after2 {
		if a.ID == p.ID && a.Enabled == orig { t.Fatal("policy toggle had no effect") }
	}
}
