// Command atazs-armatur reads Claude Code session JSON from stdin and prints a
// three-line status:
//
//	dir › branch                  ctx  ━━━━╸━━━   38%
//	                  2h41        ses  ━━━━━━╺━   75%
//	                  3d 4h      week  ━━━━━━━╸   96%
//
// Pure stdin -> stdout. No network, no files, no API.
package main

import (
	"fmt"
	"io"
	"os"
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
	out := render(stdin, env{
		cols:  terminalCols(),
		now:   float64(time.Now().UnixNano()) / 1e9,
		color: os.Getenv("NO_COLOR") == "",
		getcwd: func() string {
			// An unreachable working directory yields "", and basename("")
			// gives an empty folder name rather than a wrong one.
			wd, _ := os.Getwd()
			return wd
		},
		branchOf: gitBranch,
	})
	_, _ = os.Stdout.WriteString(out)
}
