# Bundled Palettes

Official terminal palettes that Mordye embeds in its binary, one JSON file per
palette, named by its id. `notices/` holds each upstream license, which the
release ships alongside the binary.

Don't edit these files. `tools/palettes` writes them from each theme's
official port at a pinned commit, and CI rebuilds them to prove they match.
[CONTRIBUTING.md](../CONTRIBUTING.md) covers adding one, and
[ADR 0002](../docs/decisions/0002-bundled-terminal-palettes.md) covers why.
