package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want options
		err  string
	}{
		{nil, options{}, ""},
		{[]string{"--email"}, options{email: true}, ""},
		{[]string{"--email", "--version"}, options{version: true, email: true}, ""},
		{[]string{"-v", "-h"}, options{help: true}, ""},
		{[]string{"--email", "--emial"}, options{}, `atazs-armatur: unknown option "--emial"`},
		// An unknown option wins even over help.
		{[]string{"-h", "--email=true"}, options{}, `atazs-armatur: unknown option "--email=true"`},
	} {
		got, err := parseArgs(tc.args)
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		if got != tc.want || msg != tc.err {
			t.Errorf("parseArgs(%q) = %+v, %q; want %+v, %q", tc.args, got, msg, tc.want, tc.err)
		}
	}
}

// TestMainEmail runs the binary itself: only --email shows the account
// email, and a typo exits 2 with nothing on stdout.
func TestMainEmail(t *testing.T) {
	if args, ok := os.LookupEnv("ARMATUR_ARGS"); ok {
		os.Args = append([]string{"atazs-armatur"}, strings.Fields(args)...)
		main()
		os.Exit(0) // before the test framework adds its PASS line to stdout
	}
	dir := t.TempDir()
	write(t, filepath.Join(dir, ".claude.json"), `{"oauthAccount":{"emailAddress":"user@example.com"}}`)
	run := func(args string) (stdout, stderr string, code int) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainEmail$")
		cmd.Env = append(os.Environ(), "ARMATUR_ARGS="+args, "CLAUDE_CONFIG_DIR="+dir,
			"COLUMNS=80", "NO_COLOR=1")
		cmd.Stdin = strings.NewReader(`{"context_window":{"used_percentage":38}}`)
		var out, errOut strings.Builder
		cmd.Stdout, cmd.Stderr = &out, &errOut
		err := cmd.Run()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return out.String(), errOut.String(), code
	}
	for _, tc := range []struct {
		args      string
		withEmail bool
	}{{"", false}, {"--email", true}} {
		out, _, code := run(tc.args)
		if code != 0 || strings.Count(out, "\n") != 3 || strings.Contains(out, "user@example.com") != tc.withEmail {
			t.Errorf("args %q: exit %d, want 0, three lines, email %v:\n%s", tc.args, code, tc.withEmail, out)
		}
	}
	out, errOut, code := run("--email --emial")
	if code != 2 || out != "" || !strings.Contains(errOut, `unknown option "--emial"`) {
		t.Errorf("typo: exit %d, stdout %q, stderr %q; want 2, nothing, the option named", code, out, errOut)
	}
}
