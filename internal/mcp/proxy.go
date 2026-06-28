package mcp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"sync"
	"time"
)

// SafetyConfig controls MCP proxy safety features.
type SafetyConfig struct {
	// PII redaction on server-to-client responses
	RedactPII bool

	// Rate limit (requests/minute per client IP); 0 = unlimited
	RatePerMin int

	// Token budget (estimated tokens/minute per client IP); 0 = unlimited
	TokenBudget int

	// If non-nil, called before forwarding each client request method
	CheckPolicy MCPPolicyFunc

	// If non-nil, called before forwarding tools/call requests
	ApproveTool MCPToolApprovalFunc
}

// Proxy is a transparent MCP security proxy.
type Proxy struct {
	listenAddr string
	targetURL  string
	store      *AuditStore
	safety     SafetyConfig

	rateLimiter  *RateLimiter
	tokenBudget  *TokenBudget

	mu       sync.Mutex
	listener net.Listener
	running  bool
	started  time.Time
	stopCh   chan struct{}
	wg       sync.WaitGroup
	connSem  chan struct{} // limits concurrent connections (max 100)
}

const maxProxyConns = 100

// NewProxy creates an MCP proxy that listens on listenAddr and forwards to targetURL.
func NewProxy(listenAddr, targetURL string, store *AuditStore) *Proxy {
	return &Proxy{
		listenAddr: listenAddr,
		targetURL:  targetURL,
		store:      store,
		stopCh:     make(chan struct{}),
		connSem:    make(chan struct{}, maxProxyConns),
	}
}

// WithSafety enables safety features on the proxy.
func (p *Proxy) WithSafety(cfg SafetyConfig) *Proxy {
	p.safety = cfg
	if cfg.RatePerMin > 0 {
		p.rateLimiter = NewRateLimiter(cfg.RatePerMin)
	}
	if cfg.TokenBudget > 0 {
		p.tokenBudget = NewTokenBudget(cfg.TokenBudget)
	}
	return p
}

// Start begins listening and proxying.
func (p *Proxy) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running {
		return fmt.Errorf("mcp proxy already running on %s", p.listenAddr)
	}
	l, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("mcp proxy listen: %w", err)
	}
	p.listener = l
	p.running = true
	p.started = time.Now()
	slog.Info("mcp proxy listening", "addr", p.listenAddr, "target", p.targetURL)
	go p.acceptLoop()
	return nil
}

// Stop shuts down the proxy.
func (p *Proxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return nil
	}
	p.running = false
	close(p.stopCh)
	if p.listener != nil {
		_ = p.listener.Close()
	}
	p.wg.Wait()
	p.stopCh = make(chan struct{})
	slog.Info("mcp proxy stopped")
	return nil
}

// IsRunning returns true if the proxy is active.
func (p *Proxy) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// Uptime returns the proxy uptime.
func (p *Proxy) Uptime() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.running {
		return 0
	}
	return time.Since(p.started)
}

func (p *Proxy) acceptLoop() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			select {
			case <-p.stopCh:
				return
			default:
				slog.Warn("mcp proxy accept error", "error", err)
				continue
			}
		}
		p.wg.Add(1)
		go p.handleConn(conn)
	}
}

