// Package receipt defines the ANS receipt schema and canonical serialization.
// SPDX-License-Identifier: MIT
package receipt

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

type ActionType string

const (
	ActionFileRead      ActionType = "file.read"
	ActionFileWrite     ActionType = "file.write"
	ActionFileDelete    ActionType = "file.delete"
	ActionHTTPGet       ActionType = "http.get"
	ActionHTTPPost      ActionType = "http.post"
	ActionHTTPOther     ActionType = "http.other"
	ActionShellExec     ActionType = "shell.exec"
	ActionDBRead        ActionType = "db.read"
	ActionDBWrite       ActionType = "db.write"
	ActionAgentDelegate ActionType = "agent.delegate"
	ActionMemoryRead    ActionType = "memory.read"
	ActionMemoryWrite   ActionType = "memory.write"
	ActionCustom        ActionType = "custom"
)

type PolicyDecision string

const (
	PolicyAllow               PolicyDecision = "allow"
	PolicyDeny                PolicyDecision = "deny"
	PolicyAllowWithConditions PolicyDecision = "allow_with_conditions"
)

type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomePartial Outcome = "partial"
)

type Phase string

const (
	PhasePre  Phase = "pre"
	PhasePost Phase = "post"
)

// ActionPayload is hashed by the client; only the hash is stored in the receipt.
type ActionPayload struct {
	Type   ActionType        `json:"type"`
	Target string            `json:"target,omitempty"`
	Args   map[string]string `json:"args,omitempty"`
	Raw    string            `json:"raw,omitempty"`
}

// HashHex returns the hex SHA-256 of the canonical JSON of this payload.
// Returns an empty string (and logs the error) if canonical JSON serialisation fails.
func (ap ActionPayload) HashHex() string {
	b, err := canonicalJSON(ap)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h)
}

// GenesisHash is the prev_receipt_hash of the very first receipt in a chain.
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// Receipt is one signed event in the ANS audit chain.
type Receipt struct {
	// Identity — ReceiptID is set by SetReceiptID() after all other fields are populated.
	ReceiptID     string `json:"receipt_id"`
	Phase         Phase  `json:"phase"`
	AgentID       string `json:"agent_id"`
	ParentAgentID string `json:"parent_agent_id,omitempty"`

	// Chain linkage
	PrevReceiptHash string `json:"prev_receipt_hash"`
	ChainIndex      uint64 `json:"chain_index"`

	// Action
	ActionType     ActionType `json:"action_type"`
	PayloadHash    string     `json:"payload_hash"`
	PayloadSummary string     `json:"payload_summary,omitempty"`

	// Pre-action only
	PolicyDecision     PolicyDecision `json:"policy_decision,omitempty"`
	AuthorizingContext string         `json:"authorizing_context,omitempty"`

	// Snapshot reference for time-travel state restore
	SnapshotID string `json:"snapshot_id,omitempty"`

	// Post-action only
	Outcome        Outcome `json:"outcome,omitempty"`
	OutcomeSummary string  `json:"outcome_summary,omitempty"`
	DurationMS     int64   `json:"duration_ms,omitempty"`
	PreReceiptID   string  `json:"pre_receipt_id,omitempty"`

	TimestampNS int64 `json:"timestamp_ns"`
	// Signature is set by Sign() after SetReceiptID(). Not included in SignableBytes.
	Signature string `json:"signature,omitempty"`

	// cachedHash and cachedRaw are set by ComputeHash on first call.
	cachedHash string
	cachedRaw  []byte
	mu         sync.Mutex
}

// SignableBytes returns deterministic canonical JSON of all fields except
// ReceiptID and Signature. This is what gets hashed and signed.
func (r *Receipt) SignableBytes() ([]byte, error) {
	// Build the map directly with alphabetically sorted keys to avoid
	// the expensive json.Marshal → json.Unmarshal → sortedJSON cycle.
	m := map[string]interface{}{
		"action_type":       string(r.ActionType),
		"agent_id":          r.AgentID,
		"chain_index":       r.ChainIndex,
		"payload_hash":      r.PayloadHash,
		"phase":             string(r.Phase),
		"prev_receipt_hash": r.PrevReceiptHash,
		"timestamp_ns":      r.TimestampNS,
	}
	// Optional fields — only include when non-zero (mirrors omitempty)
	if r.AuthorizingContext != "" {
		m["authorizing_context"] = r.AuthorizingContext
	}
	if r.DurationMS != 0 {
		m["duration_ms"] = r.DurationMS
	}
	if r.Outcome != "" {
		m["outcome"] = string(r.Outcome)
	}
	if r.OutcomeSummary != "" {
		m["outcome_summary"] = r.OutcomeSummary
	}
	if r.ParentAgentID != "" {
		m["parent_agent_id"] = r.ParentAgentID
	}
	if r.PayloadSummary != "" {
		m["payload_summary"] = r.PayloadSummary
	}
	if r.PolicyDecision != "" {
		m["policy_decision"] = string(r.PolicyDecision)
	}
	if r.PreReceiptID != "" {
		m["pre_receipt_id"] = r.PreReceiptID
	}
	if r.SnapshotID != "" {
		m["snapshot_id"] = r.SnapshotID
	}
	return sortedJSON(m)
}

