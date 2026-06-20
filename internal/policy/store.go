package policy

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const policyCacheTTL = 10 * int64(time.Second) // refresh policies from DB every 10s

type cachedPolicies struct {
	policies  []*Policy
	expiresAt int64 // unix nanos
}

// Store persists policies in a SQLite database.
type Store struct {
	mu       sync.RWMutex
	db       *sql.DB
	cache    *cachedPolicies
	dirty    bool // set true on Insert/Delete, forces immediate refresh
}

// NewStore creates a policy store backed by the given DB.
// The caller must ensure the policies table exists (see Schema).
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Schema returns the DDL for the policies table.
var Schema = `
CREATE TABLE IF NOT EXISTS policies (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    enabled     INTEGER DEFAULT 1,
    priority    INTEGER DEFAULT 0,
    severity    TEXT DEFAULT 'medium',
    conditions  TEXT NOT NULL,
    action      TEXT NOT NULL,
    created_ns  INTEGER NOT NULL,
    updated_ns  INTEGER NOT NULL
);
`

// Insert saves a new policy.
func (s *Store) Insert(p *Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true
	condJSON, err := json.Marshal(p.Conditions)
	if err != nil {
		return fmt.Errorf("marshaling conditions: %w", err)
	}
	actJSON, err := json.Marshal(p.Action)
	if err != nil {
		return fmt.Errorf("marshaling action: %w", err)
	}
	now := time.Now().UnixNano()
	p.CreatedNS = now
	p.UpdatedNS = now
	_, err = s.db.Exec(
		`INSERT OR REPLACE INTO policies (id, name, description, enabled, priority, severity, conditions, action, created_ns, updated_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, boolInt(p.Enabled), p.Priority, string(p.Severity),
		string(condJSON), string(actJSON), p.CreatedNS, p.UpdatedNS,
	)
	return err
}

// Get retrieves a policy by ID.
func (s *Store) Get(id string) (*Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row := s.db.QueryRow(
		`SELECT id, name, description, enabled, priority, severity, conditions, action, created_ns, updated_ns FROM policies WHERE id=?`, id,
	)
	return scanPolicy(row)
}

// List returns all policies, ordered by priority descending.
func (s *Store) List() ([]*Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listNoLock()
}

// listNoLock is the internal implementation of List; caller must hold at least a read lock.
func (s *Store) listNoLock() ([]*Policy, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, enabled, priority, severity, conditions, action, created_ns, updated_ns FROM policies ORDER BY priority DESC, name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var policies []*Policy
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

// ListEnabled returns all enabled policies, ordered by priority descending.
// Results are cached for policyCacheTTL to avoid a SQL query on every append.
func (s *Store) ListEnabled() ([]*Policy, error) {
	s.mu.RLock()
	if s.cache != nil && !s.dirty && time.Now().UnixNano() < s.cache.expiresAt {
		c := s.cache
		s.mu.RUnlock()
		return c.policies, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock
	if s.cache != nil && !s.dirty && time.Now().UnixNano() < s.cache.expiresAt {
		return s.cache.policies, nil
	}
	s.dirty = false
	all, err := s.listNoLock()
	if err != nil {
		return nil, err
	}
	var enabled []*Policy
	for _, p := range all {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	s.cache = &cachedPolicies{
		policies:  enabled,
		expiresAt: time.Now().UnixNano() + policyCacheTTL,
	}
	return enabled, nil
}

// Delete removes a policy by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true
	_, err := s.db.Exec(`DELETE FROM policies WHERE id=?`, id)
	return err
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanPolicy(row scannable) (*Policy, error) {
	var (
		id, name, desc, sev, condStr, actStr string
		enabled                              int
		prio                                 int
		createdNS, updatedNS                 int64
	)
	if err := row.Scan(&id, &name, &desc, &enabled, &prio, &sev, &condStr, &actStr, &createdNS, &updatedNS); err != nil {
		return nil, err
	}
	p := &Policy{
		ID: id, Name: name, Description: desc,
		Enabled: enabled != 0, Priority: prio, Severity: Severity(sev),
		CreatedNS: createdNS, UpdatedNS: updatedNS,
	}
	if err := json.Unmarshal([]byte(condStr), &p.Conditions); err != nil {
		return nil, fmt.Errorf("unmarshaling conditions for %q: %w", id, err)
	}
	if err := p.Conditions.CompileRegexp(); err != nil {
		return nil, fmt.Errorf("compiling regex in conditions for %q: %w", id, err)
	}
	if err := json.Unmarshal([]byte(actStr), &p.Action); err != nil {
		return nil, fmt.Errorf("unmarshaling action for %q: %w", id, err)
	}
	return p, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
