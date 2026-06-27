package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	mock "ans/internal/client"
)

func captureExit() {
	exitErr = func(code int) {
		panic(fmt.Sprintf("exit:%d", code))
	}
}

func resetExit() {
	exitErr = func(code int) { os.Exit(code) }
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = old
	return buf.String()
}

func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	f()
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = old
	return buf.String()
}

func TestRunVersion(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("version", []string{}, c) })
	if !strings.Contains(out, "ans v") {
		t.Fatalf("expected version output, got %q", out)
	}
}

func TestRunHelp(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("help", []string{}, c) })
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected help output, got %q", out)
	}
}

func TestRunInit(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("init", []string{}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunDoctor(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("doctor", []string{}, c) })
	if !strings.Contains(out, "DIAGNOSTICS") {
		t.Fatalf("expected diagnostics, got %q", out)
	}
}

func TestRunStatus(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("status", []string{}, c) })
	if !strings.Contains(out, "STATUS") {
		t.Fatalf("expected status output, got %q", out)
	}
}

func TestRunStart(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("start", []string{}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunStop(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("stop", []string{}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunUpdate(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("update", []string{}, c) })
	if !strings.Contains(out, "Updated to") {
		t.Fatalf("expected update output, got %q", out)
	}
}

func TestRunUninstall(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("uninstall", []string{}, c) })
	if !strings.Contains(out, "uninstalled") {
		t.Fatalf("expected uninstall output, got %q", out)
	}
}

func TestRunChain(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("chain", []string{}, c) })
	if !strings.Contains(out, "─") {
		t.Fatalf("expected chain output, got %q", out)
	}
}

func TestRunVerifyNoArgs(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	var exited bool
	orig := exitErr
	exitErr = func(code int) { exited = true; panic("exit") }
	func() {
		defer func() { recover() }()
		Run("verify", []string{}, c)
	}()
	if !exited {
		t.Fatal("expected exit for missing args")
	}
	exitErr = orig
}

func TestRunVerifyReceipt(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("verify", []string{"abc123"}, c) })
	if !strings.Contains(out, "VALID") {
		t.Fatalf("expected VALID, got %q", out)
	}
}

func TestRunVerifyChain(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("verify", []string{"--chain"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected verified output, got %q", out)
	}
}

func TestRunAgents(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("agents", []string{}, c) })
	if !strings.Contains(out, "AGENTS") {
		t.Fatalf("expected agents output, got %q", out)
	}
}

func TestRunRegister(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("register", []string{"--name", "test-agent"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunRegisterMissingName(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	var exited bool
	orig := exitErr
	exitErr = func(code int) { exited = true; panic("exit") }
	func() {
		defer func() { recover() }()
		Run("register", []string{}, c)
	}()
	if !exited {
		t.Fatal("expected exit for missing --name")
	}
	exitErr = orig
}

func TestRunExport(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("export", []string{"--format", "jsonl", "--output", "/tmp/out.jsonl"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunExportMissingOutput(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	var exited bool
	orig := exitErr
	exitErr = func(code int) { exited = true; panic("exit") }
	func() {
		defer func() { recover() }()
		Run("export", []string{"--format", "pdf"}, c)
	}()
	if !exited {
		t.Fatal("expected exit for missing --output")
	}
	exitErr = orig
}

func TestRunPrune(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("prune", []string{"--up-to", "10"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunRotate(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("rotate", []string{"agent_123"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunTimeTravel(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("time-travel", []string{"3"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunSnapshotTake(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("snapshot", []string{"take"}, c) })
	if !strings.Contains(out, "✓") {
		t.Fatalf("expected success, got %q", out)
	}
}

func TestRunSnapshotDiff(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("snapshot", []string{"diff"}, c) })
	if !strings.Contains(out, "+") {
		t.Fatalf("expected diff output, got %q", out)
	}
}

func TestRunSnapshotList(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("snapshot", []string{"list"}, c) })
	if !strings.Contains(out, "Slot") {
		t.Fatalf("expected snapshots, got %q", out)
	}
}

func TestRunSnapshots(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("snapshots", []string{}, c) })
	if !strings.Contains(out, "Slot") {
		t.Fatalf("expected snapshots, got %q", out)
	}
}

func TestRunCompensateMissingIndex(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	var exited bool
	orig := exitErr
	exitErr = func(code int) { exited = true; panic("exit") }
	func() {
		defer func() { recover() }()
		Run("compensate", []string{}, c)
	}()
	if !exited {
		t.Fatal("expected exit for missing index")
	}
	exitErr = orig
}

func TestRunPolicyList(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("policy", []string{"list"}, c) })
	if !strings.Contains(out, "POLICIES") {
		t.Fatalf("expected policies, got %q", out)
	}
}

func TestRunTokenList(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("token", []string{"list"}, c) })
	if !strings.Contains(out, "TOKENS") {
		t.Fatalf("expected tokens, got %q", out)
	}
}

func TestRunMCPStatus(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("mcp", []string{"status"}, c) })
	if !strings.Contains(out, "STATUS") {
		t.Fatalf("expected mcp status, got %q", out)
	}
}

func TestRunUnknownSubcmd(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	var exited bool
	orig := exitErr
	exitErr = func(code int) { exited = true; panic("exit") }
	func() {
		defer func() { recover() }()
		Run("nonexistent", []string{}, c)
	}()
	if !exited {
		t.Fatal("expected exit for unknown subcommand")
	}
	exitErr = orig
}

func TestRunChainWithFlags(t *testing.T) {
	captureExit()
	defer resetExit()
	c := mock.NewMockClient()
	out := captureStdout(func() { Run("chain", []string{"--n", "5", "--agent", "ans_test"}, c) })
	if !strings.Contains(out, "┌─") {
		t.Fatalf("expected chain output, got %q", out)
	}
}
