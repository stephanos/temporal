# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-24.

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
The Scala-era specs (fn-119 to fn-209: 58 specs, 183 tasks) were closed in Flow on 2026-10-24 at the
owner's request, unimplemented; the archived tree stays the reference. The Rust follow-ups below use the owner's priority order; their individual
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
fn-238 is complete locally: one shared front end, and the interpreter as a differential check
over every admitted Model the gate reaches. .6 kept the compiled checker as the default because
its gate was not met (Kettle upper quartile 2.01x, sets unmeasurable); the reviewed switch is kept
on `work/fn2386-interpreted-switch`. Its required gate passed on the merged tip at `d23625a390`
(572.0s gate, 8.9s inner loop, quiet machine).

Umpire in Rust lives in `temporal/model`, a Cargo workspace pinned through `mise.toml`
(`rust = "1.99.1"`). Framework crates under `model/umpire/` are grouped by authoring, front end and IR,
back ends, protocol, catalog, hosts and development fixtures; [the crate map](model/README.md) names
each package and its role. The original Rust verdict was scoped to what
`model/archived/framework/samples/Kettle.scala` uses. The verdict is one rule (R24): if the checker
runs as wasm in a Go host through wazero (pure Go, no cgo) on the stock toolchain, with Kettle's report matching
the Rust goldens byte for byte, Rust wins. Everything else in `model/DESIGN.md` is information. fn-196.8, the Scala checker speed fix, was
retired with the frozen Scala backlog, since speed no longer decides. The decisive slice passed in fn-210.9: the recommendation
is Rust. The guest now directly exports `umpire_buffer_alloc`, `umpire_buffer_release` and
`umpire_call`, with no custom linker or export rewriting (fn-210.16).

