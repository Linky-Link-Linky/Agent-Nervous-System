package broker

import (
	"context"
	"fmt"
	"time"
)

// DevProvider uses pre-configured credentials for development.
type DevProvider struct{}

func NewDevProvider() *DevProvider { return &DevProvider{} }
func (p *DevProvider) Name() string { return "dev" }

func (p *DevProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	now := time.Now()
	return &Credential{
		CredentialID:   generateRequestID(),
		AgentID:        req.AgentID,
		ProviderName:   "dev",
		Type:           "dev",
		Secret:         "dev-secret",
		Metadata:       map[string]string{"access_key": "dev-access-key"},
		Scope:          req.Scope,
		IssuedAt:       now,
		ExpiresAt:      now.Add(time.Duration(req.TTLSeconds) * time.Second),
		RequestID:      req.RequestID,
		PreReceiptID:   req.PreReceiptID,
	}, nil
}

func (p *DevProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return fmt.Errorf("dev provider: revocation not implemented")
}

func (p *DevProvider) ValidateScope(scope Scope) error {
	return nil
}
