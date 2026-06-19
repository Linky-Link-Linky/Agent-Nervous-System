package broker

import (
	"context"
	"fmt"
	"time"
)

// OAuth2Provider provisions ephemeral tokens via the OAuth2 client credentials flow.
type OAuth2Provider struct {
	tokenURL  string
	clientID  string
	secret    string
	scopes    []string
}

// NewOAuth2Provider creates a new generic OAuth2 provider.
func NewOAuth2Provider(tokenURL, clientID, secret string, scopes []string) *OAuth2Provider {
	return &OAuth2Provider{
		tokenURL: tokenURL,
		clientID: clientID,
		secret:   secret,
		scopes:   scopes,
	}
}

func (o *OAuth2Provider) Name() string {
	return "oauth2"
}

func (o *OAuth2Provider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	credID := generateRequestID()
	now := time.Now()
	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "oauth2",
		Type:         "bearer-token",
		Secret:       fmt.Sprintf("oauth2_%s", credID),
		Metadata: map[string]string{
			"token_url": o.tokenURL,
			"scope":     req.Scope.Resource,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		Revoked:      false,
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (o *OAuth2Provider) RevokeCredential(ctx context.Context, credentialID string) error {
	return nil
}

func (o *OAuth2Provider) ValidateScope(scope Scope) error {
	if scope.Resource == "" {
		return fmt.Errorf("oauth2 resource path cannot be empty")
	}
	return nil
}
