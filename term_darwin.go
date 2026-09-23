package main

// ancestorCols has no /proc to walk on macOS; /dev/tty covers it.
func ancestorCols() int { return 0 }
