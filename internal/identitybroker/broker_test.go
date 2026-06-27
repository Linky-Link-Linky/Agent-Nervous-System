package identitybroker

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type mockProvider struct {
	provisionToken  func(ctx context.Context, req *TokenRequest) (*TokenResponse, error)
	validateToken   func(ctx context.Context, tokenValue string) (*Token, error)
	revokeToken     func(ctx context.Context, tokenID string, revokedBy string) error
	getToken        func(ctx context.Context, tokenID string) (*Token, error)
	listTokens      func(ctx context.Context, agentID string, resourceType string, limit int) ([]*Token, error)
}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) ProvisionToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	return m.provisionToken(ctx, req)
}
func (m *mockProvider) ValidateToken(ctx context.Context, tokenValue string) (*Token, error) {
	return m.validateToken(ctx, tokenValue)
}
func (m *mockProvider) RevokeToken(ctx context.Context, tokenID string, revokedBy string) error {
	return m.revokeToken(ctx, tokenID, revokedBy)
}
func (m *mockProvider) GetToken(ctx context.Context, tokenID string) (*Token, error) {
	return m.getToken(ctx, tokenID)
}
func (m *mockProvider) ListTokens(ctx context.Context, agentID string, resourceType string, limit int) ([]*Token, error) {
	return m.listTokens(ctx, agentID, resourceType, limit)
}

func setupManager(t *testing.T) (*TokenManager, *SQLiteTokenStore, *mockProvider) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open(): %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(Schema); err != nil {
		t.Fatalf("Exec(Schema): %v", err)
	}
	store := NewSQLiteTokenStore(db)
	mp := &mockProvider{
		provisionToken: func(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
			return &TokenResponse{
				TokenID:      "tok_" + fmt.Sprintf("%d", time.Now().UnixNano()),
				TokenValue:   "mock-token-value",
				ExpiresAt:    time.Now().Add(time.Duration(req.TTLSeconds) * time.Second),
				Resource:     req.Resource,
				ResourceType: req.ResourceType,
			}, nil
		},
		validateToken: func(ctx context.Context, tokenValue string) (*Token, error) {
			tokens, _ := store.List("", "", 100)
			for _, t := range tokens {
				if t.ProviderData == tokenValue {
					return t, nil
				}
			}
			return nil, fmt.Errorf("token not found: %s", tokenValue)
		},
		getToken: func(ctx context.Context, tokenID string) (*Token, error) {
			return nil, fmt.Errorf("not implemented")
		},
		listTokens: func(ctx context.Context, agentID string, resourceType string, limit int) ([]*Token, error) {
			return nil, fmt.Errorf("not implemented")
		},
		revokeToken: func(ctx context.Context, tokenID string, revokedBy string) error {
			return nil
		},
	}
	tm := NewTokenManager(store, mp, 60*time.Second)
	return tm, store, mp
}

func TestProvisionToken(t *testing.T) {
	tm, _, _ := setupManager(t)
	resp, err := tm.ProvisionToken(context.Background(), &TokenRequest{
		AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
		TTLSeconds: 30, MaxUsage: 3,
	})
	if err != nil {
		t.Fatalf("ProvisionToken(): %v", err)
	}
	if resp.TokenID == "" {
		t.Error("TokenID is empty")
	}
	if resp.TokenValue != "mock-token-value" {
		t.Errorf("TokenValue = %q, want mock-token-value", resp.TokenValue)
	}
	if resp.Resource != "s3://bucket" {
		t.Errorf("Resource = %q, want s3://bucket", resp.Resource)
	}
}

func TestProvisionTokenDefaultTTL(t *testing.T) {
	tm, _, _ := setupManager(t)
	resp, err := tm.ProvisionToken(context.Background(), &TokenRequest{
		AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
		TTLSeconds: 0, MaxUsage: 1,
	})
	if err != nil {
		t.Fatalf("ProvisionToken(): %v", err)
	}
	expectedTTL := 60 * time.Second
	ttl := resp.ExpiresAt.Sub(time.Now())
	if ttl < expectedTTL-5*time.Second || ttl > expectedTTL+5*time.Second {
		t.Errorf("TTL = %v, want ~60s", ttl)
	}
}

func TestProvisionTokenProviderFailure(t *testing.T) {
	tm, _, mp := setupManager(t)
	mp.provisionToken = func(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
		return nil, fmt.Errorf("provider unavailable")
	}
	_, err := tm.ProvisionToken(context.Background(), &TokenRequest{
		AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
		TTLSeconds: 30, MaxUsage: 1,
	})
	if err == nil {
		t.Fatal("ProvisionToken() succeeded, want error")
	}
}

