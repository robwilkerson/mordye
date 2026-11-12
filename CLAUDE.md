# Mordye

Mordye completes a sparse theme palette. It returns 15 Material interface roles
and 22 terminal fields, and keeps every hue where the theme put it. It ships as
one static Go binary. Its contract is the CLI plus the JSON it writes. The root
package is importable, but it is not a promised public API.

Write the name as Mordye in prose: docs, comments, and PR or issue text. Keep
`mordye` lowercase wherever someone types it: commands, the module and package,
the binary, file paths, and CLI output.

## Stack

Go 1.27+, standard library only. A new module dependency is a deliberate
decision; raise it before adding one.

## Architecture

- **Generator** (root package): `Complete(Input) (Output, error)`. It knows nothing about any source.
- **Adapters** (`adapter/<source>/`): one package per source. `Parse` takes file contents, never paths, and returns `mordye.Input`. `adapter/dux` is the pattern for new adapters.
- **Terminal port adapters** (`adapter/kitty`) supplement a theme adapter: `Parse` returns `mordye.TerminalInput`, and the CLI sets it on the theme's input. Why kitty: `docs/decisions/0001-kitty-ports-for-terminal-colors.md`.
- **CLI** (`cmd/mordye`): the only code that touches the filesystem. It reads the files its flags name and runs one subcommand per source.
- Mordye never discovers files. Finding a source's settings is the caller's job; keep it that way.
- `color.go` holds the OKLab/LCh math. The tuning constants in `palette.go` were set against seven themes. A changed constant shifts every generated color, so check every fixture.

## Invariants

The README promises these, and tests encode them:

- A color the theme defines passes through unchanged.
- The accent moves only `primary` and `secondary`.
- Swatches never become interface roles.
- The output always has all 37 fields, plus `sources`.

Callers template against the output shape. A change to it, or to `Version`, is
breaking; flag it and apply the `👀 Attention` label.

## Tests

- Test names state the behavior as a sentence, such as `TestAccentDoesNotMoveTerminalColors`.
- Files in `adapter/*/testdata/` are unmodified upstream themes, credited in the README. Never edit them. A new fixture is copied verbatim and credited.

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
