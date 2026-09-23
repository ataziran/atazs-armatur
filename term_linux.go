package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// ancestorCols walks up the process tree and asks the first ancestor that
// still holds the pty.
//
// Measured against a build without it: where COLUMNS is not exported, every
// standard fd is a pipe and the process no longer has a controlling terminal
// -- under setsid, or simply as a grandchild of the shell that owns the pty --
// opening /dev/tty fails with ENXIO and the width falls back to 80. The walk
// is the only source that still finds the real one (123 columns against the
// 80-column fallback in the session this was measured in).
func ancestorCols() int {
	pid := "self"
	for range 6 {
		stat, err := os.ReadFile("/proc/" + pid + "/stat")
		if err != nil {
			return 0
		}
		// Fields after the parenthesised command name: state, ppid, ...
		i := strings.LastIndexByte(string(stat), ')')
		if i < 0 {
			return 0
		}
		fields := strings.Fields(string(stat[i+1:]))
		if len(fields) < 2 {
			return 0
		}
		ppid := fields[1]
		if ppid == "0" || ppid == "1" {
			return 0
		}
		for _, fd := range []int{2, 1, 0} {
			// O_NONBLOCK: open() on a FIFO with no writer would block forever.
			f, err := os.OpenFile("/proc/"+ppid+"/fd/"+strconv.Itoa(fd), os.O_RDONLY|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0)
			if err != nil {
				continue
			}
			n := ttyCols(f.Fd())
			f.Close()
			if n > 0 {
				return n
			}
		}
		pid = ppid
	}
	return 0
}
