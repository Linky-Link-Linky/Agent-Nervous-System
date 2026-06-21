// Package daemon — request handlers for each protocol message type.
// SPDX-License-Identifier: Apache-2.0
package daemon

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/broker"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identity"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/mcp"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/policy"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot"
)

// execCommand is a variable so tests can mock it.
var execCommandContext = exec.CommandContext

// safeCmdPattern allows only alphanumeric, spaces, common path chars, and shell-safe punctuation.
var safeCmdPattern = regexp.MustCompile(`^[a-zA-Z0-9_\-\/\\\.\:\@\,\=\+\~\%\s]+$`)

// writeOK writes a JSON response frame to conn. Connection errors are expected
// (e.g. client disconnect) and silently discarded after logging.
func writeOK(w io.Writer, msgType byte, v interface{}) {
	if err := WriteJSON(w, msgType, v); err != nil {
		fmt.Fprintf(os.Stderr, "ans: write response: %v\n", err)
	}
}

type Handler struct{ daemon *Daemon }

func (h *Handler) Handle(ctx context.Context, conn net.Conn) {
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(15 * time.Second)
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		f, err := ReadFrame(conn)
		if err != nil {
			return
		}
		// Re-arm the deadline for the next read (never zero — prevents indefinite hold)
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		switch f.Type {
		case MsgPing:
			if err := WriteFrame(conn, MsgPong, nil); err != nil {
				fmt.Fprintf(os.Stderr, "ans: write pong: %v\n", err)
			}
		case MsgSignAppend:
			h.handleSignAppend(conn, f.Body)
		case MsgVerify:
			h.handleVerify(conn, f.Body)
		case MsgQuery:
			h.handleQuery(conn, f.Body)
		case MsgRegister:
			h.handleRegister(conn, f.Body)
		case MsgStatus:
			h.handleStatus(conn)
		case MsgSnapshot:
			h.handleSnapshot(conn, f.Body)
		case MsgRestore:
			h.handleRestore(conn, f.Body)
		case MsgRegisterCompensate:
			h.handleRegisterCompensate(conn, f.Body)
		case MsgCompensate:
			h.handleCompensate(conn, f.Body)
		case MsgPolicyRegister:
			h.handlePolicyRegister(conn, f.Body)
		case MsgPolicyList:
			h.handlePolicyList(conn, f.Body)
		case MsgPolicyDelete:
			h.handlePolicyDelete(conn, f.Body)
		case MsgPolicyEvaluate:
			h.handlePolicyEvaluate(conn, f.Body)
		case MsgTokenRequest:
			h.handleTokenRequest(conn, f.Body, ctx)
		case MsgTokenRevoke:
			h.handleTokenRevoke(conn, f.Body, ctx)
		case MsgTokenList:
			h.handleTokenList(conn, f.Body)
		case MsgMCPStart:
			h.handleMCPStart(conn, f.Body)
		case MsgMCPStop:
			h.handleMCPStop(conn, f.Body)
		case MsgMCPStatus:
			h.handleMCPStatus(conn, f.Body)
		case MsgMCPLog:
			h.handleMCPLog(conn, f.Body)
		case MsgSnapshotList:
			h.handleSnapshotList(conn, f.Body)
		case MsgSnapshotDiff:
			h.handleSnapshotDiff(conn, f.Body)
		default:
			writeOK(conn, MsgError, ErrorResp{Message: fmt.Sprintf("unknown msg type 0x%02x", f.Type)})
		}
	}
}

