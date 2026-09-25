# How atazs-armatur works

Reference for the input it reads, the output it draws, and how it finds the terminal width. For
installation and configuration, see the [README](../README.md). For building, testing and
releases, see [CONTRIBUTING.md](../CONTRIBUTING.md).

- [Invocation](#invocation)
- [Input](#input)
- [What it reads besides the payload](#what-it-reads-besides-the-payload)
- [What it does not do](#what-it-does-not-do)
- [Output](#output)
- [Meter rows](#meter-rows)
- [Waiting and idle](#waiting-and-idle)
- [Terminal width](#terminal-width)
- [Line one and truncation](#line-one-and-truncation)
- [Display width](#display-width)
- [Sanitizing names](#sanitizing-names)
- [Characters and colours](#characters-and-colours)
- [Malformed input](#malformed-input)
- [Startup time](#startup-time)
- [Source files](#source-files)

## Invocation

Claude Code runs the configured command on every status line update, passes session JSON on
stdin, and shows what the command prints on stdout.

| Invocation                        | Behaviour                                           | Exit |
| --------------------------------- | --------------------------------------------------- | ---: |
| payload on stdin                  | prints the status line on stdout                    |    0 |
| `-v`, `--version`                 | prints `atazs-armatur <version> (<7-digit commit>)` |    0 |
| `-h`, `--help`                    | prints the usage on stdout                          |    0 |
| any other argument                | `unknown option: <arg>` and the usage on stderr     |    2 |
| no arguments, stdin is a terminal | usage on stderr instead of waiting for input        |    2 |

The version line comes from the build info the Go toolchain embeds (module version and
`vcs.revision`); nothing is stamped in with `-ldflags`. Without a revision the commit is left
out, and without build info the line reads `atazs-armatur (unknown)`.

## Input

Fields read from the payload:

| Field                                   | Used for                                        |
| --------------------------------------- | ----------------------------------------------- |
| `context_window.used_percentage`        | the `ctx` row                                   |
| `rate_limits.five_hour.used_percentage` | the `ses` row                                   |
| `rate_limits.five_hour.resets_at`       | the `ses` countdown, Unix seconds               |
| `rate_limits.seven_day.used_percentage` | the `week` row                                  |
| `rate_limits.seven_day.resets_at`       | the `week` countdown, Unix seconds              |
| `transcript_path`                       | the idle mark, via the file's modification time |
| `workspace.current_dir`, then `cwd`     | folder name and branch lookup directory         |

`used_percentage` is calculated by Claude Code. The program does not compute context use from
token counts.

The working directory is the first non-empty string among `workspace.current_dir` and `cwd`. A
value of another type counts as absent. If neither names a directory, the process's own working
directory is used.

`resets_at` is accepted as a JSON number or as a numeric string, surrounding whitespace included.
NaN and infinity count as no reset time. A value that overflows float64, such as `1e999`, does
not fail the decode (numbers are kept as `json.Number`) and shows no countdown. The time left is
capped at 10^15 seconds before the conversion to an integer, so a far-off finite value such as
`1e300` still counts down (`11574074074d 1h`).

All other fields are ignored.

## What it reads besides the payload

- **Transcript.** One `stat` on `transcript_path` for its modification time. The content is
  never opened. If the path is empty or the `stat` fails, there is no idle mark.
- **Git branch.** From files only:
  1. Resolve symlinks in the working directory, then walk up to the first `.git`.
  2. If `.git` is a directory, read `.git/HEAD`.
  3. If `.git` is a file (a linked worktree), read its `gitdir: <path>` line and read `HEAD` in
     that directory. A relative path is resolved against the worktree.
  4. `ref: refs/heads/<name>` shows `<name>`. A bare object id (detached HEAD) shows `@` and its
     first seven hex digits, so it cannot be mistaken for a branch with that name. Anything else,
     or no repository, shows no branch.

  `git` is not executed. A subprocess would cost a fork per repaint, and a `git` that hangs on a
  slow network mount or a stuck lock would hang the status line with it.
- **Environment.** `COLUMNS`, `NO_COLOR`, and on Windows `WT_SESSION`.
- **Terminal size.** See [Terminal width](#terminal-width).

## What it does not do

It does not use the network, call an API, read a config file, write to disk, spend tokens, or
start a subprocess. It writes only to stdout, and to stderr for usage errors.

## Output

Three lines, each starting and ending with an SGR reset (`ESC[0m`). The leading reset clears
stale attributes and keeps the renderer from trimming the leading spaces.

```text
dir › branch           ctx  ━━━━━━╺━━━━━━━━━   38%
                2h41   ses  ━━━━━━━━━━━━╺━━━   75%
               3d 4h  week  ━━━━━━━━━━━━━━━╺   96%
```

- Line one: folder name and branch on the left, the `ctx` row flush right.
- Lines two and three: the `ses` and `week` rows, right-aligned.

A malformed payload (see [Malformed input](#malformed-input)) prints a single line holding only a
reset. An unexpected runtime failure is recovered and prints the same; no stack trace reaches
the status line.

## Meter rows

Each row is 35 columns wide for percentages from −99 to 999 and countdowns of up to 5 characters:

| Part      | Width | Content                                                      |
| --------- | ----: | ------------------------------------------------------------ |
| countdown |    5+ | right-aligned; empty on `ctx` and when there is no countdown |
| separator |     2 | spaces                                                       |
| label     |     4 | `ctx`, `ses` or `week`, right-aligned                        |
| separator |     2 | spaces                                                       |
| bar       |    16 | see below                                                    |
| separator |     2 | spaces                                                       |
| number    |     4 | percentage, three digits right-aligned plus `%`              |

**Bar.** 16 cells with half-cell resolution, so 32 steps. The number of half cells is
`floor(32 × pct / 100)`, with `pct` clamped to 0–100 first. Rounding down means a bar is full only
at 100% and never looks fuller than the value. When the fill ends on a cell boundary and track
remains, the first track cell is a tip (`╺`, or `╶` on a thin track) instead of a blunt edge.

**Number.** The integer part of the value, clamped to −999…9999, so the number keeps at most four
digits (five columns with `%`). 99.6% shows as 99%.

**Colour.** The same thresholds on every row: green below 50%, yellow from 50%, red from 80%.
The colour is chosen from the value clamped to 0–100.

**Countdown.** Whole seconds until `resets_at`:

| Time left        | Shown   |
| ---------------- | ------- |
| under 1 minute   | `<1m`   |
| under 1 hour     | `45m`   |
| under 24 hours   | `2h41`  |
| 24 hours or more | `3d 4h` |

## Waiting and idle

A row **waits** when its value is not known: the percentage is missing or `null`, the window is
missing, or `resets_at` is less than one whole second away or in the past. A waiting row draws a
thin track (`─` across all 16 cells) and `...` in place of the number. It never shows a 0% that
was not reported. A window with a future `resets_at` but no percentage still shows its countdown.

`rate_limits` is absent with an API key, and either window can be missing on its own; each row
waits independently.

The **idle mark** appears when the transcript's modification time is more than 300 seconds before
the current time. A modification time in the future is not idle. When idle:

- `◷` (U+25F7) is placed directly before each of the three labels. On ` ctx` and ` ses` it
  replaces the label's leading space; on `week` it takes one of the two spaces before the label.
  The row width does not change.
- `ses` and `week` draw a thin track behind their unchanged fill, because other sessions and
  claude.ai draw on the same limits and may have used more since the last reading.
- `ctx` keeps its full track. The context window belongs to this session alone.

On Windows without `WT_SESSION` (that is, outside Windows Terminal), the mark is `○` (U+25CB),
because Consolas and Lucida Console, the conhost fonts, lack U+25F7.

A long tool call or a subagent does not write to the session transcript, so the mark can appear
while Claude is working.

## Terminal width

Sources, in order; the first that answers wins:

1. `COLUMNS`, if it parses as a positive integer. Claude Code sets it before running the command
   ([docs](https://code.claude.com/docs/en/statusline)). Zero, negative or non-numeric values are
   skipped rather than taken as a width, since some shells and CI runners export `COLUMNS=0`.
2. A platform query:

   | Platform | Queries, in order                                                                  |
   | -------- | ---------------------------------------------------------------------------------- |
   | Linux    | `TIOCGWINSZ` on fds 2, 1, 0; the ancestor walk; `/dev/tty`                         |
   | macOS    | `TIOCGWINSZ` on fds 2, 1, 0; `/dev/tty`                                            |
   | Windows  | `GetConsoleScreenBufferInfo` on stderr, stdout; then `CONOUT$`                     |
   | other    | none                                                                               |

3. A fixed width of 80 columns.

Every source is capped at 1000 columns, so `COLUMNS=999999` cannot pad the output to a megabyte.
All descriptor opens use `O_RDONLY|O_NOCTTY|O_NONBLOCK`, so a FIFO with no writer cannot block.

### The ancestor walk (Linux)

Under Claude Code all three standard descriptors are pipes, and when the process has no
controlling terminal (under `setsid`, or as a grandchild of the shell that owns the pty) opening
`/dev/tty` fails with `ENXIO`. Without `COLUMNS` the width would then fall back to 80. The walk
finds the real width:

1. Read `/proc/<pid>/stat`, starting with `self`, and take the parent id from the fields after
   the parenthesized command name.
2. Stop if the parent id is 0 or 1.
3. Open `/proc/<ppid>/fd/2`, `1` and `0` read-only and non-blocking, ask each for its size with
   `TIOCGWINSZ`, and close it. The first positive answer is the width.
4. Otherwise continue from the parent, at most six levels up.

### Margin

The output is laid out for the width minus 4 columns, with a floor of 20. Claude Code truncates
every status line a few columns short of the terminal edge and replaces the tail with `…`;
notifications and the verbose-mode token counter share the same row. The four columns were
measured in a live session.

The meter rows are 35 columns wide, so below about 40 terminal columns they no longer fit.

## Line one and truncation

Line one holds `folder › branch`, at least one space, and the `ctx` row. The `ctx` row is never
truncated. The space for the text is the layout width minus the row width minus 2. While the text
does not fit:

1. The branch loses its last two characters and gets `…`, repeatedly. At two characters or fewer
   it is dropped.
2. Then the folder name is shortened the same way, down to a single character (`…`).

The folder name is the last path component of the working directory. Both `/` and `\` count as
separators, and trailing separators are ignored, so native Windows paths work. The root `/` shows
as `/`.

## Display width

Widths are measured with escape sequences stripped:

- Code points in `unicode.Mn`, `unicode.Me` and `unicode.Cf` (combining marks, format characters)
  count 0.
- East Asian Wide (W) and Fullwidth (F) code points count 2. The ranges are in `width_table.go`,
  generated from `EastAsianWidth.txt` for the Unicode version of the Go standard library, so the
  zero-width and wide data agree.
- Everything else counts 1, including East Asian Ambiguous. All box-drawing characters are
  Ambiguous, and Windows Terminal, iTerm2 and Linux terminals draw them narrow.

CJK and single-code-point emoji folder names line up. Multi-code-point emoji sequences are not
handled as a unit.

## Sanitizing names

Folder and branch names are untrusted. Before display:

1. CSI sequences (`ESC [ … final byte`) with parameter bytes from `0-9;:?`, and OSC sequences
   (`ESC ] …` terminated by BEL or `ESC \`), are removed. A CSI with `<`, `=` or `>` loses only
   its `ESC` in step 2, and the rest shows as text.
2. Every remaining C0 and C1 control character is removed.

A directory named `$'\e[2J'` shows as text instead of clearing the screen.

## Characters and colours

No Nerd Font is needed. The bars use five characters from the Box Drawing block (U+2500–U+257F):

| Character | Code point | Use                          |
| --------- | ---------- | ---------------------------- |
| `━`       | U+2501     | filled cell, full track cell |
| `╸`       | U+2578     | half-filled cell             |
| `╺`       | U+257A     | tip after a flush fill       |
| `─`       | U+2500     | thin track cell              |
| `╶`       | U+2576     | thin tip                     |

Windows Terminal draws this block itself, and Consolas, the conhost default on Windows 10,
carries all of it. Block Elements (U+2580–U+259F) are avoided: Consolas lacks the eighth blocks
and conhost has no fallback, so they would show as boxes. The branch separator is a plain `›`.
The idle mark and its fallback are described under [Waiting and idle](#waiting-and-idle).

Colours:

| Element                                | SGR                       |
| -------------------------------------- | ------------------------- |
| folder name                            | bold, cyan (`1`, `36`)    |
| branch                                 | magenta (`35`)            |
| frame (labels, countdowns, `›`, `...`) | 256-colour 240            |
| bar track                              | 256-colour 236            |
| green / yellow / red                   | 256-colour 71 / 179 / 167 |

Folder and branch follow the terminal theme; meters and frame need 256-colour support in the
terminal and in tmux or screen. The dim attribute (`ESC[2m`) is never used: conhost, winpty and
some tmux setups drop it, and the grey frame would then render as bright as the content.

`NO_COLOR` set to any non-empty value removes all escape sequences from the output
([no-color.org](https://no-color.org)). The bars still show the level.

## Malformed input

The payload is decoded into typed fields, and the field types define what counts as malformed.

| Input                                               | Result                               |
| --------------------------------------------------- | ------------------------------------ |
| `{` followed by invalid or truncated JSON           | a single empty line                  |
| field of the wrong type (`"used_percentage": "50"`) | a single empty line                  |
| empty stdin, not starting with `{`, trailing data   | all rows waiting                     |
| missing or `null` field                             | that row waits                       |
| non-string `workspace.current_dir` or `cwd`         | falls through to the next source     |
| `resets_at` as a numeric string                     | accepted                             |
| `resets_at` not a number, NaN, infinite, `1e999`    | no countdown                         |
| `resets_at` in the past                             | the row waits                        |
| percentage out of range                             | clamped: bar 0–100, number −999…9999 |
| no directory in payload or process                  | empty folder name                    |
| transcript missing or unreadable                    | no idle mark                         |
| unreadable `.git/HEAD` or `gitdir:` file            | no branch                            |

## Startup time

The command runs on every update in every open session. Median per call, 21 rounds of 500
sequential calls including the shell's fork and exec, on Linux 6.6 under WSL2, Ryzen 9 7950X3D,
Go 1.27.1:

| Command                 | Median per call | Range across rounds |
| ----------------------- | --------------: | ------------------: |
| `atazs-armatur`         |          2.2 ms |          2.1–2.3 ms |
| `/bin/true`             |          0.6 ms |          0.6–0.6 ms |
| `node -e ''` (v24.18.0) |         20.2 ms |        20.0–20.5 ms |

`/bin/true` is the cost of starting any process; the remaining 1.6 ms are the Go runtime and the
binary's own work. `node -e ''` only starts the runtime, before any status line code loads.
Reading a git branch did not change the median. Process start costs differ on native Linux and
macOS. To repeat it (GNU `date`):

```bash
printf '%s' '{"context_window":{"used_percentage":38},"rate_limits":{"five_hour":{"used_percentage":75,"resets_at":1800009690},"seven_day":{"used_percentage":96,"resets_at":1800273605}}}' > payload.json
export COLUMNS=80
for r in $(seq 21); do
  s=$(date +%s%N)
  for i in $(seq 500); do ./atazs-armatur <payload.json >/dev/null; done
  echo $(( ($(date +%s%N) - s) / 500000 ))   # microseconds per call
done | sort -n | sed -n 11p                  # median of 21 rounds
```

## Source files

| File                               | Contents                                                  |
| ---------------------------------- | --------------------------------------------------------- |
| `main.go`                          | options, gathering the environment, transcript `stat`     |
| `session.go`                       | payload struct and decoding                               |
| `render.go`                        | layout, meters, countdowns, sanitizing, colours           |
| `git.go`                           | branch from `.git/HEAD` and worktree `gitdir:` files      |
| `term.go`                          | `COLUMNS` and the width cap                               |
| `term_unix.go`, `term_linux.go`    | `TIOCGWINSZ`, the ancestor walk, `/dev/tty`               |
| `term_darwin.go`, `term_other.go`  | stubs for platforms without the walk or any query         |
| `term_windows.go`                  | console queries                                           |
| `width.go`, `width_table.go`       | display width and the generated wide ranges               |
| `version.go`                       | the `--version` line                                      |
