package client

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	"ans/internal/model"
)

type MockClient struct {
	mu       sync.Mutex
	receipts []*model.Receipt
	snaps    []*model.Snapshot
	policies []*model.Policy
	tokens   []*model.Token
	mcpState *model.MCPStatus
	nextIdx  int
	start    time.Time
	tick     int
}

func (m *MockClient) Init() error {
	return nil
}

func (m *MockClient) Doctor() (*model.DoctorReport, error) {
	return &model.DoctorReport{
		Checks: []model.CheckStatus{
			{Name: "daemon", Value: "running", Status: "ok", Detail: "PID 1234"},
			{Name: "chain", Value: "valid", Status: "ok", Detail: "1247 receipts"},
			{Name: "disk", Value: "2.3 GB", Status: "ok", Detail: "/home/.ans"},
		},
		AllOK: true,
	}, nil
}

func (m *MockClient) StartDaemon(ndjson bool, webhook string) error {
	return nil
}

func (m *MockClient) StopDaemon() error {
	return nil
}

func (m *MockClient) Update() (string, error) {
	return "v0.2.0", nil
}

func (m *MockClient) Uninstall() error {
	return nil
}

func (m *MockClient) ListAgents() ([]*model.Agent, error) {
	return []*model.Agent{
		{ID: "ans_" + randHex(10), Name: "alpha", Version: "1.0.0", Owner: "dev", PublicKey: randHex(32)},
		{ID: "ans_" + randHex(10), Name: "beta", Version: "2.0.0", Owner: "ops", PublicKey: randHex(32)},
	}, nil
}

func (m *MockClient) RegisterAgent(name, version, owner string) (*model.Agent, error) {
	return &model.Agent{
		ID:        "ans_" + randHex(10),
		Name:      name,
		Version:   version,
		Owner:     owner,
		PublicKey: randHex(32),
	}, nil
}

func (m *MockClient) Export(format, output string) (int, error) {
	return 1024, nil
}

func (m *MockClient) Prune(upTo int) (string, error) {
	return randHex(32), nil
}

func (m *MockClient) RotateKey(agentID string) (*model.Agent, error) {
	return &model.Agent{ID: agentID, PublicKey: randHex(32)}, nil
}

func (m *MockClient) SnapshotTake(agentID, snapType, paths string) (*model.Snapshot, error) {
	return &model.Snapshot{
		ID:   randHex(12),
		Slot: 1,
		Type: snapType,
		Paths: paths,
		Timestamp: time.Now(),
	}, nil
}

func (m *MockClient) SnapshotList() ([]*model.Snapshot, error) {
	return m.ListSnapshots("", 10)
}

func (m *MockClient) PolicyAdd(file string) (*model.Policy, error) {
	return &model.Policy{ID: randHex(20), Name: "added-policy", Enabled: true, Effect: "allow"}, nil
}

