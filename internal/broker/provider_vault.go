package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
)

type VaultProvider struct {
	address    string
	token      string
	namespace  string
	httpClient *http.Client
}

type VaultProviderOption func(*VaultProvider)

func WithVaultHTTPClient(client *http.Client) VaultProviderOption {
	return func(p *VaultProvider) {
		p.httpClient = client
	}
}

func NewVaultProvider(address, token, namespace string, opts ...VaultProviderOption) *VaultProvider {
	p := &VaultProvider{
		address:   address,
		token:     token,
		namespace: namespace,
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

func (v *VaultProvider) Name() string {
	return "vault"
}

func (v *VaultProvider) ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error) {
	if v.address == "" {
		return nil, fmt.Errorf("vault: address is required")
	}
	if v.token == "" {
		return nil, fmt.Errorf("vault: token is required")
	}

	client, err := v.vaultClient()
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create client: %w", err)
	}

	vaultPath := strings.TrimPrefix(req.Scope.Resource, "vault://")
	if vaultPath == req.Scope.Resource {
		vaultPath = strings.TrimPrefix(vaultPath, "vault:")
	}
	vaultPath = strings.TrimPrefix(vaultPath, "/")
	vaultPath = "/v1/" + vaultPath

	secretData := map[string]interface{}{
		"ttl": fmt.Sprintf("%ds", req.TTLSeconds),
	}

	secret, err := client.Logical().WriteWithContext(ctx, vaultPath, secretData)
	if err != nil {
		return nil, fmt.Errorf("vault: failed to write dynamic secret at %s: %w", vaultPath, err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("vault: returned nil secret for path %s", vaultPath)
	}

	secretJSON, _ := mapToJSON(secret.Data)
	leaseID := secret.LeaseID

	now := time.Now()
	credID := generateRequestID()

	meta := map[string]string{
		"vault_path": vaultPath,
		"lease_id":   leaseID,
	}
	if secret.Renewable {
		meta["renewable"] = "true"
	}
	meta["request_id"] = secret.RequestID

	return &Credential{
		CredentialID: credID,
		AgentID:      req.AgentID,
		ProviderName: "vault",
		Type:         "vault-dynamic-secret",
		Secret:       secretJSON,
		Metadata:     meta,
		Scope:        req.Scope,
		IssuedAt:     now,
		ExpiresAt:    now.Add(time.Duration(req.TTLSeconds) * time.Second),
		RequestID:    req.RequestID,
		PreReceiptID: req.PreReceiptID,
	}, nil
}

func (v *VaultProvider) RevokeCredential(ctx context.Context, credentialID string) error {
	return nil
}

func (v *VaultProvider) ValidateScope(scope Scope) error {
	if scope.Resource == "" {
		return fmt.Errorf("vault resource path cannot be empty")
	}
	vaultPath := scope.Resource
	if !strings.HasPrefix(vaultPath, "vault://") && !strings.HasPrefix(vaultPath, "vault:") {
		return fmt.Errorf("invalid vault resource format: %s (expected vault://path)", scope.Resource)
	}
	if len(scope.Permissions) > 0 {
		for _, perm := range scope.Permissions {
			switch perm {
			case "read", "write", "delete", "list", "create", "update", "*":
			default:
				return fmt.Errorf("invalid vault permission: %s", perm)
			}
		}
	}
	return nil
}

func (v *VaultProvider) vaultClient() (*vault.Client, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = v.address
	if v.httpClient != http.DefaultClient {
		cfg.HttpClient = v.httpClient
	}
	client, err := vault.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	client.SetToken(v.token)
	if v.namespace != "" {
		client.SetNamespace(v.namespace)
	}
	return client, nil
}

func mapToJSON(m map[string]interface{}) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
