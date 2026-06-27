// Package daemon — lifecycle management. Owns chain, keystore, and socket listener.
// SPDX-License-Identifier: Apache-2.0
package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/broker"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/chain"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/identity"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/mcp"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/policy"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/snapshot"
)

type Daemon struct {
	chain         *chain.Chain
	keystore      *identity.Keystore
	snapStore     *snapshot.Store
	snapshotter   snapshot.Snapshotter
	policyExec    *policy.Executor
	broker        *broker.Broker
	mcpProxy      *mcp.Proxy
	mcpAudit      *mcp.AuditStore
	snapBaseDir   string
	workspaceRoot string
	startedAt     time.Time
	wg            sync.WaitGroup
	cancel        context.CancelFunc

	NDJSONWriter io.Writer
	WebhookURL   string

	mcpMu      sync.Mutex
	auditMu    sync.Mutex
	auditRing  [200]AuditEventEntry
	auditIdx   int
	auditCount int
}

// recordAuditEvent pushes an event into the ring buffer (thread-safe).
func (d *Daemon) recordAuditEvent(component, eventType, description string) {
	d.auditMu.Lock()
	d.auditRing[d.auditIdx%200] = AuditEventEntry{
		TimestampNS: time.Now().UnixNano(),
		Component:   component,
		EventType:   eventType,
		Description: description,
	}
	d.auditIdx++
	if d.auditCount < 200 {
		d.auditCount++
	}
	d.auditMu.Unlock()
}

// getAuditEvents returns up to limit recent events (thread-safe).
func (d *Daemon) getAuditEvents(limit int) []AuditEventEntry {
	d.auditMu.Lock()
	defer d.auditMu.Unlock()
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if limit > d.auditCount {
		limit = d.auditCount
	}
	start := (d.auditIdx - limit + 200) % 200
	out := make([]AuditEventEntry, limit)
	for i := 0; i < limit; i++ {
		out[i] = d.auditRing[(start+i)%200]
	}
	return out
}

// afterAppend is called after every successful receipt append.
// Emits NDJSON line to NDJSONWriter, fires webhook POST if configured.
func (d *Daemon) afterAppend(rawReceipt json.RawMessage) {
	d.recordAuditEvent("receipt-chain", "COMMIT", "receipt appended")

	if d.NDJSONWriter != nil {
		var buf bytes.Buffer
		buf.WriteString(`{"type":"receipt","data":`)
		buf.Write(rawReceipt)
		buf.WriteString("}\n")
		_, _ = d.NDJSONWriter.Write(buf.Bytes())
	}

	if d.WebhookURL == "" {
		return
	}
	// Reject non-HTTPS webhook URLs (allow http only with ANS_DEV=1)
	if !strings.HasPrefix(d.WebhookURL, "https://") && os.Getenv("ANS_DEV") != "1" {
		fmt.Fprintf(os.Stderr, "ans: webhook URL must use https (set ANS_DEV=1 to allow http): %s\n", d.WebhookURL)
		return
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		var fields map[string]interface{}
		if err := json.Unmarshal(rawReceipt, &fields); err != nil {
			return
		}
		receiptID, _ := fields["receipt_id"].(string)
		event := map[string]interface{}{
			"specversion":     "1.0",
			"id":              receiptID,
			"source":          "ans/daemon",
			"type":            "ans.receipt.append",
			"datacontenttype": "application/json",
			"time":            time.Now().UTC().Format(time.RFC3339),
			"data":            fields,
		}
		body, _ := json.Marshal(event)
		req, err := http.NewRequest(http.MethodPost, d.WebhookURL, bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/cloudevents+json")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ans: webhook POST failed: %v\n", err)
			return
		}
		_ = resp.Body.Close()
	}()
}

// New opens the chain and keystore at their default paths.
func New() (*Daemon, error) {
	c, err := chain.Open("")
	if err != nil {
		return nil, fmt.Errorf("opening chain: %w", err)
	}
	ks, err := identity.NewKeystore("")
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("opening keystore: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	d := &Daemon{chain: c, keystore: ks, workspaceRoot: cwd}
	if err := d.initSnapshotStore(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("initialising snapshot store: %w", err)
	}
	d.initPolicyExec()
	d.initBroker()
	d.initMCP()
	return d, nil
}

// NewWithPaths creates a Daemon with explicit paths (used in tests).
func NewWithPaths(chainPath, keystorePath string) (*Daemon, error) {
	c, err := chain.Open(chainPath)
	if err != nil {
		return nil, fmt.Errorf("opening chain: %w", err)
	}
	ks, err := identity.NewKeystore(keystorePath)
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("opening keystore: %w", err)
	}
	root, err := filepath.Abs(filepath.Dir(chainPath))
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("resolving workspace root: %w", err)
	}
	d := &Daemon{chain: c, keystore: ks, workspaceRoot: root}
	if err := d.initSnapshotStore(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("initialising snapshot store: %w", err)
	}
	d.initPolicyExec()
	d.initBroker()
	d.initMCP()
	return d, nil
}

