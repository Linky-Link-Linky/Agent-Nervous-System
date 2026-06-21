// Package broker — Ephemeral Identity Provisioning for Zero-Trust Agent Access.
//
// Agents execute under the developer's admin keys by default — massive insider threat.
// The Identity Broker provisions ephemeral, scoped, single-use credentials with 60-second TTLs.
//
// Supported backends:
// - HashiCorp Vault (dynamic secrets)
// - AWS IAM (temporary credentials via STS AssumeRole)
// - Google Cloud IAM (short-lived service account tokens)
// - Azure AD (ephemeral app registrations)
// - Generic OAuth2/OIDC (client credentials flow with TTL)
//
// SPDX-License-Identifier: MIT
package broker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Provider defines the interface for credential providers (Vault, AWS, GCP, etc.)
type Provider interface {
	// Name returns the provider name (e.g., "vault", "aws-iam", "gcp-iam")
	Name() string

	// ProvisionCredential generates an ephemeral credential for the given scope.
	// Returns a Credential with a 60-second TTL (or provider-defined max TTL).
	ProvisionCredential(ctx context.Context, req *ProvisionRequest) (*Credential, error)

	// RevokeCredential immediately revokes the credential (best-effort).
	RevokeCredential(ctx context.Context, credentialID string) error

	// ValidateScope checks if the scope is valid for this provider.
	ValidateScope(scope Scope) error
}

// ProvisionRequest describes what access the agent needs.
type ProvisionRequest struct {
	AgentID       string    `json:"agent_id"`        // Agent requesting the credential
	ActionType    string    `json:"action_type"`     // e.g., "file.read", "http.post", "db.query"
	Scope         Scope     `json:"scope"`           // What resources the agent can access
	TTLSeconds    int       `json:"ttl_seconds"`     // Requested TTL (max 60s enforced)
	RequestID     string    `json:"request_id"`      // Unique request ID for audit
	PreReceiptID  string    `json:"pre_receipt_id"`  // Linked pre-action receipt
	ParentAgentID string    `json:"parent_agent_id"` // Parent agent if sub-agent
	Timestamp     time.Time `json:"timestamp"`       // Request timestamp
}

// Scope defines what the credential can access.
type Scope struct {
	// Resource is the target (e.g., "s3://bucket/object", "db://prod/users", "file:///etc/nginx/conf")
	Resource string `json:"resource"`

	// Permissions is the set of allowed operations (e.g., ["read"], ["write"], ["read", "write"])
	Permissions []string `json:"permissions"`

	// Constraints are additional restrictions (e.g., "ip:192.168.1.0/24", "time:14:00-15:00")
	Constraints map[string]string `json:"constraints,omitempty"`
}

// Credential is an ephemeral, scoped credential.
type Credential struct {
	CredentialID   string            `json:"credential_id"`   // Unique credential ID
	AgentID        string            `json:"agent_id"`        // Agent that owns this credential
	ProviderName   string            `json:"provider"`        // Provider that issued it
	Type           string            `json:"type"`            // "aws-sts", "vault-token", "oauth2-bearer", etc.
	Secret         string            `json:"secret"`          // The actual secret (token, key, etc.)
	Metadata       map[string]string `json:"metadata"`        // Provider-specific metadata (e.g., "role_arn")
	Scope          Scope             `json:"scope"`           // What this credential can access
	IssuedAt       time.Time         `json:"issued_at"`       // When issued
	ExpiresAt      time.Time         `json:"expires_at"`      // When it expires
	Revoked        bool              `json:"revoked"`         // Whether it's been revoked
	RequestID      string            `json:"request_id"`      // Request ID that generated this
	PreReceiptID   string            `json:"pre_receipt_id"`  // Pre-action receipt ID
	ProvisionLogID string            `json:"provision_log_id"` // Log entry ID
}

// Broker manages credential provisioning and lifecycle.
type Broker struct {
	providers map[string]Provider // provider name → provider
	cache     *CredentialCache
	logger    Logger
	mu        sync.RWMutex
	done      chan struct{}
}

// NewBroker creates a new identity broker.
func NewBroker(logger Logger) *Broker {
	return &Broker{
		providers: make(map[string]Provider),
		cache:     NewCredentialCache(),
		logger:    logger,
		done:      make(chan struct{}),
	}
}

// RegisterProvider registers a credential provider.
func (b *Broker) RegisterProvider(p Provider) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	name := p.Name()
	if _, exists := b.providers[name]; exists {
		return fmt.Errorf("provider %q already registered", name)
	}
	b.providers[name] = p
	b.logger.Infof("Registered credential provider: %s", name)
	return nil
}

