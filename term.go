package main

import (
	"os"
	"strconv"
	"strings"
)

// maxCols caps every width source. A COLUMNS of 999999 is not a terminal, and
// without the cap the padding alone would push a megabyte into the prompt.
const maxCols = 1000

// terminalCols returns the real terminal width. Claude Code hands the command
// a pipe on all three standard fds and documents that it sets COLUMNS (and
// LINES) to the terminal size first; the platform-specific probe of the
// terminal the session runs in covers callers that do not export COLUMNS.
func terminalCols() int {
	if cols, ok := colsFromEnv(os.Getenv("COLUMNS")); ok {
		return cols
	}
	if n := platformCols(); n > 0 {
		return min(n, maxCols)
	}
	return 80
}

// colsFromEnv reads a width from the COLUMNS value. Only a positive number
// counts: a zero or negative COLUMNS -- some shells and CI runners export one
// -- means "unknown", and falling through to the probe beats collapsing the
// line to its minimum.
func colsFromEnv(value string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return 0, false
	}
	return min(n, maxCols), true
}
