//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

// EnableRawMode switches the Windows terminal to raw / non-canonical mode and returns a restore function.
func EnableRawMode() (func(), error) {
	handle := windows.Handle(os.Stdin.Fd())
	var oldMode uint32
	if err := windows.GetConsoleMode(handle, &oldMode); err != nil {
		return func() {}, err
	}

	rawMode := oldMode &^ (windows.ENABLE_LINE_INPUT | windows.ENABLE_ECHO_INPUT | windows.ENABLE_PROCESSED_INPUT)
	if err := windows.SetConsoleMode(handle, rawMode); err != nil {
		return func() {}, err
	}

	restore := func() {
		_ = windows.SetConsoleMode(handle, oldMode)
	}
	return restore, nil
}
