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
// Override with ANS_SOCK_PATH env var.
func SocketPath() string {
	if v := os.Getenv("ANS_SOCK_PATH"); v != "" {
		return v
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "ans.sock")
	}
	return "/tmp/ans.sock"
}

// Listen creates a Unix socket listener. Uses a PID lock file for exclusive
// ownership to prevent TOCTOU / symlink attacks. Socket mode 0600.
func Listen() (net.Listener, error) {
	path := SocketPath()
	lockPath := path + ".lock"
	// Acquire exclusive lock file
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("daemon already running (lock: %s)", lockPath)
		}
		return nil, fmt.Errorf("acquiring lock %s: %w", lockPath, err)
	}
	fmt.Fprintf(lf, "%d\n", os.Getpid())
	lf.Close()

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		os.Remove(lockPath)
		return nil, fmt.Errorf("removing stale socket %s: %w", path, err)
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		os.Remove(path)
		os.Remove(lockPath)
		return nil, fmt.Errorf("listening on %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		l.Close()
		os.Remove(path)
		os.Remove(lockPath)
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