func (h *Handler) handleSignAppend(conn net.Conn, body []byte) {
	var req SignAppendReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	if req.AgentID == "" || req.ActionType == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "agent_id, action_type required"})
		return
	}
	if req.Phase == "post" && req.PayloadHash == "" && req.PreReceiptID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "payload_hash or pre_receipt_id required for post phase"})
		return
	}
	if len(req.AgentID) > 128 || len(req.ActionType) > 64 || len(req.PayloadHash) > 128 ||
		len(req.PayloadSummary) > 200 || len(req.AuthContext) > 500 || len(req.OutcomeSummary) > 500 {
		writeOK(conn, MsgError, ErrorResp{Message: "field too long"})
		return
	}
	if req.Phase != "pre" && req.Phase != "post" {
		writeOK(conn, MsgError, ErrorResp{Message: "phase must be 'pre' or 'post'"})
		return
	}
	if req.Phase == "pre" && req.PolicyDecision != "" &&
		req.PolicyDecision != "allow" && req.PolicyDecision != "deny" && req.PolicyDecision != "allow_with_conditions" {
		writeOK(conn, MsgError, ErrorResp{Message: "policy_decision must be 'allow', 'deny', or 'allow_with_conditions'"})
		return
	}
	if req.Phase == "post" && req.Outcome != "" &&
		req.Outcome != "success" && req.Outcome != "failure" && req.Outcome != "partial" {
		writeOK(conn, MsgError, ErrorResp{Message: "outcome must be 'success', 'failure', or 'partial'"})
		return
	}
	if req.DurationMS < 0 {
		writeOK(conn, MsgError, ErrorResp{Message: "duration_ms must be >= 0"})
		return
	}
	// Evaluate action against dynamic policies (Biological Immune System)
	ctx := map[string]interface{}{
		"model.weight_type": "open",
	}
	facts := policy.MakeFacts(req.AgentID, req.ActionType, req.Phase, req.PayloadSummary, req.ParentAgentID, ctx)
	evalRes := h.daemon.policyExec.Evaluate(facts)
	if !evalRes.Allowed {
		n := evalRes.Nociception
		if n != nil {
			writeOK(conn, MsgNociceptionError, Nociception{
				PolicyID: n.PolicyID, PolicyName: n.PolicyName,
				Message: n.Message, Severity: n.Severity,
			})
		} else {
			writeOK(conn, MsgError, ErrorResp{Message: "action denied by policy"})
		}
		return
	}
	agent, err := h.daemon.keystore.Load(req.AgentID)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "agent not found: " + err.Error()})
		return
	}
	signer := receipt.NewSigner(agent.PrivateKey)

	// AppendNew holds the chain mutex for tip-read + build + sign + insert atomically.
	r, err := h.daemon.chain.AppendNew(func(prevHash string, nextIdx uint64) (*receipt.Receipt, error) {
		b := receipt.NewBuilder(req.AgentID, prevHash, nextIdx)
		if req.ParentAgentID != "" {
			b = b.WithParent(req.ParentAgentID)
		}
		if req.Phase == "pre" {
			payload := receipt.ActionPayload{Type: receipt.ActionType(req.ActionType)}
			r := b.PreAction(payload, req.PayloadSummary, receipt.PolicyDecision(req.PolicyDecision), req.AuthContext)
			r.PayloadHash = req.PayloadHash // use client-computed hash
			return r, nil
		}
		return b.PostAction(
			req.PreReceiptID, receipt.ActionType(req.ActionType),
			req.PayloadHash, req.PayloadSummary,
			receipt.Outcome(req.Outcome), req.OutcomeSummary, req.DurationMS,
		), nil
	}, signer)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "append failed: " + err.Error()})
		return
	}
	tip, err := r.ComputeHash()
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "compute hash failed: " + err.Error()})
		return
	}
	writeOK(conn, MsgSignAppendResp, SignAppendResp{
		ReceiptID: r.ReceiptID, ChainIndex: r.ChainIndex, ChainTip: tip, Signature: r.Signature,
	})
	h.daemon.afterAppend(r.RawJSON())
}

