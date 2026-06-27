package identitybroker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// SQLiteTokenStore implements TokenStore using SQLite.
type SQLiteTokenStore struct {
	mu sync.RWMutex
	db *sql.DB
}

// NewSQLiteTokenStore creates a new SQLiteTokenStore.
func NewSQLiteTokenStore(db *sql.DB) *SQLiteTokenStore {
	return &SQLiteTokenStore{db: db}
}

// Schema returns the DDL for the tokens table.
var Schema = `
CREATE TABLE IF NOT EXISTS tokens (
    id                TEXT    PRIMARY KEY,
    agent_id          TEXT    NOT NULL,
    resource          TEXT    NOT NULL,
    resource_type     TEXT    NOT NULL,
    expires_at        INTEGER NOT NULL,
    created_at        INTEGER NOT NULL,
    last_used_at      INTEGER NOT NULL,
    usage_count       INTEGER NOT NULL DEFAULT 0,
    max_usage         INTEGER NOT NULL DEFAULT 1,
    revoked           INTEGER NOT NULL DEFAULT 0,
    revoked_at        INTEGER NOT NULL DEFAULT 0,
    revoked_by        TEXT    NOT NULL DEFAULT '',
    provider_data     TEXT    NOT NULL DEFAULT '',
    metadata          TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_tokens_agent ON tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_tokens_resource ON tokens(resource);
CREATE INDEX IF NOT EXISTS idx_tokens_expires ON tokens(expires_at);
`

// Insert stores a new token.
func (s *SQLiteTokenStore) Insert(token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerDataJSON, err := json.Marshal(token.ProviderData)
	if err != nil {
		return fmt.Errorf("marshaling provider data: %w", err)
	}

	metadataJSON, err := json.Marshal(token.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO tokens (
		id, agent_id, resource, resource_type, expires_at, created_at, last_used_at,
		usage_count, max_usage, revoked, revoked_at, revoked_by, provider_data, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.AgentID, token.Resource, token.ResourceType,
		token.ExpiresAt.UnixNano(), token.CreatedAt.UnixNano(), token.LastUsedAt.UnixNano(),
		token.UsageCount, token.MaxUsage, boolToInt(token.Revoked),
		token.RevokedAt.UnixNano(), token.RevokedBy, string(providerDataJSON), string(metadataJSON),
	)
	return err
}

// Get retrieves a token by ID.
func (s *SQLiteTokenStore) Get(id string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(
		`SELECT id, agent_id, resource, resource_type, expires_at, created_at, last_used_at,
		usage_count, max_usage, revoked, revoked_at, revoked_by, provider_data, metadata
		FROM tokens WHERE id=?`, id,
	)
	return scanToken(row)
}

// Update updates an existing token.
func (s *SQLiteTokenStore) Update(token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerDataJSON, err := json.Marshal(token.ProviderData)
	if err != nil {
		return fmt.Errorf("marshaling provider data: %w", err)
	}

	metadataJSON, err := json.Marshal(token.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	_, err = s.db.Exec(
		`UPDATE tokens SET
		agent_id=?, resource=?, resource_type=?, expires_at=?, created_at=?, last_used_at=?,
		usage_count=?, max_usage=?, revoked=?, revoked_at=?, revoked_by=?, provider_data=?, metadata=?
		WHERE id=?`,
		token.AgentID, token.Resource, token.ResourceType,
		token.ExpiresAt.UnixNano(), token.CreatedAt.UnixNano(), token.LastUsedAt.UnixNano(),
		token.UsageCount, token.MaxUsage, boolToInt(token.Revoked),
		token.RevokedAt.UnixNano(), token.RevokedBy, string(providerDataJSON), string(metadataJSON),
		token.ID,
	)
	return err
}

// Delete removes a token by ID.
func (s *SQLiteTokenStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM tokens WHERE id=?`, id)
	return err
}

// List returns tokens matching the given filters.
func (s *SQLiteTokenStore) List(agentID string, resourceType string, limit int) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, agent_id, resource, resource_type, expires_at, created_at, last_used_at,
		usage_count, max_usage, revoked, revoked_at, revoked_by, provider_data, metadata
		FROM tokens WHERE 1=1`
	args := []interface{}{}

	if agentID != "" {
		query += ` AND agent_id=?`
		args = append(args, agentID)
	}

	if resourceType != "" {
		query += ` AND resource_type=?`
		args = append(args, resourceType)
	}

	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*Token
	for rows.Next() {
		token, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

// CleanupExpired removes expired tokens.
func (s *SQLiteTokenStore) CleanupExpired() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano()
	result, err := s.db.Exec(
		`DELETE FROM tokens WHERE expires_at < ? OR (revoked = 1 AND revoked_at < ?)`, now, now,
	)
	if err != nil {
		return 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// RevokeAllForAgent revokes all tokens for an agent.
func (s *SQLiteTokenStore) RevokeAllForAgent(agentID string, revokedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixNano()
	_, err := s.db.Exec(
		`UPDATE tokens SET revoked=1, revoked_at=?, revoked_by=? WHERE agent_id=? AND revoked=0`,
		now, revokedBy, agentID,
	)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanToken(row scanner) (*Token, error) {
	var (
		id, agentID, resource, resourceType, revokedBy, providerDataJSON, metadataJSON string
		expiresAt, createdAt, lastUsedAt, revokedAt int64
		usageCount, maxUsage, revoked int
	)

	if err := row.Scan(
		&id, &agentID, &resource, &resourceType,
		&expiresAt, &createdAt, &lastUsedAt,
		&usageCount, &maxUsage, &revoked,
		&revokedAt, &revokedBy, &providerDataJSON, &metadataJSON,
	); err != nil {
		return nil, err
	}

	var providerData string
	if err := json.Unmarshal([]byte(providerDataJSON), &providerData); err != nil {
		return nil, fmt.Errorf("unmarshaling provider data for %q: %w", id, err)
	}

	var metadata string
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("unmarshaling metadata for %q: %w", id, err)
	}

	return &Token{
		ID: id, AgentID: agentID, Resource: resource, ResourceType: resourceType,
		ExpiresAt: time.Unix(0, expiresAt), CreatedAt: time.Unix(0, createdAt),
		LastUsedAt: time.Unix(0, lastUsedAt), UsageCount: usageCount, MaxUsage: maxUsage,
		Revoked: revoked != 0, RevokedAt: time.Unix(0, revokedAt), RevokedBy: revokedBy,
		ProviderData: providerData, Metadata: metadata,
	}, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
