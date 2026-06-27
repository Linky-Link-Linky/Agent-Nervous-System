package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"golang.org/x/oauth2"
)

func TestEnvProvider(t *testing.T) {
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	os.Setenv("AWS_SESSION_TOKEN", "test-session-token")
	t.Cleanup(func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
	})

	p := NewEnvProvider()
	if p.Name() != "env" {
		t.Errorf("Name() = %q, want env", p.Name())
	}

	req := &ProvisionRequest{
		AgentID:    "ans_agent1",
		ActionType: "s3:GetObject",
		Scope:      Scope{Resource: "s3://bucket", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
	if cred.AgentID != "ans_agent1" {
		t.Errorf("AgentID = %q, want ans_agent1", cred.AgentID)
	}
	if cred.Secret != "test-secret-key" {
		t.Errorf("Secret = %q, want test-secret-key", cred.Secret)
	}
	meta := cred.Metadata
	if meta == nil || meta["access_key"] != "test-access-key" {
		t.Errorf("Metadata access_key = %q, want test-access-key", meta["access_key"])
	}
	if meta == nil || meta["session_token"] != "test-session-token" {
		t.Errorf("Metadata session_token = %q, want test-session-token", meta["session_token"])
	}
}

func TestEnvProviderMissingVars(t *testing.T) {
	os.Unsetenv("ANS_ACCESS_KEY")
	os.Unsetenv("ANS_SECRET_KEY")
	os.Unsetenv("ANS_SESSION_TOKEN")

	p := NewEnvProvider()
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "s3://bucket", Permissions: []string{"read"}},
	})
	if err == nil {
		t.Error("ProvisionCredential() succeeded with missing env vars, want error")
	}
}

type mockSTSClient struct {
	assumeRoleFunc func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

func (m *mockSTSClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	return m.assumeRoleFunc(ctx, params, optFns...)
}

func TestAWSProvider(t *testing.T) {
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			if params.RoleArn == nil || *params.RoleArn == "" {
				t.Error("AssumeRole called with empty RoleArn")
			}
			if params.RoleSessionName == nil || *params.RoleSessionName == "" {
				t.Error("AssumeRole called with empty RoleSessionName")
			}
			if params.Policy == nil || *params.Policy == "" {
				t.Error("AssumeRole called with empty Policy")
			}
			var policyDoc map[string]interface{}
			if err := json.Unmarshal([]byte(*params.Policy), &policyDoc); err != nil {
				t.Errorf("Policy is not valid JSON: %v", err)
			}
			return &sts.AssumeRoleOutput{
				Credentials: &types.Credentials{
					AccessKeyId:     aws.String("ASIA-test-access-key"),
					SecretAccessKey: aws.String("test-secret-key"),
					SessionToken:    aws.String("test-session-token"),
					Expiration:      aws.Time(time.Now().Add(15 * time.Minute)),
				},
			}, nil
		},
	}

	p := NewAWSProvider("us-east-1", "123456789012", WithSTSClient(mock))
	req := &ProvisionRequest{
		AgentID:    "ans_agent1",
		ActionType: "s3:GetObject",
		Scope:      Scope{Resource: "s3://my-bucket/data.txt", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
	if cred.Type != "aws-sts" {
		t.Errorf("Type = %q, want aws-sts", cred.Type)
	}
	if cred.Secret != "test-secret-key" {
		t.Errorf("Secret = %q, want test-secret-key", cred.Secret)
	}
	if cred.Metadata["access_key"] != "ASIA-test-access-key" {
		t.Errorf("Metadata access_key = %q, want ASIA-test-access-key", cred.Metadata["access_key"])
	}
	if cred.Metadata["session_token"] != "test-session-token" {
		t.Errorf("Metadata session_token = %q, want test-session-token", cred.Metadata["session_token"])
	}
	if cred.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt is in the past")
	}
}

func TestAWSProviderEmptyCredentials(t *testing.T) {
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			return &sts.AssumeRoleOutput{Credentials: nil}, nil
		},
	}
	p := NewAWSProvider("us-east-1", "123456789012", WithSTSClient(mock))
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "s3://bucket", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error for empty credentials, got nil")
	}
}

