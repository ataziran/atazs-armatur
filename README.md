# atazs-armatur

A minimal status line for [Claude Code](https://code.claude.com): context usage, plus the 5-hour
and weekly usage limits with reset countdowns. One static binary, no runtime, no dependencies.

The usage limits run in fixed windows and nothing counts them down while you work, so you learn
that one has closed when an answer stops coming, in the middle of something. The context window
fills just as quietly until the session has to compact. Three meters turn both into something you
can see coming.

[![CI](https://github.com/ataziran/atazs-armatur/actions/workflows/ci.yml/badge.svg)](https://github.com/ataziran/atazs-armatur/actions/workflows/ci.yml) ![Go 1.27+](https://img.shields.io/badge/Go-1.27%2B-00ADD8?logo=go&logoColor=white) ![License: GPL-3.0](https://img.shields.io/badge/license-GPL--3.0-blue)

![Status line: context 24%, 5-hour limit 91% resetting in 40m, weekly limit 15% resetting in 4d 9h](docs/statusline.svg)

## What you see

| Row    | Measures                                        | From                             |
| ------ | ----------------------------------------------- | -------------------------------- |
| `ctx`  | Context window used by the current conversation | `context_window.used_percentage` |
| `ses`  | 5-hour usage limit, with time until reset       | `rate_limits.five_hour`          |
| `week` | 7-day usage limit, with time until reset        | `rate_limits.seven_day`          |

- **Colour** means one thing on every row: how close to the wall. Green below 50%, yellow from
  50%, red from 80%.
- **Bars** are 16 cells wide with half-cell resolution. Bars and percentages round down, so 99.6%
  shows as 99% and a bar is only full at 100%: it never looks fuller than the truth.
- **Countdowns** read `<1m`, `45m`, `2h41` or `3d 4h`, and go blank once the reset time has passed.
- **Rows** for a limit Claude Code does not report are hidden rather than shown as 0%.

## Why another one

The status lines for Claude Code I looked at are Node packages, wired into `settings.json` as
something like `npx -y <package>@latest`. That pays for a runtime start on every repaint, and
`@latest` runs whatever was published since you last looked. This one is a single static binary
that Claude Code runs directly: what you installed is what runs, and it never goes looking for a
newer self.

Measured on one machine (WSL2 on linux/amd64, payload read from a file, 200 runs each, 50 for
Node):

| | per invocation |
| --- | --- |
| atazs-armatur, complete run | 2.0 ms |
| `/bin/true`, process creation alone | 0.5 ms |
| `node -e ''`, an empty runtime before any status line code | 18 ms |

A quarter of this tool's cost is the fork the operating system charges for running any binary at
all. The absolute numbers differ per machine; the gap does not, and `npx` adds a registry lookup
on top of the Node figure.

The trade is configurability. Those projects let you compose your own line from segments; this one
has a fixed layout, no config file and no options. If you want to arrange your own status line,
they are the better tool.

## Install

1. Download the binary for your platform and `SHA256SUMS` from the
   [latest release](https://github.com/ataziran/atazs-armatur/releases/latest), then check the
   download. Asset names are under [Compatibility](#compatibility); `linux-amd64` shown:

   ```bash
   curl -fLO https://github.com/ataziran/atazs-armatur/releases/latest/download/atazs-armatur-linux-amd64
   curl -fLO https://github.com/ataziran/atazs-armatur/releases/latest/download/SHA256SUMS
   sha256sum -c --ignore-missing SHA256SUMS   # macOS: shasum -a 256 -c --ignore-missing SHA256SUMS
   ```

   A good download prints `atazs-armatur-linux-amd64: OK`. On Windows without Git Bash, compare
   `Get-FileHash .\atazs-armatur-windows-amd64.exe` with the line in `SHA256SUMS`. Every release
   also carries a build provenance attestation; see [Releases](#releases).

2. Put it somewhere stable. On macOS and Linux:

   ```bash
   mkdir -p ~/.local/bin
   mv atazs-armatur-<os>-<arch> ~/.local/bin/atazs-armatur
   chmod +x ~/.local/bin/atazs-armatur
   ```

   A browser download on macOS carries the quarantine flag and Gatekeeper may refuse to run it;
   `curl` downloads do not, and `xattr -d com.apple.quarantine ~/.local/bin/atazs-armatur` clears it.

   On Windows, save it as `C:\Users\<you>\.local\bin\atazs-armatur.exe`.

   `atazs-armatur --version` prints the release and the commit it was built from.

3. Add a `statusLine` entry inside the top-level object of `~/.claude/settings.json`. On macOS and
   Linux:

   ```json
   "statusLine": {
     "type": "command",
     "command": "~/.local/bin/atazs-armatur",
     "padding": 0,
     "refreshInterval": 60
   }
   ```

   On Windows, point `command` at the `.exe` and write the path with forward slashes:

   ```json
   "statusLine": {
     "type": "command",
     "command": "C:/Users/<you>/.local/bin/atazs-armatur.exe",
     "padding": 0,
     "refreshInterval": 60
   }
   ```

   Claude Code runs the command through Git Bash when it is installed, otherwise PowerShell. Git
   Bash treats unquoted backslashes as escapes, so `C:\Users\...` fails without a visible error.
   `~` also works and expands to your Windows home directory
   ([docs](https://code.claude.com/docs/en/statusline#windows-configuration)).

   `padding` adds spacing on top of Claude Code's own; keep it at `0`, the lines are already sized
   to the terminal. `refreshInterval` re-runs the command every 60 seconds, so the countdowns keep
   moving while the session is idle.

## How it works

Claude Code runs the command on every status line update and passes session JSON on stdin.
atazs-armatur prints one to three lines on stdout and exits. Run by hand it answers `--version`
and `--help`, and with a terminal on stdin rather than a pipe it prints the usage instead of
waiting for input that is not coming.

- **Input.** It reads `context_window`, `rate_limits.five_hour`, `rate_limits.seven_day` and the
  working directory (`workspace.current_dir`, then `cwd`, then its own). `ctx` falls back to
  `total_input_tokens / context_window_size` when `used_percentage` is null, which Claude Code
  allows early in a session.
- **Nothing else.** No network, no API calls, no config file, nothing written to disk, no extra
  tokens and no subprocess. The branch comes from reading `.git/HEAD`, walking up from the working
  directory and resolving the `gitdir:` file a linked worktree leaves there. Outside a repository
  there is no branch; on a detached HEAD the short object id takes its place, marked `@` so it
  cannot be read as a branch of that name.
- **Width.** `COLUMNS` when it holds a positive number, which Claude Code sets before running the
  command ([docs](https://code.claude.com/docs/en/statusline)). Otherwise it asks the terminal, in
  this order: an ioctl on the standard descriptors, on Linux the first ancestor process that still
  holds the pty, then `/dev/tty`; Windows queries the console. The Unix opens are non-blocking, so
  a pipe with no writer cannot hang it. If nothing answers, 80. Every source is capped, so a stray
  `COLUMNS=999999` cannot pad the line into the megabytes.
- **The ancestor walk.** It reads `/proc/<pid>/stat` for the parent id, opens `/proc/<pid>/fd/2`,
  `1` and `0` read-only and non-blocking, asks each for the terminal size with `TIOCGWINSZ` and
  closes it again, at most six levels up. Under Claude Code all three standard descriptors are
  pipes and `/dev/tty` answers `ENXIO`, so the walk is the only source that finds the real width;
  without it the line falls back to 80 columns.
- **Margin.** Four columns stay free on the right, because Claude Code truncates every status line
  a few columns short of the edge and replaces the tail with `…`. Below about 40 columns the meter
  rows no longer fit.
- **Layout.** Line one has folder › branch on the left and the `ctx` meter flush right. The meter
  is the content and is never truncated: when space runs out the branch is shortened, then
  dropped, then the folder name is shortened. Display width follows Unicode East Asian Width, so
  CJK and single-codepoint emoji folder names line up.
- **Colour.** `NO_COLOR` set to any non-empty value drops the escape sequences
  ([no-color.org](https://no-color.org)); the bars still show the fill level.
- **Sanitization.** Folder and branch names are untrusted: CSI and OSC escape sequences and every
  C0 and C1 control character are removed before display. A directory named `$'\e[2J'` shows up as
  text rather than clearing your screen.
- **Bad input.** A field that is missing falls back; a field of the wrong shape
  (`"used_percentage": "50"`) leaves the line empty. Out-of-range numbers are clamped before
  display, a stale or non-finite reset time shows no countdown, and nothing ever prints a stack
  trace.

## Compatibility

| OS      | amd64                             | arm64                             |
| ------- | --------------------------------- | --------------------------------- |
| Linux   | `atazs-armatur-linux-amd64`       | `atazs-armatur-linux-arm64`       |
| macOS   | `atazs-armatur-darwin-amd64`      | `atazs-armatur-darwin-arm64`      |
| Windows | `atazs-armatur-windows-amd64.exe` | `atazs-armatur-windows-arm64.exe` |

- **No Nerd Font.** The bars use three characters from the Box Drawing block only: `━` U+2501,
  `╸` U+2578 and `╺` U+257A, not Block Elements (Consolas lacks the eighth blocks, and conhost
  would show tofu). Windows Terminal draws that block itself, and Consolas, the conhost default on
  Windows 10, carries all of it. The branch marker is a plain `›`.
- **Narrow box drawing.** Box-drawing characters are East Asian Ambiguous and are counted as one
  column, which is how Windows Terminal, iTerm2 and Linux terminals draw them.
- **256 colours.** Meters and frame use 256-colour SGR; folder and branch use your theme's cyan
  and magenta. Never the dim attribute: conhost, winpty and some tmux setups drop it, and where dim
  is dropped the grey frame would render as bright as the content.

## Troubleshooting

**The `ses` or `week` row is missing.** Claude Code sends `rate_limits` only to claude.ai Pro and
Max subscribers or behind a Claude apps gateway with a spend limit, only after the first API
response in a session, and it drops a window once its reset time has passed. With an API key you
see `ctx` alone.

**The line wraps or is cut off.** Width comes from `COLUMNS`, which Claude Code sets; when running
the binary another way, export `COLUMNS`. Four columns are already kept free, but notifications and
the verbose-mode token counter share the row and can still cut into it on narrow terminals.

**Colours look flat or the bar track is invisible.** The terminal, and tmux or screen if you use
them, needs 256-colour support. The empty part of each bar is a dark grey from that palette.

**The countdown does not move.** Claude Code re-runs the command on events such as a new message,
and an idle session has none. Set `"refreshInterval": 60`.

**Nothing shows at all.** Claude Code leaves the status line blank until you accept the workspace
trust dialog for the folder.

**Nothing shows on Windows.** Git Bash eats the backslashes in a `C:\...` path and the command
fails silently. Write it with forward slashes: `C:/Users/<you>/.local/bin/atazs-armatur.exe`.

## Build from source

Go 1.27 (the `go` line in `go.mod`), no dependencies.

```bash
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' .
go test ./...
```

`width_table.go` is generated from `EastAsianWidth.txt` for the standard library's Unicode version,
and a test fails when the two disagree. A toolchain with a newer Unicode needs `go generate`, which
downloads the table from unicode.org and rewrites the file.

The rendering tests are golden fixtures: each `testdata/golden/<case>.json` pins stdin, terminal
width, clock, git branch and working directory, and `<case>.out` holds the expected output byte for
byte. The cases cover thresholds, half cells, truncation, wide characters, escapes in names and bad
input.

## Releases

Pushing a `v*` tag runs [`release.yml`](.github/workflows/release.yml). It vets and tests, builds
all six targets from the tagged commit with `CGO_ENABLED=0 go build -trimpath -ldflags='-s -w'`,
checks that every binary is stamped with the tag and a clean tree, writes `SHA256SUMS`, attests
build provenance and creates the release.

The builds are reproducible: a clean clone checked out at the tag, built with the command above,
the same Go release (`go version -m <binary>` prints it) and the same `GOOS`/`GOARCH`, gives the
same bytes. To verify a download:

```bash
sha256sum -c --ignore-missing SHA256SUMS
gh attestation verify <file> --repo ataziran/atazs-armatur
```

[`ci.yml`](.github/workflows/ci.yml) runs `gofmt`, `go vet` and the tests on Linux, macOS and
Windows for every push to `main` and every pull request, and vets every release target.

## Contributing

- Run `gofmt -l .`, `go vet ./...` and `go test ./...`; CI runs the same on all three OSes.
- Goldens are byte-for-byte fixtures: new behaviour gets a new `.json` and `.out` pair.
- `.gitattributes` forces LF everywhere and never converts the `.out` files.
- The binary stays dependency-free: standard library only.

## License

GPL-3.0, see [LICENSE](LICENSE). Use it, change it and share it freely, at home, at work and
alongside commercial products. Whoever distributes it passes on the source and this license.
