package broker

import (
	"context"
	"fmt"
	"os"
	"time"
)

// EnvProvider reads credentials from environment variables.
type EnvProvider struct{}

func NewEnvProvider() *EnvProvider { return &EnvProvider{} }
func (p *EnvProvider) Name() string { return "env" }

func (p *EnvProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	ak := os.Getenv("AWS_ACCESS_KEY_ID")
	sk := os.Getenv("AWS_SECRET_ACCESS_KEY")
	st := os.Getenv("AWS_SESSION_TOKEN")
	if ak == "" {
		return nil, fmt.Errorf("env provider: AWS_ACCESS_KEY_ID not set")
	}
	now := time.Now()
	return &Credential{
		CredentialID:   generateRequestID(),
		AgentID:        req.AgentID,
		ProviderName:   "env",
		Type:           "access-key",
		Secret:         sk,
		Metadata:       map[string]string{"access_key": ak, "session_token": st},
		Scope:          req.Scope,
		IssuedAt:       now,
		ExpiresAt:      now.Add(time.Duration(req.TTLSeconds) * time.Second),
		RequestID:      req.RequestID,
		PreReceiptID:   req.PreReceiptID,
	}, nil
}

func (p *EnvProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return fmt.Errorf("env provider: revocation not implemented — environment variables cannot be revoked")
}

func (p *EnvProvider) ValidateScope(scope Scope) error {
	return nil
}
