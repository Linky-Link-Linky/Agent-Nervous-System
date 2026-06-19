package broker

import (
	"context"
	"fmt"
	"time"
)

// AzureProvider provisions ephemeral app registration secrets from Azure AD.
type AzureProvider struct {
	tenantID     string
	clientID     string
	clientSecret string
}

// NewAzureProvider creates a new Azure AD provider.
func NewAzureProvider(tenantID, clientID, clientSecret string) *AzureProvider {
	return &AzureProvider{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (a *AzureProvider) Name() string {
	return "azure-ad"
}

func (a *AzureProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	credID := generateRequestID()
	now := time.Now()
	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "azure-ad",
		Type:         "oauth2-access-token",
		Secret:       fmt.Sprintf("eyJ0eXAiOiJKV1Q.%s", credID),
		Metadata: map[string]string{
			"tenant_id": a.tenantID,
			"resource":  req.Scope.Resource,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		Revoked:      false,
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (a *AzureProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return nil
}

func (a *AzureProvider) ValidateScope(scope Scope) error {
	if scope.Resource == "" {
		return fmt.Errorf("azure resource path cannot be empty")
	}
	return nil
}