func TestValidateTokenValid(t *testing.T) {
	tm, store, _ := setupManager(t)
	// First provision a token so it exists in store
	resp, err := tm.ProvisionToken(context.Background(), &TokenRequest{
		AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
		TTLSeconds: 60, MaxUsage: 5,
	})
	if err != nil {
		t.Fatalf("ProvisionToken(): %v", err)
	}

	result, err := tm.ValidateToken(context.Background(), resp.TokenValue)
	if err != nil {
		t.Fatalf("ValidateToken(): %v", err)
	}
	if !result.Valid {
		t.Errorf("ValidateToken() Valid = false, want true; error: %s", result.Error)
	}
	if result.AgentID != "ans_agent1" {
		t.Errorf("AgentID = %q, want ans_agent1", result.AgentID)
	}
	// Verify usage was incremented
	store.mu.RLock()
	token, _ := store.Get(resp.TokenID)
	store.mu.RUnlock()
	if token.UsageCount != 1 {
		t.Errorf("UsageCount after validate = %d, want 1", token.UsageCount)
	}
}

func TestValidateTokenExpired(t *testing.T) {
	tm, store, mp := setupManager(t)
	mp.validateToken = func(ctx context.Context, tokenValue string) (*Token, error) {
		return &Token{
			ID: "tok_expired", AgentID: "ans_agent1", Resource: "s3://bucket",
			ResourceType: "s3", ExpiresAt: time.Now().Add(-time.Hour), MaxUsage: 5,
		}, nil
	}
	store.Insert(&Token{
		ID: "tok_expired", AgentID: "ans_agent1", Resource: "s3://bucket",
		ResourceType: "s3", ExpiresAt: time.Now().Add(-time.Hour),
		MaxUsage: 5,
	})

	result, err := tm.ValidateToken(context.Background(), "some-value")
	if err != nil {
		t.Fatalf("ValidateToken(): %v", err)
	}
	if result.Valid {
		t.Error("ValidateToken() for expired token returned Valid=true")
	}
}

func TestValidateTokenUsageLimit(t *testing.T) {
	tm, store, mp := setupManager(t)
	mp.validateToken = func(ctx context.Context, tokenValue string) (*Token, error) {
		return &Token{
			ID: "tok_usedup", AgentID: "ans_agent1", Resource: "s3://bucket",
			ResourceType: "s3", ExpiresAt: time.Now().Add(time.Hour), MaxUsage: 1,
		}, nil
	}
	store.Insert(&Token{
		ID: "tok_usedup", AgentID: "ans_agent1", Resource: "s3://bucket",
		ResourceType: "s3", ExpiresAt: time.Now().Add(time.Hour),
		UsageCount: 1, MaxUsage: 1,
	})

	result, err := tm.ValidateToken(context.Background(), "some-value")
	if err != nil {
		t.Fatalf("ValidateToken(): %v", err)
	}
	if result.Valid {
		t.Error("ValidateToken() for exhausted token returned Valid=true")
	}
}

func TestRevokeToken(t *testing.T) {
	tm, store, _ := setupManager(t)
	resp, err := tm.ProvisionToken(context.Background(), &TokenRequest{
		AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
		TTLSeconds: 60, MaxUsage: 5,
	})
	if err != nil {
		t.Fatalf("ProvisionToken(): %v", err)
	}

	if err := tm.RevokeToken(context.Background(), resp.TokenID, "admin"); err != nil {
		t.Fatalf("RevokeToken(): %v", err)
	}

	store.mu.RLock()
	token, _ := store.Get(resp.TokenID)
	store.mu.RUnlock()
	if !token.Revoked {
		t.Error("Token not marked as revoked")
	}
	if token.RevokedBy != "admin" {
		t.Errorf("RevokedBy = %q, want admin", token.RevokedBy)
	}
}

func TestBrokerCleanupExpired(t *testing.T) {
	tm, store, _ := setupManager(t)
	// Insert an expired token directly
	store.Insert(&Token{
		ID: "tok_old", AgentID: "ans_agent1", Resource: "r", ResourceType: "t",
		ExpiresAt: time.Now().Add(-time.Hour), CreatedAt: time.Now(), LastUsedAt: time.Now(),
		MaxUsage: 1,
	})
	count, err := tm.CleanupExpired()
	if err != nil {
		t.Fatalf("CleanupExpired(): %v", err)
	}
	if count != 1 {
		t.Errorf("CleanupExpired() removed %d, want 1", count)
	}
}

func TestConcurrentProvision(t *testing.T) {
	tm, _, _ := setupManager(t)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := tm.ProvisionToken(context.Background(), &TokenRequest{
				AgentID: "ans_agent1", Resource: "s3://bucket", ResourceType: "s3",
				TTLSeconds: 60, MaxUsage: 5,
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("Concurrent ProvisionToken(): %v", err)
	}
}