func (d *Daemon) initMCP() {
	d.mcpAudit = mcp.NewAuditStore(d.chain.DB())
}

// checkMCPPolicy is called by the MCP proxy to check if a method is allowed.
func (d *Daemon) checkMCPPolicy(clientAddr, method string) (bool, string) {
	ctx := map[string]interface{}{
		"mcp.client_addr": clientAddr,
		"mcp.method":      method,
	}
	// Use a placeholder agent ID — the MCP proxy may not know the agent identity.
	facts := policy.MakeFacts("mcp-proxy", "mcp."+method, "pre", "", "", ctx)
	res := d.policyExec.Evaluate(facts)
	return res.Allowed, nociceptionMessage(res.Nociception)
}

// approveToolCall is called by the MCP proxy before forwarding a tools/call.
func (d *Daemon) approveToolCall(clientAddr, toolName string, params json.RawMessage) (bool, string) {
	ctx := map[string]interface{}{
		"mcp.client_addr": clientAddr,
		"mcp.tool_name":   toolName,
	}
	facts := policy.MakeFacts("mcp-proxy", "mcp.tools.call", "pre", "tool: "+toolName, "", ctx)
	res := d.policyExec.Evaluate(facts)
	return res.Allowed, nociceptionMessage(res.Nociception)
}

func nociceptionMessage(n *policy.NociceptionError) string {
	if n == nil {
		return ""
	}
	return n.Message
}

func (d *Daemon) initBroker() {
	d.broker = broker.NewBroker(broker.DiscardLogger{})
	_ = d.broker.RegisterProvider(broker.NewEnvProvider())
	// DevProvider only registered when ANS_DEV=1 — provides fake credentials for testing
	if os.Getenv("ANS_DEV") == "1" {
		_ = d.broker.RegisterProvider(broker.NewDevProvider())
	}
}

func (d *Daemon) initPolicyExec() {
	policyStore := policy.NewStore(d.chain.DB())
	d.policyExec = policy.NewExecutor(policyStore)
}

func (d *Daemon) initSnapshotStore() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory for snapshots: %w", err)
	}
	snapDir := filepath.Join(home, ".ans", "snapshots")
	store, err := snapshot.NewStore(d.chain.DB(), filepath.Join(snapDir, "data"))
	if err != nil {
		return err
	}
	fsSnap := snapshot.NewFileSystemSnap(d.workspaceRoot)
	store.RegisterSnapshotter(fsSnap)
	d.snapshotter = fsSnap
	d.snapStore = store
	d.snapBaseDir = snapDir
	return nil
}

// Run starts the daemon on the platform socket, blocks until signal, then shuts down.
func (d *Daemon) Run() error {
	l, err := Listen()
	if err != nil {
		_ = d.chain.Close()
		return fmt.Errorf("starting listener: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.startedAt = time.Now()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sig
		fmt.Fprintln(os.Stderr, "\nans: shutting down...")
		d.cancel()
		_ = l.Close()
	}()

	fmt.Fprintf(os.Stderr, "ans daemon started — socket: %s\n", SocketPath())
	d.serveListener(ctx, l)
	_ = d.chain.Close()
	fmt.Fprintln(os.Stderr, "ans daemon stopped")
	return nil
}

// RunOnListener runs the daemon on a pre-created listener (for tests).
func (d *Daemon) RunOnListener(ctx context.Context, l net.Listener) {
	d.startedAt = time.Now()
	d.serveListener(ctx, l)
	_ = d.chain.Close()
}

func (d *Daemon) serveListener(ctx context.Context, l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				d.wg.Wait()
				return
			default:
				fmt.Fprintf(os.Stderr, "ans: accept error: %v\n", err)
				continue
			}
		}
		d.wg.Add(1)
		go func(c net.Conn) {
			defer d.wg.Done()
			defer c.Close()
			(&Handler{daemon: d}).Handle(ctx, c)
		}(conn)
	}
}
