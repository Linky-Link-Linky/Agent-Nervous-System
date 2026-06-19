//go:build windows

// Package daemon — Windows named pipe implementation.
// Uses github.com/Microsoft/go-winio. Only compiled on Windows.
// SPDX-License-Identifier: MIT
package daemon

import (
	"fmt"
	"net"

	winio "github.com/Microsoft/go-winio"
)

const windowsPipeName = `\\.\pipe\ans`

func SocketPath() string { return windowsPipeName }

// Listen creates a Windows named pipe restricted to the current user.
func Listen() (net.Listener, error) {
	l, err := winio.ListenPipe(windowsPipeName, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;OW)", // owner-only access
		InputBufferSize:    65536,
		OutputBufferSize:   65536,
	})
	if err != nil {
		return nil, fmt.Errorf("listening on named pipe %s: %w", windowsPipeName, err)
	}
	return l, nil
}

// Dial connects to the ANS daemon named pipe.
func Dial() (net.Conn, error) {
	conn, err := winio.DialPipe(windowsPipeName, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ANS daemon at %s: %w\nRun: ans start", windowsPipeName, err)
	}
	return conn, nil
}
