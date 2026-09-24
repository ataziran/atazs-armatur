// Command atazs-armatur reads Claude Code session JSON from stdin and prints a
// three-line status:
//
//	dir › branch                  ctx  ━━━━╸━━━   38%
//	                  2h41        ses  ━━━━━━╺━   75%
//	                  3d 4h      week  ━━━━━━━╸   96%
//
// Reads stdin, .git/HEAD and the transcript's modification time; writes only
// stdout. No network, no API.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

const usage = `Usage: atazs-armatur [-v|--version] [-h|--help]

Reads Claude Code's session JSON on stdin and prints the status line on stdout.`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version":
			info, ok := debug.ReadBuildInfo()
			fmt.Println(formatVersion(info, ok))
		case "-h", "--help":
			fmt.Println(usage)
		default:
			fmt.Fprintln(os.Stderr, "unknown option: "+os.Args[1])
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(2)
		}
		return
	}
	// A terminal on stdin means nobody is piping a payload in: this is a person
	// who ran the command to see what it does, not a status line call. Waiting
	// for input they will never type looks like a hang.
	if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	defer func() {
		// Never leave the status line with a stack trace: an unexpected failure
		// prints a bare reset. This is the last net, not control flow -- the
		// expected cases are handled by decode returning false.
		if recover() != nil {
			_, _ = os.Stdout.WriteString(reset + "\n")
		}
	}()
	// A read error leaves whatever arrived before it; a payload that is short
	// or empty renders with the fallbacks, which is what should happen anyway.
	stdin, _ := io.ReadAll(os.Stdin)
	s, ok := decode(stdin)
	e := env{
		cols:  terminalCols(),
		now:   float64(time.Now().UnixNano()) / 1e9,
		color: os.Getenv("NO_COLOR") == "",
		clock: clockGlyph(runtime.GOOS, os.Getenv("WT_SESSION")),
	}
	// A malformed payload renders an empty line: nothing to look up for it.
	if ok {
		if e.dir = sessionDir(s); e.dir == "" {
			// An unreachable working directory yields "", and basename("")
			// gives an empty folder name rather than a wrong one.
			e.dir, _ = os.Getwd()
		}
		e.branch = gitBranch(e.dir)
		e.modTime, e.hasModTime = transcriptModTime(s.TranscriptPath)
	}
	_, _ = os.Stdout.WriteString(render(s, ok, e))
}

// transcriptModTime returns when the session transcript last changed, in Unix
// seconds; false when there is no path or it cannot be read. One stat, the
// content is never read.
func transcriptModTime(path string) (float64, bool) {
	if path == "" {
		return 0, false
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return float64(info.ModTime().UnixNano()) / 1e9, true
}
