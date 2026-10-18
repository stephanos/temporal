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
The Scala framework is frozen under `model/archived/` for reference and is not expected to build. Every spec other
than fn-210 and its follow-ups fn-212 and fn-211 is deferred in Flow (each carries a dated Decision Context
line) and is not listed here; the earlier Scala phases and the one-language series live in those
specs. Those deferred Scala plans retain paths relative to the archived tree.

Umpire in Rust lives in `temporal/model`, a Cargo workspace pinned through `mise.toml`
(`rust = "1.99.1"`), with crates under `model/umpire/`: `dsl` (with `dsl/macros`), `checker`,
`testing` (the dev-only fixture runner), `tests` (the shared end-to-end fixtures), and `wasm`
(the generated-protobuf guest); the `samples` crate has been removed. Its current scope is exactly what
`model/archived/framework/samples/Kettle.scala` uses today. The verdict is one rule (R24): if the checker
runs as wasm in a Go host through wazero (pure Go, no cgo) on the stock toolchain, with Kettle's report matching
the Rust goldens byte for byte, Rust wins. Everything else in `model/DESIGN.md` is information. fn-196.8, the Scala checker speed fix, stays
deferred, since speed no longer decides. The decisive slice passed in fn-210.9: the recommendation
is Rust. The guest now directly exports `umpire_buffer_alloc`, `umpire_buffer_release` and
`umpire_call`, with no custom linker or export rewriting (fn-210.16).

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| Umpire in Rust (now) | fn-210.6 and .7 | The archive, registration, progress, direct wasm ABI and generic Go host relocation are complete; attribute Kettle and the comparison remain. Standalone extraction is planned, not executed, in `MIGRATE.md`. |
| After the verdict | fn-212, then fn-211 if approved; Case slice planned by fn-210.7 | fn-212 first decides whether lifting and TLC's second opinion justify their machinery; fn-211 adds Apalache over that emitter if it proceeds. The Case slice makes the Model the oracle, with one system-agnostic protobuf vocabulary and a Temporal extension (R33). Program and evidence capture stay; the Contract and all CEL go; the Model decides read completion and SDK callback responses (R35, R37); workers in other languages, such as a Java SDK worker, reach the run's single Model through the runner's local endpoint with a thin shim per SDK (R40). One host-loop command replaces run/fuzz/replay/assess/repeat modes (R38). A Run is observations plus the Model's judgment; profiles, receipts and provenance go, while the never-overwrite publish store stays (R39). Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. |

## Specs

### fn-210: Rust prototype: Kettle on a trait engine with an attribute surface (model)

Captured on 2026-10-18. Scope is Kettle's constructs only: vars with a `oneOf` start, actors with
actions and internal steps, guarded transitions with effects, invariants, postconditions with
`before` and `after`, one `without` scope, one `whenever … eventually … in` progress claim, and the
questions `explore`, `verifyAll` and `refuted`. Gate: `make model-check` (local); goldens update
with `make model-update`. [Spec](.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md).

What is fixed, all owner decisions of 2026-10-18:

- **Shape (R9 to R17).** A world value implements `Machine` with associated `State` and `Action`;
  `step` returns an ordered list of (outcome, facts, next), empty meaning disabled; overflow checks
  in every profile and a panicking effect refutes its step; verdicts are holds, fails or unknown;
  determinism is structural; dsl and checker build for
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
  API misuse, Model problems as verdicts. A future lifted tree is internal.
- **Layout (R44).** A Model is one `#[machine] mod` mirroring Scala's nested objects: the state struct, the action
  enums, then one `impl` per part kind in order (`#[transitions]`, `#[invariants]`, `#[postconditions]`, `#[scopes]`,
  `#[progress]`), each at most once; the attribute rejects any other order or a duplicate group.
- **Decisive slice and follow-up (R20, R24, R36).** The checker runs as wasm in a wazero Go host,
  which decides the Rust verdict. Lifting, subset warnings and TLC agreement move to fn-212;
  `#[machine]` stays here as registration only, without token recording or subset warnings.
- **Feedback loop (R30).** The inner loop (edit one crate, rebuild, run its fixtures) stays at 3 s or
  less and the full local gate at 30 s or less, warm (baseline 0.8 s and 0.7 s). `make model-time`
  prints each step and warns over budget; every task reports it when marked done, and a breach is
  fixed in the task or excepted by the owner. Slow tools (JVM, network) run batched or behind
  `UMPIRE_BACKENDS=require`.
- **Host interface (R31, R32, R42).** The Model talks to any host through protobuf messages over a
  three-function core-wasm ABI (`umpire_buffer_alloc`, `umpire_buffer_release`, `umpire_call`), generated on both sides from one schema in
  `model/umpire/proto/` (`prost` in Rust, `protoc-gen-go` in Go). The Go host is wazero, so Go
  stays at `CGO_ENABLED=0`; the module targets `wasm32-wasip1`. wasmtime remains the simulation's
  host; the same module loads in both.
  `oracle.proto` stays minimal (explore, rows, refusal) until the Case slice merges it with
  Testpilot's schema into one vocabulary (R33).
- **Process.** Parallel tasks touch disjoint crates; shared files (`Cargo.toml`, `Cargo.lock`,
  `Makefile`, `README.md`) are edited by one task at a time; a task that moves paths runs alone and
  every session re-reads its task after it.

| Name | Status | What |
| --- | --- | --- |
| fn-210.6 | 🔄 in progress | Parallel lanes: attribute Kettle with separate User/Heater enums and checker claim ordering; `#[machine]` enforces the Scala part order as an error and the approved nested surface (`#[machine] mod` with one impl per part kind, actor-level `#[internal]`, optional `before`, `without([..])` scopes, `whenever(..).eventually(..)` progress as per-state booleans); only the four approved action spellings change in canonical goldens; trait reference stays unchanged (R6, R14, R44) |
| fn-210.7 | ⬜ todo | After .6: `model/DESIGN.md` with the verdict and the comparison; then delete `options/` (R7, R24) |

### fn-212: TLA+ lifting and the TLC backend for Umpire in Rust (model)

After fn-210, first decide whether the independent reading and temporal checks justify the
lifting subset, emitter and JVM gate. On approval, lift shared Models, run pinned TLC and compare
reachable states and verdicts through the shared fixture consumer.
[Spec](.flow/specs/fn-212-tla-lifting-and-the-tlc-backend-for.md).

| Name | Status | What |
| --- | --- | --- |
| fn-212.1 | ⬜ todo | Decide cost, benefit and lifting source; on no, close fn-212 and fn-211 with the note |
| fn-212.2 | ⬜ todo | On yes: lift, emit TLA+, run TLC and require agreement on shared fixtures |

### fn-211: Apalache backend for Umpire in Rust (model)

Captured on 2026-10-18 as a follow-up, after fn-210 and fn-212 and only if their verdicts are to continue in
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
