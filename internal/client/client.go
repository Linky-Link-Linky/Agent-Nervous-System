package client

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	path "path/filepath"
	"time"

	"ans/internal/model"
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

type Client interface {
	DaemonStatus() (*model.DaemonStatus, error)
	ListReceipts(n int, agentID string) ([]*model.Receipt, error)
	VerifyReceipt(id string) (bool, error)
	VerifyChain() (bool, int, error)
	ListSnapshots(agentID string, n int) ([]*model.Snapshot, error)
	SnapshotDiff(agentID string) (*model.SnapshotDiff, error)
	SnapshotTake(agentID, snapType, paths string) (*model.Snapshot, error)
	SnapshotList() ([]*model.Snapshot, error)
	TimeTravel(index int) error
	ListPolicies() ([]*model.Policy, error)
	PolicyAdd(file string) (*model.Policy, error)
	PolicyList(enabled bool) ([]*model.Policy, error)
	PolicyRemove(id string) error
	PolicyToggle(id string, enabled bool) error
	PolicyEval(actionType, payload string) (*model.PolicyEvalResult, error)
	ListTokens() ([]*model.Token, error)
	TokenRequest(resource, action string, ttl int) (*model.TokenInfo, error)
	TokenRevoke(id string) error
	MCPStatus() (*model.MCPStatus, error)
	MCPStart(target, listen string, inj bool, rate int) (*model.MCPStartInfo, error)
	MCPStop() error
	MCPLog(inj bool, method string, n int) ([]*model.MCPLogEntry, error)
	Init() error
	Doctor() (*model.DoctorReport, error)
	StartDaemon(ndjson bool, webhook string) error
	StopDaemon() error
	Update() (string, error)
	Uninstall() error
	ListAgents() ([]*model.Agent, error)
	RegisterAgent(name, version, owner string) (*model.Agent, error)
	Export(format, output string) (int, error)
	Prune(upTo int) (string, error)
	RotateKey(agentID string) (*model.Agent, error)
	CompensateDryRun(index int) (*model.CompensatePlan, error)
	Compensate(index int) (*model.CompensateResult, error)
}

type SocketClient struct {
	socketPath string
}

func NewSocketClient(socketPath string) *SocketClient {
	return &SocketClient{socketPath: socketPath}
}

func DefaultSocketPath() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, ".ans", "daemon.sock")
}

func (c *SocketClient) roundTrip(reqType string, payload any, respPayload any) error {
	conn, err := net.DialTimeout("unix", c.socketPath, 3*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	var raw json.RawMessage
	if payload != nil {
		b, _ := json.Marshal(payload)
		raw = b
	}
	req := Request{Type: reqType, Payload: raw}
	reqBody, _ := json.Marshal(req)

	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(reqBody)))
	if _, err := conn.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := conn.Write(reqBody); err != nil {
		return fmt.Errorf("write body: %w", err)
	}

	if _, err := conn.Read(header); err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	bodyLen := binary.BigEndian.Uint32(header)
	body := make([]byte, bodyLen)
	if _, err := conn.Read(body); err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if !resp.OK {
		return fmt.Errorf("daemon error: %s", resp.Error)
	}
	if respPayload != nil && resp.Payload != nil {
		return json.Unmarshal(resp.Payload, respPayload)
	}
	return nil
}

