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

type AzureProvider struct {
	tenantID     string
	clientID     string
	clientSecret string
	tokenURL     string
	httpClient   *http.Client
}

type AzureProviderOption func(*AzureProvider)

func WithAzureHTTPClient(client *http.Client) AzureProviderOption {
	return func(p *AzureProvider) {
		p.httpClient = client
	}
}

func WithAzureTokenURL(url string) AzureProviderOption {
	return func(p *AzureProvider) {
		p.tokenURL = url
	}
}

func NewAzureProvider(tenantID, clientID, clientSecret string, opts ...AzureProviderOption) *AzureProvider {
	p := &AzureProvider{
		tenantID:     tenantID,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.httpClient == nil {
		p.httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSHandshakeTimeout: 5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
			},
		}
	}
	return p
}

func (a *AzureProvider) Name() string {
	return "azure-ad"
}

func (a *AzureProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	if a.tenantID == "" {
		return nil, fmt.Errorf("azure: tenantID is required")
	}
	if a.clientID == "" {
		return nil, fmt.Errorf("azure: clientID is required")
	}
	if a.clientSecret == "" {
		return nil, fmt.Errorf("azure: clientSecret is required")
	}

	scope := a.resolveScope(req.Scope.Resource)
	tokenURL := a.tokenURL
	if tokenURL == "" {
		tokenURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", a.tenantID)
	}

	data := url.Values{}
	data.Set("client_id", a.clientID)
	data.Set("client_secret", a.clientSecret)
	data.Set("scope", scope)
	data.Set("grant_type", "client_credentials")

	httpReq, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("azure: failed to create token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("azure: token endpoint request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure: token endpoint returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("azure: failed to parse token response: %w", err)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("azure: token endpoint returned empty access token")
	}

	credID := generateRequestID()
	now := time.Now()

	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "azure-ad",
		Type:         "oauth2-access-token",
		Secret:       result.AccessToken,
		Metadata: map[string]string{
			"tenant_id":    a.tenantID,
			"client_id":    a.clientID,
			"resource":     req.Scope.Resource,
			"token_type":   result.TokenType,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (a *AzureProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return fmt.Errorf("azure provider: revocation not implemented — use short TTLs")
}

func (a *AzureProvider) ValidateScope(scope Scope) error {
	if scope.Resource == "" {
		return fmt.Errorf("azure resource path cannot be empty")
	}
	if !strings.HasPrefix(scope.Resource, "https://") &&
		!strings.HasPrefix(scope.Resource, "api://") {
		return fmt.Errorf("invalid Azure resource format: %s", scope.Resource)
	}
	for _, perm := range scope.Permissions {
		switch perm {
		case "read", "write", "delete", "list", "get", "post", "put", "patch", "*":
		default:
			return fmt.Errorf("invalid Azure permission: %s", perm)
		}
	}
	return nil
}

func (a *AzureProvider) resolveScope(resource string) string {
	if strings.HasSuffix(resource, "/.default") {
		return resource
	}
	if strings.HasPrefix(resource, "https://") || strings.HasPrefix(resource, "api://") {
		if resource[len(resource)-1] == '/' {
			return resource + ".default"
		}
		return resource + "/.default"
	}
	return resource + "/.default"
}
