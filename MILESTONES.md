# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-20.

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

Execution resumed under the owner's active milestone-completion goal (2026-10-20).
fn-213, fn-215, fn-223 and fn-224 are complete locally with their completion reviews.
fn-224 passed fresh default and required-TLC model gates; fn-225 is in progress.
TLC is implemented for its admitted source subset; unsupported source,
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
| After the Rust verdict | fn-225 → fn-226 → fn-218 → fn-217 → fn-211 | Owner priority order; fn-225 and fn-226 are placed by the root agent. fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-221, fn-223 and fn-224 are complete locally with their reviews and scoped gates passed; fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The Case slice keeps the Model as oracle, one Model per run, thin SDK shims, one host loop, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

## Specs

### fn-225: umpire-lift: separate Model shape from body lowering (model)

Captured on 2026-10-19 at the owner's request. A pure refactor of `umpire-lift`: a shape module
reads the `#[machine]` module's declarations and never lowers statements or expressions; a body
module lowers expressions, effects, vectors and claim bodies and never walks items; `tree.rs`
splits into declarations and expressions. Every golden stays byte-identical. Not two crates; a
shape parser shared with `#[machine]` waits until the two actually drift.
[Spec](.flow/specs/fn-225-umpire-lift-separate-model-shape-from.md).

| Name | Status | What |
| --- | --- | --- |
| fn-225.1 | 🔄 in progress | Shape/body split and tree split, goldens byte-identical (R1) |

### fn-226: Suite search follow-ups (model)

Captured on 2026-10-20 from the re-review of fn-213's complete waypoint search (verdict ship, no
correctness defects). A panic inside a waypoint predicate is blamed on a Model action; the search
budget counts (state, progress) pairs, so large bounded patterns turn unknown; any skipped step
panic refuses an answer even when it cannot affect it. Placed before fn-218, whose `cover` reuses
the search. [Spec](.flow/specs/fn-226-suite-search-follow-ups-model.md).

| Name | Status | What |
| --- | --- | --- |
| fn-226.1 | ⬜ todo | Predicate panics, Model-state budget, panic relevance, cosmetics (R1, R2, R3) |

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