func (h *Handler) handleVerify(conn net.Conn, body []byte) {
	var req VerifyReq
	if err := json.Unmarshal(body, &req); err != nil || req.ReceiptID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "receipt_id required"})
		return
	}
	r, err := h.daemon.chain.Get(req.ReceiptID)
	if err != nil {
		writeOK(conn, MsgVerifyResp, VerifyResp{Valid: false, ReceiptID: req.ReceiptID, Error: err.Error()})
		return
	}
	var pubkeys map[string]ed25519.PublicKey
	if ag, err := h.daemon.keystore.Load(r.AgentID); err == nil {
		pubkeys = map[string]ed25519.PublicKey{r.AgentID: ag.PublicKey}
	}
	verifyErr := h.daemon.chain.VerifyReceipt(req.ReceiptID, pubkeys)
	resp := VerifyResp{
		Valid: verifyErr == nil, ReceiptID: r.ReceiptID, AgentID: r.AgentID,
		ActionType: string(r.ActionType), Phase: string(r.Phase),
		PolicyDecision: string(r.PolicyDecision), Outcome: string(r.Outcome),
		TimestampNS: r.TimestampNS, ChainIndex: r.ChainIndex,
	}
	if verifyErr != nil {
		resp.Error = verifyErr.Error()
	}
	if ag, err := h.daemon.keystore.Load(r.AgentID); err == nil {
		resp.AgentName = ag.Name
	}
	writeOK(conn, MsgVerifyResp, resp)
}

func (h *Handler) handleQuery(conn net.Conn, body []byte) {
	var req QueryReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	opts := chain.QueryOptions{AgentID: req.AgentID, ActionType: req.ActionType, Phase: req.Phase, Limit: limit, Offset: req.Offset}
	if req.SinceNS > 0 {
		opts.Since = time.Unix(0, req.SinceNS)
	}
	receipts, err := h.daemon.chain.List(opts)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: err.Error()})
		return
	}
	writeOK(conn, MsgQueryResp, map[string]interface{}{"receipts": receipts})
}

func (h *Handler) handleRegister(conn net.Conn, body []byte) {
	var req RegisterReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "invalid JSON: " + err.Error()})
		return
	}
	if req.Name == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "name required"})
		return
	}
	if len(req.Name) > 128 || len(req.Version) > 32 || len(req.Owner) > 128 || len(req.PublicKeyHex) > 256 {
		writeOK(conn, MsgError, ErrorResp{Message: "field too long"})
		return
	}
	var ag *identity.Agent
	if req.PublicKeyHex != "" {
		pubBytes, err := hex.DecodeString(req.PublicKeyHex)
		if err != nil || len(pubBytes) != ed25519.PublicKeySize {
			writeOK(conn, MsgError, ErrorResp{Message: "invalid public_key_hex"})
			return
		}
		ag = identity.NewFromKeys(req.Name, req.Version, req.Owner, ed25519.PublicKey(pubBytes), nil)
	} else {
		var err error
		ag, err = identity.New(req.Name, req.Version, req.Owner)
		if err != nil {
			writeOK(conn, MsgError, ErrorResp{Message: "keygen failed: " + err.Error()})
			return
		}
	}
	if err := h.daemon.keystore.Save(ag); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "save failed: " + err.Error()})
		return
	}
	writeOK(conn, MsgRegisterResp, RegisterResp{AgentID: ag.ID})
}

func (h *Handler) handleSnapshot(conn net.Conn, body []byte) {
	var req SnapshotReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad snapshot request: " + err.Error()})
		return
	}
	if req.AgentID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "agent_id required for snapshot"})
		return
	}
	if len(req.AgentID) > 128 || len(req.SnapType) > 32 || len(req.Paths) > 4096 {
		writeOK(conn, MsgError, ErrorResp{Message: "field too long"})
		return
	}
	snapType := snapshot.SnapFileSystem
	if req.SnapType != "" {
		snapType = snapshot.SnapType(req.SnapType)
	}
	idx, _ := h.daemon.chain.Tip()
	sn, err := h.daemon.snapStore.Capture(snapType, req.AgentID, idx, "")
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "snapshot failed: " + err.Error()})
		return
	}
	writeOK(conn, MsgSnapshotResp, SnapshotResp{
		SnapshotID: sn.ID, ChainIndex: sn.ChainIndex,
		SnapType: string(sn.SnapType), SizeBytes: sn.SizeBytes, Hash: sn.Hash,
	})
}