| Phase | Specs and tasks | Scope |
| --- | --- | --- |
| After the Rust verdict | fn-217 → fn-230 → fn-231 → fn-227 → fn-211 → fn-228 → fn-238 → fn-237 | Owner priority order (fn-231 after fn-217 and fn-229, before fn-228.2 if possible; fn-229 is complete; fn-230 has no dependency and must land before fn-228.4; fn-227 placed before fn-211 on 2026-10-20 so Apalache arrives to a parameterized Model; fn-228 depends on fn-227 only, its place relative to fn-211 is the owner's). fn-233 (Kettle service, filed 2026-10-21) depends on fn-217 alone: on 2026-10-22 the owner dropped its fn-227 dependency, since its task .1 needs only fn-227.12's request evidence, which is complete. fn-233 is complete. Owner-approved remaining order is fn-243.1–.3 → fn-228.7 → fn-243.4 → fn-237 (the owner placed all of fn-244 before fn-243 on 2026-10-24; the programmed-worker implementation precedes fn-228.7 so R40 is written once, and fn-243.4 checks the write-up before fn-237 uses the hold fault; fn-242 runs in the parallel lane before fn-237.1); fn-211 is deferred at the owner’s request on 2026-10-22; fn-240 (symbolic back end, filed 2026-10-23 after the native symbolic spike) is deferred at the owner’s request on 2026-10-23. On 2026-10-23 the owner reactivated fn-238.5/.6 and all five fn-237 tasks at the end of that queue; their existing acceptance and measurement gates remain in force. fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-218, fn-221, fn-223, fn-224, fn-225, fn-226 and fn-229 are complete locally with their scoped reviews and gates passed; fn-225 and fn-226's single-task reviews cover their full specs. fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The former "Case slice" is filed as fn-228 (Runner and Run): the Model as umpire, one Model per Run, Propose on the protocol, one Go host loop, Driver and Shim, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

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
The rename checkpoint (.1), Propose (.2), the Go `runner` package (.3) and the Temporal Driver (.4)
are complete locally. The Runner drives an incremental Driver exchange, serializes every Host call,
records a `umpire.run.v1` Run, replays it offline and publishes it never-overwrite; one `umpire`
command with five modes replaced run, fuzz, replay, assess and repeat. The Temporal Driver acts and
observes over the generated API, gates the in-process worker by the Model's `worker_tasks`
instruction, never answers for the Model, and joins all its work on Close; the live slice runs
through `umpire explore` with the Run as the only artifact. .4's required gate passed at 2054.281s
(load average about 40, not steady state); inner loop 22.546s. fn-241 (one WorkflowService table
plus per-feature claims) is complete. Four pre-existing live Testpilot failures, the canary and
publish retirements and `TestTestpilotModelSlice` are carried into .6. The owner deferred the Shim
(.5), superseded by fn-243. .6 is complete locally: the legacy Testpilot packages are retired with a
coverage map, generic runtime 23,117 -> 1,647 and total 33,851 -> 3,317 non-test lines, the canary
runs the activity Model through the Runner behind an allowlist fence, and every gate passed
(572.0s gate, 8.9s inner loop). The programmed-worker implementation (fn-243.2) precedes .7.
[Spec](.flow/specs/fn-228-runner-and-run-the-one-go-host-loop.md).

| Name | Status | What |
| --- | --- | --- |
| fn-228.7 | ⬜ todo | After .6, fn-244.2 and fn-243.2; writes R40 in its final form (programmed workers): DESIGN.md Runner and Run write-up; R33 to R40 discharged; R5 closed as superseded (R7) |

### fn-237: Activity timeouts, heartbeat and cooperative cancellation (model, temporal)

Split from fn-227 on 2026-10-21 by owner decision; depends on fn-227. Carries the four optional
timeouts, Heartbeat, Tick and cooperative cancellation with their native and source support
(optional handler facts, bounded optional values), the final one-tick compatibility refinement,
live timeout admission and cancellation evidence, and their write-up. Reactivated by the owner
on 2026-10-23, scheduled after fn-238. On 2026-10-24 the spec was aligned with fn-243: .4's worker evidence
is program rows (heartbeat, cancellation acknowledgement) judged from server history, the held attempt and
schedule-to-start expiry use the hold fault, and task paths carry the fn-244 layout mapping.
[Spec](.flow/specs/fn-237-activity-timeouts-heartbeat-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-237.1 | ⬜ todo | After fn-244.1: optional handler facts through native adapters and independent TLC replay (R1) |
| fn-237.2 | ⬜ todo | Bounded nominal optional values through source binding, emission and decode (R1) |
| fn-237.3 | ⬜ todo | After .1, .2 and fn-227.3: deadlines, Tick, Heartbeat, cooperative cancel, final compatibility refinement (R2, R3) |
| fn-237.4 | ⬜ todo | After .3: live timeout admission and cooperative cancellation evidence (R4) |
| fn-237.5 | ⬜ todo | After .4: DESIGN.md timeout and cancellation write-up (R5) |

### fn-243: Programmed workers: the Model issues each worker program before it runs (model, temporal)

Filed on 2026-10-23 after the owner deferred the Shim and asked for pre-programmed workers: in a
cloud setup no worker can reach Umpire. Reworked with the owner on 2026-10-24: the program is a
table keyed by what a worker sees, constant per Model and World, issued once on Reset and written by
`umpire program`; per-Run variation travels in an `umpire-scenario` header the Model stamps when it
proposes a Start. A feature-agnostic `WorkerProgram` envelope carries per-feature key types (activity
now; workflow by kitchen sink script later). Two executors, raw (the fake worker for features without
SDK support) and sdk, live under `model/temporal/hosts/go/reference/worker`; the worker is part of the
system under test and the Run records which ran. Worker actions are observed, never proposed (new
Read proposal kind); a hold fault in the proposal's instruction replaces the Driver's gate; judgment
reads Describe's outcome and last failure; a deviation is Unexplained, an unprogrammed task is Refused.
Supersedes the Shim (fn-228.5). Design of record:
[programmed-workers.md](model/design/programmed-workers.md). Placed by the owner on 2026-10-24 after
fn-228.6 and fn-244.2 and before fn-228.7, which writes R40 once in its final form; fn-237 needs the
hold fault and follows.
[Spec](.flow/specs/fn-243-programmed-workers-the-model-issues.md).

| Name | Status | What |
| --- | --- | --- |
| fn-243.1 | ⬜ todo | After fn-228.6 and fn-244.2: program on Reset, Read proposals, `WorkerProgram` envelope, activity rows, `umpire program` (R1) |
| fn-243.2 | ⬜ todo | After .1: raw and sdk executors, Driver hold fault, judgment from server history, live slice (R2, R3) |
| fn-243.3 | ⬜ todo | After .2: a raw executor in another language follows the program file, or the gap is recorded (R4) |
| fn-243.4 | ⬜ todo | After .2, .3 and fn-228.7: DESIGN.md R40 final form checked; design note points to code (R5) |

### fn-240: Symbolic back end: Models as BTOR2 transition systems checked by rIC3, witnesses replayed natively (model) — deferred

Captured on 2026-10-23 from the positive native symbolic spike (`model/design/native-symbolic-backend.md`;
probe preserved at `model/design/native-symbolic/`). Turns the probe into `umpire-symbolic` beside the TLC
and Apalache back ends: the compiled World's embedded admitted tree emitted as BTOR2 with one `bad` per
property, a pinned rIC3 built into `model/build/ric3` running IC3 for unbounded proofs and BMC to the native
depth plus one for violations, witnesses replayed through `checked_step` with the claim re-judged natively,
a consumer selected by `expected/symbolic.txt` under `UMPIRE_BACKENDS=require`, then symbolic Worlds (fn-211's
R6 moves here) and the rest of the admitted subset. A proof counts as evidence only with native agreement.
No liveness; TLC owns progress claims. fn-211 stays deferred until set-valued state is admitted. Deferred at
the owner's request on 2026-10-23; no task started. The probe remains runnable for measurements.
[Spec](.flow/specs/fn-240-symbolic-back-end-models-as-btor2.md).

| Name | Status | What |
| --- | --- | --- |
| fn-240.1 | ⏸️ deferred | `umpire-symbolic` crate: BTOR2 emitter over the embedded admitted tree for Kettle-shaped Models, checked arithmetic latch, one-step history, emission goldens (R1) |
| fn-240.2 | ⏸️ deferred | After .1: pinned rIC3 built into `model/build/ric3`; IC3 proofs, BMC to native depth plus one, lifecycle, verdict kinds (R2) |
| fn-240.3 | ⏸️ deferred | After .1, .2: witness decoding by declaration index, `checked_step` replay, claim re-judged, replay types shared with `umpire-apalache`, mutation controls (R3, R4) |
| fn-240.4 | ⏸️ deferred | After .3: `expected/symbolic.txt` consumer, require-mode gating, ownership, Kettle goldens, DESIGN.md and MILESTONES (R4, R5, R8) |
| fn-240.5 | ⏸️ deferred | After .4: symbolic Worlds over declared finite domains, computed starts, decoded World replay, `parameterized_world` golden (R6) |
| fn-240.6 | ⏸️ deferred | After .4: action inputs, non-unit outcomes, bounded sequences, Rust widths on `Type::Int`; `action_inputs` and `shared_returns` goldens (R7) |

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
Task .3 is complete locally: strict typed ITF replay, selected property and arithmetic refusal validation, legacy and native Vector projections passed review and both audits. Fresh guest/slice/gaps proof passed with guest `d96f123b6ec669e7385e9fff75f711f2d20e0f15a0d918928238f7e263221217`. All 17 required gate stages passed: gate 824.431s exceeded 60s; inner loop 7.632s met 10s. The first full run failed with its causal output discarded; its cause remains unknown. A reviewed timing-wrapper correction now preserves failed child diagnostics; the logged workspace diagnostic and final retry passed. Fixture integration and symbolic Worlds are deferred at the owner’s request on 2026-10-22. The unfinished .4 consumer is preserved on local branch `work/fn2114-deferred` at `fee5fc56ae`, including its tracked handoff; no completion review or final gate is claimed. Execution moves to fn-228. On 2026-10-23 the owner took Apalache out of the required gate
(`UMPIRE_APALACHE=require` runs its real-tool controls); resuming fn-211 restores it.
[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.4 | ⏸️ deferred | After .3: Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |
| fn-211.5 | ⏸️ deferred | After .3: symbolic World constants via `ConstInit`, one run per property across the declared World domain, replayed under the decoded World (R6) |
