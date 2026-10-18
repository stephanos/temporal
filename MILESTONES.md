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
(`umpire/dsl`, `umpire/checker`, `umpire/samples` after fn-210.10; `framework/…` until then), pinned through `mise.toml`
(`rust = "1.99.1"`). Its scope is exactly what `model/framework/samples/Kettle.scala` uses today.
The authoring-surface options are written out as the same Model seven ways in
`model2/framework/samples/options/`; the owner chose to build the plain-trait engine first
(every surface expands to it) and then the attribute surface (plain functions with pass-through
attributes). The write-up at the end, `model2/DESIGN.md`, recommends extending the prototype to
Bank and Voting or stopping, and that recommendation decides what is un-deferred.

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| Rust prototype (now) | fn-210.1 and .2, then the rename .10 and shared Cargo wiring .11, then .3 and .4 → .5 in parallel, with the decisive wasm slice .9 right after .10, then .6 and the TLC slice .8, then .7 | Engine on the `Machine` trait with BFS, claims, progress; derive and attribute macros; Kettle in the attribute surface with the Scala suite's questions and the measurement note's counts (81 starts, 283 states, depth 62) as parity; `DESIGN.md` with the comparison and the recommendation |
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
owner 2026-10-18): `insta` snapshots of each fixture's report and counterexample paths, `trybuild`
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
wasmtime-go, matching the Rust goldens byte for byte (R20). Verdict rule (R24, owner 2026-10-18): if the wasm slice passes on the stock toolchain, Rust
wins; every other measure is information. Conventions (R25 to R28, owner 2026-10-18): one `Spell` text form for all values in every
output, the author's Rust names in outputs, `Result<_, Refusal>` for API misuse with Model problems
as verdicts, an internal lifted tree. Parallel tasks touch disjoint crates; shared files are
edited by one task at a time; the gate runs locally. Gate: `make model2-check`; update
goldens with `make model2-update`. Test layout (R22, R23): each fixture lives in its owning crate's
`tests/fixtures/<name>/`, with its `model.rs` and `expected/` ("fixture", since Case is Testpilot's term); `samples` holds the real Models and their
end-to-end goldens; a dev-only `umpire-testing` crate owns the runner.
[Spec](.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md).

| Name | Status | What |
| --- | --- | --- |
| fn-210.3 | ⬜ todo | After .2: progress within a scope, weak fairness on internal steps, lasso counterexamples (R3) |
| fn-210.4 | ⬜ todo | After .1: `derive(State)` with `#[starts]`, `derive(Action)` with `#[internal]`; trybuild messages (R5) |
| fn-210.5 | ⬜ todo | After .4: `#[machine]` attribute, pass-through impl plus generated `Machine` impl; refusals for missing or duplicate `#[on]` (R5) |
| fn-210.6 | ⬜ todo | After .3/.5: Kettle in the attribute surface with its test; parity with the Scala suite (R6) |
| fn-210.8 | ⬜ todo | After .6: slice 1, `umpire-lift` from recorded tokens to TLA+, `umpire-tlc` runner, agreement with the checker's 283 states as goldens (R19) |
| fn-210.9 | ⬜ todo | After .2/.10, decisive (R24): the checker as wasm in a wasmtime-go host on the trait Kettle; report and rows byte-identical to its goldens on stock rustc means Rust (R20, R24) |
| fn-210.7 | ⬜ todo | After .6/.8/.9: `model2/DESIGN.md`: authoring, size, speed, tooling, lifting and hosting comparison and the recommendation; then delete `samples/options/` (R7) |
| fn-210.12 | 🔄 in progress | After .10/.11, before .3/.4/.9: `Spell` as the one value spelling in every output and `Result<_, Refusal>` for API misuse; goldens re-rendered, diff is spelling only (R25, R27) |

## Open for the owner

- The deferral is total by request; fn-196.8 (the Scala checker speed fix) stays deferred, since speed
  no longer decides the verdict. If any other Scala work should continue in parallel, name it.
