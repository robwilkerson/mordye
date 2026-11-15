# 1. Kitty Ports for Terminal Colors

Date: 2026-11-11

## Status

Accepted.

## Context

Dux theme files carry about six chromatic colors and no terminal palette. Mordye fills the 16 terminal slots by hue, and the result often differs from the theme's own terminal port ([#32](https://github.com/robwilkerson/mordye/issues/32)). Some themes fill slots by name rather than hue: Dracula's blue slot holds its purple. No hue rule can recover that.

Almost every theme publishes official terminal ports that name all 16 slots. Mordye already passes `Input.Terminal` through unchanged, so reading a port fixes the gap without changing the generator.

The question was which port format to read. The format doesn't need to match the terminal anyone runs. The port is only a source of colors, and templates write the output for any terminal.

## Decision

Read kitty theme files, through `adapter/kitty` and a `--terminal-palette FILE` flag on `mordye dux`. The flag names what the file holds, the theme's canonical terminal palette, rather than its format. A second format could then arrive behind the same flag.

- **Easiest to get:** most themes ship an official kitty port, as four of the six Dux fixtures do. Kitty's own [theme collection](https://github.com/kovidgoyal/kitty-themes) covers the other two, Nord and Gruvbox Material.
- **Easiest to read:** each line is a key and a value, so the standard library parses it in a few lines.
- **Alternatives:** Alacritty and WezTerm schemes are TOML, which would mean a hand-written partial parser. Foot's files are simple, but foot doesn't ship every fixture theme (Gruvbox Material is missing).

The maintainer runs WezTerm, with foot as a backup, and not kitty. That's expected.

## Consequences

- A user with no kitty port at hand still gets hue-matched colors, as before.
- The colors match the port passed in. Ports of the same theme can disagree with each other (Rosé Pine's foot and Alacritty ports put different colors in green).
- Another port format can be added later as another adapter and flag, if one is ever easier to get for some theme.
