package client

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	path "path/filepath"
	"time"

	"ans-tui/internal/model"
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
	SnapshotDiff(agentID string) (string, error)
	TimeTravel(index int) error
	ListPolicies() ([]*model.Policy, error)
	PolicyToggle(id string, enabled bool) error
	ListTokens() ([]*model.Token, error)
	TokenRevoke(id string) error
	MCPStatus() (*model.MCPStatus, error)
	MCPLog(n int) ([]*model.MCPLogEntry, error)
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

func (c *SocketClient) SnapshotDiff(agentID string) (string, error) {
	var out string
	if err := c.roundTrip("snapshot.diff", map[string]string{"agent_id": agentID}, &out); err != nil {
		return "", err
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

func (c *SocketClient) PolicyToggle(id string, enabled bool) error {
	return c.roundTrip("policy.toggle", map[string]any{"id": id, "enabled": enabled}, nil)
}

func (c *SocketClient) ListTokens() ([]*model.Token, error) {
	var out []*model.Token
	if err := c.roundTrip("token.list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
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

func (c *SocketClient) MCPLog(n int) ([]*model.MCPLogEntry, error) {
	var out []*model.MCPLogEntry
	if err := c.roundTrip("mcp.log", map[string]int{"n": n}, &out); err != nil {
		return nil, err
	}
	return out, nil
}
