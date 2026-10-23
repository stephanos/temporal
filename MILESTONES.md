# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-23.

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
passed all twelve stock/Temporal byte comparisons and both platforms' host suites. fn-239 renewed
that certification after fn-238.8's guest identity changes at `a96a4c6550`, with all twelve
comparisons and both host suites passing on each OS.
[Renewed certification evidence](.flow/artifacts/fn-239-renew-the-linuxmacos-guest-byte/certification-a96a4c6550/proof.md).
fn-233 is complete locally with all four tasks, three-draw task reviews, parent completion
review and correctness/standards audits. The independent Kettle service exercises shared
catalogs, envelopes, annotations, live Judge/Answer and four differential mutants with
unchanged canonical Models and semantic goldens. Full gates passed; the warm gate took
364.531s against its 60s soft target and the inner loop took 6.066s against 10s.
TLC checks its admitted source subset, including the Phased/reject activity Model
with retries; unsupported source is explicitly refused.
fn-227 is complete locally: all 13 tasks and the parent completion review passed. Declared
Worlds, finite inputs, activity retries, retained-lifecycle refinement and actual SDK retry
proof are delivered. Final required gate passed in 582.209s against its 60s soft target;
inner loop 6.112s met 10s. Timeout and cooperative-cancellation work is scheduled in fn-237 after fn-238.