func (p *Proxy) handleConn(client net.Conn) {
	defer p.wg.Done()
	defer client.Close()

	target, err := DialTarget(p.targetURL)
	if err != nil {
		slog.Warn("mcp proxy dial target failed", "target", p.targetURL, "error", err)
		return
	}
	defer target.Close()

	// Acquire semaphore after successful dial to avoid holding slots during slow dials
	select {
	case p.connSem <- struct{}{}:
	default:
		slog.Warn("mcp proxy max connections reached", "max", maxProxyConns)
		return
	}
	defer func() { <-p.connSem }()

	// Copy safety config to avoid data race if WithSafety is called concurrently
	safety := p.safety
	ctx := &proxyCtx{
		client:    client,
		target:    target,
		store:     p.store,
		stopCh:    p.stopCh,
		safety:    &safety,
		rateLim:   p.rateLimiter,
		tokBudget: p.tokenBudget,
		clientAddr: client.RemoteAddr().String(),
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go ctx.pipe(ctx.client, ctx.target, DirClientToServer, &wg)
	go ctx.pipe(ctx.target, ctx.client, DirServerToClient, &wg)
	wg.Wait()
}

type proxyCtx struct {
	client     net.Conn
	target     net.Conn
	store      *AuditStore
	stopCh     chan struct{}
	safety     *SafetyConfig
	rateLim    *RateLimiter
	tokBudget  *TokenBudget
	clientAddr string
}

func (ctx *proxyCtx) pipe(src, dst net.Conn, dir Direction, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("mcp proxy pipe panic", "dir", dir, "error", fmt.Sprintf("%v", r))
		}
	}()
	reader := bufio.NewReaderSize(src, 4*1024*1024)
	for {
		select {
		case <-ctx.stopCh:
			return
		default:
		}
		_ = src.SetReadDeadline(time.Now().Add(30 * time.Second))
		lineBytes, err := reader.ReadSlice('\n')
		if err != nil {
			if err == bufio.ErrBufferFull {
				slog.Warn("mcp proxy line too long, closing connection", "dir", dir)
				return
			}
			if err != io.EOF && !isTimeout(err) {
				slog.Warn("mcp proxy read error", "error", err)
			}
			return
		}
		line := string(bytes.TrimRight(lineBytes, "\r\n"))
		if line == "" {
			continue
		}

		// Parse and analyze
		entry := ctx.analyze(line, dir)

		// Check for injection
		if entry.Injection {
			slog.Warn("mcp proxy INJECTION DETECTED", "type", entry.InjectionTy, "method", entry.Method, "content", TruncateString(entry.Content, 80))
		}

		// Client-to-server: apply safety checks before forwarding
		if dir == DirClientToServer {
			forward := ctx.checkRequestSafety(line, entry, src)
			if !forward {
				continue
			}
		}

		// Server-to-client: apply response safety
		if dir == DirServerToClient {
			line = ctx.applyResponseSafety(line, entry)
		}

		// Optimize context if needed
		if entry.ToksEst > 500 && dir == DirServerToClient {
			opt := OptimizeContext(line)
			if opt.Pruned {
				entry.Pruned = true
				entry.PrunedChars = opt.PrunedLen
				line = opt.Output
				entry.ToksEst = EstimateTokens(line)
			}
		}

		// Audit
		if ctx.store != nil {
			_ = ctx.store.Insert(entry)
		}

		// Forward the (possibly modified) message
		if _, err := dst.Write([]byte(line + "\n")); err != nil {
			return
		}
	}
}

// checkRequestSafety applies rate limiting, policy checks, tool approval, and
// token budget for client-to-server messages. Returns false to drop the message.
func (ctx *proxyCtx) checkRequestSafety(line string, entry *LogEntry, clientConn net.Conn) bool {
	safety := ctx.safety
	if safety == nil {
		return true
	}

	// Rate limiting
	if ctx.rateLim != nil && !ctx.rateLim.Allow(ctx.clientAddr) {
		slog.Warn("mcp proxy rate limit exceeded", "client", ctx.clientAddr, "method", entry.Method)
		writeError(clientConn, entry.RequestID, -32000, "rate limit exceeded: too many requests per minute")
		return false
	}

	// Token budget check (request tokens)
	if ctx.tokBudget != nil {
		reqTokens := EstimateTokens(line)
		if !ctx.tokBudget.Allow(ctx.clientAddr, reqTokens) {
			slog.Warn("mcp proxy token budget exceeded", "client", ctx.clientAddr)
			writeError(clientConn, entry.RequestID, -32001, "token budget exceeded for current window")
			return false
		}
	}

	// Policy check on method
	if safety.CheckPolicy != nil && entry.Method != "" {
		allowed, reason := safety.CheckPolicy(ctx.clientAddr, entry.Method)
		if !allowed {
			slog.Warn("mcp proxy policy denied", "method", entry.Method, "client", ctx.clientAddr, "reason", reason)
			writeError(clientConn, entry.RequestID, -32002, fmt.Sprintf("method %s denied by policy: %s", entry.Method, reason))
			return false
		}
	}

	// Tool-use approval
	if safety.ApproveTool != nil && entry.Method == MethodToolsCall {
		var msg JSONRPC
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			toolName := extractToolName(msg.Params)
			approved, reason := safety.ApproveTool(ctx.clientAddr, toolName, msg.Params)
			if !approved {
				slog.Warn("mcp proxy tool call denied", "tool", toolName, "client", ctx.clientAddr, "reason", reason)
				errMsg := fmt.Sprintf("tool %s not approved: %s", toolName, reason)
				writeError(clientConn, entry.RequestID, -32003, errMsg)
				return false
			}
		} else {
			slog.Warn("mcp proxy failed to parse tools/call message", "error", err)
			writeError(clientConn, entry.RequestID, -32003, "failed to parse tool call")
			return false
		}
	}

	return true
}

