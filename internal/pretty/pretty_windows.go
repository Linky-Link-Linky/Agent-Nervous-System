//go:build windows
// +build windows

package pretty

import (
	"os"

	"golang.org/x/sys/windows"
)

// Enable ANSI color processing on Windows consoles.
func init() {
	for _, f := range []*os.File{os.Stderr, os.Stdout} {
		handle := windows.Handle(f.Fd())
		var mode uint32
		if err := windows.GetConsoleMode(handle, &mode); err != nil {
			continue
		}
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		windows.SetConsoleMode(handle, mode)
	}
}
