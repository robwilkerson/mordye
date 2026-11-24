# 3. Format Versioning

Date: 2026-11-24

## Status

Accepted ([#13](https://github.com/robwilkerson/mordye/issues/13)).

## Context

One `version` field covers both the input and output formats, and input must match it exactly. Nothing said which changes bump it.

Callers mostly don't need it. The only caller today pins the Mordye binary, and the release number already signals breaking changes. The field earns its place in one case: a stored palette that outlives the binary that wrote it. A caller can check `version` and refuse a palette it doesn't understand, instead of misreading it. Removing the field would itself be a breaking change, so it stays, with a short rule for when it moves.

## Decision

- **One `version` for input and output.** They move together.
- **It bumps when an existing field is removed or renamed, or its value changes meaning or format.**
- **Additions don't bump it.** A new optional input field or a new output field leaves existing callers working.
- **Generated colors don't bump it.** A change to how Mordye fills a missing color is tuning, noted in the changelog. The README's invariants, such as defined colors passing through unchanged, are the contract.
- **`sources` is outside the contract.** It explains each color for debugging, and its shape and values can change in any release without a bump. Nothing reads it programmatically, and a format for it would only constrain how it explains things ([#14](https://github.com/robwilkerson/mordye/issues/14)).
- **Input must match exactly.** A mismatch fails with an error, never a palette built from a format Mordye doesn't speak.
- **A bump ships as a minor release before 1.0, and a major release after.**

## Consequences

- Callers can template against the output fields and rely on them until `version` changes.
- Most improvements, including better generated colors, ship without a format change.
- A caller that wants to catch a stale palette compares `version` with the one it was written for.