func (h *Handler) handleRestore(conn net.Conn, body []byte) {
	var req RestoreReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad restore request: " + err.Error()})
		return
	}
	if req.TargetIndex == 0 {
		writeOK(conn, MsgError, ErrorResp{Message: "target_index must be > 0"})
		return
	}
	if len(req.SnapType) > 32 {
		writeOK(conn, MsgError, ErrorResp{Message: "snap_type too long"})
		return
	}
	snapType := snapshot.SnapFileSystem
	if req.SnapType != "" {
		snapType = snapshot.SnapType(req.SnapType)
	}
	if err := h.daemon.snapStore.Restore(snapType, req.TargetIndex); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "restore failed: " + err.Error()})
		return
	}
	writeOK(conn, MsgRestoreResp, RestoreResp{
		Success: true, TargetIndex: req.TargetIndex, Message: "state restored",
	})
}

func (h *Handler) handleTokenRequest(conn net.Conn, body []byte, ctx context.Context) {
	var req TokenRequestReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	if req.AgentID == "" || req.Resource == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "agent_id and resource required"})
		return
	}
	action := req.Action
	if action == "" {
		action = "read"
	}
	ttl := req.TTLSeconds
	if ttl <= 0 {
		ttl = 60
	}
	if ttl > 60 {
		ttl = 60
	}
	scope := broker.Scope{
		Resource:    req.Resource,
		Permissions: []string{action},
	}
	provReq := &broker.ProvisionRequest{
		AgentID:    req.AgentID,
		ActionType: action,
		Scope:      scope,
		TTLSeconds: ttl,
	}
	// Try registered providers in order
	var (
		cred *broker.Credential
		provErr error
	)
	for _, name := range []string{"dev", "env"} {
		cred, provErr = h.daemon.broker.Provision(ctx, name, provReq)
		if provErr == nil {
			break
		}
	}
	if provErr != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "provisioning failed: " + provErr.Error()})
		return
	}
	writeOK(conn, MsgTokenResp, TokenRequestResp{
		Success:      true,
		TokenID:      cred.CredentialID,
		TokenType:    cred.Type,
		AccessKey:    cred.Metadata["access_key"],
		SecretKey:    cred.Secret,
		SessionToken: cred.Metadata["session_token"],
		Resource:     cred.Scope.Resource,
		ExpiresNS:    cred.ExpiresAt.UnixNano(),
	})
}

func (h *Handler) handleTokenRevoke(conn net.Conn, body []byte, ctx context.Context) {
	var req TokenRevokeReq
	if err := json.Unmarshal(body, &req); err != nil || req.TokenID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "token_id required"})
		return
	}
	if err := h.daemon.broker.Revoke(ctx, req.TokenID); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "revoke failed: " + err.Error()})
		return
	}
	writeOK(conn, MsgTokenRevokeResp, TokenRevokeResp{Success: true})
}

func (h *Handler) handleTokenList(conn net.Conn, body []byte) {
	active := h.daemon.broker.ListActive()
	entries := make([]TokenEntry, len(active))
	for i, c := range active {
		entries[i] = TokenEntry{
			TokenID:   c.CredentialID,
			Provider:  c.ProviderName,
			TokenType: c.Type,
			Resource:  c.Scope.Resource,
			Action:    strings.Join(c.Scope.Permissions, ","),
			AgentID:   c.AgentID,
			CreatedNS: c.IssuedAt.UnixNano(),
			ExpiresNS: c.ExpiresAt.UnixNano(),
			SingleUse: false,
			State:     "active",
		}
	}
	writeOK(conn, MsgTokenListResp, TokenListResp{Tokens: entries})
}

