// Package daemon — wire protocol: length-prefixed frames over Unix socket or named pipe.
//
// Frame layout: [4-byte big-endian uint32 payload_size][1-byte msg_type][N-byte JSON body]
// Size is validated BEFORE allocation to prevent memory exhaustion attacks.
// SPDX-License-Identifier: Apache-2.0
package daemon

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot"
)

const (
	MsgSignAppend     byte = 0x01
	MsgSignAppendResp byte = 0x02
	MsgVerify         byte = 0x03
	MsgVerifyResp     byte = 0x04
	MsgQuery          byte = 0x05
	MsgQueryResp      byte = 0x06
	MsgRegister       byte = 0x07
	MsgRegisterResp   byte = 0x08
	MsgStatus         byte = 0x09
	MsgStatusResp     byte = 0x0A
	MsgPing           byte = 0x0B
	MsgPong           byte = 0x0C
	MsgSnapshot            byte = 0x0D
	MsgSnapshotResp        byte = 0x0E
	MsgRestore             byte = 0x0F
	MsgRestoreResp         byte = 0x10
	MsgSnapshotList        byte = 0x11
	MsgSnapshotListResp    byte = 0x12
	MsgRegisterCompensate  byte = 0x13
	MsgRegisterCompensateResp byte = 0x14
	MsgCompensate          byte = 0x15
	MsgCompensateResp      byte = 0x16
	MsgPolicyRegister      byte = 0x17
	MsgPolicyRegisterResp  byte = 0x18
	MsgPolicyList          byte = 0x19
	MsgPolicyListResp      byte = 0x1A
	MsgPolicyDelete        byte = 0x1B
	MsgPolicyDeleteResp    byte = 0x1C
	MsgPolicyEvaluate      byte = 0x1D
	MsgPolicyEvaluateResp  byte = 0x1E
	MsgNociceptionError    byte = 0x1F
	MsgTokenRequest        byte = 0x21
	MsgTokenResp           byte = 0x22
	MsgTokenRevoke         byte = 0x23
	MsgTokenRevokeResp     byte = 0x24
	MsgTokenList           byte = 0x25
	MsgTokenListResp       byte = 0x26
	MsgMCPStart            byte = 0x31
	MsgMCPStartResp        byte = 0x32
	MsgMCPStop             byte = 0x33
	MsgMCPStopResp         byte = 0x34
	MsgMCPStatus           byte = 0x35
	MsgMCPStatusResp       byte = 0x36
	MsgMCPLog              byte = 0x37
	MsgMCPLogResp          byte = 0x38
	MsgSnapshotDiff        byte = 0x39
	MsgSnapshotDiffResp    byte = 0x3A
	MsgAuditEvents         byte = 0x3B
	MsgAuditEventsResp     byte = 0x3C
	MsgError               byte = 0xFF

	MaxFrameSize uint32 = 4 * 1024 * 1024 // 4 MB
)

type Frame struct {
	Type byte
	Body []byte
}

// WriteFrame writes [4-byte size][1-byte type][body] to w as a single write.
func WriteFrame(w io.Writer, msgType byte, body []byte) error {
	bodyLen := uint64(len(body))
	if bodyLen+1 > uint64(MaxFrameSize) {
		return fmt.Errorf("frame too large: %d (max %d)", len(body), MaxFrameSize)
	}
	payloadLen := uint32(1 + bodyLen)
	// Pre-build a single buffer to issue one syscall instead of three.
	buf := make([]byte, 5+len(body))
	binary.BigEndian.PutUint32(buf, payloadLen)
	buf[4] = msgType
	copy(buf[5:], body)
	_, err := w.Write(buf)
	return err
}

