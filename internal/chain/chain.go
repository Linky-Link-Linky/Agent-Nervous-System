// Package chain manages the ANS append-only receipt chain stored in SQLite.
// Uses modernc.org/sqlite (pure Go, no CGO) for static linking.
// SPDX-License-Identifier: MIT
package chain

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/ans-project/ans/internal/clock"
	"github.com/ans-project/ans/internal/receipt"
)

var (
	reReceiptID      = regexp.MustCompile(`^[a-f0-9]{32}$`)
	rePrevHash       = regexp.MustCompile(`^[a-f0-9]{64}$`)
	reSignature      = regexp.MustCompile(`^[a-f0-9]{128}$`)
	reAgentID        = regexp.MustCompile(`^ans_[a-zA-Z0-9_\-]{3,}$`)

	validPhases    = map[string]bool{"pre": true, "post": true}
	validOutcomes  = map[string]bool{"success": true, "failure": true, "partial": true}
	validPolicies  = map[string]bool{"allow": true, "deny": true, "allow_with_conditions": true}
	validActionTypes = map[string]bool{
		"file.read": true, "file.write": true, "file.delete": true,
		"http.get": true, "http.post": true, "http.other": true,
		"shell.exec": true, "db.read": true, "db.write": true,
		"agent.delegate": true, "memory.read": true, "memory.write": true,
		"custom": true,
	}
)

// validateReceipt checks a receipt against the ANS JSON Schema constraints.
// Returns nil on success, or a descriptive error on first violation.
func validateReceipt(r *receipt.Receipt) error {
	if !reReceiptID.MatchString(r.ReceiptID) {
		return fmt.Errorf("receipt_id: must match ^[a-f0-9]{32}$, got %q", r.ReceiptID)
	}
	if !reAgentID.MatchString(r.AgentID) {
		return fmt.Errorf("agent_id: must match ^ans_[a-zA-Z0-9_-]{3,}$, got %q", r.AgentID)
	}
	if !rePrevHash.MatchString(r.PrevReceiptHash) && r.PrevReceiptHash != receipt.GenesisHash {
		return fmt.Errorf("prev_receipt_hash: must match ^[a-f0-9]{64}$, got %q", r.PrevReceiptHash)
	}
	if !validPhases[string(r.Phase)] {
		return fmt.Errorf("phase: must be \"pre\" or \"post\", got %q", r.Phase)
	}
	if !validActionTypes[string(r.ActionType)] {
		return fmt.Errorf("action_type: invalid, got %q", r.ActionType)
	}
	if r.ChainIndex < 1 {
		return fmt.Errorf("chain_index: must be >= 1, got %d", r.ChainIndex)
	}
	if len(r.PayloadSummary) > 80 {
		return fmt.Errorf("payload_summary: max length 80, got %d", len(r.PayloadSummary))
	}
	if len(r.AuthorizingContext) > 200 {
		return fmt.Errorf("authorizing_context: max length 200, got %d", len(r.AuthorizingContext))
	}
	if len(r.OutcomeSummary) > 120 {
		return fmt.Errorf("outcome_summary: max length 120, got %d", len(r.OutcomeSummary))
	}
	if r.PolicyDecision != "" && !validPolicies[string(r.PolicyDecision)] {
		return fmt.Errorf("policy_decision: invalid, got %q", r.PolicyDecision)
	}
	if r.Outcome != "" && !validOutcomes[string(r.Outcome)] {
		return fmt.Errorf("outcome: invalid, got %q", r.Outcome)
	}
	if r.Signature != "" && !reSignature.MatchString(r.Signature) {
		return fmt.Errorf("signature: must match ^[a-f0-9]{128}$, got len=%d", len(r.Signature))
	}
	if r.PreReceiptID != "" && !reReceiptID.MatchString(r.PreReceiptID) {
		return fmt.Errorf("pre_receipt_id: must match ^[a-f0-9]{32}$, got %q", r.PreReceiptID)
	}
	if r.ParentAgentID != "" && !reAgentID.MatchString(r.ParentAgentID) {
		return fmt.Errorf("parent_agent_id: must match ^ans_[a-zA-Z0-9_-]{3,}$, got %q", r.ParentAgentID)
	}
	if r.DurationMS < 0 {
		return fmt.Errorf("duration_ms: must be >= 0, got %d", r.DurationMS)
	}
	return nil
}

