# Contributing to Mordye

<!-- Outline. Each section's notes say what it will cover. -->

## Before You Start

- Open an issue first for anything bigger than a small fix, and say what problem it solves.

## Setup

- Go 1.27+, standard library only.
- Run `just bootstrap`.

## Making a Change

- Run `just check` before pushing; CI runs the same checks.
- Test naming, and why upstream fixtures are never edited.
- When a change needs a changelog entry.
- Link to CLAUDE.md for the project's rules, rather than repeating them.

## Adding a Terminal Palette

- What qualifies: the theme authors' own port, under an MIT-compatible license.
- The four steps from ADR 0002, with exact commands once the converters exist.
- Link to [ADR 0002](docs/decisions/0002-bundled-terminal-palettes.md).

## Adding a Source Adapter

- `adapter/dux` is the pattern.
- Adapters take file contents, never paths, and map their theme names to palette ids.

## Pull Requests

- What the PR description covers.
- CodeRabbit and PR-Agent review against CLAUDE.md.
