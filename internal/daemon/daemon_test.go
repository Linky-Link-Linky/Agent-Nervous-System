package daemon

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/receipt"
)

func TestPingPong(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	// Create a test listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)

	// Give daemon time to start
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	// Send ping
	if err := WriteFrame(conn, MsgPing, nil); err != nil {
		t.Fatalf("WriteFrame(Ping) failed: %v", err)
	}

	// Expect pong
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	f, err := ReadFrame(conn)
	if err != nil {
		t.Fatalf("ReadFrame() failed: %v", err)
	}
	if f.Type != MsgPong {
		t.Errorf("Response type = 0x%02x, want MsgPong (0x%02x)", f.Type, MsgPong)
	}
}

func TestRegisterAndSignAppend(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	// Register agent
	regReq := RegisterReq{Name: "test-agent", Version: "1.0.0", Owner: "tester"}
	if err := WriteJSON(conn, MsgRegister, regReq); err != nil {
		t.Fatalf("WriteJSON(Register) failed: %v", err)
	}
	var regResp RegisterResp
	if _, err := ReadJSON(conn, &regResp); err != nil {
		t.Fatalf("ReadJSON(RegisterResp) failed: %v", err)
	}
	if regResp.AgentID == "" {
		t.Fatal("RegisterResp.AgentID is empty")
	}

	// Sign pre-action
	preReq := SignAppendReq{
		AgentID: regResp.AgentID, Phase: "pre", ActionType: "file.write",
		PayloadHash: "abc123", PayloadSummary: "write test.txt",
		PolicyDecision: "allow", AuthContext: "test",
	}
	if err := WriteJSON(conn, MsgSignAppend, preReq); err != nil {
		t.Fatalf("WriteJSON(SignAppend pre) failed: %v", err)
	}
	var preResp SignAppendResp
	if _, err := ReadJSON(conn, &preResp); err != nil {
		t.Fatalf("ReadJSON(SignAppendResp pre) failed: %v", err)
	}
	if preResp.ReceiptID == "" {
		t.Error("Pre-receipt ReceiptID is empty")
	}
	if preResp.ChainIndex != 1 {
		t.Errorf("Pre-receipt ChainIndex = %d, want 1", preResp.ChainIndex)
	}

	// Sign post-action
	postReq := SignAppendReq{
		AgentID: regResp.AgentID, Phase: "post", ActionType: "file.write",
		PayloadHash: "abc123", PayloadSummary: "write test.txt",
		Outcome: "success", OutcomeSummary: "wrote 42 bytes", DurationMS: 10,
		PreReceiptID: preResp.ReceiptID,
	}
	if err := WriteJSON(conn, MsgSignAppend, postReq); err != nil {
		t.Fatalf("WriteJSON(SignAppend post) failed: %v", err)
	}
	var postResp SignAppendResp
	if _, err := ReadJSON(conn, &postResp); err != nil {
		t.Fatalf("ReadJSON(SignAppendResp post) failed: %v", err)
	}
	if postResp.ReceiptID == "" {
		t.Error("Post-receipt ReceiptID is empty")
	}
	if postResp.ChainIndex != 2 {
		t.Errorf("Post-receipt ChainIndex = %d, want 2", postResp.ChainIndex)
	}
}

func TestVerifyValid(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	// Register and append
	regReq := RegisterReq{Name: "test", Version: "1.0.0"}
	WriteJSON(conn, MsgRegister, regReq)
	var regResp RegisterResp
	ReadJSON(conn, &regResp)

	preReq := SignAppendReq{
		AgentID: regResp.AgentID, Phase: "pre", ActionType: "custom",
		PayloadHash: "abc", PayloadSummary: "test", PolicyDecision: "allow",
	}
	WriteJSON(conn, MsgSignAppend, preReq)
	var preResp SignAppendResp
	ReadJSON(conn, &preResp)

	// Verify
	verReq := VerifyReq{ReceiptID: preResp.ReceiptID}
	if err := WriteJSON(conn, MsgVerify, verReq); err != nil {
		t.Fatalf("WriteJSON(Verify) failed: %v", err)
	}
	var verResp VerifyResp
	if _, err := ReadJSON(conn, &verResp); err != nil {
		t.Fatalf("ReadJSON(VerifyResp) failed: %v", err)
	}
	if !verResp.Valid {
		t.Errorf("VerifyResp.Valid = false, want true; error: %s", verResp.Error)
	}
}

