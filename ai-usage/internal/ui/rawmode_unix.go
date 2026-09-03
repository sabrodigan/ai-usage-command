//go:build !windows

package ui

import (
	"os"

	"golang.org/x/sys/unix"
)

// EnableRawMode switches the terminal to non-canonical / raw mode and returns a restore function.
func EnableRawMode() (func(), error) {
	fd := int(os.Stdin.Fd())
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return func() {}, err
	}

	oldState := *termios
	termios.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return func() {}, err
	}

	restore := func() {
		_ = unix.IoctlSetTermios(fd, unix.TCSETS, &oldState)
	}
	return restore, nil
}
