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
New crate `model/umpire/suite` (`umpire-suite`, deps umpire-dsl, umpire-checker, thiserror; dev umpire-testing): `Run::from(state)`, `step`, `repeat`, `state`, `path`, `Display` (spelled as the checker's `paths/` goldens), and `Run::along(path!)` (fn-215 R4). `expect! { k = run.state(); a; b; }` panics at the first false line with `stringify!` text and the spelled state. Suite `Refusal`: Disabled, Ambiguous, Unanchored, Unbridged { waypoint, state, cut }, Checker(..). Fixtures in `umpire/suite/tests/fixtures/` (expect_line, disabled_step, ambiguous_step, along) pin every message.

Shared leg search for fn-213.3: `suite/src/search.rs::through(world, starts, &path, budget) -> Result<Trace, Refusal>`; first leg from `starts` to waypoint 0, then shortest leg per gap, each leg answered by `Checker::explore` on a claim-free `Rooted` view of the world (the suite adds no exploration logic). `find` should call it with the space's starts and budget; `Unbridged.cut` tells unknown from none.

Deviations:
- `Run<S>` is generic over the state, not the machine: `Run::from(state)` and `Run::along(path)` must infer the world from a state alone. Added `umpire_dsl::Registered` (state -> world), implemented by `#[machine]` (expansion goldens updated); manual Models implement it by hand (the suite's coin fixture does).
- Checker additions: `Checker::BUDGET` (the 1<<20 default, now used by `refuted`) and `Result::path_to(StateId)` (public, checked wrapper over the parent-path walk).
- `expect!` names its state: `expect! { k = run.state(); ..; }`, since the macro cannot otherwise spell the state.
- `Run::along` takes `Path` by value (spec form), with a local `needless_pass_by_value` allow.
- README gained a short `umpire-suite` paragraph; no Makefile change was needed (the gate runs `--workspace`).

model-time:
```
timing-controls 0.015s, capability-controls 0.095s, source-boundary 0.112s, protobuf-freshness 0.060s,
temporal-protobuf-freshness 0.055s, fmt 0.222s, clippy 0.090s, docs 0.155s, tests 2.497s,
wasm32-wasip1 0.058s, core-wasm-clippy 0.069s, wasm-guest 0.063s, go-host 1.184s, temporal-go-host 5.885s
inner loop (edit+test)            2.746s
gate total                       10.783s  (soft target 30s)
inner loop                        2.746s  (soft target 10s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f943c47cd6
- Tests: make model-check, make model-time
- PRs: