package main

import (
	"syscall"
	"unsafe"
)

var getConsoleScreenBufferInfo = syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleScreenBufferInfo")

type consoleInfo struct {
	size, cursor          struct{ x, y int16 }
	attributes            uint16
	left, top, right, bot int16
	maxSize               struct{ x, y int16 }
}

func platformCols() int {
	for _, h := range []syscall.Handle{syscall.Stderr, syscall.Stdout} {
		if n := consoleCols(h); n > 0 {
			return n
		}
	}
	// Standard handles are pipes under Claude Code; the console itself is
	// still reachable by name if the process is attached to one.
	name, _ := syscall.UTF16PtrFromString("CONOUT$")
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		return 0
	}
	defer syscall.CloseHandle(h)
	return consoleCols(h)
}

func consoleCols(h syscall.Handle) int {
	var info consoleInfo
	if ok, _, _ := getConsoleScreenBufferInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&info))); ok == 0 {
		return 0
	}
	return int(info.right-info.left) + 1
}