const schema = `
CREATE TABLE IF NOT EXISTS receipts (
	chain_index  INTEGER PRIMARY KEY,
	receipt_id   TEXT    NOT NULL UNIQUE,
	agent_id     TEXT    NOT NULL,
	phase        TEXT    NOT NULL,
	action_type  TEXT    NOT NULL,
	prev_hash    TEXT    NOT NULL,
	receipt_hash TEXT    NOT NULL,
	timestamp_ns INTEGER NOT NULL,
	raw_receipt  TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent  ON receipts(agent_id);
CREATE INDEX IF NOT EXISTS idx_phase  ON receipts(phase);
CREATE INDEX IF NOT EXISTS idx_action ON receipts(action_type);
CREATE INDEX IF NOT EXISTS idx_ts     ON receipts(timestamp_ns);
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
`

// snapshotSchema creates the snapshots table for time-travel state restore.
// Kept here to avoid a circular dependency with the snapshot package.
const snapshotSchema = `
CREATE TABLE IF NOT EXISTS snapshots (
	snapshot_id   TEXT    NOT NULL PRIMARY KEY,
	chain_index   INTEGER NOT NULL,
	agent_id      TEXT    NOT NULL,
	receipt_id    TEXT    NOT NULL DEFAULT '',
	snap_type     TEXT    NOT NULL,
	storage_path  TEXT    NOT NULL,
	size_bytes    INTEGER NOT NULL DEFAULT 0,
	hash          TEXT    NOT NULL,
	timestamp_ns  INTEGER NOT NULL,
	metadata      TEXT    NOT NULL DEFAULT '{}'
);
	CREATE INDEX IF NOT EXISTS idx_snap_agent ON snapshots(agent_id);
	CREATE INDEX IF NOT EXISTS idx_snap_chain  ON snapshots(chain_index);
	CREATE INDEX IF NOT EXISTS idx_snap_type   ON snapshots(snap_type);
	`
const compensationSchema = `
CREATE TABLE IF NOT EXISTS compensations (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	chain_index   INTEGER NOT NULL,
	agent_id      TEXT    NOT NULL,
	receipt_id    TEXT    NOT NULL,
	action_type   TEXT    NOT NULL,
	reverse_action TEXT   NOT NULL DEFAULT '',
	reverse_cmd   TEXT   NOT NULL DEFAULT '',
	executed      INTEGER NOT NULL DEFAULT 0,
	created_ns    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_comp_chain ON compensations(chain_index);
`
const brokerSchema = `
CREATE TABLE IF NOT EXISTS broker_tokens (
	id            TEXT PRIMARY KEY,
	agent_id      TEXT NOT NULL,
	provider      TEXT NOT NULL DEFAULT '',
	resource      TEXT NOT NULL DEFAULT '',
	access_key    TEXT NOT NULL,
	secret_key    TEXT NOT NULL,
	session_token TEXT DEFAULT '',
	region        TEXT DEFAULT '',
	expires_at    INTEGER NOT NULL,
	used          INTEGER DEFAULT 0,
	created_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_broker_agent ON broker_tokens(agent_id);
CREATE INDEX IF NOT EXISTS idx_broker_exp  ON broker_tokens(expires_at);
CREATE TABLE IF NOT EXISTS broker_providers (
	name          TEXT PRIMARY KEY,
	provider_type TEXT NOT NULL,
	config_json   TEXT DEFAULT '{}',
	enabled       INTEGER DEFAULT 1
);
`
const policySchema = `
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
const mcpSchema = `
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

// Chain is the local append-only receipt chain.
type Chain struct {
	db       *sql.DB
	mu       sync.Mutex
	lastIdx  uint64
	lastHash string
	hlc      clock.HLC
}

// Open opens (or creates) the chain at path. Empty path → ~/.ans/chain.db.
func Open(path string) (*Chain, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".ans", "chain.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	db.SetMaxOpenConns(1)
	// Apply main schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}
	// Apply anchor schema (for chain pruning — defined in prune.go, same package)
	if _, err := db.Exec(anchorSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying anchor schema: %w", err)
	}
	// Migration: add merkle_tree column to anchors for pre-existing databases
	db.Exec(`ALTER TABLE anchors ADD COLUMN merkle_tree TEXT NOT NULL DEFAULT ''`)
	// Apply snapshot schema (for time-travel state restore)
	if _, err := db.Exec(snapshotSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying snapshot schema: %w", err)
	}
	// Apply compensation schema (for compensating actions on rollback)
	if _, err := db.Exec(compensationSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying compensation schema: %w", err)
	}
	// Apply policy schema (for Policy-as-Code enforcement)
	if _, err := db.Exec(policySchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying policy schema: %w", err)
	}
	// Apply MCP audit log schema
	if _, err := db.Exec(mcpSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying mcp schema: %w", err)
	}
	// Apply broker schema (for ephemeral identity provisioning)
	if _, err := db.Exec(brokerSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying broker schema: %w", err)
	}
	c := &Chain{db: db, lastHash: receipt.GenesisHash}
	row := db.QueryRow(`SELECT chain_index, receipt_hash FROM receipts ORDER BY chain_index DESC LIMIT 1`)
	var idx uint64
	var hash string
	if err := row.Scan(&idx, &hash); err == nil {
		c.lastIdx = idx
		c.lastHash = hash
	}
	return c, nil
}

