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

On 2026-10-18 the owner paused everything else for one question: should Umpire be built in Rust?
The Scala framework under `model/` keeps working as it is and is not modified. Every spec other
than fn-210 and its follow-up fn-211 is deferred in Flow (each carries a dated Decision Context
line) and is not listed here; the earlier Scala phases and the one-language series live in those
specs.

Umpire in Rust lives in `temporal/model2`, a Cargo workspace pinned through `mise.toml`
(`rust = "1.99.1"`), with crates under `model2/umpire/`: `dsl` (with `dsl/macros`), `checker`,
`testing` (the dev-only fixture runner), `tests` (the shared end-to-end fixtures), and `wasm`
(the generated-protobuf guest); the `samples` crate has been removed. Its current scope is exactly what
`model/framework/samples/Kettle.scala` uses today. The verdict is one rule (R24): if the checker
runs as wasm in a Go host through wazero (pure Go, no cgo) on the stock toolchain, with Kettle's report matching
the Rust goldens byte for byte, Rust wins. Everything else in `model2/DESIGN.md` is information. fn-196.8, the Scala checker speed fix, stays
deferred, since speed no longer decides. The decisive slice passed in fn-210.9: the recommendation
is Rust, with the explicit export-name-only linker step documented in `model2/DESIGN.md`.

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| Umpire in Rust (now) | fn-210.14 protocol wiring alone, then .3 beside .4 → .5, then .6 and the TLC slice .8, then .7 | Engine, conventions, fixtures and decisive wasm slice are complete; the Rust recommendation is recorded; the macro surface, progress, TLC slice and full write-up follow |
| After the verdict | fn-211, then decided by fn-210.7 | Apalache over the same emitter and fixtures (fn-211); then the DSL grows one construct at a time (inputs, collections, parameters, compositions, refinement), each landing through the checker, TLA and the Case together, with real Testpilot starting on the mini activity. If the verdict is no, fn-210 closes with its evidence and the Scala phases resume |

## Specs

### fn-210: Rust prototype: Kettle on a trait engine with an attribute surface (model2)

Captured on 2026-10-18. Scope is Kettle's constructs only: vars with a `oneOf` start, actors with
actions and internal steps, guarded transitions with effects, invariants, postconditions with
`before` and `after`, one `without` scope, one `whenever … eventually … in` progress claim, and the
questions `explore`, `verifyAll` and `refuted`. Gate: `make model2-check` (local); goldens update
with `make model2-update`. [Spec](.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md).

What is fixed, all owner decisions of 2026-10-18:

- **Shape (R9 to R17).** A world value implements `Machine` with associated `State` and `Action`;
  `step` returns an ordered list of (outcome, facts, next), empty meaning disabled; overflow checks
  in every profile and a panicking effect refutes its step; verdicts are holds, fails or unknown;
  a liftable subset is warned on; determinism is structural; dsl and checker build for
  `wasm32-wasip1` in the gate; the visited set sits behind a `Visited` seam; `imbl` collections and
  `Finite`-derived inputs are recorded for Bank.
- **Dependencies (R18).** Only admitted crates (a release in the last 12 months, or finished and
  very widely used): `indexmap`, `petgraph`, `smallvec`, `rustc-hash`, `fixedbitset`,
  `enum-iterator`, `imbl`, `thiserror`, the `syn` 3 macro stack with `darling`, `manyhow` and
  `prettyplease`, and `insta`, `trybuild`, `proptest`. Unmaintained alternatives are banned in
  `deny.toml`.