func (c *SocketClient) DaemonStatus() (*model.DaemonStatus, error) {
	var s model.DaemonStatus
	if err := c.roundTrip("daemon.status", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *SocketClient) ListReceipts(n int, agentID string) ([]*model.Receipt, error) {
	payload := map[string]any{"n": n, "agent_id": agentID}
	var out []*model.Receipt
	if err := c.roundTrip("chain.list", payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) VerifyReceipt(id string) (bool, error) {
	var out bool
	if err := c.roundTrip("chain.verify", map[string]string{"receipt_id": id}, &out); err != nil {
		return false, err
	}
	return out, nil
}

func (c *SocketClient) VerifyChain() (bool, int, error) {
	var out struct {
		Verified bool `json:"verified"`
		Count    int  `json:"count"`
	}
	if err := c.roundTrip("chain.verify_all", nil, &out); err != nil {
		return false, 0, err
	}
	return out.Verified, out.Count, nil
}

func (c *SocketClient) ListSnapshots(agentID string, n int) ([]*model.Snapshot, error) {
	payload := map[string]any{"agent_id": agentID, "n": n}
	var out []*model.Snapshot
	if err := c.roundTrip("snapshot.list", payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) SnapshotDiff(agentID string) (*model.SnapshotDiff, error) {
	var out model.SnapshotDiff
	if err := c.roundTrip("snapshot.diff", map[string]string{"agent_id": agentID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) SnapshotTake(agentID, snapType, paths string) (*model.Snapshot, error) {
	var out model.Snapshot
	if err := c.roundTrip("snapshot.take", map[string]string{"agent_id": agentID, "type": snapType, "paths": paths}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) SnapshotList() ([]*model.Snapshot, error) {
	var out []*model.Snapshot
	if err := c.roundTrip("snapshot.list_all", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) TimeTravel(index int) error {
	return c.roundTrip("time-travel", map[string]int{"index": index}, nil)
}

func (c *SocketClient) ListPolicies() ([]*model.Policy, error) {
	var out []*model.Policy
	if err := c.roundTrip("policy.list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) PolicyAdd(file string) (*model.Policy, error) {
	var out model.Policy
	if err := c.roundTrip("policy.add", map[string]string{"file": file}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) PolicyList(enabled bool) ([]*model.Policy, error) {
	var out []*model.Policy
	if err := c.roundTrip("policy.list_all", map[string]bool{"enabled": enabled}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) PolicyRemove(id string) error {
	return c.roundTrip("policy.remove", map[string]string{"id": id}, nil)
}

func (c *SocketClient) PolicyToggle(id string, enabled bool) error {
	return c.roundTrip("policy.toggle", map[string]any{"id": id, "enabled": enabled}, nil)
}

func (c *SocketClient) PolicyEval(actionType, payload string) (*model.PolicyEvalResult, error) {
	var out model.PolicyEvalResult
	if err := c.roundTrip("policy.eval", map[string]string{"action_type": actionType, "payload_summary": payload}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) ListTokens() ([]*model.Token, error) {
	var out []*model.Token
	if err := c.roundTrip("token.list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) TokenRequest(resource, action string, ttl int) (*model.TokenInfo, error) {
	var out model.TokenInfo
	if err := c.roundTrip("token.request", map[string]any{"resource": resource, "action": action, "ttl": ttl}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) TokenRevoke(id string) error {
	return c.roundTrip("token.revoke", map[string]string{"credential_id": id}, nil)
}

func (c *SocketClient) MCPStatus() (*model.MCPStatus, error) {
	var out model.MCPStatus
	if err := c.roundTrip("mcp.status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) MCPStart(target, listen string, inj bool, rate int) (*model.MCPStartInfo, error) {
	var out model.MCPStartInfo
	if err := c.roundTrip("mcp.start", map[string]any{"target": target, "listen": listen, "safety_disable": inj, "rate_limit": rate}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) MCPStop() error {
	return c.roundTrip("mcp.stop", nil, nil)
}

func (c *SocketClient) MCPLog(inj bool, method string, n int) ([]*model.MCPLogEntry, error) {
	var out []*model.MCPLogEntry
	if err := c.roundTrip("mcp.log", map[string]any{"inj": inj, "method": method, "n": n}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) Init() error {
	return c.roundTrip("daemon.init", nil, nil)
}

func (c *SocketClient) Doctor() (*model.DoctorReport, error) {
	var out model.DoctorReport
	if err := c.roundTrip("daemon.doctor", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) StartDaemon(ndjson bool, webhook string) error {
	return c.roundTrip("daemon.start", map[string]any{"ndjson": ndjson, "webhook": webhook}, nil)
}

func (c *SocketClient) StopDaemon() error {
	return c.roundTrip("daemon.stop", nil, nil)
}

func (c *SocketClient) Update() (string, error) {
	var out string
	if err := c.roundTrip("daemon.update", nil, &out); err != nil {
		return "", err
	}
	return out, nil
}

func (c *SocketClient) Uninstall() error {
	return c.roundTrip("daemon.uninstall", nil, nil)
}

func (c *SocketClient) ListAgents() ([]*model.Agent, error) {
	var out []*model.Agent
	if err := c.roundTrip("agent.list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *SocketClient) RegisterAgent(name, version, owner string) (*model.Agent, error) {
	var out model.Agent
	if err := c.roundTrip("agent.register", map[string]string{"name": name, "version": version, "owner": owner}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) Export(format, output string) (int, error) {
	var out int
	if err := c.roundTrip("chain.export", map[string]string{"format": format, "output": output}, &out); err != nil {
		return 0, err
	}
	return out, nil
}

func (c *SocketClient) Prune(upTo int) (string, error) {
	var out string
	if err := c.roundTrip("chain.prune", map[string]int{"up_to": upTo}, &out); err != nil {
		return "", err
	}
	return out, nil
}

func (c *SocketClient) RotateKey(agentID string) (*model.Agent, error) {
	var out model.Agent
	if err := c.roundTrip("agent.rotate", map[string]string{"agent_id": agentID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) CompensateDryRun(index int) (*model.CompensatePlan, error) {
	var out model.CompensatePlan
	if err := c.roundTrip("compensate.dryrun", map[string]int{"index": index}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *SocketClient) Compensate(index int) (*model.CompensateResult, error) {
	var out model.CompensateResult
	if err := c.roundTrip("compensate.apply", map[string]int{"index": index}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
