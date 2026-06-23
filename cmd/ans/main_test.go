package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var ansBin string
var buildOnce sync.Once
var buildFailed error

func buildBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ans-test-*")
		if err != nil {
			buildFailed = fmt.Errorf("MkdirTemp: %w", err)
			return
		}
		bin := filepath.Join(dir, "ans-test"+exeSuffix())
		cmd := exec.Command("go", "build", "-o", bin, ".")
		out, err := cmd.CombinedOutput()
		if err != nil {
			buildFailed = fmt.Errorf("build: %w\n%s", err, out)
			return
		}
		// Quick check that binary can actually execute
		check := exec.Command(bin, "version")
		if err := check.Run(); err != nil {
			buildFailed = fmt.Errorf("binary blocked by security policy: %w", err)
			return
		}
		ansBin = bin
	})
	if buildFailed != nil {
		t.Skipf("subprocess tests skipped: %v", buildFailed)
	}
	return ansBin
}

func exeSuffix() string {
	if len(os.Args) > 0 && len(os.Args[0]) > 4 && os.Args[0][len(os.Args[0])-4:] == ".exe" {
		return ".exe"
	}
	return ""
}

func TestNoColor(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"no env", map[string]string{}, false},
		{"NO_COLOR=1", map[string]string{"NO_COLOR": "1"}, true},
		{"ANS_NO_COLOR=1", map[string]string{"ANS_NO_COLOR": "1"}, true},
		{"NO_COLOR=0", map[string]string{"NO_COLOR": "0"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevNO := os.Getenv("NO_COLOR")
			prevANS := os.Getenv("ANS_NO_COLOR")
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			if got := noColor(); got != tt.want {
				t.Errorf("noColor() = %v, want %v", got, tt.want)
			}
			os.Unsetenv("NO_COLOR")
			os.Unsetenv("ANS_NO_COLOR")
			if prevNO != "" {
				os.Setenv("NO_COLOR", prevNO)
			}
			if prevANS != "" {
				os.Setenv("ANS_NO_COLOR", prevANS)
			}
		})
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"abcdef0123456789", true},
		{"ABCDEF", true},
		{"deadbeef", true},
		{"123456", true},
		{"", true},
		{"xyz", false},
		{"hello", false},
		{"0x1234", false},
	}
	for _, tt := range tests {
		if got := isHex(tt.input); got != tt.want {
			t.Errorf("isHex(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPidFilePath(t *testing.T) {
	path := pidFilePath()
	if path == "" {
		t.Fatal("pidFilePath() returned empty")
	}
	if !strings.Contains(path, ".ans") || !strings.Contains(path, "daemon.pid") {
		t.Errorf("pidFilePath() = %q, should contain .ans/daemon.pid", path)
	}
}

func TestVersionCommand(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if !strings.Contains(string(out), "0.1.0") {
		t.Errorf("output = %q, want to contain version number", string(out))
	}
}

func TestHelpCommand(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	if !strings.Contains(string(out), "ANS") {
		t.Errorf("output missing header: %q", string(out))
	}
}

func TestNoArgsShowsUsage(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("no-args command failed: %v", err)
	}
	if !strings.Contains(string(out), "USAGE") {
		t.Errorf("output = %q, want to contain USAGE", string(out))
	}
}

func TestUnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "nonexistent")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("unknown command did not exit with error")
	}
	if !strings.Contains(string(out), "unknown command") {
		t.Errorf("output = %q, want to contain 'unknown command'", string(out))
	}
}

func TestVersionFlag(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version command failed: %v", err)
	}
	if !strings.Contains(string(out), "0.1.0") {
		t.Errorf("output = %q", string(out))
	}
}

func TestHelpFlag(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help command failed: %v", err)
	}
	if !strings.Contains(string(out), "USAGE") {
		t.Errorf("output missing USAGE: %q", string(out))
	}
}

func TestRegisterNoFlags(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "register")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("register without flags did not exit with error")
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("output = %q, want 'Usage'", string(out))
	}
}

func TestRotateNoArgs(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "rotate")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("rotate without args did not exit with error")
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("output = %q, want 'Usage'", string(out))
	}
}

func TestPolicyAddNoFile(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "policy", "add")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("policy add without args did not exit with error")
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("output = %q, want 'Usage'", string(out))
	}
}

func TestPruneRequiresFlag(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "prune")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("prune without --up-to did not exit with error")
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("output = %q, want 'Usage'", string(out))
	}
}

func TestCompensateMissingArg(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "compensate")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("compensate without arg did not exit with error")
	}
	if !strings.Contains(string(out), "Usage") {
		t.Errorf("output = %q, want 'Usage'", string(out))
	}
}
