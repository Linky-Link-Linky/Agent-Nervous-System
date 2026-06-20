package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OAuth2Provider struct {
	tokenURL   string
	clientID   string
	secret     string
	scopes     []string
	httpClient *http.Client
}

type OAuth2ProviderOption func(*OAuth2Provider)

func WithOAuth2HTTPClient(client *http.Client) OAuth2ProviderOption {
	return func(p *OAuth2Provider) {
		p.httpClient = client
	}
}

func NewOAuth2Provider(tokenURL, clientID, secret string, scopes []string, opts ...OAuth2ProviderOption) *OAuth2Provider {
	p := &OAuth2Provider{
		tokenURL: tokenURL,
		clientID: clientID,
		secret:   secret,
		scopes:   scopes,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.httpClient == nil {
		p.httpClient = http.DefaultClient
	}
	return p
}

func (o *OAuth2Provider) Name() string {
	return "oauth2"
}

func (o *OAuth2Provider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	if o.tokenURL == "" {
		return nil, fmt.Errorf("oauth2: token URL is required")
	}
	if o.clientID == "" {
		return nil, fmt.Errorf("oauth2: client ID is required")
	}
	if o.secret == "" {
		return nil, fmt.Errorf("oauth2: client secret is required")
	}

	scope := strings.Join(o.scopes, " ")
	if scope == "" {
		scope = req.Scope.Resource
	}

	data := url.Values{}
	data.Set("client_id", o.clientID)
	data.Set("client_secret", o.secret)
	data.Set("grant_type", "client_credentials")
	data.Set("scope", scope)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("oauth2: failed to create token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("oauth2: token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth2: token endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("oauth2: failed to parse token response: %w", err)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("oauth2: token endpoint returned empty access token")
	}

	credID := generateRequestID()
	now := time.Now()

	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "oauth2",
		Type:         "bearer-token",
		Secret:       result.AccessToken,
		Metadata: map[string]string{
			"token_url":  o.tokenURL,
			"token_type": result.TokenType,
			"scope":      result.Scope,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
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
