# ModelSuite in Rust: examples and explorations (model)

## Conversation Evidence

> user (2026-10-18): "do we have a plan for the equivlent of ModelSpec from Scala?" — answered: no; nothing on the Rust side replaces fn-195's ModelSuite, and the Case slice has no author-facing front door without it.
> user (2026-10-18): "let's wait with conformance for now." and, on the test file's place, "should this live inside the model.rs or next to it?" — answered: next to it, `tests.rs` beside `model.rs`; user: "yes".

## Goal & Context

In Scala a Model's test file (fn-195, ModelSuite) does two jobs: it tests the Model by authored example, by exhaustive claims over a space and by sampling, and it declares the conformance tests that become Cases. Umpire in Rust has no equivalent: the only authored test interface is the `umpire_testing::fixture!` golden hook inside a fixture's `model.rs`, which serves the framework, not the Model's author. This spec gives every shared Model a plain-Rust test file beside it with two of the four Scala sections, `examples` and `explorations`. `conformance` (and with it `sample`, since generated paths on the Model and on the system are one activity) waits for the Case slice, by owner decision of 2026-10-18.

Everything is plain Rust: `#[test]` functions in two modules, `cargo test` as the runner, failures spelled with `Spell` so they read like the checker's reports.

## Architecture & Data Models

- Layout: a shared fixture is `umpire/tests/fixtures/<name>/{model.rs, tests.rs, expected/}`; `build.rs` declares both files as modules (`<name>` and `<name>_tests`), and the golden hook (`fixture!`) moves from `model.rs` into `tests.rs`, so `model.rs` is only the Model (what `#[machine]` registers, what `umpire-wasm` includes, what fn-212 would lift). A sibling module cannot see private fields, so the state struct's fields are `pub` (as the trait reference's are, and as Scala's `var`s were).
- Crate: `umpire/suite/` (`umpire-suite`), depending on `umpire-dsl` and `umpire-checker`; test files `use umpire_suite::*`. It holds no checker logic: an authored path replays through the same `Machine::step` the checker uses, and explorations call the checker's `explore`, `verify_all` and `refuted`.
- `examples`: an authored path. `Run::from(state)` starts at a concrete state; `.step(action)` takes the one emitted step of an enabled action; `.repeat(n, action)` takes it `n` times; `.state()` reads the current state. A disabled action or one emitting several steps is a `Refusal` naming the action and spelling the state, never a panic. `expect! { a; b; c; }` evaluates each Boolean line and fails the test at the first false one, printing that line's source text and the spelled state (the Scala `expect:` block, as a declarative macro; no compiler-phase work).
- `explorations`: a `Space` is a set of start states and a budget (`Space::from([state]).budget(n)`, default the checker's); `verify_all(&space)`, `refuted(&space, predicate)` (a `Result` carrying the counterexample path as spelled actions) and `find(&space, predicate)` (the shortest path to a matching state, or a `Refusal` that none exists within the budget). Scopes are the Model's own (`#[scopes]`); a space does not redefine them.
- The two sections are modules `mod examples` and `mod explorations` in `tests.rs`; nothing registers them, `cargo test` discovers them. Failure messages of the suite API are pinned by fixtures in `umpire/suite/tests/fixtures/` (a wrong `expect!` line, a disabled step, an unreachable `find`).

## API Contracts

- `Run::from(state) -> Run<M>`; `Run::step(self, action) -> Result<Run<M>, Refusal>`; `Run::repeat(self, n, action) -> Result<Run<M>, Refusal>`; `Run::state(&self) -> &M::State`.
- `expect! { expr; expr; .. }` inside a `#[test]`.
- `Space::from(starts) -> Space<M>`; `.budget(n)`; `verify_all(&Space) -> Result<Report, Refusal>`; `refuted(&Space, fn(&State) -> bool) -> Result<Counterexample, Refusal>`; `find(&Space, fn(&State) -> bool) -> Result<Path, Refusal>`.
- Paths and counterexamples spell as the checker's `paths/<claim>.txt` do.

## Edge Cases & Constraints

- fn-210 R30 applies: the suite adds no slow tool; Kettle's `tests.rs` runs inside the inner-loop and gate budgets, and `make model-time` output is pasted into each task's done summary.
- A budget cut answers unknown, never verified; `refuted` on a cut exploration still stands when a counterexample was found before the cut.
- `expect!` reports the first failing line only; it does not render subexpressions.

## Acceptance Criteria

- **R1:** Every shared fixture is `{model.rs, tests.rs, expected/}`: `build.rs` declares both, the golden hook lives in `tests.rs`, `model.rs` holds only the Model, and the state struct's fields are `pub`. Goldens are unchanged by the move. Errors: a fixture without `tests.rs` fails the run naming it.
- **R2:** `examples`: `Run::from`, `step`, `repeat` and `state` replay an authored path through `Machine::step`; a disabled or ambiguous action is a `Refusal` naming the action and spelling the state. `expect!` fails at the first false line with its source text and the spelled state. Errors pinned by suite fixtures.
- **R3:** `explorations`: `Space`, `verify_all`, `refuted` and `find` answer the checker's questions from the space's starts and budget, with paths spelled as the checker's goldens; an unreachable `find` and a cut exploration are refusals and unknowns, never panics or false verdicts. Errors pinned by suite fixtures.
- **R4:** Kettle's `tests.rs` asks the questions of `model/archived/framework/samples/Kettle.test.scala` that the surface admits: the press/heat/depress enabledness table from the coldest and room starts, the 99-to-100 boundary and the `>=` failover guard, `boils` from room temperature in 81 steps within `never_switched_off`, `never_beeps` refuted by press, 80 heats and failover, a budget of 10 answering unknown, and vacuity of `fails_over_at_boiling` when the heater is excluded. No error surface.
- **R5:** Plain Rust throughout: `#[test]` functions in `mod examples` and `mod explorations`, run by `cargo test` and the existing gate; no registration, no runner beyond cargo, and the inner-loop and gate budgets of fn-210 R30 hold. No error surface.

## Boundaries

- No `conformance` section, no Cases, no `sample`: they arrive with the Case slice.
- No `unit` section: an ordinary `#[test]` outside the two modules is already that.
- No change to the Model surface of fn-210.6 beyond `pub` fields; no lifting (fn-212), no wasm host changes.
- Does not start before fn-210 is done.

## Decision Context

2026-10-18, owner: the ModelSuite equivalent is planned now but after fn-210.7; conformance waits ("let's wait with conformance for now"); the test file lives next to `model.rs`, not inside it, so the Model file stays the whole liftable unit and the fixture folder is model + tests + goldens; `pub` fields are the accepted consequence.

2026-10-19, owner: fn-215's `Path` is the suite's pattern too; `find` takes a `path!` through waypoints and `Run::along` fills a milestone story (fn-215 R4, delivered by .3 and .2). Tasks .2/.3 wait for fn-215 R1.