func TestQueryFilter(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	// Register 2 agents
	WriteJSON(conn, MsgRegister, RegisterReq{Name: "agent1", Version: "1.0"})
	var reg1 RegisterResp
	ReadJSON(conn, &reg1)

	WriteJSON(conn, MsgRegister, RegisterReq{Name: "agent2", Version: "1.0"})
	var reg2 RegisterResp
	ReadJSON(conn, &reg2)

	// Append receipts for both
	for _, aid := range []string{reg1.AgentID, reg2.AgentID, reg1.AgentID} {
		req := SignAppendReq{
			AgentID: aid, Phase: "pre", ActionType: "custom",
			PayloadHash: "test", PolicyDecision: "allow",
		}
		WriteJSON(conn, MsgSignAppend, req)
		var resp SignAppendResp
		ReadJSON(conn, &resp)
	}

	// Query by agent1
	queryReq := QueryReq{AgentID: reg1.AgentID, Limit: 10}
	if err := WriteJSON(conn, MsgQuery, queryReq); err != nil {
		t.Fatalf("WriteJSON(Query) failed: %v", err)
	}
	var queryResp struct {
		Receipts []*receipt.Receipt `json:"receipts"`
	}
	if _, err := ReadJSON(conn, &queryResp); err != nil {
		t.Fatalf("ReadJSON(QueryResp) failed: %v", err)
	}
	if len(queryResp.Receipts) != 2 {
		t.Errorf("Query returned %d receipts, want 2", len(queryResp.Receipts))
	}
	for _, r := range queryResp.Receipts {
		if r.AgentID != reg1.AgentID {
			t.Errorf("Query returned receipt with AgentID=%s, want %s", r.AgentID, reg1.AgentID)
		}
	}
}

func TestStatus(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	if err := WriteJSON(conn, MsgStatus, struct{}{}); err != nil {
		t.Fatalf("WriteJSON(Status) failed: %v", err)
	}
	var status StatusResp
	if _, err := ReadJSON(conn, &status); err != nil {
		t.Fatalf("ReadJSON(StatusResp) failed: %v", err)
	}
	if status.Uptime == "" {
		t.Error("StatusResp.Uptime is empty")
	}
	if status.TotalReceipts < 0 {
		t.Errorf("StatusResp.TotalReceipts = %d, want >= 0", status.TotalReceipts)
	}
}

func TestConcurrentClients(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	// 20 goroutines each register an agent and append 5 receipts
	var wg sync.WaitGroup
	errors := make(chan error, 100)
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", l.Addr().String())
			if err != nil {
				errors <- err
				return
			}
			defer conn.Close()

			// Register
			regReq := RegisterReq{Name: "agent", Version: "1.0"}
			WriteJSON(conn, MsgRegister, regReq)
			var regResp RegisterResp
			ReadJSON(conn, &regResp)

			// Append 5 receipts
			for i := 0; i < 5; i++ {
				req := SignAppendReq{
					AgentID: regResp.AgentID, Phase: "pre", ActionType: "custom",
					PayloadHash: "test", PolicyDecision: "allow",
				}
				WriteJSON(conn, MsgSignAppend, req)
				var resp SignAppendResp
				if _, err := ReadJSON(conn, &resp); err != nil {
					errors <- err
					return
				}
			}
		}(g)
	}
	wg.Wait()

	// Check status
	conn, _ := net.Dial("tcp", l.Addr().String())
	defer conn.Close()
	WriteJSON(conn, MsgStatus, struct{}{})
	var status StatusResp
	ReadJSON(conn, &status)

	if status.TotalReceipts != 100 {
		t.Errorf("TotalReceipts = %d, want 100", status.TotalReceipts)
	}

	select {
	case err := <-errors:
		t.Errorf("Concurrent client error: %v", err)
	default:
	}
}

