// Command atazs-armatur reads Claude Code session JSON from stdin and prints a
// three-line status:
//
//	dir › branch              Opus 5.5               ctx  ━━━━━━╺━━━━━━━━━   38%
//	atazs-armatur-06                          2h41   ses  ━━━━━━━━━━━━╺━━━   75%
//	me@example.com                           3d 4h  week  ━━━━━━━━━━━━━━━╺   96%
//
// Reads stdin, .git/HEAD, the transcript's modification time, Claude Code's
// session registry (for the name on line 2) and, with --email, its config
// (for the account email on line 3); writes only stdout, and stderr for usage
// errors. No network, no API.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

const usage = `Usage: atazs-armatur [--email] [-v|--version] [-h|--help]

Reads Claude Code's session JSON on stdin and prints the status line on stdout.

  --email        also show the logged-in account's email, from Claude Code's config
  -v, --version  print the version and exit
  -h, --help     print this help and exit`

func main() {
	opts, err := parseArgs(os.Args[1:])
	switch {
	case err != nil:
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	case opts.help:
		fmt.Println(usage)
		return
	case opts.version:
		info, ok := debug.ReadBuildInfo()
		fmt.Println(formatVersion(info, ok))
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
		e.peer = peerName(sessionID(s))
		if opts.email {
			e.email = accountEmail()
		}
		e.modTime, e.hasModTime = transcriptModTime(s.TranscriptPath)
	}
	_, _ = os.Stdout.WriteString(render(s, ok, e))
}

type options struct{ help, version, email bool }

// parseArgs reads the whole command line. An unknown argument is an error, so
// a typo in the statusLine command shows up instead of being ignored. Help
// wins over version.
func parseArgs(args []string) (options, error) {
	var o options
	for _, a := range args {
		switch a {
		case "-h", "--help":
			o.help = true
		case "-v", "--version":
			o.version = true
		case "--email":
			o.email = true
		default:
			return options{}, fmt.Errorf("atazs-armatur: unknown option %q", a)
		}
	}
	o.version = o.version && !o.help
	return o, nil
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