func TestAWSProviderAssumeRoleError(t *testing.T) {
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			return nil, fmt.Errorf("AccessDenied: User is not authorized")
		},
	}
	p := NewAWSProvider("us-east-1", "123456789012", WithSTSClient(mock))
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "s3://bucket", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error from AssumeRole failure, got nil")
	}
	if !strings.Contains(err.Error(), "AccessDenied") {
		t.Errorf("Expected AccessDenied error, got: %v", err)
	}
}

func TestAWSProviderUnsupportedResource(t *testing.T) {
	p := NewAWSProvider("us-east-1", "123456789012")
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "unsupported://format", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("ProvisionCredential() with unsupported resource succeeded, want error")
	}
}

func TestAWSProviderValidateScope(t *testing.T) {
	p := NewAWSProvider("us-east-1", "123456789012")
	tests := []struct {
		scope   Scope
		wantErr bool
	}{
		{Scope{Resource: "s3://bucket", Permissions: []string{"read"}}, false},
		{Scope{Resource: "arn:aws:s3:::bucket", Permissions: []string{"write"}}, false},
		{Scope{Resource: "s3://bucket", Permissions: []string{"delete"}}, false},
		{Scope{Resource: "arn:aws-cn:s3:::bucket", Permissions: []string{"list"}}, false},
		{Scope{Resource: "arn:aws-us-gov:s3:::bucket", Permissions: []string{"*"}}, false},
		{Scope{Resource: "s3://bucket", Permissions: []string{"invalid"}}, true},
		{Scope{Resource: "", Permissions: []string{"read"}}, true},
		{Scope{Resource: "invalid", Permissions: []string{"read"}}, true},
	}
	for _, tt := range tests {
		err := p.ValidateScope(tt.scope)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateScope(%+v) err=%v, wantErr=%v", tt.scope, err, tt.wantErr)
		}
	}
}

func TestAWSProviderResolveARN(t *testing.T) {
	mock := &mockSTSClient{
		assumeRoleFunc: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
			if *params.RoleArn != "arn:aws:iam::123456789012:role/custom-role" {
				t.Errorf("Expected custom ARN, got %s", *params.RoleArn)
			}
			return &sts.AssumeRoleOutput{
				Credentials: &types.Credentials{
					AccessKeyId: aws.String("AKIA-test"), SecretAccessKey: aws.String("sk-test"),
					SessionToken: aws.String("tok-test"), Expiration: aws.Time(time.Now().Add(15 * time.Minute)),
				},
			}, nil
		},
	}
	p := NewAWSProvider("us-east-1", "123456789012", WithSTSClient(mock))
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "arn:aws:iam::123456789012:role/custom-role", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
}

func TestGCPProvider(t *testing.T) {
	tokenURL := "http://mock-token.example.com/token"

	httpClient := &http.Client{Transport: &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "iamcredentials.googleapis.com" {
				return &http.Response{
					StatusCode: 200,
					Body:       bodyReader(`{"accessToken":"ya29.mock-generated-token","expireTime":"2026-12-31T23:59:59Z"}`),
					Header:     http.Header{"Content-Type": {"application/json"}},
				}, nil
			}
			if req.URL.String() == tokenURL {
				return &http.Response{
					StatusCode: 200,
					Body:       bodyReader(`{"access_token":"mock-gcp-token","expires_in":3600,"token_type":"Bearer"}`),
					Header:     http.Header{"Content-Type": {"application/json"}},
				}, nil
			}
			return http.DefaultTransport.RoundTrip(req)
		},
	}}

	p := NewGCPProvider("", "my-project",
		WithGCPHTTPClient(httpClient),
		WithGCPTokenSource(&mockTokenSource{token: &oauth2.Token{AccessToken: "mock-gcp-token"}}),
	)

	if p.Name() != "gcp-iam" {
		t.Errorf("Name() = %q, want gcp-iam", p.Name())
	}
	req := &ProvisionRequest{
		AgentID:    "ans_agent1",
		ActionType: "storage.objects.get",
		Scope:      Scope{Resource: "gs://my-bucket/data.txt", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
	if cred.Type != "oauth2-access-token" {
		t.Errorf("Type = %q, want oauth2-access-token", cred.Type)
	}
	if cred.Metadata["project_id"] != "my-project" {
		t.Errorf("project_id = %q, want my-project", cred.Metadata["project_id"])
	}
}

func TestGCPProviderIAMAPIError(t *testing.T) {
	httpClient := &http.Client{Transport: &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "iamcredentials.googleapis.com" {
				return &http.Response{
					StatusCode: 403,
					Body:       bodyReader(`{"error":"permission denied"}`),
					Header:     http.Header{"Content-Type": {"application/json"}},
				}, nil
			}
			return http.DefaultTransport.RoundTrip(req)
		},
	}}

	p := NewGCPProvider("", "my-project",
		WithGCPHTTPClient(httpClient),
		WithGCPTokenSource(&mockTokenSource{token: &oauth2.Token{AccessToken: "mock-token"}}),
	)
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "gs://bucket", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error from IAM API rejection, got nil")
	}
}

