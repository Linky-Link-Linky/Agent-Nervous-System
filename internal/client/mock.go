package client

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
)

type MockClient struct {
	mu         sync.Mutex
	receipts   []*model.Receipt
	callCount  int
	tokens     []*model.Token
	policies   []*model.Policy
	mcpRunning bool
	mcpStatus  *model.MCPStatus
	mcpLog     []*model.MCPLogEntry
	issuedAt   time.Time
}

func NewMock() *MockClient {
	m := &MockClient{
		issuedAt: time.Now(),
	}
	m.seed()
	return m
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (m *MockClient) seed() {
	m.receipts = make([]*model.Receipt, 0, 50)
	agents := []string{"ans_3vQb7uL6x9", "ans_9yWc2kM4x1", "ans_1aB2cD3eF4"}
	actions := []string{"file.write", "file.read", "shell.exec", "http.post", "http.get", "agent.delegate", "db.query", "db.insert"}
	outcomes := []string{"success", "failure", "partial", "denied"}
	phases := []string{"pre", "post"}
	decisions := []string{"allow", "deny"}

	var prevHash string
	for i := 0; i < 50; i++ {
		idx := 1247 - i
		id := randHex(16)
		if prevHash == "" {
			prevHash = randHex(16)
		}
		rcpt := &model.Receipt{
			Index:          idx,
			ID:             id,
			PrevHash:       prevHash,
			AgentID:        agents[i%len(agents)],
			ActionType:     actions[i%len(actions)],
			Phase:          phases[i%len(phases)],
			Outcome:        outcomes[i%len(outcomes)],
			DurationMS:     int64(100 + (i*37)%5000),
			Timestamp:      time.Now().Add(-time.Duration(i) * 30 * time.Second),
			PolicyDecision: decisions[i%len(decisions)],
			PayloadSummary: fmt.Sprintf("payload data for receipt %d", idx),
			SnapshotID:     randHex(12),
			Signature:      randHex(32),
		}
		prevHash = id
		m.receipts = append(m.receipts, rcpt)
	}

	m.policies = []*model.Policy{
		{ID: "no-pii-on-open-models", Name: "Block PII on Open Models", Enabled: true, Priority: 100, Severity: "HIGH", Effect: "deny"},
		{ID: "rate-limit-shell-exec", Name: "Limit shell.exec rate", Enabled: true, Priority: 50, Severity: "MEDIUM", Effect: "deny"},
		{ID: "warn-large-payloads", Name: "Warn on large payloads", Enabled: false, Priority: 10, Severity: "LOW", Effect: "warn"},
	}

	m.tokens = []*model.Token{
		{ID: "cred_a1b2c3d4", Type: "aws-sts", AgentID: "ans_3vQb7uL6x9", Resource: "s3://prod-bucket/config.json", Permissions: []string{"read"}, IssuedAt: m.issuedAt, ExpiresAt: m.issuedAt.Add(60 * time.Second)},
		{ID: "cred_b2c3d4e5", Type: "vault", AgentID: "ans_9yWc2kM4x1", Resource: "vault://database/creds/readonly", Permissions: []string{"read", "list"}, IssuedAt: m.issuedAt, ExpiresAt: m.issuedAt.Add(45 * time.Second)},
		{ID: "cred_c3d4e5f6", Type: "oauth2", AgentID: "ans_3vQb7uL6x9", Resource: "gcp://project/sa/deployer", Permissions: []string{"write"}, IssuedAt: m.issuedAt, ExpiresAt: m.issuedAt.Add(30 * time.Second)},
	}

	m.mcpStatus = &model.MCPStatus{
		Running:       true,
		UptimeSeconds: 13320,
		ListenAddr:    ":8080",
		TargetURL:     "http://localhost:9090",
		TotalMessages: 1523,
		TotalTokens:   284501,
		BurnRate:      832.4,
		Injections:    3,
		Pruned:        47,
		RateLimited:   2,
		PolicyDenied:  1,
		ToolsDenied:   0,
		ReqHistory:    make([]float64, 30),
		TokHistory:    make([]float64, 30),
	}

	for i := range m.mcpStatus.ReqHistory {
		n, _ := rand.Int(rand.Reader, big.NewInt(20))
		m.mcpStatus.ReqHistory[i] = float64(n.Int64())
		n2, _ := rand.Int(rand.Reader, big.NewInt(1000))
		m.mcpStatus.TokHistory[i] = float64(n2.Int64())
	}

	m.mcpLog = make([]*model.MCPLogEntry, 50)
	dirs := []string{"request", "response"}
	methods := []string{"tools/call", "tools/list", "resources/read", "resources/list", "prompts/get", "tools/execute"}
	now := time.Now()
	for i := range m.mcpLog {
		m.mcpLog[i] = &model.MCPLogEntry{
			Timestamp:      now.Add(-time.Duration(i) * 3 * time.Second),
			Direction:      dirs[i%2],
			Method:         methods[i%len(methods)],
			TokenEstimate:  50 + (i*17)%2000,
			InjDetected:    i%17 == 0,
			PIIFound:       i%23 == 0,
			PolicyResult:   []string{"allow", "allow", "allow", "deny"}[i%4],
			ContentPreview: fmt.Sprintf("content excerpt for log entry %d", i),
		}
	}
}

func (m *MockClient) prependReceipt() {
	var prevHash string
	if len(m.receipts) > 0 {
		prevHash = m.receipts[0].ID
	}
	idx := 1248 + len(m.receipts)
	agents := []string{"ans_3vQb7uL6x9", "ans_9yWc2kM4x1", "ans_1aB2cD3eF4"}
	actions := []string{"file.write", "file.read", "shell.exec", "http.post"}
	outcomes := []string{"success", "failure", "partial", "denied"}
	i := len(m.receipts) % 4
	rcpt := &model.Receipt{
		Index:          idx,
		ID:             randHex(16),
		PrevHash:       prevHash,
		AgentID:        agents[i%len(agents)],
		ActionType:     actions[i%len(actions)],
		Phase:          "post",
		Outcome:        outcomes[i%len(outcomes)],
		DurationMS:     int64(100 + (i*37)%5000),
		Timestamp:      time.Now(),
		PolicyDecision: "allow",
		PayloadSummary: fmt.Sprintf("new payload for receipt %d", idx),
		SnapshotID:     randHex(12),
		Signature:      randHex(32),
	}
	m.receipts = append([]*model.Receipt{rcpt}, m.receipts...)
}

func (m *MockClient) Init() error {
	return nil
}

func (m *MockClient) DaemonStatus() (*model.DaemonStatus, error) {
	return &model.DaemonStatus{
		Running:       true,
		PID:           12345,
		UptimeSeconds: 13320,
		ChainLength:   len(m.receipts),
		AgentCount:    3,
		DBSizeMB:      4.2,
		ChainVerified: true,
		Version:       "v0.1.0",
	}, nil
}

func (m *MockClient) StartDaemon(bool, string) error {
	return nil
}

func (m *MockClient) StopDaemon() error {
	return nil
}

func (m *MockClient) Doctor() (*model.DoctorReport, error) {
	return &model.DoctorReport{
		Checks: []*model.DoctorCheck{
			{Name: "Socket", Value: "~/.ans/daemon.sock", Status: "ok", Detail: "exists mode 0600"},
			{Name: "Daemon PID", Value: "12345", Status: "ok", Detail: "running"},
			{Name: "Config", Value: "~/.ans/config.json", Status: "ok", Detail: "valid"},
			{Name: "Chain DB", Value: "~/.ans/chain.db", Status: "ok", Detail: "4.2 MB"},
			{Name: "Chain Length", Value: "1,247 receipts", Status: "ok"},
			{Name: "Chain Verify", Value: "intact", Status: "ok"},
			{Name: "Key Store", Value: "3 agents", Status: "ok"},
			{Name: "Snapshots", Value: "47 files 5.8 MB", Status: "ok"},
		},
		AllOK: true,
	}, nil
}

func (m *MockClient) Update() (string, error) {
	return "v0.2.0", nil
}

func (m *MockClient) Uninstall() error {
	return nil
}

func (m *MockClient) ListReceipts(n int, agentID string) ([]*model.Receipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	if m.callCount > 1 {
		m.prependReceipt()
	}
	var filtered []*model.Receipt
	for _, r := range m.receipts {
		if agentID != "" && r.AgentID != agentID {
			continue
		}
		filtered = append(filtered, r)
	}
	if n > 0 && n < len(filtered) {
		filtered = filtered[:n]
	}
	return filtered, nil
}

func (m *MockClient) VerifyReceipt(id string) (bool, error) {
	return true, nil
}

func (m *MockClient) VerifyChain() (bool, int, error) {
	return true, len(m.receipts), nil
}

func (m *MockClient) ListAgents() ([]*model.Agent, error) {
	return []*model.Agent{
		{ID: "ans_3vQb7uL6x9", Name: "main-agent", Version: "1.0.0", Owner: "acme-corp", PublicKey: randHex(32), CreatedAt: time.Now().Add(-72 * time.Hour)},
		{ID: "ans_9yWc2kM4x1", Name: "backup-agent", Version: "1.0.0", Owner: "acme-corp", PublicKey: randHex(32), CreatedAt: time.Now().Add(-48 * time.Hour)},
	}, nil
}

func (m *MockClient) RegisterAgent(name, version, owner string) (*model.Agent, error) {
	return &model.Agent{
		ID: "ans_" + randHex(10), Name: name, Version: version,
		Owner: owner, PublicKey: randHex(32), CreatedAt: time.Now(),
	}, nil
}

func (m *MockClient) Export(format, outputPath string) (int64, error) {
	return 284000, nil
}

func (m *MockClient) Prune(upTo int) (string, error) {
	return randHex(32), nil
}

func (m *MockClient) RotateKey(agentID string) (*model.Agent, error) {
	return &model.Agent{ID: agentID, PublicKey: randHex(32)}, nil
}

func (m *MockClient) ListSnapshots(agentID string, n int) ([]*model.Snapshot, error) {
	snaps := make([]*model.Snapshot, 10)
	for i := range snaps {
		snaps[i] = &model.Snapshot{
			ID: randHex(12), AgentID: agentID,
			ChainIndex: 1247 - i, Type: "filesystem",
			SizeBytes: int64(128000 + i*100),
			Timestamp: time.Now().Add(-time.Duration(i) * 30 * time.Second),
		}
	}
	return snaps, nil
}

func (m *MockClient) SnapshotTake(agentID, snapType string, paths []string) (*model.Snapshot, error) {
	return &model.Snapshot{
		ID: randHex(12), AgentID: agentID,
		Type: snapType, SizeBytes: 128000,
		Timestamp: time.Now(),
	}, nil
}

func (m *MockClient) SnapshotDiff(agentID string) (string, error) {
	return "MODIFIED  /etc/nginx/nginx.conf\nADDED     /etc/nginx/conf.d/ssl.conf\nDELETED   /tmp/deploy.lock", nil
}

func (m *MockClient) TimeTravel(string, string) error {
	return nil
}

func (m *MockClient) CompensateDryRun(index int) (*model.CompensationPlan, error) {
	return &model.CompensationPlan{
		Steps: []*model.CompensationStep{
			{ChainIndex: index, ActionType: "file.delete", Command: "restore-backup.sh /etc/nginx/nginx.conf", HasComp: true},
			{ChainIndex: index - 2, ActionType: "db.query", HasComp: false},
		},
		WouldRun: 1, Skipped: 1,
	}, nil
}

func (m *MockClient) Compensate(index int) (*model.CompensationResult, error) {
	return &model.CompensationResult{
		Ran: 1, Failed: 0,
		Steps: []struct {
			ChainIndex int    `json:"chain_index"`
			Command    string `json:"command"`
			ExitCode   int    `json:"exit_code"`
			Stderr     string `json:"stderr"`
		}{
			{ChainIndex: index, Command: "restore-backup.sh", ExitCode: 0},
		},
	}, nil
}

func (m *MockClient) ListPolicies() ([]*model.Policy, error) {
	return m.policies, nil
}

func (m *MockClient) PolicyAdd(policyJSON string) (*model.Policy, error) {
	p := &model.Policy{
		ID: randHex(16), Name: "custom-policy",
		Enabled: true, Priority: 50,
		Severity: "MEDIUM", Effect: "deny",
	}
	m.policies = append(m.policies, p)
	return p, nil
}

func (m *MockClient) PolicyRemove(id string) error {
	var kept []*model.Policy
	for _, p := range m.policies {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	m.policies = kept
	return nil
}

func (m *MockClient) PolicyToggle(id string, enabled bool) error {
	for _, p := range m.policies {
		if p.ID == id {
			p.Enabled = enabled
			break
		}
	}
	return nil
}

func (m *MockClient) PolicyEval(actionType, payloadSummary string) (*model.PolicyEvalResult, error) {
	if payloadSummary != "" && len(payloadSummary) > 0 {
		return &model.PolicyEvalResult{
			Allowed:       false,
			DenyingPolicy: "no-pii-on-open-models",
			DenyReason:    "Cannot send PII to open-weight models",
			ErrorType:     "NociceptionError",
			ErrorCode:     "0x1F",
			Evaluated:     3,
		}, nil
	}
	return &model.PolicyEvalResult{Allowed: true, Evaluated: 3}, nil
}

func (m *MockClient) ListTokens() ([]*model.Token, error) {
	return m.tokens, nil
}

func (m *MockClient) TokenRequest(resource, action string, ttl int) (*model.Token, error) {
	t := &model.Token{
		ID: "cred_" + randHex(8), Type: "aws-sts",
		AgentID: "ans_3vQb7uL6x9", Resource: resource,
		Permissions: []string{action},
		IssuedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
	}
	m.tokens = append(m.tokens, t)
	return t, nil
}

func (m *MockClient) TokenRevoke(id string) error {
	var kept []*model.Token
	for _, t := range m.tokens {
		if t.ID != id {
			kept = append(kept, t)
		}
	}
	m.tokens = kept
	return nil
}

func (m *MockClient) MCPStatus() (*model.MCPStatus, error) {
	m.mcpStatus.TotalMessages += int64(1 + m.callCount%5)
	m.mcpStatus.ReqHistory = append(m.mcpStatus.ReqHistory, float64(5+m.callCount%20))
	if len(m.mcpStatus.ReqHistory) > 60 {
		m.mcpStatus.ReqHistory = m.mcpStatus.ReqHistory[1:]
	}
	m.mcpStatus.TokHistory = append(m.mcpStatus.TokHistory, float64(200+m.callCount%800))
	if len(m.mcpStatus.TokHistory) > 60 {
		m.mcpStatus.TokHistory = m.mcpStatus.TokHistory[1:]
	}
	m.callCount++
	return m.mcpStatus, nil
}

func (m *MockClient) MCPStart(MCPStartOptions) error {
	m.mcpRunning = true
	return nil
}

func (m *MockClient) MCPStop() error {
	m.mcpRunning = false
	return nil
}

func (m *MockClient) MCPLog(n int, method string, injOnly bool) ([]*model.MCPLogEntry, error) {
	var filtered []*model.MCPLogEntry
	for _, e := range m.mcpLog {
		if injOnly && !e.InjDetected {
			continue
		}
		if method != "" && e.Method != method {
			continue
		}
		filtered = append(filtered, e)
	}
	if n > 0 && n < len(filtered) {
		filtered = filtered[:n]
	}
	return filtered, nil
}
