package main

// Pure rendering: session JSON in, status lines out. Everything the output
// depends on besides stdin -- terminal width, clock, git branch, working
// directory -- is passed in, so golden tests can pin it.

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	reset, bold   = "\033[0m", "\033[1m"
	cyan, magenta = "\033[36m", "\033[35m"

	// 256-colour, never \033[2m: the dim attribute is dropped by conhost, winpty
	// and some tmux setups, and a frame rendered at full brightness collapses
	// the look.
	frame, track       = "\033[38;5;240m", "\033[38;5;236m"
	green, yellow, red = "\033[38;5;71m", "\033[38;5;179m", "\033[38;5;167m"

	// Only U+2500-U+257F (Box Drawing): Windows Terminal draws this range
	// itself (AtlasEngine, BuiltinGlyphs.cpp) and Consolas -- the conhost
	// default on Windows 10 -- carries all of it. Block Elements
	// (U+2580-U+259F) are NOT safe: Consolas lacks the eighth blocks, conhost
	// has no fallback -> tofu. Same for symbols like U+2387; the branch marker
	// is a plain '›'.
	bar, half, tip = "━", "╸", "╺" // U+2501, U+2578, U+257A

	barWidth = 16
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;:?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)`)

type env struct {
	cols     int
	now      float64
	color    bool
	getcwd   func() string
	branchOf func(cwd string) string
}

// render returns the complete output for one invocation.
func render(stdin []byte, e env) string {
	out := reset + "\n"
	if s, ok := decode(stdin); ok {
		out = renderLines(s, e)
	}
	// NO_COLOR is honoured at the exit rather than in every writer: the same
	// regexp that measures display width strips the sequences again.
	if !e.color {
		out = ansi.ReplaceAllString(out, "")
	}
	return out
}

func renderLines(s session, e env) string {
	// Slack on the right: Claude Code truncates every status line a few columns
	// short of the terminal edge (and replaces the tail with a single '…'), on
	// top of notifications and the verbose-mode token counter. Measured
	// empirically -- do not shave this down without re-checking in a live session.
	cols := max(20, e.cols-4)

	// Only a non-empty string is a path; anything else counts as absent and
	// falls through to the next source.
	cwd, _ := s.Workspace.CurrentDir.(string)
	if cwd == "" {
		cwd, _ = s.CWD.(string)
	}
	if cwd == "" {
		cwd = e.getcwd()
	}
	name := sanitize(basename(cwd))
	branch := sanitize(e.branchOf(cwd))

	var rows []string

	// used_percentage is pre-calculated by Claude Code from context_window
	// (input + cache_creation + cache_read tokens) / context_window_size.
	// Fall back to a manual calculation only if it's null/absent.
	var used float64
	ctx := s.ContextWindow
	switch {
	case ctx.UsedPercentage != nil:
		used = *ctx.UsedPercentage
	case ctx.TotalInput != nil && ctx.WindowSize != nil && *ctx.WindowSize != 0:
		used = *ctx.TotalInput / *ctx.WindowSize * 100
	}
	rows = append(rows, meterRow("", "ctx", used))

	// rate_limits is absent entirely for API-key use, and either window can be
	// missing on its own. A missing window drops its whole row -- better than
	// claiming 0%.
	for _, w := range [...]struct {
		win   *window
		label string
	}{{s.RateLimits.FiveHour, "ses"}, {s.RateLimits.SevenDay, "week"}} {
		if w.win == nil || w.win.UsedPercentage == nil {
			continue
		}
		rows = append(rows, meterRow(fmtReset(w.win.ResetsAt, e.now), w.label, *w.win.UsedPercentage))
	}

	blockWidth := 0
	for _, r := range rows {
		blockWidth = max(blockWidth, displayWidth(r))
	}

	// Line 1: text left, ctx meter flush right. The meter is never truncated --
	// it is the content; the branch gives way first, then the folder name.
	budget := cols - blockWidth - 2
	for branch != "" && displayWidth(plain(name, branch)) > budget {
		branch = shorten(branch, "")
	}
	for len([]rune(name)) > 1 && displayWidth(plain(name, branch)) > budget {
		name = shorten(name, "…")
	}

	left := compose(name, branch)
	pad := max(1, cols-blockWidth-displayWidth(left))
	lines := []string{left + strings.Repeat(" ", pad) + rightAlign(rows[0], blockWidth)}
	for _, r := range rows[1:] {
		lines = append(lines, rightAlign(r, cols))
	}

	// Each line starts with RESET too: it clears stale SGR state and keeps the
	// leading whitespace from being trimmed by the renderer.
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(reset + l + reset + "\n")
	}
	return b.String()
}

// shorten drops the last two runes and appends '…'; at two runes or fewer it
// returns short instead.
func shorten(s, short string) string {
	r := []rune(s)
	if len(r) > 2 {
		return string(r[:len(r)-2]) + "…"
	}
	return short
}

func compose(name, branch string) string {
	if branch != "" {
		return bold + cyan + name + reset + " " + frame + "›" + reset + " " + magenta + branch + reset
	}
	return bold + cyan + name + reset
}

func plain(name, branch string) string {
	if branch != "" {
		return name + " › " + branch
	}
	return name
}

// basename accepts both separators so native Windows paths work too.
func basename(p string) string {
	s := strings.TrimRight(p, `/\`)
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}
	if s == "" {
		return p
	}
	return s
}

// sanitize strips ANSI and Cc controls: folder/branch names are untrusted and
// would otherwise inject escape sequences into the terminal.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.ReplaceAllString(s, ""))
}

func rightAlign(s string, cols int) string {
	return strings.Repeat(" ", max(0, cols-displayWidth(s))) + s
}

func level(pct float64) string {
	// Same thresholds for all three meters, so the colour means one thing:
	// how close to the wall.
	switch {
	case pct >= 80:
		return red
	case pct >= 50:
		return yellow
	}
	return green
}

// meter draws a half-cell bar after rich/progress_bar.py. Display width is
// always width.
func meter(pct float64, width int) string {
	pct = max(0, min(100, pct))
	col := level(pct)
	// Floor, not round: a bar that reads full at 96% would lie in the
	// direction that hurts.
	halves := int(float64(width*2) * pct / 100)
	full, hasHalf := halves/2, halves%2
	var b strings.Builder
	b.WriteString(col + strings.Repeat(bar, full))
	if hasHalf == 1 {
		b.WriteString(half)
	}
	rem := width - full - hasHalf
	b.WriteString(track)
	if hasHalf == 0 && full > 0 && rem > 0 {
		// Flush transition: a tip reads cleaner than a blunt edge.
		b.WriteString(tip)
		rem--
	}
	b.WriteString(strings.Repeat(bar, rem) + reset)
	return b.String()
}

// fmtReset turns a Unix epoch into at most 5 chars of countdown; "" when
// stale or absent.
func fmtReset(resetsAt any, now float64) string {
	var at float64
	switch v := resetsAt.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return ""
		}
		at = f
	case string:
		// A numeric string counts too, surrounding space included.
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return ""
		}
		at = f
	default:
		return ""
	}
	// inf and NaN never count down.
	if math.IsNaN(at) || math.IsInf(at, 0) {
		return ""
	}
	// Cap before the conversion: float->int past the type's range is
	// implementation-defined, and that far out reads the same as "never".
	// int64 rather than int, so the cap holds on a 32-bit build too.
	delta := int64(max(-1, min(at-now, 1e15)))
	if delta <= 0 {
		return ""
	}
	if delta < 60 {
		return "<1m"
	}
	minutes, hours := delta/60%60, delta/3600
	if hours < 1 {
		return fmt.Sprintf("%dm", minutes)
	}
	if hours < 24 {
		return fmt.Sprintf("%dh%02d", hours, minutes)
	}
	return fmt.Sprintf("%dd %dh", hours/24, hours%24)
}

func meterRow(timer, label string, pct float64) string {
	// Clamp before int(): converting a float beyond int64 is undefined, and
	// the number stays at most 4 digits.
	p := int(max(-999, min(9999, pct)))
	col := level(float64(max(0, min(100, p))))
	return fmt.Sprintf("%s%5s%s  %s%4s%s  %s  %s%3d%%%s",
		frame, timer, reset, frame, label, reset, meter(pct, barWidth), col, p, reset)
}
