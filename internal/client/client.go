package client

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/daemon"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
)

type Request struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Response struct {
	Type    string          `json:"type"`
	OK      bool            `json:"ok"`
	Error   string          `json:"error,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type MCPStartOptions struct {
	Listen        string
	Target        string
	SafetyDisable bool
	RateLimit     int
	TokenBudget   int
	PIIRedact     bool
}

type Client interface {
	Init() error
	DaemonStatus() (*model.DaemonStatus, error)
	StartDaemon(ndjson bool, webhookURL string) error
	StopDaemon() error
	Doctor() (*model.DoctorReport, error)
	Update() (string, error)
	Uninstall() error
	ListReceipts(n int, agentID string) ([]*model.Receipt, error)
	VerifyReceipt(id string) (bool, error)
	VerifyChain() (bool, int, error)
	ListAgents() ([]*model.Agent, error)
	RegisterAgent(name, version, owner string) (*model.Agent, error)
	Export(format, outputPath string) (int64, error)
	Prune(upTo int) (string, error)
	RotateKey(agentID string) (*model.Agent, error)
	ListSnapshots(agentID string, n int) ([]*model.Snapshot, error)
	SnapshotTake(agentID, snapType string, paths []string) (*model.Snapshot, error)
	SnapshotDiff(agentID string) (string, error)
	TimeTravel(indexOrHash string, snapType string) error
	CompensateDryRun(index int) (*model.CompensationPlan, error)
	Compensate(index int) (*model.CompensationResult, error)
	ListPolicies() ([]*model.Policy, error)
	PolicyAdd(policyJSON string) (*model.Policy, error)
	PolicyRemove(id string) error
	PolicyToggle(id string, enabled bool) error
	PolicyEval(actionType, payloadSummary string) (*model.PolicyEvalResult, error)
	ListTokens() ([]*model.Token, error)
	TokenRequest(resource, action string, ttl int) (*model.Token, error)
	TokenRevoke(id string) error
	MCPStatus() (*model.MCPStatus, error)
	MCPStart(opts MCPStartOptions) error
	MCPStop() error
	MCPLog(n int, method string, injOnly bool) ([]*model.MCPLogEntry, error)
}

type SocketClient struct {
	sockPath string
}

func NewSocket(sockPath string) *SocketClient {
	return &SocketClient{sockPath: sockPath}
}

func DefaultSockPath() string {
	if v := os.Getenv("ANS_SOCK_PATH"); v != "" {
		return v
	}
	return daemon.SocketPath()
}

func (c *SocketClient) roundTrip(req *Request) (*Response, error) {
	conn, err := net.DialTimeout("unix", c.sockPath, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(body)))
	if _, err := conn.Write(header); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}
	if _, err := conn.Write(body); err != nil {
		return nil, fmt.Errorf("write body: %w", err)
	}

	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	respLen := binary.BigEndian.Uint32(header)
	respBody := make([]byte, respLen)
	if _, err := io.ReadFull(conn, respBody); err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if !resp.OK {
		return &resp, errors.New(resp.Error)
	}
	return &resp, nil
}

func (c *SocketClient) do(reqType string, payload, result any) error {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		raw = b
	}
	resp, err := c.roundTrip(&Request{Type: reqType, Payload: raw})
	if err != nil {
		return err
	}
	if result != nil && resp.Payload != nil {
		return json.Unmarshal(resp.Payload, result)
	}
	return nil
}

func (c *SocketClient) Init() error {
	return c.do("init", nil, nil)
}

func (c *SocketClient) DaemonStatus() (*model.DaemonStatus, error) {
	var s model.DaemonStatus
	err := c.do("daemon_status", nil, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *SocketClient) StartDaemon(ndjson bool, webhookURL string) error {
	return c.do("start_daemon", map[string]any{"ndjson": ndjson, "webhook": webhookURL}, nil)
}

func (c *SocketClient) StopDaemon() error {
	return c.do("stop_daemon", nil, nil)
}

func (c *SocketClient) Doctor() (*model.DoctorReport, error) {
	var r model.DoctorReport
	err := c.do("doctor", nil, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *SocketClient) Update() (string, error) {
	var v string
	err := c.do("update", nil, &v)
	if err != nil {
		return "", err
	}
	return v, nil
}

func (c *SocketClient) Uninstall() error {
	return c.do("uninstall", nil, nil)
}

func (c *SocketClient) ListReceipts(n int, agentID string) ([]*model.Receipt, error) {
	var rcpts []*model.Receipt
	err := c.do("list_receipts", map[string]any{"n": n, "agent_id": agentID}, &rcpts)
	if err != nil {
		return nil, err
	}
	return rcpts, nil
}

func (c *SocketClient) VerifyReceipt(id string) (bool, error) {
	var ok bool
	err := c.do("verify_receipt", map[string]string{"id": id}, &ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (c *SocketClient) VerifyChain() (bool, int, error) {
	var r struct {
		Verified bool `json:"verified"`
		Count    int  `json:"count"`
	}
	err := c.do("verify_chain", nil, &r)
	if err != nil {
		return false, 0, err
	}
	return r.Verified, r.Count, nil
}

func (c *SocketClient) ListAgents() ([]*model.Agent, error) {
	var agents []*model.Agent
	err := c.do("list_agents", nil, &agents)
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (c *SocketClient) RegisterAgent(name, version, owner string) (*model.Agent, error) {
	var a model.Agent
	err := c.do("register_agent", map[string]string{"name": name, "version": version, "owner": owner}, &a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *SocketClient) Export(format, outputPath string) (int64, error) {
	var size int64
	err := c.do("export", map[string]string{"format": format, "output": outputPath}, &size)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (c *SocketClient) Prune(upTo int) (string, error) {
	var root string
	err := c.do("prune", map[string]int{"up_to": upTo}, &root)
	if err != nil {
		return "", err
	}
	return root, nil
}

func (c *SocketClient) RotateKey(agentID string) (*model.Agent, error) {
	var a model.Agent
	err := c.do("rotate_key", map[string]string{"agent_id": agentID}, &a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *SocketClient) ListSnapshots(agentID string, n int) ([]*model.Snapshot, error) {
	var snaps []*model.Snapshot
	err := c.do("list_snapshots", map[string]any{"agent_id": agentID, "n": n}, &snaps)
	if err != nil {
		return nil, err
	}
	return snaps, nil
}

func (c *SocketClient) SnapshotTake(agentID, snapType string, paths []string) (*model.Snapshot, error) {
	var s model.Snapshot
	err := c.do("snapshot_take", map[string]any{"agent_id": agentID, "type": snapType, "paths": paths}, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *SocketClient) SnapshotDiff(agentID string) (string, error) {
	var diff string
	err := c.do("snapshot_diff", map[string]string{"agent_id": agentID}, &diff)
	if err != nil {
		return "", err
	}
	return diff, nil
}

func (c *SocketClient) TimeTravel(indexOrHash string, snapType string) error {
	return c.do("time_travel", map[string]string{"index_or_hash": indexOrHash, "type": snapType}, nil)
}

func (c *SocketClient) CompensateDryRun(index int) (*model.CompensationPlan, error) {
	var p model.CompensationPlan
	err := c.do("compensate_dry_run", map[string]int{"index": index}, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *SocketClient) Compensate(index int) (*model.CompensationResult, error) {
	var r model.CompensationResult
	err := c.do("compensate", map[string]int{"index": index}, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *SocketClient) ListPolicies() ([]*model.Policy, error) {
	var policies []*model.Policy
	err := c.do("list_policies", nil, &policies)
	if err != nil {
		return nil, err
	}
	return policies, nil
}

func (c *SocketClient) PolicyAdd(policyJSON string) (*model.Policy, error) {
	var p model.Policy
	err := c.do("policy_add", map[string]string{"json": policyJSON}, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *SocketClient) PolicyRemove(id string) error {
	return c.do("policy_remove", map[string]string{"id": id}, nil)
}

func (c *SocketClient) PolicyToggle(id string, enabled bool) error {
	return c.do("policy_toggle", map[string]any{"id": id, "enabled": enabled}, nil)
}

func (c *SocketClient) PolicyEval(actionType, payloadSummary string) (*model.PolicyEvalResult, error) {
	var r model.PolicyEvalResult
	err := c.do("policy_eval", map[string]string{"action_type": actionType, "payload_summary": payloadSummary}, &r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *SocketClient) ListTokens() ([]*model.Token, error) {
	var tokens []*model.Token
	err := c.do("list_tokens", nil, &tokens)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

func (c *SocketClient) TokenRequest(resource, action string, ttl int) (*model.Token, error) {
	var t model.Token
	err := c.do("token_request", map[string]any{"resource": resource, "action": action, "ttl": ttl}, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *SocketClient) TokenRevoke(id string) error {
	return c.do("token_revoke", map[string]string{"id": id}, nil)
}

func (c *SocketClient) MCPStatus() (*model.MCPStatus, error) {
	var s model.MCPStatus
	err := c.do("mcp_status", nil, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *SocketClient) MCPStart(opts MCPStartOptions) error {
	return c.do("mcp_start", opts, nil)
}

func (c *SocketClient) MCPStop() error {
	return c.do("mcp_stop", nil, nil)
}

func (c *SocketClient) MCPLog(n int, method string, injOnly bool) ([]*model.MCPLogEntry, error) {
	var entries []*model.MCPLogEntry
	err := c.do("mcp_log", map[string]any{"n": n, "method": method, "inj_only": injOnly}, &entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}
