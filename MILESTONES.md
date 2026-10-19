# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-19.

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
The Scala framework is frozen under `model/archived/` for reference and is not expected to build.
Earlier Scala phases and the one-language series remain deferred in Flow and retain paths relative
to the archived tree. The Rust follow-ups below use the owner's priority order; their individual
readiness gates and dependencies still apply.

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
| Umpire in Rust (now) | fn-210 final review | All implementation tasks and the measured DESIGN comparison are complete. Recommendation: Rust, under stock WASM/Go parity alone. Cross-task spec review and final quality audits remain. Standalone extraction is planned, not executed, in `MIGRATE.md`. |
| After the verdict | fn-214 → fn-216 → fn-213 → fn-215 → fn-212 → fn-218 → fn-217 → fn-211 | Owner priority order. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 decides whether lifting and TLC justify their machinery; fn-211 adds Apalache if approved. The Case slice planned by fn-210.7 keeps the Model as oracle, one Model per run, thin SDK shims, one host loop, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. |

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
- **Layout (R44, flat form since 2026-10-19).** A Model is one `#[machine] mod` with the state struct, the action
  enums and one `impl Model` whose methods carry per-item attributes (`#[on(Actor)]` match methods, optional
  `#[effect]`, `#[invariant]`, `#[postcondition]`, `#[assumption]`, `#[progress]`); `#[machine]` enforces that
  kinds are grouped and in that order, that only `#[on]`/`#[effect]` take `&mut self`, and emits
  `#[forbid(unsafe_code)]`; every attribute is pass-through. Claims are `check!` statements (.19); a progress claim
  names its assumption in its expression (`.assuming(..)`).
- **Decisive slice and follow-up (R20, R24, R36).** The checker runs as wasm in a wazero Go host,
  which decides the Rust verdict. Lifting, subset warnings and TLC agreement move to fn-212;
  `#[machine]` stays here as registration only, without token recording or subset warnings.
- **Feedback loop (R30).** The warm soft targets are 10 s for the inner loop (edit one crate,
  rebuild, run its fixtures) and 30 s for the full local gate (baseline 0.8 s and 0.7 s). `make model-time`
  prints each step and notes overruns; every task records the result. Small isolated overruns do
  not block completion; investigate substantial or repeated overruns. Slow tools (JVM, network) run batched or behind
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

All implementation tasks are complete. DESIGN records the Rust recommendation, final measurements,
parameterized-Model readiness and the historical options comparison; `options/` is removed.
Cross-task spec review and final quality audits remain before the local spec closure.

### fn-214: Progress vocabulary on Kettle: scopes, bounds, fairness and verdicts (model)

Captured on 2026-10-19, after fn-210.19. Kettle parity with its archived test's progress questions:
`Assumption` (renamed from `Scope`; `#[assumption]` methods, `.assuming(..)`) gains `only` and `fair` beside `without`; fairness is explicit, never implied by
`#[internal]` (owner); `Progress::within(n)` bounds a claim; verdicts are named (deadlock, cycle,
missed deadline, verified with longest, unknown) with Scala's hint lines. Kettle's goldens do not
change. [Spec](.flow/specs/fn-214-progress-vocabulary-on-kettle-scopes.md).

| Name | Status | What |
| --- | --- | --- |
| fn-214.1 | ⬜ todo | `Assumption` with `only`/`fair`; fairness from the assumption only (R1) |
| fn-214.2 | ⬜ todo | `.within(n)`; named verdicts and hints in the checker (R2, R3) |
| fn-214.3 | ⬜ todo | After .1 and .2: Kettle's progress questions as fixtures; DESIGN.md fairness and `WF_vars` contract (R4, R5) |

### fn-216: Mini activity vertical slice: the Model judges and answers one real SDK run (model, testpilot)

Captured on 2026-10-19, after fn-210.7 and in parallel with the Kettle-only DSL specs. The thinnest
real Temporal slice, on a new Temporal side of the workspace, `model/temporal/` (nothing Temporal
ever enters `model/umpire`; a gate check enforces it): the product-level mini activity Model in Rust (one
enum state, six actions, one safety property) with its `temporal.rs` realization, two Model-interface methods (`judge` an observation by replay, `answer` an SDK callback; the proto
becomes `umpire.model.v1`) with a Temporal extension of three envelopes over the Temporal API itself (`temporal.api.*` from `proto/api.binpb`), and one Go test against an in-process server where a real Go SDK
worker's activity body returns what the Model answers and every `DescribeActivityExecution`
observation is judged by the Model; scenarios completed and failed. No Case, Program, Contract or
CEL; nothing from Testpilot's proto is used (R33 amended). The write-up feeds the Case slice.
[Spec](.flow/specs/fn-216-mini-activity-vertical-slice-the-model.md).

| Name | Status | What |
| --- | --- | --- |
| fn-216.1 | ⬜ todo | Mini activity Model and `temporal.rs` under `model/temporal/features/`, goldens; generic `Realize` trait in umpire; DSL gaps noted (R1) |
| fn-216.2 | ⬜ todo | After .1: proto renamed `umpire.model.v1` with `Any` payloads; `umpire.temporal.v1` under `model/temporal/proto`: three envelopes over `temporal.api` messages; `umpire-wasm` as a library, Temporal guest in `model/temporal/wasm`; Go host; no-Temporal gate check (R2, R5) |
| fn-216.3 | ⬜ todo | After .2: Go test, in-process server, SDK worker answering from the Model, observations judged; completed and failed (R3) |
| fn-216.4 | ⬜ todo | After .3: DESIGN.md "Mini activity slice" findings for the Case slice (R4) |

### fn-213: ModelSuite in Rust: examples and explorations (model)