- **Testing (R8, R22, R23, R29).** End to end by golden files, no unit tests by default, one
  brute-force oracle property test. A fixture is a folder with its `model.rs` and `expected/`
  ("fixture", since Case is Testpilot's term). Fixtures every backend must agree on live in
  `umpire/tests/fixtures/<name>/`, and their expected files select the consumers (checker, TLC,
  Apalache); checker-only fixtures (panics, overflow, holes, budget cuts, refusals) stay in
  `umpire/checker/tests/fixtures/`. A Model's source is its fixture's `model.rs`; there is no
  `samples` crate.
- **Conventions (R25 to R28).** One `Spell` text form for every value in every output; the
  author's Rust names in outputs, mapped only at a future Case boundary; `Result<_, Refusal>` for
  API misuse, Model problems as verdicts; the lifted tree is internal.
- **Vertical before broad (R19, R20, R24).** Two thin slices before the write-up: Kettle lifted
  from the macro's tokens to TLA+ and agreeing with TLC on all 283 states, and the checker as wasm
  in a wazero Go host. The wasm slice alone decides the verdict.
- **Feedback loop (R30).** The inner loop (edit one crate, rebuild, run its fixtures) stays at 3 s or
  less and the full local gate at 30 s or less, warm (baseline 0.8 s and 0.7 s). `make model2-time`
  prints each step and warns over budget; every task reports it when marked done, and a breach is
  fixed in the task or excepted by the owner. Slow tools (JVM, network) run batched or behind
  `UMPIRE_BACKENDS=require`.
- **Host interface (R31, R32).** The Model talks to any host through protobuf messages over a
  three-function core-wasm ABI (`alloc`, `free`, `call`), generated on both sides from one schema in
  `model2/umpire/proto/` (`prost` in Rust, `protoc-gen-go` in Go). The Go host is wazero, so Go
  stays at `CGO_ENABLED=0`; the module targets `wasm32-wasip1`. wasmtime remains the simulation's
  host; the same module loads in both.
- **Process.** Parallel tasks touch disjoint crates; shared files (`Cargo.toml`, `Cargo.lock`,
  `Makefile`, `README.md`) are edited by one task at a time; a task that moves paths runs alone and
  every session re-reads its task after it.

| Name | Status | What |
| --- | --- | --- |
| fn-210.14 | 🔄 in progress | After .9, alone: pre-wire the R27 scope-action Refusal through the generated host protocol, keeping .3/.4 disjoint |
| fn-210.3 | ⬜ todo | After .14: progress within a scope, weak fairness on internal steps, lasso and dead-end counterexamples (R3) |
| fn-210.4 | ⬜ todo | After .14, beside .3: `derive(State)` and `derive(Action)` with `Spell`; trybuild messages (R5, R17, R25) |
| fn-210.5 | ⬜ todo | After .4: `#[machine]` attribute, pass-through impl plus generated `Machine` impl; refusals for missing or duplicate `#[on]` (R5, R13, R26) |
| fn-210.6 | ⬜ todo | After .3/.5: Kettle in the attribute surface as its fixture's `model.rs`; goldens identical to the trait Kettle's (R6) |
| fn-210.8 | ⬜ todo | After .6: TLC slice: `umpire-lift` from recorded tokens to TLA+, `umpire-tlc` (in `umpire/backends/tlc/`) runner and consumer, `tlc.txt` agreement on 283 states (R19, R26, R28) |
| fn-210.7 | ⬜ todo | After .6/.8/.9: `model2/DESIGN.md` with the verdict and the comparison; then delete `options/` (R7, R24) |

### fn-211: Apalache backend for Umpire in Rust (model2)

Captured on 2026-10-18 as a follow-up, after fn-210 and only if its verdict is to continue in
Rust. Adds Apalache as a second TLA+ backend over the same lifted tree and emitter: Snowcat type
annotations in the emitter, a pinned Apalache release run on the mise JDK with one bounded check
per invariant at the native depth plus one, ITF counterexamples decoded through `Spell` and
replayed through the Model, and an Apalache consumer selected by `expected/apalache.txt`. No
liveness through Apalache; TLC owns progress claims.
[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.1 | ⬜ todo | Snowcat type annotations in the TLA+ emitter; TLC results unchanged (R1) |
| fn-211.2 | ⬜ todo | After .1: `umpire-apalache` in `umpire/backends/apalache/`, pinned release, bounded check per invariant, deterministic lifecycle (R2) |
| fn-211.3 | ⬜ todo | After .2: ITF counterexamples decoded through `Spell` and replayed through the Model (R3) |
| fn-211.4 | ⬜ todo | After .3: Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |

## Open for the owner

Nothing open.
