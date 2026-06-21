package identitybroker

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Token represents an ephemeral credential for agent execution.
type Token struct {
	ID           string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	Resource      string    `json:"resource"`
	ResourceType  string    `json:"resource_type"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsedAt    time.Time `json:"last_used_at"`
	UsageCount    int       `json:"usage_count"`
	MaxUsage      int       `json:"max_usage"`
	Revoked       bool      `json:"revoked"`
	RevokedAt     time.Time `json:"revoked_at,omitempty"`
	RevokedBy     string    `json:"revoked_by,omitempty"`
	ProviderData  string    `json:"provider_data,omitempty"`
	Metadata      string    `json:"metadata,omitempty"`
}

// TokenRequest defines the parameters for requesting an ephemeral token.
type TokenRequest struct {
	AgentID       string `json:"agent_id"`
	Resource      string `json:"resource"`
	ResourceType  string `json:"resource_type"`
	Purpose       string `json:"purpose"`
	TTLSeconds    int    `json:"ttl_seconds"`
	MaxUsage      int    `json:"max_usage"`
	Metadata      string `json:"metadata,omitempty"`
}

// TokenResponse is the result of token provisioning.
type TokenResponse struct {
	TokenID      string    `json:"token_id"`
	TokenValue   string    `json:"token_value"`
	ExpiresAt    time.Time `json:"expires_at"`
	Resource     string    `json:"resource"`
	ResourceType string    `json:"resource_type"`
	Metadata     string    `json:"metadata,omitempty"`
}

// ValidationResult represents the result of token validation.
type ValidationResult struct {
	Valid        bool   `json:"valid"`
	TokenID      string `json:"token_id"`
	AgentID      string `json:"agent_id"`
	Resource     string `json:"resource"`
	Error        string `json:"error,omitempty"`
	UsageCount   int    `json:"usage_count"`
}

// Provider defines the interface for identity providers.
type Provider interface {
	// Name returns the provider's name.
	Name() string

	// ProvisionToken creates a new ephemeral token for the given request.
	ProvisionToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error)

	// ValidateToken validates a token value and returns the associated token.
	ValidateToken(ctx context.Context, tokenValue string) (*Token, error)

	// RevokeToken revokes a token by ID.
	RevokeToken(ctx context.Context, tokenID string, revokedBy string) error

	// GetToken retrieves a token by ID.
	GetToken(ctx context.Context, tokenID string) (*Token, error)

	// ListTokens returns tokens matching the given criteria.
	ListTokens(ctx context.Context, agentID string, resourceType string, limit int) ([]*Token, error)
}

// TokenStore manages token persistence.
type TokenStore interface {
	// Insert stores a new token.
	Insert(token *Token) error

	// Get retrieves a token by ID.
	Get(id string) (*Token, error)

	// Update updates an existing token.
	Update(token *Token) error

	// Delete removes a token by ID.
	Delete(id string) error

	// List returns tokens matching the given filters.
	List(agentID string, resourceType string, limit int) ([]*Token, error)

	// CleanupExpired removes expired tokens.
	CleanupExpired() (int, error)

	// RevokeAllForAgent revokes all tokens for an agent.
	RevokeAllForAgent(agentID string, revokedBy string) error
}

// TokenManager orchestrates token lifecycle and provider interactions.
type TokenManager struct {
	store    TokenStore
	provider Provider
	ttl      time.Duration
	mu       sync.RWMutex
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager(store TokenStore, provider Provider, ttl time.Duration) *TokenManager {
	return &TokenManager{store: store, provider: provider, ttl: ttl}
}

// ProvisionToken creates a new ephemeral token for the given request.
func (tm *TokenManager) ProvisionToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if req.TTLSeconds <= 0 {
		req.TTLSeconds = int(tm.ttl.Seconds())
	}

	// Generate token ID
	tokenID := fmt.Sprintf("tok_%s", generateTokenID())

	// Create token
	token := &Token{
		ID: tokenID,
		AgentID:      req.AgentID,
		Resource:     req.Resource,
		ResourceType: req.ResourceType,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(time.Duration(req.TTLSeconds) * time.Second),
		MaxUsage:     req.MaxUsage,
		Metadata:     req.Metadata,
	}

	// Store token
	if err := tm.store.Insert(token); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	// Get token value from provider
	resp, err := tm.provider.ProvisionToken(ctx, req)
	if err != nil {
		// If provider fails, delete the stored token
		_ = tm.store.Delete(tokenID)
		return nil, fmt.Errorf("failed to provision token from provider: %w", err)
	}

	// Update token with provider data
	token.ProviderData = resp.TokenValue
	if err := tm.store.Update(token); err != nil {
		return nil, fmt.Errorf("failed to update token with provider data: %w", err)
	}

	return &TokenResponse{
		TokenID:      tokenID,
		TokenValue:   resp.TokenValue,
		ExpiresAt:    token.ExpiresAt,
		Resource:     token.Resource,
		ResourceType: token.ResourceType,
		Metadata:     token.Metadata,
	}, nil
}

// ValidateToken validates a token value and returns the validation result.
func (tm *TokenManager) ValidateToken(ctx context.Context, tokenValue string) (*ValidationResult, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Get token from provider
	token, err := tm.provider.ValidateToken(ctx, tokenValue)
	if err != nil {
		return &ValidationResult{Valid: false, Error: err.Error()}, nil
	}

	// Check if token exists in store
	stored, err := tm.store.Get(token.ID)
	if err != nil {
		return &ValidationResult{Valid: false, Error: "token not found in store"}, nil
	}

	// Check if token is revoked
	if stored.Revoked {
		return &ValidationResult{Valid: false, Error: "token revoked"}, nil
	}

	// Check if token is expired
	if time.Now().After(stored.ExpiresAt) {
		return &ValidationResult{Valid: false, Error: "token expired"}, nil
	}

	// Check usage count
	if stored.UsageCount >= stored.MaxUsage {
		return &ValidationResult{Valid: false, Error: "token usage limit exceeded"}, nil
	}

	// Update last used time and usage count
	stored.LastUsedAt = time.Now()
	stored.UsageCount++
	if err := tm.store.Update(stored); err != nil {
		return nil, fmt.Errorf("failed to update token usage: %w", err)
	}

	return &ValidationResult{
		Valid:      true,
		TokenID:    stored.ID,
		AgentID:    stored.AgentID,
		Resource:   stored.Resource,
		UsageCount: stored.UsageCount,
	}, nil
}

// RevokeToken revokes a token by ID.
func (tm *TokenManager) RevokeToken(ctx context.Context, tokenID string, revokedBy string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Get token from store
	token, err := tm.store.Get(tokenID)
	if err != nil {
		return fmt.Errorf("token not found: %w", err)
	}

	// Revoke token
	token.Revoked = true
	token.RevokedAt = time.Now()
	token.RevokedBy = revokedBy

	// Update in store
	if err := tm.store.Update(token); err != nil {
		return fmt.Errorf("failed to update token: %w", err)
	}

	// Revoke from provider
	if err := tm.provider.RevokeToken(ctx, tokenID, revokedBy); err != nil {
		return fmt.Errorf("failed to revoke token from provider: %w", err)
	}

	return nil
}

// CleanupExpired removes expired tokens.
func (tm *TokenManager) CleanupExpired() (int, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	return tm.store.CleanupExpired()
}

// generateTokenID generates a unique token ID.
func generateTokenID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
