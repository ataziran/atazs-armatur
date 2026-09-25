# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- The session's peer name (the name other sessions use with `SendMessage`, e.g.
  `atazs-armatur-06`) under the folder on line 2, read from Claude Code's session registry in
  `~/.claude/sessions/` (or `$CLAUDE_CONFIG_DIR`). Without a registry entry the spot stays
  empty. It gives way before the `ses` meter on a narrow terminal. Thanks @ioxoi.

### Fixed

- A working directory reached through a symlink shows the branch of the repository it points
  into, not of the repository around the link. Thanks @ioxoi.
- A JSON object followed by a stray `}` or `]` shows waiting rows like any other trailing data,
  instead of being read as a valid payload. Thanks @ioxoi.

## [0.2.0] - 2026-09-24

Upgrading from 0.1.0: the entries marked **Behaviour change** change what the line shows.

### Added

- **Behaviour change:** The idle mark `◷` before the labels when the session transcript has not
  changed for more than 5 minutes, meaning no new reading; `○` on Windows outside Windows
  Terminal. `ses` and `week` then draw a thin track behind the unchanged fill, since other
  sessions share those limits. It needs `refreshInterval` to appear while the session is idle.
- A limit with a future reset time but no value keeps its countdown while the number shows `...`.

### Changed

- **Behaviour change:** Every meter row is always drawn. A meter without a value keeps its row
  and shows `...` on a thin track, instead of a false `0%` for `ctx` or no row at all for `ses`
  and `week`. With an API key, `ses` and `week` now show `...`. Empty stdin and input that is not
  a JSON object show three waiting rows. A JSON object with a field of the wrong type still gives
  one empty line.
- **Behaviour change:** A limit whose reset time has passed shows `...` until Claude Code sends
  the next value, instead of the old percentage without a countdown.
- **Behaviour change:** A `transcript_path` that is not a string blanks the line. The binary now
  calls `stat` on it once per run, for the idle mark, and never opens the file.

### Removed

- **Behaviour change:** The `ctx` fallback to `total_input_tokens / context_window_size`. It only
  applied before the first response and then showed a false `0%`; `ctx` now shows `...` until
  then. The token fields are no longer read, so a wrong type in either no longer blanks the line.

## [0.1.0] - 2026-09-23

First release.

### Added

- Rows `ctx` (context window), `ses` (5-hour limit) and `week` (7-day limit), the two limits with
  a countdown to the next reset. A limit Claude Code does not report is hidden, for example with
  an API key.
- Bars and percentages round down; green below 50%, yellow from 50%, red from 80%. The bars use
  only box-drawing characters found in Consolas, so no Nerd Font is needed. `NO_COLOR` drops the
  escape sequences.
- Folder name and git branch on the first line, read from `.git/HEAD` without running `git`,
  linked worktrees included. A detached HEAD shows `@<short id>`. On a narrow terminal the branch
  is shortened first, then the folder, so the meter stays whole.
- Width from `COLUMNS`, otherwise from a non-blocking probe of the terminal. Every source is
  capped, so `COLUMNS=999999` does not pad the line.
- No network, no API calls, no extra tokens: all values come from the session data Claude Code
  passes on stdin.
- Hostile or malformed input is handled: escape sequences and control characters are stripped
  from folder and branch names, a field of the wrong type blanks the line instead of showing a
  wrong value, a `resets_at` sent as a numeric string still counts, and out-of-range or
  non-finite numbers are clamped or ignored.
- `--version`/`-v` prints the release and commit, `-h`/`--help` or a terminal on stdin prints the
  usage, and an unknown option exits 2.
- Release binaries for six targets with `SHA256SUMS` and a build provenance attestation.

[Unreleased]: https://github.com/ataziran/atazs-armatur/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/ataziran/atazs-armatur/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/ataziran/atazs-armatur/releases/tag/v0.1.0