func TestGCPProviderValidateScope(t *testing.T) {
	p := NewGCPProvider("/c.json", "p")
	tests := []struct {
		scope   Scope
		wantErr bool
	}{
		{Scope{Resource: "gs://bucket", Permissions: []string{"read"}}, false},
		{Scope{Resource: "projects/my-proj", Permissions: []string{"write"}}, false},
		{Scope{Resource: "//compute.googleapis.com/projects/", Permissions: []string{"*"}}, false},
		{Scope{Resource: "", Permissions: []string{"read"}}, true},
		{Scope{Resource: "invalid", Permissions: []string{"read"}}, true},
		{Scope{Resource: "gs://bucket", Permissions: []string{"invalid"}}, true},
	}
	for _, tt := range tests {
		err := p.ValidateScope(tt.scope)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateScope(%+v): err=%v, wantErr=%v", tt.scope, err, tt.wantErr)
		}
	}
}

func TestAzureProvider(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm(): %v", err)
		}
		if r.Form.Get("client_id") != "test-client-id" {
			t.Errorf("client_id = %q, want test-client-id", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") != "test-secret" {
			t.Errorf("client_secret = %q, want test-secret", r.Form.Get("client_secret"))
		}
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", r.Form.Get("grant_type"))
		}
		if !strings.Contains(r.Form.Get("scope"), ".default") {
			t.Errorf("scope = %q, expected /.default", r.Form.Get("scope"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6InRlc3QifQ.test-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer ts.Close()

	p := NewAzureProvider("test-tenant", "test-client-id", "test-secret",
		WithAzureHTTPClient(ts.Client()),
		WithAzureTokenURL(ts.URL),
	)

	if p.Name() != "azure-ad" {
		t.Errorf("Name() = %q, want azure-ad", p.Name())
	}

	req := &ProvisionRequest{
		AgentID:    "ans_agent1",
		ActionType: "blob.read",
		Scope:      Scope{Resource: "https://storage.blob.core.windows.net/container", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
	if cred.Type != "oauth2-access-token" {
		t.Errorf("Type = %q, want oauth2-access-token", cred.Type)
	}
	if cred.Metadata["tenant_id"] != "test-tenant" {
		t.Errorf("tenant_id = %q, want test-tenant", cred.Metadata["tenant_id"])
	}
}

func TestAzureProviderMissingConfig(t *testing.T) {
	tests := []struct {
		name   string
		tenant string
		client string
		secret string
	}{
		{"empty tenant", "", "c", "s"},
		{"empty client", "t", "", "s"},
		{"empty secret", "t", "c", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewAzureProvider(tt.tenant, tt.client, tt.secret)
			_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
				AgentID: "ans_agent1", ActionType: "read",
				Scope: Scope{Resource: "https://example.com", Permissions: []string{"read"}}, TTLSeconds: 60,
			})
			if err == nil {
				t.Error("Expected error for missing config, got nil")
			}
		})
	}
}

func TestAzureProviderTokenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_client",
			"error_description": "Invalid client credentials",
		})
	}))
	defer ts.Close()

	p := NewAzureProvider("test-tenant", "bad-client", "bad-secret",
		WithAzureHTTPClient(ts.Client()),
		WithAzureTokenURL(ts.URL),
	)

	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "https://example.com", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error from Azure token endpoint, got nil")
	}
}