func (m *MockClient) PolicyList(enabled bool) ([]*model.Policy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*model.Policy, 0, len(m.policies))
	for _, p := range m.policies {
		if enabled && !p.Enabled {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (m *MockClient) PolicyRemove(id string) error {
	return nil
}

func (m *MockClient) PolicyEval(actionType, payload string) (*model.PolicyEvalResult, error) {
	allowed := true
	matched := "default-allow"
	if payload == "deny" || actionType == "shell.exec" {
		allowed = false
		matched = "deny-shell-exec"
	}
	return &model.PolicyEvalResult{Allowed: allowed, MatchedPolicy: matched}, nil
}

func (m *MockClient) TokenRequest(resource, action string, ttl int) (*model.TokenInfo, error) {
	return &model.TokenInfo{
		ID:       "tok_" + randHex(8),
		Resource: resource,
		Action:   action,
		Expiry:   time.Now().Add(time.Duration(ttl) * time.Second),
		Token:    randHex(32),
	}, nil
}

func (m *MockClient) MCPStart(target, listen string, inj bool, rate int) (*model.MCPStartInfo, error) {
	return &model.MCPStartInfo{ListenAddr: listen}, nil
}

func (m *MockClient) MCPStop() error {
	return nil
}

func (m *MockClient) MCPLog(inj bool, method string, n int) ([]*model.MCPLogEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n > len(m.mcpState.RecentLog) {
		n = len(m.mcpState.RecentLog)
	}
	out := make([]*model.MCPLogEntry, n)
	copy(out, m.mcpState.RecentLog[:n])
	return out, nil
}

func (m *MockClient) CompensateDryRun(index int) (*model.CompensatePlan, error) {
	return &model.CompensatePlan{
		Steps: []model.CompensateStep{
			{ChainIndex: index, ActionType: "file.write", HasComp: true, Command: "cp backup /restore"},
		},
		WouldRun: 1,
		Skipped:  0,
	}, nil
}

func (m *MockClient) Compensate(index int) (*model.CompensateResult, error) {
	return &model.CompensateResult{
		Steps: []model.CompensateStep{
			{ChainIndex: index, ActionType: "file.write", HasComp: true, Command: "cp backup /restore"},
		},
		Ran:    1,
		Failed: 0,
	}, nil
}

func NewMockClient() *MockClient {
	start := time.Now().Add(-4 * time.Hour)
	m := &MockClient{
		start:   start,
		nextIdx: 1247,
	}

	m.receipts = m.genReceipts(50, start)
	m.snaps = m.genSnapshots(10, start)
	m.policies = m.genPolicies()
	m.tokens = m.genTokens(start)
	m.mcpState = m.genMCPStatus(start)
	return m
}

func randHex(n int) string {
	b := make([]byte, (n+1)/2)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func randInt(n int) int {
	bi, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(bi.Int64())
}

func randFloat(min, max float64) float64 {
	return min + float64(randInt(1000))/1000*(max-min)
}

var actionTypes = []string{
	"file.write", "file.read", "file.delete",
	"http.post", "http.get", "http.put",
	"agent.delegate", "agent.spawn",
	"db.read", "db.write",
	"shell.exec", "shell.pipe",
	"tools.call", "tools.list",
	"mcp.message", "mcp.subscribe",
}

var outcomes = []string{"success", "success", "success", "failure", "partial", "denied"}

func pickActionType() string {
	return actionTypes[randInt(len(actionTypes))]
}

func pickOutcome() string {
	return outcomes[randInt(len(outcomes))]
}

func (m *MockClient) genReceipts(n int, start time.Time) []*model.Receipt {
	out := make([]*model.Receipt, n)
	for i := 0; i < n; i++ {
		idx := m.nextIdx - i
		out[i] = &model.Receipt{
			Index:      idx,
			ID:         randHex(64),
			PrevHash:   randHex(64),
			AgentID:    fmt.Sprintf("ans_%s", randHex(10)),
			ActionType: pickActionType(),
			Phase:      []string{"pre", "post"}[randInt(2)],
			Outcome:    pickOutcome(),
			DurationMS: int64(100 + randInt(9000)),
			Timestamp:  start.Add(time.Duration(i) * time.Second),
			PolicyDecision: []string{"allow", "allow", "allow", "deny", "warn"}[randInt(5)],
			PayloadSummary: fmt.Sprintf("payload for action at idx %d", idx),
			Signature:    randHex(64),
		}
	}
	return out
}

func (m *MockClient) genSnapshots(n int, start time.Time) []*model.Snapshot {
	out := make([]*model.Snapshot, n)
	for i := 0; i < n; i++ {
		idx := m.nextIdx - i*5
		size := 100*1024 + randInt(200*1024)
		isDiff := i > 0 && randInt(2) == 0
		baseID := ""
		if isDiff {
			baseID = out[i-1].ID
		}
		out[i] = &model.Snapshot{
			ID:         randHex(12),
			AgentID:    fmt.Sprintf("ans_%s", randHex(10)),
			ChainIndex: idx,
			Type:       "filesystem",
			SizeBytes:  int64(size),
			Timestamp:  start.Add(time.Duration(i) * 30 * time.Second),
			IsDiff:     isDiff,
			BaseID:     baseID,
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChainIndex > out[j].ChainIndex })
	return out
}

func (m *MockClient) genPolicies() []*model.Policy {
	return []*model.Policy{
		{
			ID:       randHex(20),
			Name:     "Block PII on Open Models",
			Enabled:  true,
			Priority: 100,
			Severity: "high",
			Effect:   "deny",
		},
		{
			ID:       randHex(20),
			Name:     "Rate Limit Shell Exec",
			Enabled:  true,
			Priority: 50,
			Severity: "medium",
			Effect:   "deny",
		},
		{
			ID:       randHex(20),
			Name:     "Warn Large Payloads",
			Enabled:  false,
			Priority: 10,
			Severity: "low",
			Effect:   "warn",
		},
	}
}

func (m *MockClient) genTokens(start time.Time) []*model.Token {
	now := time.Now()
	return []*model.Token{
		{
			ID:          "cred_" + randHex(8),
			Type:        "aws-sts",
			AgentID:     "ans_" + randHex(10),
			Resource:    "s3://prod-bucket/config",
			Permissions: []string{"read", "write"},
			ExpiresAt:   now.Add(42 * time.Second),
			IssuedAt:    now.Add(-18 * time.Second),
		},
		{
			ID:          "cred_" + randHex(8),
			Type:        "vault",
			AgentID:     "ans_" + randHex(10),
			Resource:    "vault://database/creds/reader",
			Permissions: []string{"read"},
			ExpiresAt:   now.Add(18 * time.Second),
			IssuedAt:    now.Add(-42 * time.Second),
		},
		{
			ID:          "cred_" + randHex(8),
			Type:        "gcp-iam",
			AgentID:     "ans_" + randHex(10),
			Resource:    "gcp://project/sa/deployer",
			Permissions: []string{"deploy"},
			ExpiresAt:   now.Add(5 * time.Second),
			IssuedAt:    now.Add(-55 * time.Second),
		},
	}
}

func (m *MockClient) genMCPStatus(start time.Time) *model.MCPStatus {
	now := time.Now()
	uptime := int64(now.Sub(start).Seconds())
	n := 60
	reqH := make([]float64, n)
	tokH := make([]float64, n)
	for i := 0; i < n; i++ {
		reqH[i] = 10 + 18*math.Sin(float64(i)*0.3) + float64(randInt(5))
		tokH[i] = 200 + 1000*math.Sin(float64(i)*0.2) + float64(randInt(100))
	}
	log := make([]*model.MCPLogEntry, 20)
	for i := range log {
		dir := "request"
		if randInt(2) == 0 {
			dir = "response"
		}
		methods := []string{"tools/call", "resources/read", "tools/list", "resources/write", "mcp/subscribe"}
		log[i] = &model.MCPLogEntry{
			Timestamp:     now.Add(time.Duration(i-20) * time.Second),
			Direction:     dir,
			Method:        methods[randInt(len(methods))],
			TokenEstimate: 50 + randInt(2000),
			InjDetected:   randInt(10) == 0,
			PIIFound:      randInt(8) == 0,
			PolicyResult:  []string{"allow", "allow", "allow", "deny"}[randInt(4)],
			ContentPreview: fmt.Sprintf(`{"%s":"%s"}`, randHex(4), randHex(20)),
		}
	}
	return &model.MCPStatus{
		Running:        true,
		UptimeSeconds:  uptime,
		ListenAddr:     ":8080",
		TargetURL:      "localhost:9090",
		TotalMessages:  1523,
		TotalTokens:    284501,
		BurnRate:       randFloat(600, 1100),
		Injections:     3,
		Pruned:         47,
		RateLimited:    2,
		BudgetExceeded: 0,
		PolicyDenied:   1,
		ToolsDenied:    0,
		RecentLog:      log,
		ReqHistory:     reqH,
		TokHistory:     tokH,
	}
}

func (m *MockClient) DaemonStatus() (*model.DaemonStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &model.DaemonStatus{
		Running:       true,
		Uptime:        time.Since(m.start).Round(time.Second).String(),
		ChainLength:   m.nextIdx,
		AgentCount:    3,
		DBSizeMB:      randFloat(3.5, 5.0),
		ChainVerified: true,
		Version:       "v0.1.0",
	}, nil
}

func (m *MockClient) ListReceipts(n int, agentID string) ([]*model.Receipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tick++
	if m.tick%2 == 0 {
		newRcpt := &model.Receipt{
			Index:      m.nextIdx + 1,
			ID:         randHex(64),
			PrevHash:   randHex(64),
			AgentID:    fmt.Sprintf("ans_%s", randHex(10)),
			ActionType: pickActionType(),
			Phase:      "post",
			Outcome:    pickOutcome(),
			DurationMS: int64(100 + randInt(9000)),
			Timestamp:  time.Now(),
			PolicyDecision: []string{"allow", "allow", "deny"}[randInt(3)],
			PayloadSummary: fmt.Sprintf("payload for idx %d", m.nextIdx+1),
			Signature:    randHex(64),
		}
		m.receipts = append([]*model.Receipt{newRcpt}, m.receipts...)
		m.nextIdx++
	}

	if n > len(m.receipts) {
		n = len(m.receipts)
	}
	out := make([]*model.Receipt, n)
	copy(out, m.receipts[:n])
	return out, nil
}

func (m *MockClient) VerifyReceipt(id string) (bool, error) {
	return true, nil
}

func (m *MockClient) VerifyChain() (bool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return true, m.nextIdx, nil
}

func (m *MockClient) ListSnapshots(agentID string, n int) ([]*model.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if n > len(m.snaps) {
		n = len(m.snaps)
	}
	out := make([]*model.Snapshot, n)
	copy(out, m.snaps[:n])
	return out, nil
}

func (m *MockClient) SnapshotDiff(agentID string) (*model.SnapshotDiff, error) {
	return &model.SnapshotDiff{
		Added:    []string{"new-file.txt"},
		Removed:  []string{"old-file.txt"},
		Modified: []string{"changed-file.txt"},
	}, nil
}

func (m *MockClient) TimeTravel(index int) error {
	return nil
}

func (m *MockClient) ListPolicies() ([]*model.Policy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*model.Policy, len(m.policies))
	copy(out, m.policies)
	return out, nil
}

func (m *MockClient) PolicyToggle(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.policies {
		if p.ID == id {
			p.Enabled = enabled
		}
	}
	return nil
}

func (m *MockClient) ListTokens() ([]*model.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	active := make([]*model.Token, 0, len(m.tokens))
	for _, t := range m.tokens {
		if t.Revoked {
			continue
		}
		if t.ExpiresAt.Before(now) {
			continue
		}
		active = append(active, t)
	}

	if len(active) == 0 && m.tick%3 == 0 {
		newTok := &model.Token{
			ID:          "cred_" + randHex(8),
			Type:        []string{"aws-sts", "vault", "gcp-iam", "oauth2"}[randInt(4)],
			AgentID:     "ans_" + randHex(10),
			Resource:    fmt.Sprintf("resource://%s", randHex(12)),
			Permissions: []string{"read"},
			ExpiresAt:   now.Add(time.Duration(20+randInt(40)) * time.Second),
			IssuedAt:    now,
		}
		m.tokens = append(m.tokens, newTok)
		active = append(active, newTok)
	}

	out := make([]*model.Token, len(active))
	copy(out, active)
	return out, nil
}

func (m *MockClient) TokenRevoke(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tokens {
		if t.ID == id {
			t.Revoked = true
		}
	}
	return nil
}

func (m *MockClient) MCPStatus() (*model.MCPStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := *m.mcpState
	s.Running = true
	s.BurnRate = 600 + 400*math.Sin(float64(m.tick)*0.1)
	s.UptimeSeconds = int64(time.Since(m.start).Seconds())

	reqH := make([]float64, len(s.ReqHistory))
	tokH := make([]float64, len(s.TokHistory))
	for i := 0; i < len(s.ReqHistory)-1; i++ {
		reqH[i] = s.ReqHistory[i+1]
		tokH[i] = s.TokHistory[i+1]
	}
	reqH[len(reqH)-1] = 10 + 18*math.Sin(float64(m.tick)*0.3) + float64(randInt(5))
	tokH[len(tokH)-1] = 200 + 1000*math.Sin(float64(m.tick)*0.2) + float64(randInt(100))
	s.ReqHistory = reqH
	s.TokHistory = tokH

	out := &model.MCPStatus{
		Running:        s.Running,
		UptimeSeconds:  s.UptimeSeconds,
		ListenAddr:     s.ListenAddr,
		TargetURL:      s.TargetURL,
		TotalMessages:  s.TotalMessages + int64(randInt(3)),
		TotalTokens:    s.TotalTokens + int64(randInt(500)),
		BurnRate:       s.BurnRate,
		Injections:     s.Injections,
		Pruned:         s.Pruned,
		RateLimited:    s.RateLimited,
		BudgetExceeded: s.BudgetExceeded,
		PolicyDenied:   s.PolicyDenied,
		ToolsDenied:    s.ToolsDenied,
		RecentLog:      s.RecentLog,
		ReqHistory:     reqH,
		TokHistory:     tokH,
	}
	return out, nil
}



