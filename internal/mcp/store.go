package mcp

import (
	"database/sql"
	"sync"
	"time"
)

// AuditStore persists MCP traffic logs.
type AuditStore struct {
	mu sync.RWMutex
	db *sql.DB
}

// NewAuditStore creates an audit store backed by SQLite.
func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Schema DDL for the mcp_log table.
const Schema = `
CREATE TABLE IF NOT EXISTS mcp_log (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    direction      TEXT NOT NULL,
    method         TEXT NOT NULL DEFAULT '',
    request_id     TEXT NOT NULL DEFAULT '',
    content        TEXT NOT NULL DEFAULT '',
    toks_est       INTEGER NOT NULL DEFAULT 0,
    injection      INTEGER NOT NULL DEFAULT 0,
    injection_type TEXT NOT NULL DEFAULT '',
    pruned         INTEGER NOT NULL DEFAULT 0,
    pruned_chars   INTEGER NOT NULL DEFAULT 0,
    timestamp_ns   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mcp_ts    ON mcp_log(timestamp_ns);
CREATE INDEX IF NOT EXISTS idx_mcp_method ON mcp_log(method);
CREATE INDEX IF NOT EXISTS idx_mcp_inj   ON mcp_log(injection);
`

// Insert saves a log entry.
func (as *AuditStore) Insert(entry *LogEntry) error {
	as.mu.Lock()
	defer as.mu.Unlock()
	_, err := as.db.Exec(
		`INSERT INTO mcp_log (direction, method, request_id, content, toks_est, injection, injection_type, pruned, pruned_chars, timestamp_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.Direction, entry.Method, entry.RequestID, entry.Content,
		entry.ToksEst, boolInt(entry.Injection), entry.InjectionTy,
		boolInt(entry.Pruned), entry.PrunedChars, entry.TimestampNS,
	)
	return err
}

// QueryRecent returns the N most recent log entries.
func (as *AuditStore) QueryRecent(limit int) ([]*LogEntry, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	rows, err := as.db.Query(
		`SELECT id, direction, method, request_id, content, toks_est, injection, injection_type, pruned, pruned_chars, timestamp_ns
		FROM mcp_log ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*LogEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// QueryByMethod returns log entries filtered by method.
func (as *AuditStore) QueryByMethod(method string, limit int) ([]*LogEntry, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	rows, err := as.db.Query(
		`SELECT id, direction, method, request_id, content, toks_est, injection, injection_type, pruned, pruned_chars, timestamp_ns
		FROM mcp_log WHERE method=? ORDER BY id DESC LIMIT ?`, method, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*LogEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// QueryInjections returns all entries with detected injections.
func (as *AuditStore) QueryInjections(limit int) ([]*LogEntry, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()
	if limit <= 0 {
		limit = 100
	}
	rows, err := as.db.Query(
		`SELECT id, direction, method, request_id, content, toks_est, injection, injection_type, pruned, pruned_chars, timestamp_ns
		FROM mcp_log WHERE injection=1 ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*LogEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetStats computes summary statistics from the log.
func (as *AuditStore) GetStats(uptimeSeconds int64) (*Stats, error) {
	as.mu.RLock()
	defer as.mu.RUnlock()

	s := &Stats{}
	row := as.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(toks_est),0), COALESCE(SUM(injection),0), COALESCE(SUM(pruned),0), COALESCE(SUM(pruned_chars),0) FROM mcp_log`)
	if err := row.Scan(&s.TotalMessages, &s.TotalToks, &s.InjectionCount, &s.PrunedCount, &s.PrunedBytes); err != nil {
		return nil, err
	}
	s.UptimeSeconds = uptimeSeconds

	// Token burn rate: tokens in last 60s
	var recentToks int64
	cutoff := time.Now().UnixNano() - 60*1e9
	if err := as.db.QueryRow(`SELECT COALESCE(SUM(toks_est),0) FROM mcp_log WHERE timestamp_ns > ?`, cutoff).Scan(&recentToks); err != nil {
		recentToks = 0
	}
	if uptimeSeconds > 0 {
		s.TokenBurnRate = float64(recentToks) / 60.0
	}

	return s, nil
}

type entryScannable interface {
	Scan(dest ...interface{}) error
}

func scanEntry(row entryScannable) (*LogEntry, error) {
	var (
		id                                   int64
		direction, method, reqID, content    string
		toksEst                              int
		injection                            int
		injectionType                        string
		pruned                               int
		prunedChars                          int
		timestampNS                          int64
	)
	if err := row.Scan(&id, &direction, &method, &reqID, &content, &toksEst, &injection, &injectionType, &pruned, &prunedChars, &timestampNS); err != nil {
		return nil, err
	}
	return &LogEntry{
		ID: id, Direction: Direction(direction), Method: method, RequestID: reqID,
		Content: content, ToksEst: toksEst, Injection: injection != 0,
		InjectionTy: injectionType, Pruned: pruned != 0, PrunedChars: prunedChars,
		TimestampNS: timestampNS,
	}, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