// ReadFrame reads and validates one frame. Checks MaxFrameSize BEFORE allocating.
func ReadFrame(r io.Reader) (*Frame, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(hdr)
	// Validate BEFORE allocating — prevents 4 GB allocation from malicious client.
	if size == 0 || size > MaxFrameSize {
		return nil, fmt.Errorf("invalid frame size: %d", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return &Frame{Type: buf[0], Body: buf[1:]}, nil
}

func WriteJSON(w io.Writer, msgType byte, v interface{}) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return WriteFrame(w, msgType, body)
}

func ReadJSON(r io.Reader, v interface{}) (byte, error) {
	f, err := ReadFrame(r)
	if err != nil {
		return 0, err
	}
	if f.Type == MsgError {
		var e ErrorResp
		_ = json.Unmarshal(f.Body, &e)
		return MsgError, fmt.Errorf("daemon error: %s", e.Message)
	}
	if err := json.Unmarshal(f.Body, v); err != nil {
		return f.Type, fmt.Errorf("decoding frame body: %w", err)
	}
	return f.Type, nil
}

// --- Request / Response types ---

type SignAppendReq struct {
	AgentID        string `json:"agent_id"`
	Phase          string `json:"phase"`
	ActionType     string `json:"action_type"`
	PayloadHash    string `json:"payload_hash"`
	PayloadSummary string `json:"payload_summary,omitempty"`
	PolicyDecision string `json:"policy_decision,omitempty"`
	AuthContext    string `json:"auth_context,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	OutcomeSummary string `json:"outcome_summary,omitempty"`
	DurationMS     int64  `json:"duration_ms,omitempty"`
	PreReceiptID   string `json:"pre_receipt_id,omitempty"`
	ParentAgentID  string `json:"parent_agent_id,omitempty"`
}

type SignAppendResp struct {
	ReceiptID  string `json:"receipt_id"`
	ChainIndex uint64 `json:"chain_index"`
	ChainTip   string `json:"chain_tip"`
	Signature  string `json:"signature"`
}

type VerifyReq struct {
	ReceiptID string `json:"receipt_id"`
}

type VerifyResp struct {
	Valid          bool   `json:"valid"`
	ReceiptID      string `json:"receipt_id"`
	AgentID        string `json:"agent_id"`
	AgentName      string `json:"agent_name,omitempty"`
	ActionType     string `json:"action_type"`
	Phase          string `json:"phase"`
	PolicyDecision string `json:"policy_decision,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	TimestampNS    int64  `json:"timestamp_ns"`
	ChainIndex     uint64 `json:"chain_index"`
	Error          string `json:"error,omitempty"`
}

type QueryReq struct {
	AgentID    string `json:"agent_id,omitempty"`
	ActionType string `json:"action_type,omitempty"`
	Phase      string `json:"phase,omitempty"`
	SinceNS    int64  `json:"since_ns,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

type RegisterReq struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Owner        string            `json:"owner,omitempty"`
	PublicKeyHex string            `json:"public_key_hex,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type RegisterResp struct {
	AgentID string `json:"agent_id"`
}

type StatusResp struct {
	Uptime        string    `json:"uptime"`
	ChainLength   uint64    `json:"chain_length"`
	TotalReceipts int64     `json:"total_receipts"`
	TotalAgents   int64     `json:"total_agents"`
	DBSizeBytes   int64     `json:"db_size_bytes"`
	LastReceiptTS int64     `json:"last_receipt_ts,omitempty"`
	StartedAt     time.Time `json:"started_at"`
}

type ErrorResp struct {
	Message string `json:"message"`
}

// SnapshotReq requests a state snapshot be taken at the current chain tip.
type SnapshotReq struct {
	AgentID     string `json:"agent_id"`
	SnapType    string `json:"snap_type,omitempty"`    // "filesystem", "memory", "database" (default "filesystem")
	Paths       string `json:"paths,omitempty"`        // comma-separated — files to snapshot (empty = full workspace)
}

// SnapshotResp returns metadata for the captured snapshot.
type SnapshotResp struct {
	SnapshotID  string `json:"snapshot_id"`
	ChainIndex  uint64 `json:"chain_index"`
	SnapType    string `json:"snap_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Hash        string `json:"hash"`
	StoragePath string `json:"storage_path,omitempty"`
}

// RestoreReq requests restoration to a specific chain index.
type RestoreReq struct {
	TargetIndex uint64 `json:"target_index"`
	SnapType    string `json:"snap_type,omitempty"`
}

// RestoreResp reports the result of a restore operation.
type RestoreResp struct {
	Success      bool   `json:"success"`
	TargetIndex  uint64 `json:"target_index"`
	RestoredSnap string `json:"restored_snapshot_id,omitempty"`
	Message      string `json:"message,omitempty"`
}

// SnapshotListReq lists snapshots for an agent.
type SnapshotListReq struct {
	AgentID  string `json:"agent_id"`
	SnapType string `json:"snap_type,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// SnapshotListResp returns a list of snapshots.
type SnapshotListResp struct {
	Snapshots []*snapshot.Snapshot `json:"snapshots"`
}

// SnapshotDiffReq requests a file-level diff between the 2 most recent snapshots.
type SnapshotDiffReq struct {
	AgentID     string `json:"agent_id"`
	SnapType    string `json:"snap_type,omitempty"`
}

// SnapshotDiffResp returns added, modified, and deleted files.
type SnapshotDiffResp struct {
	Added    []string `json:"added,omitempty"`
	Modified []string `json:"modified,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
	Message  string   `json:"message,omitempty"`
}

// RegisterCompensateReq registers a compensating action for a given receipt.
type RegisterCompensateReq struct {
	AgentID       string `json:"agent_id"`
	ReceiptID     string `json:"receipt_id"`
	ActionType    string `json:"action_type"`
	ReverseAction string `json:"reverse_action"`     // description of the reverse action
	ReverseCmd    string `json:"reverse_cmd"`        // shell command or URL to call
}

// RegisterCompensateResp confirms registration.
type RegisterCompensateResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// CompensateReq triggers compensating actions for a given chain index.
type CompensateReq struct {
	TargetIndex uint64 `json:"target_index"`
	DryRun      bool   `json:"dry_run,omitempty"`
}

// CompensateResp reports compensation results.
type CompensateResp struct {
	Success        bool     `json:"success"`
	ActionsRun     int      `json:"actions_run"`
	ActionsFailed  int      `json:"actions_failed"`
	Details        []string `json:"details,omitempty"`
	Message        string   `json:"message,omitempty"`
}

// PolicyRegisterReq registers a new policy.
type PolicyRegisterReq struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
	Severity    string `json:"severity,omitempty"`
	Conditions  string `json:"conditions"`  // JSON string of conditions
	Action      string `json:"action"`      // JSON string of action
}

// PolicyRegisterResp confirms registration.
type PolicyRegisterResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// PolicyListReq requests policy listing.
type PolicyListReq struct {
	EnabledOnly bool `json:"enabled_only,omitempty"`
}

// PolicyListResp returns policies.
type PolicyListResp struct {
	Policies []PolicyEntry `json:"policies"`
}

// PolicyEntry is a serializable policy entry for the wire.
type PolicyEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
	Severity    string `json:"severity,omitempty"`
	CreatedNS   int64  `json:"created_ns,omitempty"`
	UpdatedNS   int64  `json:"updated_ns,omitempty"`
}

// PolicyDeleteReq deletes a policy by ID.
type PolicyDeleteReq struct {
	ID string `json:"id"`
}

// PolicyDeleteResp confirms deletion.
type PolicyDeleteResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// PolicyEvaluateReq evaluates an action against policies.
type PolicyEvaluateReq struct {
	AgentID       string `json:"agent_id"`
	ActionType    string `json:"action_type"`
	Phase         string `json:"phase"`
	PayloadHash   string `json:"payload_hash,omitempty"`
	PayloadSummary string `json:"payload_summary,omitempty"`
	ParentAgentID string `json:"parent_agent_id,omitempty"`
}

// PolicyEvaluateResp returns evaluation results.
type PolicyEvaluateResp struct {
	Allowed       bool            `json:"allowed"`
	Denied        bool            `json:"denied"`
	Nociception   *Nociception    `json:"nociception,omitempty"`
	PolicyResults []PolicyResult  `json:"policy_results,omitempty"`
}

// Nociception describes a policy violation "pain signal".
type Nociception struct {
	PolicyID   string `json:"policy_id"`
	PolicyName string `json:"policy_name"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
}

// PolicyResult captures a single policy evaluation outcome.
type PolicyResult struct {
	PolicyID     string `json:"policy_id"`
	PolicyName   string `json:"policy_name"`
	Effect       string `json:"effect"`
	Matched      bool   `json:"matched"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// TokenRequestReq requests an ephemeral token from the identity broker.
type TokenRequestReq struct {
	AgentID    string `json:"agent_id"`
	Resource   string `json:"resource"`
	Action     string `json:"action"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
	SingleUse  bool   `json:"single_use,omitempty"`
}

// TokenRequestResp returns the provisioned token.
type TokenRequestResp struct {
	Success      bool   `json:"success"`
	TokenID      string `json:"token_id,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	AccessKey    string `json:"access_key,omitempty"`
	SecretKey    string `json:"secret_key,omitempty"`
	SessionToken string `json:"session_token,omitempty"`
	BearerToken  string `json:"bearer_token,omitempty"`
	Resource     string `json:"resource,omitempty"`
	ExpiresNS    int64  `json:"expires_ns,omitempty"`
	Message      string `json:"message,omitempty"`
}

// TokenRevokeReq revokes a token.
type TokenRevokeReq struct {
	TokenID string `json:"token_id"`
}

// TokenRevokeResp confirms revocation.
type TokenRevokeResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// TokenListReq lists tokens.
type TokenListReq struct {
	AgentID string `json:"agent_id,omitempty"`
}

// TokenListResp returns tokens.
type TokenListResp struct {
	Tokens []TokenEntry `json:"tokens"`
}

// TokenEntry is a serializable token entry for the wire (no secrets).
type TokenEntry struct {
	TokenID   string `json:"token_id"`
	Provider  string `json:"provider"`
	TokenType string `json:"token_type"`
	Resource  string `json:"resource"`
	Action    string `json:"action"`
	AgentID   string `json:"agent_id"`
	CreatedNS int64  `json:"created_ns"`
	ExpiresNS int64  `json:"expires_ns"`
	SingleUse bool   `json:"single_use"`
	State     string `json:"state"`
}

// MCPStartReq starts the MCP proxy.
type MCPStartReq struct {
	ListenAddr   string `json:"listen_addr"`
	TargetURL    string `json:"target_url"`
	SafetyDisable bool  `json:"safety_disable,omitempty"`
	RedactPII    *bool  `json:"redact_pii,omitempty"`
	RateLimit    *int   `json:"rate_limit,omitempty"`
	TokenBudget  *int   `json:"token_budget,omitempty"`
}

// MCPStartResp confirms the proxy started.
type MCPStartResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// MCPStopReq stops the MCP proxy.
type MCPStopReq struct{}

// MCPStopResp confirms the proxy stopped.
type MCPStopResp struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// MCPStatusResp returns proxy status and stats.
type MCPStatusResp struct {
	Running      bool    `json:"running"`
	ListenAddr   string  `json:"listen_addr,omitempty"`
	TargetURL    string  `json:"target_url,omitempty"`
	UptimeSecs   int64   `json:"uptime_secs"`
	TotalMsgs    int64   `json:"total_msgs"`
	TotalToks    int64   `json:"total_toks"`
	BurnRate     float64 `json:"burn_rate"`
	InjCount     int64   `json:"inj_count"`
	PrunedCount  int64   `json:"pruned_count"`
	PrunedBytes  int64   `json:"pruned_bytes"`
	Message      string  `json:"message,omitempty"`
}

// MCPLogReq requests MCP audit log entries.
type MCPLogReq struct {
	Limit  int    `json:"limit,omitempty"`
	Method string `json:"method,omitempty"`
	InjOnly bool  `json:"injections_only,omitempty"`
}

// MCPLogResp returns log entries.
type MCPLogResp struct {
	Entries []MCPLogEntry `json:"entries"`
}

// MCPLogEntry is a serializable MCP log entry.
type MCPLogEntry struct {
	ID            int64  `json:"id"`
	Direction     string `json:"direction"`
	Method        string `json:"method"`
	RequestID     string `json:"request_id,omitempty"`
	Content       string `json:"content,omitempty"`
	ToksEst       int    `json:"toks_est"`
	Injection     bool   `json:"injection"`
	InjectionType string `json:"injection_type,omitempty"`
	Pruned        bool   `json:"pruned"`
	PrunedChars   int    `json:"pruned_chars"`
	TimestampNS   int64  `json:"timestamp_ns"`
}

// AuditEventsReq requests recent daemon audit events.
type AuditEventsReq struct {
	Limit int `json:"limit,omitempty"`
}

// AuditEventsResp returns recent daemon audit events.
type AuditEventsResp struct {
	Events []AuditEventEntry `json:"events"`
}

// AuditEventEntry is a single daemon audit event.
type AuditEventEntry struct {
	TimestampNS int64  `json:"timestamp_ns"`
	Component   string `json:"component"`
	EventType   string `json:"event_type"`
	Description string `json:"description,omitempty"`
}
