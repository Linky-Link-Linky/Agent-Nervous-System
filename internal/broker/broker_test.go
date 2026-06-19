package broker

import (
	"context"
	"testing"
	"time"
)

func TestDevProvider(t *testing.T) {
	p := NewDevProvider()
	if p.Name() != "dev" {
		t.Fatalf("expected name 'dev', got %q", p.Name())
	}
	req := &ProvisionRequest{
		AgentID:    "agent1",
		ActionType: "s3:GetObject",
		Scope:      Scope{Resource: "s3://bucket/file", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if cred.AgentID != "agent1" {
		t.Fatalf("expected agent1, got %s", cred.AgentID)
	}
	if cred.Scope.Resource != "s3://bucket/file" {
		t.Fatalf("expected s3://bucket/file, got %s", cred.Scope.Resource)
	}
	if cred.ExpiresAt.Before(time.Now()) {
		t.Fatal("expected future expiry")
	}
	ttl := cred.ExpiresAt.Sub(cred.IssuedAt)
	if ttl < 55*time.Second || ttl > 65*time.Second {
		t.Fatalf("expected ~60s TTL, got %v", ttl)
	}
	if err := p.RevokeCredential(context.Background(), cred.CredentialID); err != nil {
		t.Fatal(err)
	}
	if err := p.ValidateScope(req.Scope); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerProvision(t *testing.T) {
	b := NewBroker(DiscardLogger{})
	if err := b.RegisterProvider(NewDevProvider()); err != nil {
		t.Fatal(err)
	}
	req := &ProvisionRequest{
		AgentID:    "agent1",
		ActionType: "read",
		Scope:      Scope{Resource: "s3://bucket/file", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := b.Provision(context.Background(), "dev", req)
	if err != nil {
		t.Fatal(err)
	}
	if cred.CredentialID == "" {
		t.Fatal("expected credential ID")
	}
	active := b.ListActive()
	if len(active) != 1 {
		t.Fatalf("expected 1 active, got %d", len(active))
	}
	if err := b.Revoke(context.Background(), cred.CredentialID); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerEnforcesTTL(t *testing.T) {
	b := NewBroker(DiscardLogger{})
	b.RegisterProvider(NewDevProvider())
	req := &ProvisionRequest{
		AgentID:    "agent1",
		ActionType: "read",
		Scope:      Scope{Resource: "db://prod", Permissions: []string{"read"}},
		TTLSeconds: 120, // over max of 60
	}
	cred, err := b.Provision(context.Background(), "dev", req)
	if err != nil {
		t.Fatal(err)
	}
	ttl := cred.ExpiresAt.Sub(cred.IssuedAt)
	if ttl > 61*time.Second {
		t.Fatalf("expected TTL capped at ~60s, got %v", ttl)
	}
}
