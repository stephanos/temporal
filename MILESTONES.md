# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-18.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's remaining tasks in a `Name | Status | What` table: the task ID, its status
  (🔄 in progress, ⬜ todo, ⏸️ deferred) and a brief description. Remove a task's row once it is done,
  and the spec once it is closed. Flow and git keep the history.
- Only the root agent edits this page. Workers report status in their handoffs.
- Update the "As of" date with every edit.
- Task/spec branch integration follows `AGENTS.md`'s Local milestone execution policy.

## Direction

On 2026-10-18 the owner paused everything else for one question: is Rust the better base for the
framework? The Scala framework under `model/` keeps working as it is and is not modified. Every
spec other than fn-210 is deferred in Flow (each carries a dated Decision Context line) and is not
listed here; the previous version of this page, with the Scala phases and the one-language
series, is in Flow's history and in the specs themselves.

The prototype lives in `temporal/model2`, a Cargo workspace mirroring `model/framework`'s layout
(`framework/dsl`, `framework/checker`, `framework/samples`), pinned through `mise.toml`
(`rust = "1.99.1"`). Its scope is exactly what `model/framework/samples/Kettle.scala` uses today.
The authoring-surface options are written out as the same Model seven ways in
`model2/framework/samples/options/`; the owner chose to build the plain-trait engine first
(every surface expands to it) and then the attribute surface (plain functions with pass-through
attributes). The write-up at the end, `model2/DESIGN.md`, recommends extending the prototype to
Bank and Voting or stopping, and that recommendation decides what is un-deferred.

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| Rust prototype (now) | fn-210.1 → .2 → .3, with .4 → .5 in parallel after .1, then .6 → .7 | Engine on the `Machine` trait with BFS, claims, progress; derive and attribute macros; Kettle in the attribute surface with the Scala suite's questions and the measurement note's counts (81 starts, 283 states, depth 62) as parity; `DESIGN.md` with the comparison and the recommendation |
| After the recommendation | decided by fn-210.7 | Either extend to Bank and Voting and re-plan the deferred specs in Rust terms, or close fn-210 with its evidence and resume the Scala phases |

## Specs

### fn-210: Rust prototype: Kettle on a trait engine with an attribute surface (model2)

Captured on 2026-10-18 from the owner's decision to prototype the framework in Rust before going
deeper on the Scala side. Scope is Kettle only: vars with a `oneOf` start, actors with actions and
internal steps, guarded transitions with effects, invariants, postconditions with `before` and
`after`, one `without` scope, one `whenever … eventually … in` progress claim, and the questions
`explore`, `verifyAll` and `refuted`. Engine first (a `Machine` trait, BFS with a shared visited
set, claims per state and per edge, progress with weak fairness and lasso counterexamples), then
the attribute surface (`#[derive(State)]`, `#[derive(Action)]`, a pass-through `#[machine]` on the
impl collecting `#[on]`, `#[invariant]`, `#[postcondition]`, `#[scope]`, `#[progress]`), then
Kettle itself with its test, then the design note. Gate: `mise exec -- cargo test --workspace`.
[Spec](.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md).

| Name | Status | What |
| --- | --- | --- |
| fn-210.1 | ⬜ todo | `framework-dsl`: `Machine`, `Claim`, `StepClaim`, `Scope`, `Progress`, `Outcome`; `Kettle.trait.rs` compiles as a fixture (R1) |
| fn-210.2 | ⬜ todo | After .1: `framework-checker` BFS explore with budget, invariants per state, postconditions per edge, `verify_all`, `refuted`; Kettle counts match the Scala note (R2, R4) |
| fn-210.3 | ⬜ todo | After .2: progress within a scope, weak fairness on internal steps, lasso counterexamples (R3) |
| fn-210.4 | ⬜ todo | After .1: `derive(State)` with `#[starts]`, `derive(Action)` with `#[internal]`; trybuild messages (R5) |
| fn-210.5 | ⬜ todo | After .4: `#[machine]` attribute, pass-through impl plus generated `Machine` impl; refusals for missing or duplicate `#[on]` (R5) |
| fn-210.6 | ⬜ todo | After .3/.5: Kettle in the attribute surface with its test; parity with the Scala suite (R6) |
| fn-210.7 | ⬜ todo | After .6: `model2/DESIGN.md`: authoring, size, speed, tooling comparison and the recommendation (R7) |

## Open for the owner

- Whether `temporal/model2`'s option sketches stay in the tree after fn-210.6 lands Kettle, or move
  to `model2/DESIGN.md` as excerpts.
- The deferral is total by request; if any Scala work should continue in parallel (for example
  fn-196.8, the checker performance fix, which is needed for a fair speed comparison in fn-210.7),
  name it and it returns to this page.
