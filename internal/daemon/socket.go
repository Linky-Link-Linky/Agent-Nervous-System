//go:build !windows

// Package daemon — Unix domain socket implementation for Linux and macOS.
// The Windows named pipe implementation is in socket_windows.go.
// SPDX-License-Identifier: Apache-2.0
package daemon

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// SocketPath returns the Unix socket path.
func SocketPath() string {
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "ans.sock")
	}
	return "/tmp/ans.sock"
}

// Listen creates a Unix socket listener. Removes stale socket; sets mode 0600.
func Listen() (net.Listener, error) {
	path := SocketPath()
	// Remove any existing socket file (ignore ENOENT to avoid TOCTOU)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("removing stale socket %s: %w", path, err)
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		l.Close()
		return nil, fmt.Errorf("setting socket permissions: %w", err)
	}
	return l, nil
}

// Dial connects to the ANS daemon Unix socket with a 5-second timeout.
func Dial() (net.Conn, error) {
	path := SocketPath()
	conn, err := net.DialTimeout("unix", path, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ANS daemon at %s: %w\nRun: ans start", path, err)
	}
	return conn, nil
}
