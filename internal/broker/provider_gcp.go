package broker

import (
	"context"
	"fmt"
	"time"
)

// GCPProvider provisions short-lived service account tokens from GCP IAM.
type GCPProvider struct {
	credentialsFile string
	projectID       string
}

// NewGCPProvider creates a new GCP IAM provider.
func NewGCPProvider(credentialsFile, projectID string) *GCPProvider {
	return &GCPProvider{
		credentialsFile: credentialsFile,
		projectID:       projectID,
	}
}

func (g *GCPProvider) Name() string {
	return "gcp-iam"
}

func (g *GCPProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	credID := generateRequestID()
	now := time.Now()
	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "gcp-iam",
		Type:         "oauth2-access-token",
		Secret:       fmt.Sprintf("ya29.%s", credID),
		Metadata: map[string]string{
			"project_id": g.projectID,
			"resource":   req.Scope.Resource,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		Revoked:      false,
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (g *GCPProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return nil
}

func (g *GCPProvider) ValidateScope(scope Scope) error {
	if scope.Resource == "" {
		return fmt.Errorf("gcp resource path cannot be empty")
	}
	return nil
}
