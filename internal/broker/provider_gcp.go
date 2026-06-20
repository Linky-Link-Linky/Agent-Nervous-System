package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GCPProvider struct {
	credentialsFile string
	projectID       string
	httpClient      *http.Client
	tokenSource     oauth2.TokenSource
}

type GCPProviderOption func(*GCPProvider)

func WithGCPHTTPClient(client *http.Client) GCPProviderOption {
	return func(p *GCPProvider) {
		p.httpClient = client
	}
}

func WithGCPTokenSource(ts oauth2.TokenSource) GCPProviderOption {
	return func(p *GCPProvider) {
		p.tokenSource = ts
	}
}

func NewGCPProvider(credentialsFile, projectID string, opts ...GCPProviderOption) *GCPProvider {
	p := &GCPProvider{
		credentialsFile: credentialsFile,
		projectID:       projectID,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.httpClient == nil {
		p.httpClient = http.DefaultClient
	}
	return p
}

func (g *GCPProvider) Name() string {
	return "gcp-iam"
}

func (g *GCPProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	ts, err := g.getTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: failed to get token source: %w", err)
	}

	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("gcp: failed to get access token for IAM API call: %w", err)
	}

	serviceAccount := g.resolveServiceAccount(req.Scope.Resource)
	lifetime := fmt.Sprintf("%ds", req.TTLSeconds)
	if req.TTLSeconds > 3600 {
		lifetime = "3600s"
	}

	iamURL := fmt.Sprintf("https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken", serviceAccount)
	body := map[string]interface{}{
		"scope":   []string{"https://www.googleapis.com/auth/cloud-platform"},
		"lifetime": lifetime,
	}
	bodyJSON, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", iamURL, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("gcp: failed to create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gcp: IAM API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gcp: IAM API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"accessToken"`
		ExpireTime  string `json:"expireTime"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("gcp: failed to parse IAM API response: %w", err)
	}

	credID := generateRequestID()
	now := time.Now()

	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "gcp-iam",
		Type:         "oauth2-access-token",
		Secret:       result.AccessToken,
		Metadata: map[string]string{
			"project_id":       g.projectID,
			"service_account":  serviceAccount,
			"resource":         req.Scope.Resource,
			"expire_time":      result.ExpireTime,
		},
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
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
	if !strings.HasPrefix(scope.Resource, "gs://") &&
		!strings.HasPrefix(scope.Resource, "projects/") &&
		!strings.HasPrefix(scope.Resource, "//") {
		return fmt.Errorf("invalid GCP resource format: %s", scope.Resource)
	}
	for _, perm := range scope.Permissions {
		switch perm {
		case "read", "write", "delete", "list", "get", "update", "create", "*":
		default:
			return fmt.Errorf("invalid GCP permission: %s", perm)
		}
	}
	return nil
}

func (g *GCPProvider) getTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if g.tokenSource != nil {
		return g.tokenSource, nil
	}
	if g.credentialsFile != "" {
		creds, err := google.CredentialsFromJSON(ctx, []byte(g.credentialsFile),
			"https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials file: %w", err)
		}
		return creds.TokenSource, nil
	}
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("gcp: no credentials found (set GOOGLE_APPLICATION_CREDENTIALS or provide credentials file): %w", err)
	}
	return creds.TokenSource, nil
}

func (g *GCPProvider) resolveServiceAccount(resource string) string {
	if strings.HasPrefix(resource, "gs://") {
		bucket := strings.Split(strings.TrimPrefix(resource, "gs://"), "/")[0]
		if g.projectID != "" {
			return fmt.Sprintf("ans-storage-sa@%s.iam.gserviceaccount.com", g.projectID)
		}
		_ = bucket
		return "ans-storage-sa@PROJECT_ID.iam.gserviceaccount.com"
	}
	if strings.HasPrefix(resource, "projects/") {
		parts := strings.Split(resource, "/")
		if len(parts) >= 2 {
			return fmt.Sprintf("ans-compute-sa@%s.iam.gserviceaccount.com", parts[1])
		}
	}
	if g.projectID != "" {
		return fmt.Sprintf("ans-default-sa@%s.iam.gserviceaccount.com", g.projectID)
	}
	return "ans-default-sa@PROJECT_ID.iam.gserviceaccount.com"
}