func (h *Handler) handleMCPStart(conn net.Conn, body []byte) {
	var req MCPStartReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	if req.ListenAddr == "" || req.TargetURL == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "listen_addr and target_url required"})
		return
	}
	h.daemon.mcpMu.Lock()
	if h.daemon.mcpProxy != nil && h.daemon.mcpProxy.IsRunning() {
		h.daemon.mcpMu.Unlock()
		writeOK(conn, MsgMCPStartResp, MCPStartResp{Success: false, Message: "proxy already running"})
		return
	}
	h.daemon.mcpProxy = mcp.NewProxy(req.ListenAddr, req.TargetURL, h.daemon.mcpAudit)
	h.daemon.mcpMu.Unlock()

	// Build SafetyConfig from request fields; default to enabled.
	safe := mcp.SafetyConfig{
		CheckPolicy: h.daemon.checkMCPPolicy,
		ApproveTool: h.daemon.approveToolCall,
	}
	if !req.SafetyDisable {
		safe.RedactPII = true
		safe.RatePerMin = 60
		safe.TokenBudget = 50000
		if req.RedactPII != nil {
			safe.RedactPII = *req.RedactPII
		}
		if req.RateLimit != nil {
			safe.RatePerMin = *req.RateLimit
		}
		if req.TokenBudget != nil {
			safe.TokenBudget = *req.TokenBudget
		}
	}
	h.daemon.mcpProxy.WithSafety(safe)
	if err := h.daemon.mcpProxy.Start(); err != nil {
		writeOK(conn, MsgMCPStartResp, MCPStartResp{Success: false, Message: err.Error()})
		return
	}
	writeOK(conn, MsgMCPStartResp, MCPStartResp{Success: true, Message: "mcp proxy started"})
}

func (h *Handler) handleMCPStop(conn net.Conn, body []byte) {
	h.daemon.mcpMu.Lock()
	if h.daemon.mcpProxy == nil || !h.daemon.mcpProxy.IsRunning() {
		h.daemon.mcpMu.Unlock()
		writeOK(conn, MsgMCPStopResp, MCPStopResp{Success: false, Message: "proxy not running"})
		return
	}
	if err := h.daemon.mcpProxy.Stop(); err != nil {
		h.daemon.mcpMu.Unlock()
		writeOK(conn, MsgMCPStopResp, MCPStopResp{Success: false, Message: err.Error()})
		return
	}
	h.daemon.mcpProxy = nil
	h.daemon.mcpMu.Unlock()
	writeOK(conn, MsgMCPStopResp, MCPStopResp{Success: true, Message: "mcp proxy stopped"})
}

func (h *Handler) handleMCPStatus(conn net.Conn, body []byte) {
	h.daemon.mcpMu.Lock()
	proxy := h.daemon.mcpProxy
	h.daemon.mcpMu.Unlock()
	if proxy == nil || !proxy.IsRunning() {
		writeOK(conn, MsgMCPStatusResp, MCPStatusResp{Running: false, Message: "proxy not running"})
		return
	}
	stats, err := h.daemon.mcpAudit.GetStats(int64(proxy.Uptime().Seconds()))
	if err != nil {
		stats = &mcp.Stats{}
	}
	writeOK(conn, MsgMCPStatusResp, MCPStatusResp{
		Running: true, UptimeSecs: int64(proxy.Uptime().Seconds()),
		TotalMsgs: stats.TotalMessages, TotalToks: stats.TotalToks,
		BurnRate: stats.TokenBurnRate, InjCount: stats.InjectionCount,
		PrunedCount: stats.PrunedCount, PrunedBytes: stats.PrunedBytes,
	})
}

func (h *Handler) handleMCPLog(conn net.Conn, body []byte) {
	var req MCPLogReq
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			fmt.Fprintf(os.Stderr, "ans: mcp log: ignoring malformed body: %v\n", err)
		}
	}
	var entries []*mcp.LogEntry
	var err error
	if req.InjOnly {
		entries, err = h.daemon.mcpAudit.QueryInjections(req.Limit)
	} else if req.Method != "" {
		entries, err = h.daemon.mcpAudit.QueryByMethod(req.Method, req.Limit)
	} else {
		entries, err = h.daemon.mcpAudit.QueryRecent(req.Limit)
	}
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "querying mcp log: " + err.Error()})
		return
	}
	wire := make([]MCPLogEntry, len(entries))
	for i, e := range entries {
		wire[i] = MCPLogEntry{
			ID: e.ID, Direction: string(e.Direction), Method: e.Method,
			RequestID: e.RequestID, Content: e.Content, ToksEst: e.ToksEst,
			Injection: e.Injection, InjectionType: e.InjectionTy,
			Pruned: e.Pruned, PrunedChars: e.PrunedChars,
			TimestampNS: e.TimestampNS,
		}
	}
	writeOK(conn, MsgMCPLogResp, MCPLogResp{Entries: wire})
}

