---
satisfies: [R2]
---
# fn-213-modelsuite-in-rust-examples-and.2 umpire-suite examples: Run, step, repeat, expect!

## Description
New crate `model/umpire/suite/` (`umpire-suite`, depends on `umpire-dsl` and `umpire-checker`): `Run::from(state)`, `Run::step(self, action) -> Result<Run, Refusal>` taking the one emitted step of an enabled action through `Machine::step`, `Run::repeat(self, n, action)`, `Run::state()`. A disabled action or one emitting several steps is a `Refusal` naming the action and spelling the state. `expect! { a; b; }` is a declarative macro evaluating each Boolean line in order and failing the test at the first false one with that line's source text (`stringify!`) and the spelled state. Failure messages pinned by fixtures in `umpire/suite/tests/fixtures/` (wrong expect line, disabled step, ambiguous step).

**Touches:** `model/umpire/suite/**`, workspace `Cargo.toml`/`Cargo.lock` (one task at a time), `Makefile` model section (exact minimal replacement).

**Feedback loop (R30):** paste `make model-time` into the done summary.

**Added 2026-10-19 (fn-215 R4):** `Run::along(path: Path<State>) -> Result<Run, Refusal>`: the first waypoint must be a concrete state (else refused); each gap is bridged by the shortest admitted path to the next waypoint (`..=n` bounds it), then replayed through `Machine::step`, so the result is an ordinary `Run`; an unbridgeable gap is a `Refusal` naming the waypoint index and the last reached state. Needs fn-215 R1 (`Path`) first; parallel with .3 otherwise.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
