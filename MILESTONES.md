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
| Rust prototype (now) | fn-210.1 → .2 → .3, with .4 → .5 in parallel after .1, then .6, then the vertical slices .8 and .9 in parallel, then .7 | Engine on the `Machine` trait with BFS, claims, progress; derive and attribute macros; Kettle in the attribute surface with the Scala suite's questions and the measurement note's counts (81 starts, 283 states, depth 62) as parity; `DESIGN.md` with the comparison and the recommendation |
| After the recommendation | decided by fn-210.7 | Either grow the Rust DSL one construct at a time (inputs, collections, parameters, compositions, refinement), each landing through checker, TLA and Case together, with real Testpilot starting on the mini activity; or close fn-210 with its evidence and resume the Scala phases |

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
Kettle itself with its test, then the design note. Testing is end to end by golden files (R8,
owner 2026-10-18): `insta` snapshots of each Model's report and counterexample paths, `trybuild`
refusals, insta snapshots of macro expansions (formatted by `prettyplease`), one brute-force oracle
property test, no unit tests by default.
Architecture fixed before the checker (R9 to R17, owner 2026-10-18): a world value implements
`Machine` with associated `State` and `Action` (constants and compositions live on the world);
`step` returns an ordered list of (outcome, facts, next), empty meaning disabled; overflow checks
in every profile and a panicking effect refutes its step; tri-state verdicts with holes; a
liftable subset warned on now; structural determinism; a `wasm32-wasip2` build of dsl and checker
in the gate; a `Visited` seam over full states; `imbl` collections and `Finite`-derived inputs
recorded for Bank. Dependencies come from an admitted list (R18, owner 2026-10-18): `indexmap`
(visited set and state ids), `petgraph` (SCCs for progress), `smallvec`, `rustc-hash`, `fixedbitset`,
`enum-iterator`, `imbl`, `thiserror`, the `syn` 3 macro stack with `darling`, `manyhow` and
`prettyplease`, and `insta`, `trybuild`, `proptest`; unmaintained alternatives are banned in
`deny.toml`. Vertical before broad (owner, 2026-10-18): before the write-up, two thin slices test
the reasons Rust is on the table: Kettle lifted from the macro's tokens to TLA+ and agreeing with TLC
on all 283 states (R19), and the checker compiled to wasm and driven from a Go host through
wasmtime-go, matching the Rust goldens byte for byte (R20). Gate: `make model2-check`; update
goldens with `make model2-update`.
[Spec](.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md).

| Name | Status | What |
| --- | --- | --- |
| fn-210.1 | 🔄 in progress | Reopened for R9–R12: world-based `Machine`, associated state/actions, ordered `Steps`, and tri-state claims; adapt the compilation fixture |
| fn-210.2 | 🔄 in progress | Checker work preserved while .1 adapts; then BFS, panic paths, unknown holes, deterministic goldens, the oracle property test, and the visited-state seam (R2, R4, R8, R11, R14, R16) |
| fn-210.3 | ⬜ todo | After .2: progress within a scope, weak fairness on internal steps, lasso counterexamples (R3) |
| fn-210.4 | ⬜ todo | After .1: `derive(State)` with `#[starts]`, `derive(Action)` with `#[internal]`; trybuild messages (R5) |
| fn-210.5 | ⬜ todo | After .4: `#[machine]` attribute, pass-through impl plus generated `Machine` impl; refusals for missing or duplicate `#[on]` (R5) |
| fn-210.6 | ⬜ todo | After .3/.5: Kettle in the attribute surface with its test; parity with the Scala suite (R6) |
| fn-210.8 | ⬜ todo | After .6: slice 1, `framework-lift` from recorded tokens to TLA+, `framework-tlc` runner, agreement with the checker's 283 states as goldens (R19) |
| fn-210.9 | ⬜ todo | After .6: slice 2, `framework-wasm` exports driven by a wasmtime-go host in `model2/hosts/go`; report and rows byte-identical to the goldens; size and latency measured (R20) |
| fn-210.7 | ⬜ todo | After .6/.8/.9: `model2/DESIGN.md`: authoring, size, speed, tooling, lifting and hosting comparison and the recommendation; then delete `samples/options/` (R7) |

## Open for the owner

- The deferral is total by request; if any Scala work should continue in parallel (for example
  fn-196.8, the checker performance fix, which is needed for a fair speed comparison in fn-210.7),
  name it and it returns to this page.
