package broker

import (
	"context"
	"os"
	"testing"
	"time"
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

func TestAWSProvider(t *testing.T) {
	p := NewAWSProvider("us-east-1", "123456789012")
	if p.Name() != "aws-iam" {
		t.Errorf("Name() = %q, want aws-iam", p.Name())
	}
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
	if cred.AgentID != "ans_agent1" {
		t.Errorf("AgentID = %q, want ans_agent1", cred.AgentID)
	}
	if cred.Type != "aws-sts" {
		t.Errorf("Type = %q, want aws-sts", cred.Type)
	}
	if cred.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt is in the past")
	}
	if err := p.RevokeCredential(context.Background(), cred.CredentialID); err != nil {
		t.Errorf("RevokeCredential(): %v", err)
	}
}

func TestAWSProviderUnsupportedResource(t *testing.T) {
	p := NewAWSProvider("us-east-1", "123456789012")
	req := &ProvisionRequest{
		AgentID: "ans_agent1", ActionType: "read",
		Scope: Scope{Resource: "unsupported://format", Permissions: []string{"read"}},
	}
	_, err := p.ProvisionCredential(context.Background(), req)
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

func TestGCPProvider(t *testing.T) {
	p := NewGCPProvider("/path/to/creds.json", "my-project")
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
	if err := p.RevokeCredential(context.Background(), cred.CredentialID); err != nil {
		t.Errorf("RevokeCredential(): %v", err)
	}
}

func TestGCPProviderValidateScope(t *testing.T) {
	p := NewGCPProvider("/c.json", "p")
	if err := p.ValidateScope(Scope{Resource: "gs://bucket"}); err != nil {
		t.Errorf("ValidateScope(valid) = %v, want nil", err)
	}
	if err := p.ValidateScope(Scope{Resource: ""}); err == nil {
		t.Error("ValidateScope(empty) = nil, want error")
	}
}

func TestAzureProvider(t *testing.T) {
	p := NewAzureProvider("tenant-1", "client-1", "secret-1")
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
	if cred.Metadata["tenant_id"] != "tenant-1" {
		t.Errorf("tenant_id = %q, want tenant-1", cred.Metadata["tenant_id"])
	}
	if err := p.RevokeCredential(context.Background(), cred.CredentialID); err != nil {
		t.Errorf("RevokeCredential(): %v", err)
	}
}

func TestAzureProviderValidateScope(t *testing.T) {
	p := NewAzureProvider("t", "c", "s")
	if err := p.ValidateScope(Scope{Resource: "https://storage.azure.com/container"}); err != nil {
		t.Errorf("ValidateScope(valid) = %v, want nil", err)
	}
	if err := p.ValidateScope(Scope{Resource: ""}); err == nil {
		t.Error("ValidateScope(empty) = nil, want error")
	}
}

func TestOAuth2Provider(t *testing.T) {
	p := NewOAuth2Provider("https://auth.example.com/token", "client-id", "client-secret", []string{"openid", "profile"})
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
	if cred.Secret == "" {
		t.Error("Secret is empty")
	}
	if err := p.RevokeCredential(context.Background(), cred.CredentialID); err != nil {
		t.Errorf("RevokeCredential(): %v", err)
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
	p := NewVaultProvider("https://vault.example.com:8200", "hvs.root-token", "ns1")
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
	if cred.Type != "vault-token" {
		t.Errorf("Type = %q, want vault-token", cred.Type)
	}
	if cred.Metadata["vault_path"] != "vault://aws/creds/deploy-role" {
		t.Errorf("vault_path = %q, want vault://aws/creds/deploy-role", cred.Metadata["vault_path"])
	}
	if err := p.RevokeCredential(context.Background(), cred.CredentialID); err != nil {
		t.Errorf("RevokeCredential(): %v", err)
	}
}

func TestVaultProviderValidateScope(t *testing.T) {
	p := NewVaultProvider("https://vault.example.com:8200", "token", "")
	if err := p.ValidateScope(Scope{Resource: "vault://aws/creds/role"}); err != nil {
		t.Errorf("ValidateScope(valid) = %v, want nil", err)
	}
	if err := p.ValidateScope(Scope{Resource: ""}); err == nil {
		t.Error("ValidateScope(empty) = nil, want error")
	}
}