func (h *Handler) handleSnapshotList(conn net.Conn, body []byte) {
	var req SnapshotListReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad snapshot list request: " + err.Error()})
		return
	}
	if req.AgentID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "agent_id required"})
		return
	}
	if len(req.AgentID) > 128 || len(req.SnapType) > 32 {
		writeOK(conn, MsgError, ErrorResp{Message: "field too long"})
		return
	}
	snapType := snapshot.SnapFileSystem
	if req.SnapType != "" {
		snapType = snapshot.SnapType(req.SnapType)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	snaps, err := h.daemon.snapStore.List(req.AgentID, snapType, limit, req.Offset)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "listing snapshots: " + err.Error()})
		return
	}
	writeOK(conn, MsgSnapshotListResp, SnapshotListResp{Snapshots: snaps})
}

func (h *Handler) handleSnapshotDiff(conn net.Conn, body []byte) {
	var req SnapshotDiffReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad snapshot diff request: " + err.Error()})
		return
	}
	if req.AgentID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "agent_id required"})
		return
	}
	if len(req.AgentID) > 128 || len(req.SnapType) > 32 {
		writeOK(conn, MsgError, ErrorResp{Message: "field too long"})
		return
	}
	snapType := snapshot.SnapFileSystem
	if req.SnapType != "" {
		snapType = snapshot.SnapType(req.SnapType)
	}
	// Get 2 most recent snapshots
	snaps, err := h.daemon.snapStore.List(req.AgentID, snapType, 2, 0)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "listing snapshots: " + err.Error()})
		return
	}
	if len(snaps) < 2 {
		writeOK(conn, MsgSnapshotDiffResp, SnapshotDiffResp{Message: "Need at least 2 snapshots to compute a diff"})
		return
	}
	// snaps are newest-first; base is the older one (index 1)
	baseSnap := snaps[1]
	fs, ok := h.daemon.snapshotter.(*snapshot.FileSystemSnap)
	if !ok {
		writeOK(conn, MsgError, ErrorResp{Message: "filesystem snapshotter not available"})
		return
	}
	added, modified, deleted, err := fs.Diff(baseSnap.StoragePath)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "diff failed: " + err.Error()})
		return
	}
	writeOK(conn, MsgSnapshotDiffResp, SnapshotDiffResp{
		Added:    added,
		Modified: modified,
		Deleted:  deleted,
	})
}

func (h *Handler) handlePolicyRegister(conn net.Conn, body []byte) {
	var req PolicyRegisterReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	if req.ID == "" || req.Name == "" || req.Conditions == "" || req.Action == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "id, name, conditions, action required"})
		return
	}
	pol := &policy.Policy{
		ID: req.ID, Name: req.Name, Description: req.Description,
		Enabled: req.Enabled, Priority: req.Priority,
		Severity: policy.Severity(req.Severity),
	}
	if err := json.Unmarshal([]byte(req.Conditions), &pol.Conditions); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "invalid conditions JSON: " + err.Error()})
		return
	}
	if err := json.Unmarshal([]byte(req.Action), &pol.Action); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "invalid action JSON: " + err.Error()})
		return
	}
	// Pre-compile regex patterns to avoid per-call compilation
	if err := pol.Conditions.CompileRegexp(); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "invalid regex in conditions: " + err.Error()})
		return
	}
	// Validate effect
	if pol.Action.Effect != policy.EffectAllow && pol.Action.Effect != policy.EffectDeny &&
		pol.Action.Effect != policy.EffectWarn && pol.Action.Effect != policy.EffectAudit {
		writeOK(conn, MsgError, ErrorResp{Message: "effect must be 'allow', 'deny', 'warn', or 'audit'"})
		return
	}
	// Get the underlying policy store from executor
	polStore := policy.NewStore(h.daemon.chain.DB())
	if err := polStore.Insert(pol); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "saving policy: " + err.Error()})
		return
	}
	writeOK(conn, MsgPolicyRegisterResp, PolicyRegisterResp{Success: true, Message: "policy registered"})
}