Umpire in Rust lives in `temporal/model`, a Cargo workspace pinned through `mise.toml`
(`rust = "1.99.1"`), with crates under `model/umpire/`: `dsl` (with `dsl/macros`), `checker`,
`testing` (the dev-only fixture runner), `fixtures` (the shared end-to-end fixtures), `runtime`
(the generated-protobuf Model runtime), and `lift` plus `backends/tlc` (the independent source backend);
the `samples` crate has been removed. The original Rust verdict was scoped to what
`model/archived/framework/samples/Kettle.scala` uses. The verdict is one rule (R24): if the checker
runs as wasm in a Go host through wazero (pure Go, no cgo) on the stock toolchain, with Kettle's report matching
the Rust goldens byte for byte, Rust wins. Everything else in `model/DESIGN.md` is information. fn-196.8, the Scala checker speed fix, stays
deferred, since speed no longer decides. The decisive slice passed in fn-210.9: the recommendation
is Rust. The guest now directly exports `umpire_buffer_alloc`, `umpire_buffer_release` and
`umpire_call`, with no custom linker or export rewriting (fn-210.16).

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| After the Rust verdict | fn-217 → fn-230 → fn-231 → fn-227 → fn-211 → fn-228 → fn-238 → fn-237 | Owner priority order (fn-231 after fn-217 and fn-229, before fn-228.2 if possible; fn-229 is complete; fn-230 has no dependency and must land before fn-228.4; fn-227 placed before fn-211 on 2026-10-20 so Apalache arrives to a parameterized Model; fn-228 depends on fn-227 only, its place relative to fn-211 is the owner's). fn-233 (Kettle service, filed 2026-10-21) depends on fn-217 alone: on 2026-10-22 the owner dropped its fn-227 dependency, since its task .1 needs only fn-227.12's request evidence, which is complete. fn-233 is complete. Owner-approved remaining order is fn-228 → fn-238 → fn-237; fn-211 is deferred at the owner’s request on 2026-10-22. On 2026-10-23 the owner reactivated fn-238.5/.6 and all five fn-237 tasks at the end of that queue; their existing acceptance and measurement gates remain in force. fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-218, fn-221, fn-223, fn-224, fn-225, fn-226 and fn-229 are complete locally with their scoped reviews and gates passed; fn-225 and fn-226's single-task reviews cover their full specs. fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The former "Case slice" is filed as fn-228 (Runner and Run): the Model as umpire, one Model per Run, Propose on the protocol, one Go host loop, Driver and Shim, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

## Specs

### fn-228: Runner and Run: the one Go host loop over the Model, Propose on the protocol, and the Testpilot shrink (model, temporal)

Captured on 2026-10-20, after fn-227. Files the post-verdict obligations R33 to R40 as one spec under
the owner's direction that Go is dumb and the Model holds the smarts, and retires the "Case slice"
name. Names settled: the renamed crates are `umpire-runtime` and `umpire-fixtures`,
a Realization connects a Model to a system; new `Propose` on `umpire.model.v1`; Go package `runner`
with `Runner`, `Run`, `Driver` and `Shim`. The Model is the umpire; the Runtime hosts one Model; a
Realization binds it to one system; the Runner drives a Driver, asks the Model to Propose, Judge
and Answer, and records a Run. Deletes verification, evaluation, CEL, contract, casefile, campaign
and replay; one `umpire` command with five modes. The reviewed contract includes Start bootstrap,
incremental correlated Driver events, Model-owned polling completion, ordered protocol replay
including Answer, and immutable publication by model/seed/content hash.
The rename checkpoint (.1), Propose (.2) and the Go `runner` package (.3) are complete locally.
Propose is on the protocol, Runtime, `Realize` and Go `Host`. The Runner drives an incremental
Driver exchange, serializes every Host call, records a `umpire.run.v1` Run, replays it offline and
publishes it never-overwrite; one `umpire` command with five modes replaced run, fuzz, replay, assess
and repeat. .3 passed three review rounds; required gate 872.085s exceeds the 60s soft target, inner
loop 6.687s meets 10s. Four live Testpilot failures pre-date .3 and are carried into .6 with the
canary and publish retirements. The Temporal Driver (.4) is next.
[Spec](.flow/specs/fn-228-runner-and-run-the-one-go-host-loop.md).

| Name | Status | What |
| --- | --- | --- |
| fn-228.4 | 🔄 in progress | After .3: Temporal Driver as act-and-observe; live slice through `umpire explore` (R4) |
| fn-228.5 | ⬜ todo | After .4: Shim endpoint for non-Go SDK workers (R5) |
| fn-228.6 | ⬜ todo | After .4 and .5: delete and shrink Testpilot packages with replacement coverage; include relocated Runner code in counts (R6) |
| fn-228.7 | ⬜ todo | After .5 and .6: DESIGN.md Runner and Run write-up; R33 to R40 discharged (R7) |

### fn-241: One WorkflowService classification for all Temporal features (model, temporal)

Filed on 2026-10-23 at the owner's request. `classify` in `model/temporal/feature/activity/api.rs`
is an exhaustive match over every WorkflowService RPC, but only eight arms are activity-specific;
the rest classify RPCs by resource family. Splits it into one service table outside `feature/`,
which owns the exhaustiveness check, and per-feature claim lists, so a second Temporal feature
does not copy the table. Goldens stay byte-identical except out-of-scope reason text. Runs after
fn-228.4, which edits the same realization files, and before fn-228.5.
[Spec](.flow/specs/fn-241-one-workflowservice-classification-for.md).

| Name | Status | What |
| --- | --- | --- |
| fn-241.1 | ⬜ todo | After fn-228.4: one service table plus per-feature claims; duplicate-claim and unclassified-RPC controls (R1, R2) |

### fn-238: Shared front end: one Model reading for the native checker and the lifter (model)

Filed on 2026-10-21 at the owner's direction: both checkers share all code except the back end. The
native checker executes the Model and the lifter emits TLA+ (and later Apalache); the owner accepts
the risk of a shared misreading. Typed counterexample replay stays as the cross-check. Runs after
fn-227.14 and .8, before fn-227.11, so later constructs are built once.
The design ([shared-front-end.md](model/design/shared-front-end.md)) measured about 1,000 duplicated
lines per side and recommends one shared admission crate first (Option A, .2-.4), then the
interpreted checker behind a measured gate (Option B, .5-.6); the guest and replay stay compiled.
.2 is complete locally: `umpire-front` owns admission, goldens and IR unchanged; timing was
measured under concurrent load (119.6s); idle re-measurements since are 50-54s.
.8 is complete locally: guest identities hash only each crate's lockfile closure, so unrelated
workspace crates no longer stale the guest; timing 50.199s gate, 6.128s inner loop.
.3 is complete locally: the macros read Models only through `umpire-front` (dsl/macros
3,306 -> 2,153 lines). Net size was flat because most measured lifter "duplication" was lowering
rustc does natively; .4's deletion of the source re-read and re-validation is where lines drop.
.4's review found Option A met its structure (one reading, one diagnostic, cheap new back ends)
but not its size (about +336 lines against -850 to -1,600 promised), and that it halves cost only
for declaration-level constructs. Option B adds work for body-level constructs; the owner
reactivated .5 and .6 on 2026-10-23, scheduled after fn-228 with their measured gates intact.
.4 is complete locally: the macro embeds the admitted Model and the lifter emits from it, with
no `model.rs` re-read; core crates 8,297 lines against 8,190 before .2; timing 47.6s gate, 7.3s inner loop.
.7 recorded the boundary and measurements in [DESIGN.md](model/DESIGN.md#front-end-and-back-ends-fn-238).
The approved IR token-printer cleanup (.9) is complete locally before fn-227.16 adds IR
constructs. All 782 golden/snapshot/refusal files and the 26-Model IR/token comparisons
are exact; slice, gaps and required-TLC gates passed, and all three review draws returned
SHIP. Production Rust shrank 60 lines against the roughly 190-line estimate. Required gate
532.677s exceeded its 60s soft target; inner loop 9.272s met 10s. Only .5 and .6 remain open, queued after fn-228.
[Spec](.flow/specs/fn-238-shared-front-end-one-model-reading-for.md).

| Name | Status | What |
| --- | --- | --- |
| fn-238.5 | ⬜ todo | After fn-228: interpreted checker as a differential check (R2, R3) |
| fn-238.6 | ⬜ todo | After .5: interpreted checker by default behind the 2x / 25% gate (R2, R3) |

### fn-237: Activity timeouts, heartbeat and cooperative cancellation (model, temporal)

Split from fn-227 on 2026-10-21 by owner decision; depends on fn-227. Carries the four optional
timeouts, Heartbeat, Tick and cooperative cancellation with their native and source support
(optional handler facts, bounded optional values), the final one-tick compatibility refinement,
live timeout admission and cancellation evidence, and their write-up. Reactivated by the owner
on 2026-10-23, scheduled after fn-238.
[Spec](.flow/specs/fn-237-activity-timeouts-heartbeat-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-237.1 | ⬜ todo | Optional handler facts through native adapters and independent TLC replay (R1) |
| fn-237.2 | ⬜ todo | Bounded nominal optional values through source binding, emission and decode (R1) |
| fn-237.3 | ⬜ todo | After .1, .2 and fn-227.3: deadlines, Tick, Heartbeat, cooperative cancel, final compatibility refinement (R2, R3) |
| fn-237.4 | ⬜ todo | After .3: live timeout admission and cooperative cancellation evidence (R4) |
| fn-237.5 | ⬜ todo | After .4: DESIGN.md timeout and cancellation write-up (R5) |

### fn-211: Apalache backend for Umpire in Rust (model) — deferred

Captured on 2026-10-18 as a follow-up, after fn-210, fn-212 and (since 2026-10-20) fn-227, and only if their verdicts are to continue in
Rust. Adds Apalache as a second TLA+ backend over the same lifted tree and emitter: Snowcat type
annotations in the emitter, a pinned Apalache release run on the mise JDK with one bounded check
per invariant at the native depth plus one, ITF counterexamples decoded through typed source values and
replayed through the Model, and an Apalache consumer selected by `expected/apalache.txt`. No
liveness through Apalache; TLC owns progress claims. Since 2026-10-21 it also checks symbolic Worlds: every World in a
declared finite domain in one run per property (owner request, toward full Quint parity).
Task .1 is complete locally: typed safety emission, ordered checked arithmetic and one-step
history passed implementation review, both audits, actual pinned-tool controls and fresh live
slice/gap proof. The required full gate passed in 562.098s against its 60s soft target;
inner loop 7.893s met 10s. Task .2 is complete locally: pinned bounded checking, native-derived
Kettle bound 63, installer admission and shared process ownership passed review, both audits and
all 17 required gate stages. Gate 903.818s exceeded its 60s soft target; inner loop 7.229s met 10s.
Task .3 is complete locally: strict typed ITF replay, selected property and arithmetic refusal validation, legacy and native Vector projections passed review and both audits. Fresh guest/slice/gaps proof passed with guest `d96f123b6ec669e7385e9fff75f711f2d20e0f15a0d918928238f7e263221217`. All 17 required gate stages passed: gate 824.431s exceeded 60s; inner loop 7.632s met 10s. The first full run failed with its causal output discarded; its cause remains unknown. A reviewed timing-wrapper correction now preserves failed child diagnostics; the logged workspace diagnostic and final retry passed. Fixture integration and symbolic Worlds are deferred at the owner’s request on 2026-10-22. The unfinished .4 consumer is preserved on local branch `work/fn2114-deferred` at `fee5fc56ae`, including its tracked handoff; no completion review or final gate is claimed. Execution moves to fn-228.
[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.4 | ⏸️ deferred | After .3: Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |
| fn-211.5 | ⏸️ deferred | After .3: symbolic World constants via `ConstInit`, one run per property across the declared World domain, replayed under the decoded World (R6) |