// Provision generates an ephemeral credential.
func (b *Broker) Provision(ctx context.Context, providerName string, req *ProvisionRequest) (*Credential, error) {
	b.mu.RLock()
	provider, ok := b.providers[providerName]
	b.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerName)
	}

	// Enforce max TTL of 60 seconds
	if req.TTLSeconds > 60 || req.TTLSeconds <= 0 {
		req.TTLSeconds = 60
	}

	// Validate scope
	if err := provider.ValidateScope(req.Scope); err != nil {
		return nil, fmt.Errorf("invalid scope: %w", err)
	}

	// Generate request ID if not provided
	if req.RequestID == "" {
		req.RequestID = generateRequestID()
	}
	req.Timestamp = time.Now()

	// Provision the credential
	b.logger.Infof("Provisioning credential for agent %s (provider: %s, scope: %s)",
		req.AgentID, providerName, req.Scope.Resource)

	cred, err := provider.ProvisionCredential(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("provider %s failed to provision: %w", providerName, err)
	}

	// Cache the credential
	b.cache.Store(cred)

	// Schedule auto-revocation — no request context; background goroutine uses its own timeout
	go b.scheduleRevocation(cred) // #nosec G118

	b.logger.Infof("Credential %s issued for agent %s (expires in %ds)",
		cred.CredentialID, cred.AgentID, req.TTLSeconds)

	return cred, nil
}

// Revoke immediately revokes a credential.
func (b *Broker) Revoke(ctx context.Context, credentialID string) error {
	cred := b.cache.Get(credentialID)
	if cred == nil {
		return fmt.Errorf("credential %q not found", credentialID)
	}

	b.mu.RLock()
	provider, ok := b.providers[cred.ProviderName]
	b.mu.RUnlock()

	if !ok {
		return fmt.Errorf("provider %q not found", cred.ProviderName)
	}

	b.logger.Infof("Revoking credential %s (agent: %s)", credentialID, cred.AgentID)

	if err := provider.RevokeCredential(ctx, credentialID); err != nil {
		return fmt.Errorf("provider %s failed to revoke: %w", cred.ProviderName, err)
	}

	b.cache.MarkRevoked(credentialID) // Update cache with revoked status

	b.logger.Infof("Credential %s revoked", credentialID)
	return nil
}

// Get retrieves a credential from the cache.
func (b *Broker) Get(credentialID string) *Credential {
	return b.cache.Get(credentialID)
}

// ListActive returns all active (non-expired, non-revoked) credentials.
func (b *Broker) ListActive() []*Credential {
	return b.cache.ListActive()
}

// Close stops all pending revocation goroutines.
func (b *Broker) Close() {
	close(b.done)
}

// scheduleRevocation automatically revokes the credential when it expires.
func (b *Broker) scheduleRevocation(cred *Credential) {
	ttl := time.Until(cred.ExpiresAt)
	if ttl <= 0 {
		return
	}

	select {
	case <-b.done:
		return
	case <-time.After(ttl):
	}

	// Best-effort revocation; no request context available in background goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.Revoke(ctx, cred.CredentialID); err != nil {
		b.logger.Errorf("Auto-revoke failed for credential %s: %v", cred.CredentialID, err)
	}
}

// CredentialCache is a thread-safe cache for credentials.
type CredentialCache struct {
	creds map[string]*Credential
	mu    sync.RWMutex
}

func NewCredentialCache() *CredentialCache {
	return &CredentialCache{
		creds: make(map[string]*Credential),
	}
}

func (c *CredentialCache) Store(cred *Credential) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.creds[cred.CredentialID] = cred
}

func (c *CredentialCache) Get(credentialID string) *Credential {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.creds[credentialID]
}

func (c *CredentialCache) MarkRevoked(credentialID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cred, ok := c.creds[credentialID]; ok {
		cred.Revoked = true
	}
}

func (c *CredentialCache) ListActive() []*Credential {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	active := make([]*Credential, 0)
	for _, cred := range c.creds {
		if !cred.Revoked && now.Before(cred.ExpiresAt) {
			active = append(active, cred)
		}
	}
	return active
}

// Logger interface for broker logging.
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to nanosecond timestamp if crypto/rand fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:8])
}

// Sensitive metadata keys that should be redacted in JSON output.
var sensitiveMetaKeys = map[string]bool{
	"access_key":    true,
	"session_token": true,
	"secret_key":    true,
}

// MarshalJSON redacts the secret and sensitive metadata when marshaling to JSON.
func (c *Credential) MarshalJSON() ([]byte, error) {
	type Alias Credential
	redactedMeta := make(map[string]string, len(c.Metadata))
	for k, v := range c.Metadata {
		if sensitiveMetaKeys[k] {
			redactedMeta[k] = "[REDACTED]"
		} else {
			redactedMeta[k] = v
		}
	}
	return json.Marshal(&struct {
		Secret   string            `json:"secret"`
		Metadata map[string]string `json:"metadata"`
		*Alias
	}{
		Secret:   "[REDACTED]",
		Metadata: redactedMeta,
		Alias:    (*Alias)(c),
	})
}
