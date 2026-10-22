# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-21.

## Keeping this page current

- This page describes the present. Rewrite a status in place; do not append dated entries.
- List each open spec's remaining tasks in a `Name | Status | What` table: the task ID, its status
  (🔄 in progress, ⬜ todo, ⛔ blocked, ⏸️ deferred) and a brief description. Remove a task's row once it is done,
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

Execution resumed under the owner's active milestone-completion goal (2026-10-20), renewed on
2026-10-21; tasks run one at a time so each gate is attributable.
fn-213, fn-215, fn-223 and fn-224 are complete locally with their completion reviews.
fn-225 is complete locally: its single task passed implementation review, correctness and
standards audits, and default/required-TLC gates with byte-identical goldens. fn-226 is also
complete locally with all three search requirements covered, reviews passed and model gates green.
fn-218 is complete locally with all five requirements covered, real-model expansions,
default/required-TLC gates and the completion review passed.
fn-229 is complete locally with both tasks, all three requirements, implementation
and completion reviews, correctness/standards audits and integrated/live gates passed.
fn-217 is complete locally with all seven tasks, implementation and parent completion
reviews passed, exhaustive admission, black-box gaps and complete live/privacy gates.
fn-230 is complete locally with both tasks and parent completion review passed,
the embedded guest, freshness controls and renewed two-checkout Linux proof.
fn-231 is complete locally with both tasks, implementation and parent completion
reviews passed, all three conductor gates and unchanged live gap/privacy proof.
Its final timing gate took 87.140s, above the 30s soft target; inner loop was 1.672s.
fn-236's permanent guest compiler-input fix and fn-232's Linux/macOS certification are
complete locally. Four clean clones at `3c314fb5b85427471a1525d6f190fe5d1fb9f3b2`
passed all twelve stock/Temporal byte comparisons and both platforms' host suites. fn-238.8 later
changed guest identity inputs, so that certification needs renewal (fn-239).
[Certification evidence](.flow/artifacts/fn-232-linux-macos-guest-byte-identity/certification-3c314fb5b8/proof.md).
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
| After the Rust verdict | fn-217 → fn-230 → fn-231 → fn-227 → fn-211 → fn-228 | Owner priority order (fn-231 after fn-217 and fn-229, before fn-228.2 if possible; fn-229 is complete; fn-230 has no dependency and must land before fn-228.4; fn-227 placed before fn-211 on 2026-10-20 so Apalache arrives to a parameterized Model; fn-228 depends on fn-227 only, its place relative to fn-211 is the owner's). fn-233 (Kettle service, filed 2026-10-21) depends on fn-217 and fn-227 (preserving request evidence during envelope extraction); its place relative to the other successors is the owner's. fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-218, fn-221, fn-223, fn-224, fn-225, fn-226 and fn-229 are complete locally with their scoped reviews and gates passed; fn-225 and fn-226's single-task reviews cover their full specs. fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The former "Case slice" is filed as fn-228 (Runner and Run): the Model as umpire, one Model per Run, Propose on the protocol, one Go host loop, Driver and Shim, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

## Specs

### fn-233: Kettle service: a framework-owned end-to-end reference system (model, umpire)

Captured on 2026-10-21, after fn-217. The only end-to-end chain (proto, catalog, realization, guest,
Go host, live system, Judge, Answer, Annotate, gap report) runs against Temporal;
under `umpire/` the pieces exist only in isolation. This spec gives the canonical Kettle Model a
framework-owned Kettle service: an awkward-on-purpose proto, a Go reference server with named
validators and scoped limits, a `derive(Catalog)` catalog, a `Realize` binding and guest, a
live run through the Go host, and seeded server mutants the framework must name. Its first task
moves the envelopes and descriptor coverage check out of `temporal/`, measuring what is
generic today; validator generation and Go source walking are outside the updated scope.
Spec-level dependencies on fn-217 and fn-227 serialize envelope extraction after actual-request
evidence. Kettle is the already-shipped derive's second consumer (R5), not a retroactive first spike.
The reviewed plan covers generic catalog/code-generation extraction, selected live initialization
without changing the canonical Model, and differential healthy-versus-mutant detection.
[Spec](.flow/specs/fn-233-kettle-service-a-framework-owned-end-to.md).

| Name | Status | What |
| --- | --- | --- |
| fn-233.1 | ⬜ todo | Extract generic catalog tooling and envelopes; preserve Model/semantic goldens and explicitly migrate Any identities (R1) |
| fn-233.2 | ⬜ todo | Kettle service proto, Go reference server, realization, guest, selected-start Judge/Answer live run with exact RPC projections and transcript golden (R2) |
| fn-233.3 | ⬜ todo | Kettle `derive(Catalog)` catalog, `config.rs`, Annotate in the live run, `coverage.snap` and `gaps.txt` (R3, R5) |
| fn-233.4 | ⬜ todo | Seeded mutants, one-line switches, each named by the framework in `mutants.snap.txt` (R4) |

### fn-227: Parameterized Models: declared worlds, action inputs, and the activity Model with retries (model, temporal)

Captured on 2026-10-20, after fn-218 and fn-217 and before fn-211. Closes the DSL gap DESIGN.md's
readiness table leaves open: `#[derive(World)]` with `&World` access on handlers and claims, a
`#[starts]` block for computed starts, `Finite` and action inputs with `#[domain]`, and lifting of
worlds as `CONSTANT`s and inputs as finite choice. Delivered through the activity Model extended
in place: `maximum_attempts` and `RespondFailed { retryable }`, then the four optional timeouts,
`Heartbeat`, `Tick` and cooperative cancel with progress and bound claims; a one-attempt
compatibility World proves lifecycle refinement against the retained six-phase graph.
The realization reads the world from the observed start
request and the live slice proves one retry then completion. The lifter admits `impl Phased` and
`reject!` so TLC checks the activity Model.
The native World checkpoint is complete locally with SHIP review, all three final gates
and unchanged original artifacts. Its final timing was 93.288s against the 30s soft target,
with a 5.035s inner loop. The revised parent plan passed SHIP review.
Source/TLC World binding (.7) is complete locally: one owned World and one start enumeration
feed binding, native checking and same-World TLC replay across all five CFG families, with the
new `parameterized_world` fixture and unchanged baseline goldens and guest. Its second review
passed after World integers outside TLC's `.cfg` range became located refusals. Final timing
was 52.327s against the 30s soft target, with a 7.362s inner loop.
Native `Finite`, World-dependent `#[domain]` inputs and checked catalog admission (.2) are
complete locally with SHIP review, required/timing gates and the refreshed guest's live slice
and gap proof; timing 56.276s gate, 9.852s inner loop (just under its 10s target).
On 2026-10-21 the owner paused fn-227 during .8's review; its successors fn-233, fn-211
and fn-228 wait with it.
The same day the owner moved the timeout semantics (old R5: facts .9, optional values .10,
deadlines/Tick/cooperative cancel .4, plus the cooperative-cancellation part of .5 and the final
one-tick compatibility re-run) to fn-237. fn-227 keeps R3 in full, including .11, so TLC checks
the real activity Model; the owner holds TLC and Apalache scaling as crucial.
After a complexity audit and a simplification scout, the owner approved .13 and .14
(about 1,200 source and 850 test lines removed, goldens and TLC unchanged), run before .8's
review: [simplification report](.flow/artifacts/fn-227-parameterized-models-declared-worlds/simplify-report.md).
.13 is complete locally with SHIP review and full gates: one machine generator, no module
generics, method-level `#[starts]`; net -788 source and about -655 test lines, goldens and TLC
unchanged; timing 51.337s gate, 8.868s inner loop.
.14 is complete locally after a second review: derives export values to the lifter, replacing
the augmented-copy binding route; net about -91 source and -70 test lines, goldens and TLC
unchanged; timing 50.532s gate, 7.908s inner loop.
.8 (source inputs and exact-payload fairness) is complete locally with SHIP review at HEAD after
the .13/.14 migration; its minor follow-ups moved to fn-238.3.
.12 (actual request at Observation tag 4, owned same-invocation capture) is complete locally,
merged from its worktree with the guest rebuilt; timing 53.993s gate, 7.088s inner loop.
On 2026-10-21 the owner approved all three contract amendments: lifecycle refinement
in place of impossible old-byte equality; cooperative cancellation with explicit protocol
evidence; and optional actual-request Observation evidence at tag 4. Heartbeat signals a
pending request without closing, and completion/failure races remain. Preserve old lifecycle
evidence and prove the stated projection, including the final one-tick compatibility World.
The twelve-task decomposition separates native/source inputs, facts, optional ticks,
phase/rejection lifting and request capture from their activity/live consumers.
The review passed; each consumer still waits for its declared support dependencies.
[Spec](.flow/specs/fn-227-parameterized-models-declared-worlds.md).

| Name | Status | What |
| --- | --- | --- |
| fn-227.11 | ⬜ todo | After fn-238.4: source `Phased`/`reject!` and closure laws, built once on the shared front end, with unchanged activity behavior (R3) |
| fn-227.3 | ⬜ todo | After .2 and .11: activity attempts/retryable failure and retained-lifecycle refinement (R3, R4) |
| fn-227.5 | ⬜ todo | After .3 and .12: transactional live World admission (`maximum_attempts`) and SDK retry proof (R6) |
| fn-227.6 | ⬜ todo | After .5: DESIGN.md parameterized worlds, inputs and retries write-up and readiness table closure (R7) |

### fn-238: Shared front end: one Model reading for the native checker and the lifter (model)

Filed on 2026-10-21 at the owner's direction: both checkers share all code except the back end. The
native checker executes the Model and the lifter emits TLA+ (and later Apalache); the owner accepts
the risk of a shared misreading. Typed counterexample replay stays as the cross-check. Runs after
fn-227.14 and .8, before fn-227.11, so later constructs are built once.
The design ([shared-front-end.md](model/design/shared-front-end.md)) measured about 1,000 duplicated
lines per side and recommends one shared admission crate first (Option A, .2-.4), then the
interpreted checker behind a measured gate (Option B, .5-.6); the guest and replay stay compiled.
.2 is complete locally: `umpire-front` owns admission, goldens and IR unchanged; timing was
measured under concurrent load (119.6s) and is re-measured at the next idle gate.
[Spec](.flow/specs/fn-238-shared-front-end-one-model-reading-for.md).

| Name | Status | What |
| --- | --- | --- |
| fn-238.8 | 🔄 in progress | Guest identity hashes only the guest's dependency closure, so unrelated workspace crates stop staling the guest; before .3 (R3) |
| fn-238.3 | ⬜ todo | After .2: macros read through `umpire-front`; one diagnostic per form; fn-227.8 follow-ups (R2, R3) |
| fn-238.4 | ⬜ todo | After .3: macro embeds the admitted Model; lifter emits from it, no `model.rs` re-read (R2-R4) |
| fn-238.5 | ⬜ todo | After .4: interpreted checker as a differential check (R2, R3) |
| fn-238.6 | ⬜ todo | After .5: interpreted checker by default only if at most 2x slower and at most 25% more per-construct cost (R2, R3) |
| fn-238.7 | ⬜ todo | After .4 (and .6 if taken): boundary write-up and measurements (R4) |

### fn-239: Renew the Linux/macOS guest byte-identity certification (model)

fn-238.8 changed guest identity inputs (per-crate lockfile closure), so fn-232's certification at
`3c314fb5b8` needs renewal at a revision at or after `cbe2c6fcb7`. Standalone, by owner request;
nothing waits on it. Needs a native macOS run on the owner's Mac.
[Spec](.flow/specs/fn-239-renew-the-linuxmacos-guest-byte.md).

| Name | Status | What |
| --- | --- | --- |
| fn-239.1 | ⛔ blocked | Four clean builds on Linux and native macOS, twelve comparisons and both host suites; commands in the task (R1, R2) |

### fn-237: Activity timeouts, heartbeat and cooperative cancellation (model, temporal)

Split from fn-227 on 2026-10-21 by owner decision; depends on fn-227. Carries the four optional
timeouts, Heartbeat, Tick and cooperative cancellation with their native and source support
(optional handler facts, bounded optional values), the final one-tick compatibility refinement,
live timeout admission and cancellation evidence, and their write-up. Not scheduled.
[Spec](.flow/specs/fn-237-activity-timeouts-heartbeat-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-237.1 | ⏸️ deferred | Optional handler facts through native adapters and independent TLC replay (R1) |
| fn-237.2 | ⏸️ deferred | Bounded nominal optional values through source binding, emission and decode (R1) |
| fn-237.3 | ⏸️ deferred | After .1, .2 and fn-227.3: deadlines, Tick, Heartbeat, cooperative cancel, final compatibility refinement (R2, R3) |
| fn-237.4 | ⏸️ deferred | After .3: live timeout admission and cooperative cancellation evidence (R4) |
| fn-237.5 | ⏸️ deferred | After .4: DESIGN.md timeout and cancellation write-up (R5) |

### fn-211: Apalache backend for Umpire in Rust (model)

Captured on 2026-10-18 as a follow-up, after fn-210, fn-212 and (since 2026-10-20) fn-227, and only if their verdicts are to continue in
Rust. Adds Apalache as a second TLA+ backend over the same lifted tree and emitter: Snowcat type
annotations in the emitter, a pinned Apalache release run on the mise JDK with one bounded check
per invariant at the native depth plus one, ITF counterexamples decoded through typed source values and
replayed through the Model, and an Apalache consumer selected by `expected/apalache.txt`. No
liveness through Apalache; TLC owns progress claims. Since 2026-10-21 it also checks symbolic Worlds: every World in a
declared finite domain in one run per property (owner request, toward full Quint parity).
[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.1 | ⬜ todo | Typed Apalache safety relation, checked arithmetic and postcondition history; TLC results unchanged (R1) |
| fn-211.2 | ⬜ todo | After .1: `umpire-apalache` in `umpire/backends/apalache/`, pinned release, bounded check per invariant, deterministic lifecycle (R2) |
| fn-211.3 | ⬜ todo | After .2: ITF counterexamples decoded through typed source values and replayed through the Model (R3) |
| fn-211.4 | ⬜ todo | After .3: Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |
| fn-211.5 | ⬜ todo | After .3: symbolic World constants via `ConstInit`, one run per property across the declared World domain, replayed under the decoded World (R6) |

### fn-228: Runner and Run: the one Go host loop over the Model, Propose on the protocol, and the Testpilot shrink (model, temporal)

Captured on 2026-10-20, after fn-227. Files the post-verdict obligations R33 to R40 as one spec under
the owner's direction that Go is dumb and the Model holds the smarts, and retires the "Case slice"
name. Names settled: `umpire-wasm` becomes `umpire-runtime`, `umpire-tests` becomes `umpire-fixtures`,
Realization is the one word for a binding; new `Propose` on `umpire.model.v1`; Go package `runner`
with `Runner`, `Run`, `Driver` and `Shim`. The Model is the umpire; the Runtime hosts one Model; a
Realization binds it to one system; the Runner drives a Driver, asks the Model to Propose, Judge
and Answer, and records a Run. Deletes verification, evaluation, CEL, contract, casefile, campaign
and replay; one `umpire` command with five modes. The reviewed contract includes Start bootstrap,
incremental correlated Driver events, Model-owned polling completion, ordered protocol replay
including Answer, and immutable publication by model/seed/content hash.
[Spec](.flow/specs/fn-228-runner-and-run-the-one-go-host-loop.md).

| Name | Status | What |
| --- | --- | --- |
| fn-228.1 | ⬜ todo | Renames alone: `umpire-runtime`, `umpire-fixtures`, Realization in prose (R1) |
| fn-228.2 | ⬜ todo | After .1: `Propose` on protocol, Runtime, `Realize`, `Host`; activity Realization proposes (R2) |
| fn-228.3 | ⬜ todo | After .2: Go `runner`: `Runner`, `Run`, `Driver`, five modes, one `umpire` command, publish (R3) |
| fn-228.4 | ⬜ todo | After .3: Temporal Driver as act-and-observe; live slice through `umpire explore` (R4) |
| fn-228.5 | ⬜ todo | After .4: Shim endpoint for non-Go SDK workers (R5) |
| fn-228.6 | ⬜ todo | After .4 and .5: delete and shrink Testpilot packages with replacement coverage; include relocated Runner code in counts (R6) |
| fn-228.7 | ⬜ todo | After .5 and .6: DESIGN.md Runner and Run write-up; R33 to R40 discharged (R7) |