func TestAzureProviderValidateScope(t *testing.T) {
	p := NewAzureProvider("t", "c", "s")
	tests := []struct {
		scope   Scope
		wantErr bool
	}{
		{Scope{Resource: "https://storage.azure.com/container", Permissions: []string{"read"}}, false},
		{Scope{Resource: "api://my-api", Permissions: []string{"write"}}, false},
		{Scope{Resource: "https://example.com", Permissions: []string{"*"}}, false},
		{Scope{Resource: "", Permissions: []string{"read"}}, true},
		{Scope{Resource: "not-a-url", Permissions: []string{"read"}}, true},
	}
	for _, tt := range tests {
		err := p.ValidateScope(tt.scope)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateScope(%+v): err=%v, wantErr=%v", tt.scope, err, tt.wantErr)
		}
	}
}

func TestOAuth2Provider(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm(): %v", err)
		}
		if r.Form.Get("client_id") != "test-client-id" {
			t.Errorf("client_id = %q, want test-client-id", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") != "test-secret" {
			t.Errorf("client_secret = %q, want test-secret", r.Form.Get("client_secret"))
		}
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "oauth2_test_token_value",
			"expires_in":   3600,
			"token_type":   "Bearer",
			"scope":        "openid profile",
		})
	}))
	defer ts.Close()

	p := NewOAuth2Provider(ts.URL, "test-client-id", "test-secret", []string{"openid", "profile"},
		WithOAuth2HTTPClient(ts.Client()),
	)

	if p.Name() != "oauth2" {
		t.Errorf("Name() = %q, want oauth2", p.Name())
	}

	req := &ProvisionRequest{
		AgentID:    "ans_agent1",
		ActionType: "api.read",
		Scope:      Scope{Resource: "https://api.example.com/resource", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
	if cred.Type != "bearer-token" {
		t.Errorf("Type = %q, want bearer-token", cred.Type)
	}
	if cred.Secret != "oauth2_test_token_value" {
		t.Errorf("Secret = %q, want oauth2_test_token_value", cred.Secret)
	}
}

func TestOAuth2ProviderMissingConfig(t *testing.T) {
	tests := []struct {
		name     string
		tokenURL string
		clientID string
		secret   string
	}{
		{"empty token URL", "", "c", "s"},
		{"empty client ID", "https://example.com", "", "s"},
		{"empty secret", "https://example.com", "c", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewOAuth2Provider(tt.tokenURL, tt.clientID, tt.secret, nil)
			_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
				AgentID: "ans_agent1", ActionType: "read",
				Scope: Scope{Resource: "https://api.example.com", Permissions: []string{"read"}}, TTLSeconds: 60,
			})
			if err == nil {
				t.Error("Expected error for missing config, got nil")
			}
		})
	}
}

func TestOAuth2ProviderTokenError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Invalid client credentials",
		})
	}))
	defer ts.Close()

	p := NewOAuth2Provider(ts.URL, "bad-client", "bad-secret", nil,
		WithOAuth2HTTPClient(ts.Client()),
	)
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "https://api.example.com", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error from OAuth2 token endpoint, got nil")
	}
}

func TestOAuth2ProviderEmptyTokenResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "",
			"expires_in":   3600,
		})
	}))
	defer ts.Close()

	p := NewOAuth2Provider(ts.URL, "client", "secret", nil,
		WithOAuth2HTTPClient(ts.Client()),
	)
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "https://api.example.com", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error for empty access token, got nil")
	}
}

func TestOAuth2ProviderValidateScope(t *testing.T) {
	p := NewOAuth2Provider("https://auth.example.com/token", "c", "s", nil)
	if err := p.ValidateScope(Scope{Resource: "https://api.example.com"}); err != nil {
		t.Errorf("ValidateScope(valid) = %v, want nil", err)
	}
	if err := p.ValidateScope(Scope{Resource: ""}); err == nil {
		t.Error("ValidateScope(empty) = nil, want error")
	}
}