// SetReceiptID hashes SignableBytes and sets r.ReceiptID.
// Must be called before Sign().
func (r *Receipt) SetReceiptID() error {
	b, err := r.SignableBytes()
	if err != nil {
		return err
	}
	h := sha256.Sum256(b)
	r.ReceiptID = fmt.Sprintf("%x", h[:16])
	return nil
}

// ComputeHash returns the hex SHA-256 of the full marshalled receipt JSON.
// Used for hash-chaining between receipts in the chain store.
// Called AFTER Sign() so the signature is included in the chain hash.
// The result is cached on first call.
func (r *Receipt) ComputeHash() (string, error) {
	r.mu.Lock()
	if r.cachedHash != "" {
		r.mu.Unlock()
		return r.cachedHash, nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		r.mu.Unlock()
		return "", err
	}
	r.cachedRaw = b
	h := sha256.Sum256(b)
	r.cachedHash = fmt.Sprintf("%x", h)
	r.mu.Unlock()
	return r.cachedHash, nil
}

// RawJSON returns the cached raw JSON from a prior ComputeHash call.
// Returns nil if ComputeHash has not been called yet.
func (r *Receipt) RawJSON() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cachedRaw
}

// canonicalJSON serializes v with keys sorted lexicographically, no whitespace.
func canonicalJSON(v interface{}) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return sortedJSON(m)
}

func sortedJSON(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var buf bytes.Buffer
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			vb, err := sortedJSON(val[k])
			if err != nil {
				return nil, err
			}
			buf.Write(vb)
		}
		buf.WriteByte('}')
		return buf.Bytes(), nil
	case []interface{}:
		var buf bytes.Buffer
		buf.WriteByte('[')
		for i, item := range val {
			if i > 0 {
				buf.WriteByte(',')
			}
			b, err := sortedJSON(item)
			if err != nil {
				return nil, err
			}
			buf.Write(b)
		}
		buf.WriteByte(']')
		return buf.Bytes(), nil
	default:
		return json.Marshal(v)
	}
}

// Builder creates receipts with the correct chain linkage pre-set.
type Builder struct {
	agentID       string
	parentAgentID string
	prevHash      string
	chainIndex    uint64
}

// NewBuilder creates a Builder. prevHash and chainIndex come from chain.Tip().
func NewBuilder(agentID, prevHash string, chainIndex uint64) *Builder {
	return &Builder{agentID: agentID, prevHash: prevHash, chainIndex: chainIndex}
}

// WithParent sets the parent agent ID for sub-agent receipts.
func (b *Builder) WithParent(parentID string) *Builder {
	b.parentAgentID = parentID
	return b
}

// PreAction builds a pre-action receipt. Call SetReceiptID() then Sign() before Append().
func (b *Builder) PreAction(payload ActionPayload, summary string, decision PolicyDecision, authContext string) *Receipt {
	return &Receipt{
		Phase:              PhasePre,
		AgentID:            b.agentID,
		ParentAgentID:      b.parentAgentID,
		PrevReceiptHash:    b.prevHash,
		ChainIndex:         b.chainIndex,
		ActionType:         payload.Type,
		PayloadHash:        payload.HashHex(),
		PayloadSummary:     truncate(summary, 80),
		PolicyDecision:     decision,
		AuthorizingContext: truncate(authContext, 200),
		TimestampNS:        time.Now().UnixNano(),
	}
}

// PostAction builds a post-action receipt. Call SetReceiptID() then Sign() before Append().
func (b *Builder) PostAction(preReceiptID string, actionType ActionType, payloadHash, payloadSummary string, outcome Outcome, summary string, durationMS int64) *Receipt {
	return &Receipt{
		Phase:           PhasePost,
		AgentID:         b.agentID,
		ParentAgentID:   b.parentAgentID,
		PrevReceiptHash: b.prevHash,
		ChainIndex:      b.chainIndex,
		ActionType:      actionType,
		PayloadHash:     payloadHash,
		PayloadSummary:  payloadSummary,
		Outcome:         outcome,
		OutcomeSummary:  truncate(summary, 120),
		DurationMS:      durationMS,
		PreReceiptID:    preReceiptID,
		TimestampNS:     time.Now().UnixNano(),
	}
}

func truncate(s string, n int) string {
	if n < 1 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