func (h *Handler) handlePolicyList(conn net.Conn, body []byte) {
	var req PolicyListReq
	if len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}
	polStore := policy.NewStore(h.daemon.chain.DB())
	var policies []*policy.Policy
	var err error
	if req.EnabledOnly {
		policies, err = polStore.ListEnabled()
	} else {
		policies, err = polStore.List()
	}
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "listing policies: " + err.Error()})
		return
	}
	entries := make([]PolicyEntry, len(policies))
	for i, p := range policies {
		entries[i] = PolicyEntry{
			ID: p.ID, Name: p.Name, Description: p.Description,
			Enabled: p.Enabled, Priority: p.Priority, Severity: string(p.Severity),
			CreatedNS: p.CreatedNS, UpdatedNS: p.UpdatedNS,
		}
	}
	writeOK(conn, MsgPolicyListResp, PolicyListResp{Policies: entries})
}

func (h *Handler) handlePolicyDelete(conn net.Conn, body []byte) {
	var req PolicyDeleteReq
	if err := json.Unmarshal(body, &req); err != nil || req.ID == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "id required"})
		return
	}
	polStore := policy.NewStore(h.daemon.chain.DB())
	if err := polStore.Delete(req.ID); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "deleting policy: " + err.Error()})
		return
	}
	writeOK(conn, MsgPolicyDeleteResp, PolicyDeleteResp{Success: true, Message: "policy deleted"})
}

func (h *Handler) handlePolicyEvaluate(conn net.Conn, body []byte) {
	var req PolicyEvaluateReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	ctx := map[string]interface{}{
		"model.weight_type": "open",
	}
	facts := policy.MakeFacts(req.AgentID, req.ActionType, req.Phase, req.PayloadSummary, req.ParentAgentID, ctx)
	evalRes := h.daemon.policyExec.Evaluate(facts)
	wireResults := make([]PolicyResult, len(evalRes.PolicyResults))
	for i, pr := range evalRes.PolicyResults {
		wireResults[i] = PolicyResult{
			PolicyID: pr.PolicyID, PolicyName: pr.PolicyName,
			Effect: string(pr.Effect), Matched: pr.Matched,
			ErrorMessage: pr.ErrorMessage,
		}
	}
	resp := PolicyEvaluateResp{
		Allowed: evalRes.Allowed, Denied: evalRes.Denied,
		PolicyResults: wireResults,
	}
	if evalRes.Nociception != nil {
		resp.Nociception = &Nociception{
			PolicyID: evalRes.Nociception.PolicyID,
			PolicyName: evalRes.Nociception.PolicyName,
			Message:   evalRes.Nociception.Message,
			Severity:  evalRes.Nociception.Severity,
		}
	}
	writeOK(conn, MsgPolicyEvaluateResp, resp)
}