func TestDaemonTokenLifecycle(t *testing.T) {
	t.Setenv("ANS_DEV", "1")
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	// Request token
	body, _ := json.Marshal(TokenRequestReq{
		AgentID:    "test_agent",
		Resource:   "s3://bucket/key",
		Action:     "file.read",
		TTLSeconds: 30,
	})
	if err := WriteFrame(conn, MsgTokenRequest, body); err != nil {
		t.Fatalf("WriteFrame(MsgTokenRequest) failed: %v", err)
	}
	var tokenResp TokenRequestResp
	recvExpect(t, conn, MsgTokenResp, &tokenResp)
	if !tokenResp.Success {
		t.Fatalf("Token request failed: %s", tokenResp.Message)
	}
	if tokenResp.TokenID == "" {
		t.Error("TokenID is empty")
	}
	if tokenResp.TokenType == "" {
		t.Error("TokenType is empty")
	}

	// List tokens
	body, _ = json.Marshal(TokenListReq{})
	if err := WriteFrame(conn, MsgTokenList, body); err != nil {
		t.Fatalf("WriteFrame(MsgTokenList) failed: %v", err)
	}
	var listResp TokenListResp
	recvExpect(t, conn, MsgTokenListResp, &listResp)
	if len(listResp.Tokens) < 1 {
		t.Error("TokenList returned 0 tokens, want >= 1")
	}
	found := false
	for _, entry := range listResp.Tokens {
		if entry.TokenID == tokenResp.TokenID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Listed tokens does not contain newly created token")
	}

	// Revoke token — expected to fail because dev provider revocation is not implemented
	body, _ = json.Marshal(TokenRevokeReq{TokenID: tokenResp.TokenID})
	if err := WriteFrame(conn, MsgTokenRevoke, body); err != nil {
		t.Fatalf("WriteFrame(MsgTokenRevoke) failed: %v", err)
	}
	var errResp ErrorResp
	recvExpect(t, conn, MsgError, &errResp)
	if errResp.Message == "" {
		t.Error("expected error message from revoke")
	}
}

func recvExpect(t *testing.T, conn net.Conn, expectedType byte, v interface{}) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	f, err := ReadFrame(conn)
	if err != nil {
		t.Fatalf("ReadFrame() failed (expected type 0x%02x): %v", expectedType, err)
	}
	if f.Type != expectedType {
		t.Fatalf("response type = 0x%02x, want 0x%02x", f.Type, expectedType)
	}
	if err := json.Unmarshal(f.Body, v); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}
}

func TestUnknownMessageType(t *testing.T) {
	chainPath := filepath.Join(t.TempDir(), "chain.db")
	keystorePath := t.TempDir()
	d, err := NewWithPaths(chainPath, keystorePath)
	if err != nil {
		t.Fatalf("NewWithPaths() failed: %v", err)
	}

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go d.RunOnListener(ctx, l)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial() failed: %v", err)
	}
	defer conn.Close()

	// Send unknown message type
	if err := WriteFrame(conn, 0xFE, []byte("{}")); err != nil {
		t.Fatalf("WriteFrame(0xFE) failed: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	f, err := ReadFrame(conn)
	if err != nil {
		t.Fatalf("ReadFrame() failed: %v", err)
	}
	if f.Type != MsgError {
		t.Errorf("Response type = 0x%02x, want MsgError (0x%02x)", f.Type, MsgError)
	}

	var errResp ErrorResp
	json.Unmarshal(f.Body, &errResp)
	if errResp.Message == "" {
		t.Error("ErrorResp.Message is empty")
	}
}