Captured on 2026-10-18, after fn-210. Every shared fixture becomes `{model.rs, tests.rs, expected/}`:
`model.rs` is only the Model, `tests.rs` beside it holds `mod examples` (authored paths: `Run::from`,
`step`, `repeat`, `expect!`) and `mod explorations` (`Space`, `verify_all`, `refuted`, `find`), as
plain `#[test]` functions run by `cargo test`; the golden hook moves there too and state fields are
`pub`. Conformance, Cases and `sample` wait for the Case slice (owner, 2026-10-18). Kettle's
`tests.rs` asks the questions of the archived `Kettle.test.scala` that the surface admits.
[Spec](.flow/specs/fn-213-modelsuite-in-rust-examples-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-213.1 | ⬜ todo | Fixture layout: `tests.rs` beside `model.rs`, `build.rs` declares both, golden hook moves, `pub` fields (R1) |
| fn-213.2 | ⬜ todo | After .1 and fn-215: `umpire-suite` examples: `Run`, `step`, `repeat`, `Run::along(path!)`, `expect!`, refusal fixtures (R2, fn-215 R4) |
| fn-213.3 | ⬜ todo | After .1 and fn-215: `umpire-suite` explorations: `Space`, `verify_all`, `refuted`, `find` through `path!` waypoints, fixtures (R3, fn-215 R4) |
| fn-213.4 | ⬜ todo | After .2, .3 and fn-214: Kettle's `tests.rs` with the Scala suite's questions; gate and model-time (R4, R5) |

### fn-215: The path! pattern: progressions for claims, witnesses and examples (model)

Captured on 2026-10-19, after fn-214. A list with gaps is a pattern over paths:
`path![self.heating_on, .., self.beeping]` (`..` = any steps, `..=n` = at most `n`). The question
decides the reading: `#[progress]` (every admitted path; exactly two waypoints, more are refused),
`find` (some path, shortest leg by leg) and `Run::along` (a milestone story the checker fills in),
the latter two delivered by fn-213.3/.2. Kettle's goldens do not change.
[Spec](.flow/specs/fn-215-the-path-pattern-progressions-for.md).

| Name | Status | What |
| --- | --- | --- |
| fn-215.1 | ⬜ todo | `Path`/`path!` with refusals; `#[progress]` refuses >2 waypoints; Kettle in `path!` form; reading rules (R1, R2, R3) |

### fn-212: TLA+ lifting and the TLC backend for Umpire in Rust (model)

After fn-210, first decide whether the independent reading and temporal checks justify the
lifting subset, emitter and JVM gate. On approval, lift shared Models, run pinned TLC and compare
reachable states and verdicts through the shared fixture consumer.
[Spec](.flow/specs/fn-212-tla-lifting-and-the-tlc-backend-for.md).

| Name | Status | What |
| --- | --- | --- |
| fn-212.1 | ⬜ todo | Decide cost, benefit and lifting source; on no, close fn-212 and fn-211 with the note |
| fn-212.2 | ⬜ todo | On yes: lift, emit TLA+, run TLC and require agreement on shared fixtures |

### fn-218: Test-space patterns: cover and the full path! grammar (model)

Captured on 2026-10-19, after fn-215 and fn-213. Tests describe their space in Rust's own pattern
notation: actions, input ranges, outcomes, faults, alternatives `(a | b)`, repetition `[x; 0..=n]`,
permutations `unordered![..]`, exclusion gaps `..!(x)`. `cover(&space, Cases(n), ..)` requires the author's expected
case count, finds one shortest witness per alternative and fails on any alternative without a path (no silent vacuity); the expansion is a
golden a reviewer reads as the list of cases; a coverage table shows what the tests touched against
the Model's catalog. The claim form stays two state waypoints.
[Spec](.flow/specs/fn-218-test-space-patterns-cover-and-the-full.md).

| Name | Status | What |
| --- | --- | --- |
| fn-218.1 | ⬜ todo | Full `path!` grammar and alternative enumeration; claims refuse it (R1) |
| fn-218.2 | ⬜ todo | After .1 and fn-213.3: `cover`/`sample` with expansion and coverage goldens (R2, R3, R5) |
| fn-218.3 | ⬜ todo | After .2: Kettle and mini activity `cover` tests; README grammar (R3, R4) |

### fn-217: Annotation catalog over the Temporal API: kinds, tags, rules, expected errors and gap analysis (model, temporal)

Captured on 2026-10-19, after fn-216. A typed sidecar catalog over the upstream `temporal.api`
messages (one kind per field, any tags with consequences, JSON Schema presence rules, AIP
`field_behavior` roles, protovalidate constraint names), exhaustive by construction at four levels
(fields, enum values, RPC methods, Model actions) so an `api.binpb` bump fails the build until
classified; every request-side rule maps to the server's expected error and a probe produces
`gaps.txt` (`Confirmed`/`Mismatch`/`Accepted`/`Unverifiable`) plus reverse coverage of the server's
validators. Generic vocabulary in `umpire-annotations`, Temporal tags in `model/temporal`.
[Spec](.flow/specs/fn-217-annotation-catalog-over-the-temporal.md).

| Name | Status | What |
| --- | --- | --- |
| fn-217.1 | ⬜ todo | `umpire-annotations`: kinds, tags, rules, expectations, `validate`, `fill` (R2, R6) |
| fn-217.2 | ⬜ todo | After .1: `annotate!` with four-level exhaustiveness and the descriptor-set build check (R1, R2) |
| fn-217.3 | ⬜ todo | After .2: mini activity catalog with expected errors; `fill`/`validate` in the slice test (R3, R5) |
| fn-217.4 | ⬜ todo | After .3: gap probe and reverse coverage; `gaps.txt` golden (R3, R4) |
| fn-217.5 | ⬜ todo | After .4: DESIGN.md write-up (R5) |

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