func (h *Handler) handleRegisterCompensate(conn net.Conn, body []byte) {
	var req RegisterCompensateReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	if req.AgentID == "" || req.ReceiptID == "" || req.ReverseCmd == "" {
		writeOK(conn, MsgError, ErrorResp{Message: "agent_id, receipt_id, reverse_cmd required"})
		return
	}
	if !safeCmdPattern.MatchString(req.ReverseCmd) {
		writeOK(conn, MsgError, ErrorResp{Message: "reverse_cmd contains unsafe characters (only alphanumeric, path chars, and spaces allowed)"})
		return
	}
	if len(req.ReverseCmd) > 500 {
		writeOK(conn, MsgError, ErrorResp{Message: "reverse_cmd too long (max 500)"})
		return
	}
	// Look up the receipt to find its chain index
	receipt, err := h.daemon.chain.Get(req.ReceiptID)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "receipt not found: " + err.Error()})
		return
	}
	rec := &chain.CompensationRecord{
		ChainIndex:   receipt.ChainIndex,
		AgentID:      req.AgentID,
		ReceiptID:    req.ReceiptID,
		ActionType:   req.ActionType,
		ReverseAction: req.ReverseAction,
		ReverseCmd:   req.ReverseCmd,
		CreatedNS:    time.Now().UnixNano(),
	}
	if err := h.daemon.chain.SaveCompensation(rec); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "saving compensation: " + err.Error()})
		return
	}
	writeOK(conn, MsgRegisterCompensateResp, RegisterCompensateResp{Success: true, Message: "compensation registered"})
}

func (h *Handler) handleCompensate(conn net.Conn, body []byte) {
	var req CompensateReq
	if err := json.Unmarshal(body, &req); err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "bad request: " + err.Error()})
		return
	}
	if req.TargetIndex == 0 {
		writeOK(conn, MsgError, ErrorResp{Message: "target_index required"})
		return
	}
	comps, err := h.daemon.chain.GetCompensations(req.TargetIndex)
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: "querying compensations: " + err.Error()})
		return
	}
	if len(comps) == 0 {
		writeOK(conn, MsgCompensateResp, CompensateResp{
			Success: true, Message: "no compensations found at or above this index",
		})
		return
	}
	if req.DryRun {
		details := make([]string, len(comps))
		for i, c := range comps {
			details[i] = fmt.Sprintf("[%d] %s: %s (cmd: %s)", c.ChainIndex, c.ActionType, c.ReverseAction, c.ReverseCmd)
		}
		writeOK(conn, MsgCompensateResp, CompensateResp{
			Success: true, ActionsRun: len(comps), Details: details,
			Message: "dry run — no actions executed",
		})
		return
	}
	// Execute compensations in reverse order (newest first)
	var details []string
	ran, failed := 0, 0
	for _, c := range comps {
		if c.ReverseCmd == "" {
			continue
		}
		if err := h.executeCompensation(c, context.Background()); err != nil {
			details = append(details, fmt.Sprintf("[%d] FAIL: %s", c.ChainIndex, err))
			failed++
		} else {
			details = append(details, fmt.Sprintf("[%d] OK: %s", c.ChainIndex, c.ReverseAction))
			_ = h.daemon.chain.MarkCompensationExecuted(c.ID)
			ran++
		}
	}
	writeOK(conn, MsgCompensateResp, CompensateResp{
		Success: failed == 0, ActionsRun: ran, ActionsFailed: failed,
		Details: details, Message: fmt.Sprintf("executed %d/%d compensations", ran, len(comps)),
	})
}



func (h *Handler) executeCompensation(c chain.CompensationRecord, parentCtx context.Context) error {
	// Parse the command into executable + args for safe execution without shell injection.
	// The command has been validated by safeCmdPattern at registration time.
	parts := strings.Fields(c.ReverseCmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty reverse_cmd")
	}
	ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
	defer cancel()
	cmd := execCommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compensation %q failed: %w\noutput: %s", c.ReverseCmd, err, string(output))
	}
	return nil
}

func (h *Handler) handleStatus(conn net.Conn) {
	stats, err := h.daemon.chain.GetStats()
	if err != nil {
		writeOK(conn, MsgError, ErrorResp{Message: err.Error()})
		return
	}
	writeOK(conn, MsgStatusResp, StatusResp{
		Uptime: time.Since(h.daemon.startedAt).Round(time.Second).String(),
		ChainLength: stats.ChainLength, TotalReceipts: stats.TotalReceipts,
		TotalAgents: stats.TotalAgents, DBSizeBytes: stats.DBSizeBytes,
		LastReceiptTS: stats.NewestReceiptNS, StartedAt: h.daemon.startedAt,
	})
}