func TestVaultProvider(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") == "" {
			t.Error("Missing Vault token header")
		}
		if r.Header.Get("X-Vault-Namespace") != "test-ns" {
			t.Errorf("X-Vault-Namespace = %q, want test-ns", r.Header.Get("X-Vault-Namespace"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"access_key": "AKIA-test",
				"secret_key": "test-secret",
				"security_token": "test-token",
			},
			"lease_id":  "test-lease-id",
			"renewable": true,
			"request_id": "test-req-id",
		})
	}))
	defer ts.Close()

	p := NewVaultProvider(ts.URL, "hvs.test-token", "test-ns",
		WithVaultHTTPClient(ts.Client()),
	)

	if p.Name() != "vault" {
		t.Errorf("Name() = %q, want vault", p.Name())
	}

	req := &ProvisionRequest{
		AgentID:    "ans_agent1",
		ActionType: "secret.read",
		Scope:      Scope{Resource: "vault://aws/creds/deploy-role", Permissions: []string{"read"}},
		TTLSeconds: 60,
	}
	cred, err := p.ProvisionCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("ProvisionCredential(): %v", err)
	}
	if cred.Type != "vault-dynamic-secret" {
		t.Errorf("Type = %q, want vault-dynamic-secret", cred.Type)
	}
	if cred.Metadata["vault_path"] != "/v1/aws/creds/deploy-role" {
		t.Errorf("vault_path = %q, want /v1/aws/creds/deploy-role", cred.Metadata["vault_path"])
	}
	if cred.Metadata["lease_id"] != "test-lease-id" {
		t.Errorf("lease_id = %q, want test-lease-id", cred.Metadata["lease_id"])
	}
	if cred.Metadata["renewable"] != "true" {
		t.Errorf("renewable = %q, want true", cred.Metadata["renewable"])
	}
}

func TestVaultProviderNilSecret(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	p := NewVaultProvider(ts.URL, "hvs.test-token", "",
		WithVaultHTTPClient(ts.Client()),
	)
	_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "vault://aws/creds/role", Permissions: []string{"read"}}, TTLSeconds: 60,
	})
	if err == nil {
		t.Error("Expected error for nil secret, got nil")
	}
}

func TestVaultProviderMissingConfig(t *testing.T) {
	tests := []struct {
		name    string
		address string
		token   string
	}{
		{"empty address", "", "hvs.token"},
		{"empty token", "https://vault.example.com", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewVaultProvider(tt.address, tt.token, "")
			_, err := p.ProvisionCredential(context.Background(), &ProvisionRequest{
				AgentID: "ans_agent1", ActionType: "read",
				Scope: Scope{Resource: "vault://aws/creds/role", Permissions: []string{"read"}}, TTLSeconds: 60,
			})
			if err == nil {
				t.Error("Expected error for missing config, got nil")
			}
		})
	}
}

func TestVaultProviderValidateScope(t *testing.T) {
	p := NewVaultProvider("https://vault.example.com:8200", "token", "")
	tests := []struct {
		scope   Scope
		wantErr bool
	}{
		{Scope{Resource: "vault://aws/creds/role", Permissions: []string{"read"}}, false},
		{Scope{Resource: "vault:aws/creds/role", Permissions: []string{"write"}}, false},
		{Scope{Resource: "vault://database/creds/db-role", Permissions: []string{"*"}}, false},
		{Scope{Resource: "", Permissions: []string{"read"}}, true},
		{Scope{Resource: "invalid", Permissions: []string{"read"}}, true},
		{Scope{Resource: "vault://aws/creds/role", Permissions: []string{"invalid"}}, true},
	}
	for _, tt := range tests {
		err := p.ValidateScope(tt.scope)
		gotErr := err != nil
		if gotErr != tt.wantErr {
			t.Errorf("ValidateScope(%+v): err=%v, wantErr=%v", tt.scope, err, tt.wantErr)
		}
	}
}

type mockTransport struct {
	roundTripFunc func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

type mockTokenSource struct {
	token *oauth2.Token
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, nil
}

func bodyReader(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}