// applyResponseSafety applies PII redaction and tracks response tokens for
// server-to-client messages. Returns the (possibly modified) line.
func (ctx *proxyCtx) applyResponseSafety(line string, entry *LogEntry) string {
	if ctx.safety == nil {
		return line
	}

	// PII redaction
	if ctx.safety.RedactPII {
		// Only redact content-bearing responses, not the JSON envelope
		var msg JSONRPC
		if err := json.Unmarshal([]byte(line), &msg); err == nil && msg.IsResponse() && msg.Result != nil {
			resultStr := string(msg.Result)
			redacted := RedactPII(resultStr)
			if redacted != resultStr {
				entry.Pruned = true
				entry.PrunedChars += len(resultStr) - len(redacted)
				// Rebuild the line with redacted result
				msg.Result = json.RawMessage(redacted)
				if rebuilt, err := json.Marshal(msg); err == nil {
					line = string(rebuilt)
				}
			}
		}
	}

	// Token budget tracking for response tokens (monitoring only — doesn't block)
	if ctx.tokBudget != nil {
		respTokens := EstimateTokens(line)
		if !ctx.tokBudget.Allow(ctx.clientAddr, respTokens) {
			slog.Warn("mcp proxy response token budget exceeded", "client", ctx.clientAddr)
		}
	}

	return line
}

// writeError sends a JSON-RPC error response to the client connection.
func writeError(conn net.Conn, reqID string, code int, message string) {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	if reqID != "" && reqID != "(notification)" {
		resp["id"] = reqID
	} else {
		resp["id"] = nil
	}
	body, err := json.Marshal(resp)
	if err != nil {
		return
	}
	body = append(body, '\n')
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, _ = conn.Write(body)
}

func extractToolName(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(params, &p); err == nil {
		return p.Name
	}
	// Try nested arguments
	var p2 struct {
		Arguments struct {
			Name string `json:"name"`
		} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p2); err == nil && p2.Arguments.Name != "" {
		return p2.Arguments.Name
	}
	return ""
}

func (ctx *proxyCtx) analyze(line string, dir Direction) *LogEntry {
	entry := &LogEntry{
		Direction:   dir,
		TimestampNS: time.Now().UnixNano(),
	}

	var msg JSONRPC
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		entry.Content = TruncateString(line, 200)
		entry.ToksEst = EstimateTokens(line)
		if ty, ok := CheckInjection(line); ok {
			entry.Injection = true
			entry.InjectionTy = string(ty)
		}
		return entry
	}

	entry.Method = msg.Method
	if msg.IsNotification() {
		entry.RequestID = "(notification)"
	} else {
		entry.RequestID = string(msg.ID)
		// Strip JSON string quotes from string-typed IDs
		if len(entry.RequestID) >= 2 && entry.RequestID[0] == '"' {
			entry.RequestID = entry.RequestID[1 : len(entry.RequestID)-1]
		}
	}

	// Extract content for injection scanning
	if msg.IsRequest() && msg.Params != nil {
		entry.Content = TruncateString(ScanParams(msg.Params), 500)
	} else if msg.IsResponse() && msg.Result != nil {
		entry.Content = TruncateString(string(msg.Result), 500)
	}
	entry.ToksEst = EstimateTokens(entry.Content)

	// Injection check
	if ty, ok := CheckInjection(entry.Content); ok {
		entry.Injection = true
		entry.InjectionTy = string(ty)
	}

	return entry
}

// DialTarget parses the target URL and dials the MCP server.
func DialTarget(targetURL string) (net.Conn, error) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %w", err)
	}
	switch u.Scheme {
	case "http", "ws":
		return net.DialTimeout("tcp", u.Host, 5*time.Second)
	case "https", "wss":
		return tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", u.Host, &tls.Config{})
	case "tcp":
		return net.DialTimeout("tcp", u.Host, 5*time.Second)
	default:
		return net.DialTimeout("tcp", targetURL, 5*time.Second)
	}
}

func isTimeout(err error) bool {
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return true
	}
	return false
}
