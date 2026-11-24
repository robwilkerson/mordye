# Contributing to Mordye

Thanks for helping. Most contributions are one of three kinds: a fix, a
bundled terminal palette for a theme you use, or an adapter for a new source.
This guide covers each.

## Before You Start

For anything bigger than a small fix, please
[open an issue](https://github.com/robwilkerson/mordye/issues) first and say
what problem it solves. Mordye keeps a small surface on purpose, so it's
worth agreeing on the shape before you write the code.

## Setup

You need Go 1.27 or later and [just](https://github.com/casey/just). Mordye
uses the standard library only; adding a module dependency is a decision to
raise in the issue first.

After cloning, run:

```sh
just bootstrap
```

It turns on the git hooks in `.githooks/` and checks the tools a fresh clone
needs. It's safe to run again.

## Making a Change

The project's rules live in [CLAUDE.md](CLAUDE.md): the architecture, the
invariants the README promises, and how tests are written. Despite the name,
it's for people too; the AI reviewers read it as their rules, so it stays the
one place they're kept.

A few that come up most:

- **Run `just check` before you push.** It formats, vets, and tests, and CI
  runs the same checks on every push and PR.
- **Name tests as a sentence stating the behavior,** such as
  `TestAccentDoesNotMoveTerminalColors`.
- **Never edit a file under `testdata/`.** Those are unmodified upstream themes
  and ports, credited in the README. A new one is copied verbatim and
  credited, and new behavior is tested against those real files.
- **Add a changelog entry** under `## Unreleased` in `CHANGELOG.md` when
  someone using Mordye would notice the change. Write what changed for them.
  Refactors, tests, and tooling don't need one.

## Adding a Terminal Palette

Mordye bundles each theme's official terminal palette, so a theme's terminal
colors come out as its authors made them.
[ADR 0002](docs/decisions/0002-bundled-terminal-palettes.md) explains the
design.

**What qualifies:** a port published by the theme's own authors, under a
license compatible with MIT (MIT, Apache-2.0, BSD, and the like). Any format
works if there's a converter for it. Ports by third parties don't qualify,
even popular ones, because ports of one theme can disagree.

**Steps:**

1. **Check the format.** `tools/palettes` reads `kitty`, `xresources`, and
   `vim` today. A new format needs a converter in `converters/`: one file and
   one function, tested against the real upstream port copied into
   `converters/testdata/<format>/`.
2. **Write the palette** from the port at a full commit SHA, from the
   repository root:

   ```sh
   go run ./tools/palettes add \
     -id catppuccin-mocha -mode dark \
     -repo catppuccin/kitty \
     -commit <40-character SHA> \
     -path themes/mocha.conf -format kitty \
     -spdx MIT -license-path LICENSE
   ```

   The id follows the port's own file name. When the license is stated only
   in a README, pass `-license-path README.md -license-line <line>`. The tool
   writes `palettes/<id>.json` and the license notice under
   `palettes/notices/`. Never edit either by hand.

   A theme whose terminal colors live only in its Vim colorscheme uses
   `-format vim` with `-path colors/<name>.vim`. The tool loads it in headless
   Neovim (`nvim` must be on your `PATH`), with the background from `-mode`
   and each `-option name=value` set as a global variable to pick the variant,
   such as `-option everforest_background=hard`. Name the id
   `<theme>-<mode>-<variant>`.
3. **Map the theme** to the new id in each adapter that knows it, such as the
   `palettes` map in `adapter/dux/dux.go`. A test there fails if an adapter
   maps to a palette that isn't bundled.
4. **Credit the port** in the README's Bundled Palettes table. A new converter
   fixture is credited in its Credits section too.
5. **Check it reproduces:**

   ```sh
   go run ./tools/palettes check
   ```

   It rebuilds every palette from its recorded source and fails if any file
   differs. CI runs it too.

## Adding a Source Adapter

An adapter reads one source's theme settings, such as Dux's, and turns them
into Mordye's input. [`adapter/dux`](adapter/dux) is the pattern to follow.

- One package per source, under `adapter/<source>/`.
- `Parse` takes file contents, never paths, and returns `mordye.Input`. The CLI
  reads the files; Mordye never discovers them.
- The adapter maps its own theme names to bundled palette ids. Knowledge of a
  source stays in its adapter.
- A malformed or unexpected file fails with a clear error, never a partial
  palette.
- Add a subcommand for the source in `cmd/mordye`, and a row to the source
  table in the README's How It Fits Together section.

## Pull Requests

- Branch from `main`, and say in the description why the change exists and how
  you tested it. Link the issue it closes.
- CodeRabbit reviews every PR against CLAUDE.md, and PR-Agent does too on
  PRs from this repository; a maintainer can run it on a fork's. Their
  findings are suggestions: fix them, or reply with why not. Either counts as
  addressed.
- A maintainer reviews once the bots are done.
