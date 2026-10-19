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

Execution resumed on 2026-10-19 at the owner's request: complete the remaining specs in
priority order, following the root agent's own recommendations where a decision is open.
fn-216, fn-221 and fn-212 are complete locally, including their completion reviews. TLC is implemented for its admitted source subset; unsupported source,
including the current Phased/reject activity Model, is explicitly refused.

Umpire in Rust lives in `temporal/model`, a Cargo workspace pinned through `mise.toml`
(`rust = "1.99.1"`), with crates under `model/umpire/`: `dsl` (with `dsl/macros`), `checker`,
`testing` (the dev-only fixture runner), `tests` (the shared end-to-end fixtures), `wasm`
(the generated-protobuf guest), and `lift` plus `backends/tlc` (the independent source backend);
the `samples` crate has been removed. The original Rust verdict was scoped to what
`model/archived/framework/samples/Kettle.scala` uses. The verdict is one rule (R24): if the checker
runs as wasm in a Go host through wazero (pure Go, no cgo) on the stock toolchain, with Kettle's report matching
the Rust goldens byte for byte, Rust wins. Everything else in `model/DESIGN.md` is information. fn-196.8, the Scala checker speed fix, stays
deferred, since speed no longer decides. The decisive slice passed in fn-210.9: the recommendation
is Rust. The guest now directly exports `umpire_buffer_alloc`, `umpire_buffer_release` and
`umpire_call`, with no custom linker or export rewriting (fn-210.16).

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| After the Rust verdict (in progress) | fn-213 → fn-215 → fn-223 → fn-224 → fn-218 → fn-217 → fn-211 | Owner priority order; fn-223 and fn-224 are placed by the root agent. fn-210, fn-212, fn-214, fn-215, fn-216 and fn-221 are complete locally with their reviews and scoped gates passed; fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The Case slice keeps the Model as oracle, one Model per run, thin SDK shims, one host loop, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

## Specs

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
| fn-213.2 | ⬜ todo | After .1 and fn-215: `umpire-suite` examples: `Run`, `step`, `repeat`, `Run::along(path!)`, `expect!`, refusal fixtures (R2, fn-215 R4) |
| fn-213.3 | ⬜ todo | After .1 and fn-215: `umpire-suite` explorations: `Space`, `verify_all`, `refuted`, `find` through `path!` waypoints, fixtures (R3, fn-215 R4) |
| fn-213.4 | ⬜ todo | After .2, .3 and fn-214: Kettle's `tests.rs` with the Scala suite's questions; gate and model-time (R4, R5) |

### fn-223: One lifted source form: the lifter reads attribute Models only (model)

Captured on 2026-10-19 at the owner's request. About 2,000 of `umpire-lift`'s 5,000 source lines
lower the hand-written `Machine` trait form, used only by eight shared test fixtures. Each fixture
is rewritten as a `#[machine]` Model with unchanged behaviour goldens, or moved checker-only with
its surface gap recorded; then the trait lowering is deleted and hand-written source is refused
by location. The `Machine` trait itself stays. Placed after fn-215 and before fn-213.2 so later
suite, cover and Apalache work builds on one front end.
[Spec](.flow/specs/fn-223-one-lifted-source-form-the-lifter-reads.md).

| Name | Status | What |
| --- | --- | --- |

### fn-224: Accepted self-loops in the attribute surface (model)

Captured on 2026-10-19 by the root agent after fn-223 moved thirteen trait-form fixtures
checker-only. A handler is enabled only when it changes state, so an enabled action that leaves
the state unchanged (heartbeats, idempotent requests) has no spelling except a `reject!` self-loop.
One marker beside `reject!` makes it an accepted self-loop; the lifter admits it through the
existing self-loop emission, and the fixtures whose only gap was the self-loop return to TLC.
Choice, typed outcomes and facts, and computed starts stay gaps until a Temporal Model needs them.
[Spec](.flow/specs/fn-224-accepted-self-loops-in-the-attribute.md).

| Name | Status | What |
| --- | --- | --- |
| fn-224.1 | ⬜ todo | After fn-213 and fn-223: self-loop marker, lifter admission, fixtures back on TLC (R1, R2) |

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
