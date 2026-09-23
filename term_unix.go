//go:build linux || darwin

package main

import (
	"os"
	"syscall"
	"unsafe"
)

func platformCols() int {
	for _, fd := range []uintptr{2, 1, 0} {
		if n := ttyCols(fd); n > 0 {
			return n
		}
	}
	if n := ancestorCols(); n > 0 {
		return n
	}
	// The controlling terminal is still reachable by name even when every
	// standard fd is a pipe.
	if f, err := os.OpenFile("/dev/tty", os.O_RDONLY|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0); err == nil {
		defer f.Close()
		return ttyCols(f.Fd())
	}
	return 0
}

// ttyCols asks the terminal behind fd for its size (TIOCGWINSZ); 0 if fd is
// not a terminal.
func ttyCols(fd uintptr) int {
	var ws struct{ row, col, xpixel, ypixel uint16 }
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd,
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws))); errno != 0 {
		return 0
	}
	return int(ws.col)
}
