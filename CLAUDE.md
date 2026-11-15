# Mordye

Mordye completes a sparse theme palette. It returns 15 Material interface roles
and 22 terminal fields, and keeps every hue where the theme put it. It ships as
one static Go binary. Its contract is the CLI plus the JSON it writes. The root
package is importable, but it is not a promised public API.

Write the name as Mordye in prose: docs, comments, and PR or issue text. Keep
`mordye` lowercase wherever someone types it: commands, the module and package,
the binary, file paths, and CLI output.

The AI reviewers (CodeRabbit and PR-Agent) read this file as their review
rules. Keep the project's rules here, in one place, rather than in their
configs.

## Stack

Go 1.27+, standard library only. A new module dependency is a deliberate
decision; raise it before adding one.

## Architecture

- **Generator** (root package): `Complete(Input) (Output, error)`. It knows nothing about any source.
- **Adapters** (`adapter/<source>/`): one package per source. `Parse` takes file contents, never paths, and returns `mordye.Input`. `adapter/dux` is the pattern for new adapters.
- **Terminal port adapters** (`adapter/kitty`) supplement a theme adapter: `Parse` returns `mordye.TerminalInput`, and the CLI sets it on the theme's input. Why kitty: `docs/decisions/0001-kitty-ports-for-terminal-colors.md`.
- **CLI** (`cmd/mordye`): the only code that touches the filesystem. It reads the files its flags name and runs one subcommand per source. The palette goes to stdout and diagnostics to stderr. An error exits non-zero, and a zero exit never comes with a partial palette. Flags and subcommands are a public interface; renaming one breaks callers.
- Mordye never discovers files. Finding a source's settings is the caller's job; keep it that way. Adapters don't read environment variables or assume a home directory either, and source-specific knowledge stays in its adapter.
- A malformed or unexpected source file fails with a clear error, never a silently partial palette.
- `color.go` holds the OKLab/LCh math. The tuning constants in `palette.go` were set against seven themes. A changed constant shifts every generated color, so check every fixture.

## Invariants

The README promises these, and tests encode them:

- A color the theme defines passes through unchanged.
- The accent moves only `primary` and `secondary`.
- Swatches never become interface roles.
- The output always has all 37 fields, plus `sources`.
- Every output field records its origin in `sources`; a missing or wrong entry is a bug.
- A generated color keeps its slot's usual hue, at the palette's own chroma and lightness. Nothing pulls it toward the accent.
- The output is deterministic: the same input gives byte-identical output, with no map-order or float-formatting drift.
- The README's field counts, adapter table, and command examples match what the code does.

Callers template against the output shape. A change to it, or to `Version`, is
breaking; flag it and apply the `👀 Attention` label.

## Tests

- Test names state the behavior as a sentence, such as `TestAccentDoesNotMoveTerminalColors`.
- Files in `adapter/*/testdata/` are unmodified upstream themes, credited in the README. Never edit them. A new fixture is copied verbatim and credited.
- New adapter behavior is tested against those real theme files, not hand-built approximations.

## Commands

- `just bootstrap`: turn on the git hooks in `.githooks/` and check the tools a fresh clone needs (idempotent).
- `just check`: formatting, vet, and tests. Run it before calling work done; CI runs the same checks on every push and PR.
- `just build`, `just test`, `just fmt`.
- `just release major|minor|patch|X.Y.Z`: cut a release from `main`. Long recipe logic lives in `bin/justfile/<recipe>.sh`.

## Changelog

A change someone using Mordye would notice gets a plain-language entry under
`## Unreleased` in `CHANGELOG.md`, in the same PR. Write it for humans: what
changed for them, not which commits did it. Refactors, tests, and tooling don't
get entries.

## Issues

GitHub Issues on `robwilkerson/mordye`. Each issue gets one type label
(`🏷️  FEATURE`, `🏷️  BUG`, `🏷️  CHORE`) and one priority label (`🎯 Low`, `🎯 Medium`,
`🎯 High`). The type labels have two spaces after the emoji; copy names from
`gh label list`.