// Close closes the database.
func (c *Chain) Close() error { return c.db.Close() }

// DB returns the underlying SQLite database handle.
func (c *Chain) DB() *sql.DB { return c.db }

// AppendNew atomically reads the tip, calls buildFn to create a receipt,
// signs it, and inserts it — all under the chain mutex.
// This eliminates the TOCTOU race between tip-read and insert.
// buildFn receives (prevHash, nextIdx) and returns an unsigned receipt with all
// other fields populated. AppendNew calls Sign and ComputeHash internally.
//
// SECURITY: after buildFn returns, AppendNew validates that the receipt's
// ChainIndex and PrevReceiptHash actually match the (prevHash, nextIdx) values
// that were passed in. This is defense-in-depth: buildFn closures are trusted
// code today, but any future caller, test helper, or refactor that constructs
// a receipt independently of the supplied parameters must not be able to
// silently corrupt the chain. A mismatch is a programming error and is
// rejected before any database write occurs.
func (c *Chain) AppendNew(
	buildFn func(prevHash string, nextIdx uint64) (*receipt.Receipt, error),
	signer *receipt.Signer,
) (*receipt.Receipt, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	nextIdx := c.lastIdx + 1
	expectedPrevHash := c.lastHash
	r, err := buildFn(expectedPrevHash, nextIdx)
	if err != nil {
		return nil, fmt.Errorf("building receipt: %w", err)
	}
	if r.ChainIndex != nextIdx {
		return nil, fmt.Errorf(
			"buildFn returned ChainIndex %d, expected %d (chain tip mismatch)",
			r.ChainIndex, nextIdx)
	}
	if r.PrevReceiptHash != expectedPrevHash {
		return nil, fmt.Errorf(
			"buildFn returned PrevReceiptHash %q, expected %q (chain tip mismatch)",
			r.PrevReceiptHash, expectedPrevHash)
	}
	// Stamp with HLC timestamp for monotonicity across the chain
	r.TimestampNS = c.hlc.Now()
	if err := signer.Sign(r); err != nil {
		return nil, fmt.Errorf("signing: %w", err)
	}
	if err := validateReceipt(r); err != nil {
		return nil, fmt.Errorf("schema validation: %w", err)
	}
	hash, err := r.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("computing hash: %w", err)
	}
	_, err = c.db.Exec(
		`INSERT INTO receipts
		(chain_index,receipt_id,agent_id,phase,action_type,prev_hash,receipt_hash,timestamp_ns,raw_receipt)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		r.ChainIndex, r.ReceiptID, r.AgentID, string(r.Phase),
		string(r.ActionType), r.PrevReceiptHash, hash, r.TimestampNS, string(r.RawJSON()),
	)
	if err != nil {
		return nil, fmt.Errorf("inserting: %w", err)
	}
	c.lastIdx = r.ChainIndex
	c.lastHash = hash
	return r, nil
}

// Tip returns the current (lastIdx, lastHash). Read-only; safe for status display.
func (c *Chain) Tip() (uint64, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastIdx, c.lastHash
}

// GetByIndex retrieves a receipt by chain_index (1-based).
func (c *Chain) GetByIndex(idx uint64) (*receipt.Receipt, error) {
	var raw string
	if err := c.db.QueryRow(`SELECT raw_receipt FROM receipts WHERE chain_index=?`, idx).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("receipt at index %d not found", idx)
		}
		return nil, err
	}
	var r receipt.Receipt
	return &r, json.Unmarshal([]byte(raw), &r)
}

// Get retrieves a receipt by receipt_id.
func (c *Chain) Get(receiptID string) (*receipt.Receipt, error) {
	var raw string
	if err := c.db.QueryRow(`SELECT raw_receipt FROM receipts WHERE receipt_id=?`, receiptID).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("receipt %s not found", receiptID)
		}
		return nil, err
	}
	var r receipt.Receipt
	return &r, json.Unmarshal([]byte(raw), &r)
}

// QueryOptions are filters for List.
type QueryOptions struct {
	AgentID    string
	ActionType string
	Phase      string
	Since      time.Time
	Limit      int
	Offset     int
}

// List returns receipts matching opts, ordered by chain_index DESC.
func (c *Chain) List(opts QueryOptions) ([]*receipt.Receipt, error) {
	q := `SELECT raw_receipt FROM receipts WHERE 1=1`
	args := []interface{}{}
	if opts.AgentID != "" {
		q += " AND agent_id=?"
		args = append(args, opts.AgentID)
	}
	if opts.ActionType != "" {
		q += " AND action_type=?"
		args = append(args, opts.ActionType)
	}
	if opts.Phase != "" {
		q += " AND phase=?"
		args = append(args, opts.Phase)
	}
	if !opts.Since.IsZero() {
		q += " AND timestamp_ns>=?"
		args = append(args, opts.Since.UnixNano())
	}
	q += " ORDER BY chain_index DESC"
	if opts.Limit > 0 {
		if opts.Offset < 0 {
			opts.Offset = 0
		}
		q += " LIMIT ? OFFSET ?"
		args = append(args, opts.Limit, opts.Offset)
	}
	rows, err := c.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receipts []*receipt.Receipt
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r receipt.Receipt
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, err
		}
		receipts = append(receipts, &r)
	}
	return receipts, rows.Err()
}

// Count returns total receipts.
func (c *Chain) Count() (int64, error) {
	var n int64
	return n, c.db.QueryRow(`SELECT COUNT(*) FROM receipts`).Scan(&n)
}

// Stats holds aggregate chain statistics.
type Stats struct {
	TotalReceipts   int64
	TotalAgents     int64
	ChainLength     uint64
	OldestReceiptNS int64
	NewestReceiptNS int64
	DBSizeBytes     int64
}

// GetStats returns chain statistics. Uses sql.NullInt64 so empty chains return
// zero values rather than errors.
func (c *Chain) GetStats() (Stats, error) {
	var s Stats
	c.mu.Lock()
	s.ChainLength = c.lastIdx
	c.mu.Unlock()

	var oldest, newest sql.NullInt64
	if err := c.db.QueryRow(
		`SELECT COUNT(*), COUNT(DISTINCT agent_id), MIN(timestamp_ns), MAX(timestamp_ns) FROM receipts`,
	).Scan(&s.TotalReceipts, &s.TotalAgents, &oldest, &newest); err != nil {
		return s, err
	}
	if oldest.Valid {
		s.OldestReceiptNS = oldest.Int64
	}
	if newest.Valid {
		s.NewestReceiptNS = newest.Int64
	}
	var pageCount, pageSize int64
	_ = c.db.QueryRow(`PRAGMA page_count`).Scan(&pageCount)
	_ = c.db.QueryRow(`PRAGMA page_size`).Scan(&pageSize)
	s.DBSizeBytes = pageCount * pageSize

	return s, nil
}

// CompensationRecord stores a compensating action for a receipt.
type CompensationRecord struct {
	ID           int64  `json:"id"`
	ChainIndex   uint64 `json:"chain_index"`
	AgentID      string `json:"agent_id"`
	ReceiptID    string `json:"receipt_id"`
	ActionType   string `json:"action_type"`
	ReverseAction string `json:"reverse_action"`
	ReverseCmd   string `json:"reverse_cmd"`
	Executed     bool   `json:"executed"`
	CreatedNS    int64  `json:"created_ns"`
}

// SaveCompensation persists a compensating action.
func (c *Chain) SaveCompensation(rc *CompensationRecord) error {
	_, err := c.db.Exec(
		`INSERT INTO compensations (chain_index, agent_id, receipt_id, action_type, reverse_action, reverse_cmd, executed, created_ns)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rc.ChainIndex, rc.AgentID, rc.ReceiptID, rc.ActionType,
		rc.ReverseAction, rc.ReverseCmd, boolToInt(rc.Executed), rc.CreatedNS,
	)
	return err
}

// GetCompensations returns all compensations at or above the given chain index.
func (c *Chain) GetCompensations(fromIndex uint64) ([]CompensationRecord, error) {
	rows, err := c.db.Query(
		`SELECT id, chain_index, agent_id, receipt_id, action_type, reverse_action, reverse_cmd, executed, created_ns
		FROM compensations WHERE chain_index >= ? ORDER BY chain_index DESC`, fromIndex,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []CompensationRecord
	for rows.Next() {
		var r CompensationRecord
		var exec int
		if err := rows.Scan(&r.ID, &r.ChainIndex, &r.AgentID, &r.ReceiptID, &r.ActionType,
			&r.ReverseAction, &r.ReverseCmd, &exec, &r.CreatedNS); err != nil {
			return nil, err
		}
		r.Executed = exec != 0
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// MarkCompensationExecuted marks a compensation as executed.
func (c *Chain) MarkCompensationExecuted(id int64) error {
	_, err := c.db.Exec(`UPDATE compensations SET executed=1 WHERE id=?`, id)
	return err
}

func boolToInt(b bool) int {
	if b { return 1 }
	return 0
}