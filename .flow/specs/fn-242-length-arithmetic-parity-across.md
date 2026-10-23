# Length arithmetic parity: compiled Rust, the lifter and TLC agree on len() (model)

## Conversation Evidence

> fn-238.5 review (2026-10-23): arithmetic on `len()` is checked against a 32-bit range, while Rust computes it in `usize`; a probe Model with `len() * 1_000_000_000` is Verified compiled and Refuted interpreted.
> fn-238.5 implementer: the interpreter now computes lengths in `usize`; the lifter and TLC keep the mismatch. The lifter asserts the product lies in `0..2147483647`, TLC fails first with "Overflow when computing 3*1000000000" at `values = <<1, 1, 1>>`, and the TLC runner refuses that output as "mismatched TLC message terminator". Reproduction and log: `.flow/artifacts/fn-238.5/` (`lengths-tlc-probe.log`) and the `usize_lengths` control in `model/umpire/fixtures/lengths.rs`.

## Goal & Context

An admitted Model can make compiled Rust and the TLC backend disagree: the front spells `usize` as `Int{0, i32::MAX}`, the lifter asserts that range in TLA+, and TLC's 32-bit integers overflow before the assertion. The shared front end (fn-238) promises one reading for every back end, so an admitted Model must mean the same thing everywhere or be refused with a located diagnostic. Separately, the TLC runner cannot parse TLC's evaluation-error output and turns a real finding into an opaque runner refusal.

## Architecture & Data Models

- Recommended: the front admits `len()` only where every back end agrees: comparison, indexing and arithmetic whose result is provably within the TLC integer domain; other arithmetic on `len()` is refused at its source span. The interpreter, compiled checker and lifter then agree by construction. The implementer may instead bound emission with a located refusal if that keeps more existing Models admitted; record the choice.
- The TLC runner parses TLC evaluation errors (overflow and other evaluation failures) into a typed, located backend error instead of "mismatched TLC message terminator".

## Acceptance Criteria

- **R1:** `len()` arithmetic means the same in compiled Rust, the interpreted checker and TLC, or the front refuses it with a located diagnostic; the `usize_lengths` probe is either refused or agrees on all three. Existing admitted fixtures and goldens are unchanged unless they used the refused form, each listed.
- **R2:** TLC evaluation errors, including integer overflow, surface as a typed backend error naming the property and state, with a control; no "mismatched TLC message terminator" for TLC's own error output.

## Boundaries

- No change to TLC itself or its integer width; no new back end.
- No Model semantics change beyond refusing the disagreeing form.

## Decision Context

2026-10-23, conductor (owner's standing instruction to follow own recommendations): filed from the fn-238.5 review; queued in lane B after fn-238.6 and before fn-237.1, since it touches the front, lifter and TLC runner.
