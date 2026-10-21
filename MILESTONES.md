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

Execution resumed under the owner's active milestone-completion goal (2026-10-20).
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
| After the Rust verdict | fn-217 → fn-230 → fn-231 → fn-227 → fn-211 → fn-228 | Owner priority order (fn-231 after fn-217 and fn-229, before fn-228.2 if possible; fn-229 is complete; fn-230 has no dependency and must land before fn-228.4; fn-227 placed before fn-211 on 2026-10-20 so Apalache arrives to a parameterized Model; fn-228 depends on fn-227 only, its place relative to fn-211 is the owner's). fn-233 (Kettle service, filed 2026-10-21) depends on fn-217; its place in the order is the owner's. fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-218, fn-221, fn-223, fn-224, fn-225, fn-226 and fn-229 are complete locally with their scoped reviews and gates passed; fn-225 and fn-226's single-task reviews cover their full specs. fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The former "Case slice" is filed as fn-228 (Runner and Run): the Model as umpire, one Model per Run, Propose on the protocol, one Go host loop, Driver and Shim, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

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
Spec-level dependency on fn-217; .3 needs the fn-217.6 derive. R5's Kettle-first spike
chronology still needs reconciliation with the already-completed Temporal catalog.
[Spec](.flow/specs/fn-233-kettle-service-a-framework-owned-end-to.md).

| Name | Status | What |
| --- | --- | --- |
| fn-233.1 | ⬜ todo | Move envelopes and the descriptor coverage check under `umpire/`; Temporal goldens byte-identical (R1) |
| fn-233.2 | ⬜ todo | Kettle service proto, Go reference server, realization, guest, Judge-only live run with transcript golden (R2) |
| fn-233.3 | ⬜ todo | Kettle `derive(Catalog)` catalog, `config.rs`, Annotate in the live run, `coverage.snap` and `gaps.txt` (R3, R5) |
| fn-233.4 | ⬜ todo | Seeded mutants, one-line switches, each named by the framework in `mutants.snap.txt` (R4) |

### fn-232: Linux/macOS guest byte-identity certification

The owner separated this proof from fn-230 so Linux embedding and freshness work can proceed.
The proof attempt completed four clean builds and both platforms' host suites. Both guests
reproduce across paths within each OS, but all eight Linux/macOS byte comparisons fail.
Certification remains open pending a separately scoped reproducibility fix; it does not gate
fn-230 or its consumers. The retained proof identifies the source pin, toolchains and artifacts.
[Spec](.flow/specs/fn-232-linux-macos-guest-byte-identity.md).

| Name | Status | What |
| --- | --- | --- |
| fn-232.1 | ⛔ blocked | Four clean builds and within-OS checks passed; both guests differ across Linux/macOS. Fix reproducibility separately before renewed certification (R1, R2) |

### fn-227: Parameterized Models: declared worlds, action inputs, and the activity Model with retries and timeouts (model, temporal)

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
and unchanged original artifacts. Source/TLC binding in .7 is next; parent plan review
remains NEEDS_WORK. Final timing was 93.288s against the 30s soft target, with a 5.035s inner loop.
On 2026-10-21 the owner approved all three contract amendments: lifecycle refinement
in place of impossible old-byte equality; cooperative cancellation with explicit protocol
evidence; and optional actual-request Observation evidence at tag 4. Heartbeat signals a
pending request without closing, and completion/failure races remain. Preserve old lifecycle
evidence and prove the stated projection, including the final one-tick compatibility World.
Fact emission, bounded optional ticks, transport/capture and transactional Start admission
still need task sizing and plan review before activity/live execution. No task or review
was completed by the approvals.
[Spec](.flow/specs/fn-227-parameterized-models-declared-worlds.md).

| Name | Status | What |
| --- | --- | --- |
| fn-227.7 | ⬜ todo | After .1: independent typed World/source binding, whole-state Init, all CFGs and `parameterized_world` TLC proof (R1, R3) |
| fn-227.2 | ⬜ todo | After .1 and .7: `Finite` and action inputs; `action_inputs` fixture with TLC (R2, R3) |
| fn-227.3 | ⬜ todo | After .2: lifter admits `Phased`/`reject!`; activity `maximum_attempts` and retryable failure (R3, R4) |
| fn-227.4 | ⬜ todo | After .3: timeouts, Heartbeat, Tick, cooperative cancel; final refinement, progress and bound claims (R3, R4, R5) |
| fn-227.5 | ⬜ todo | After .4: world from observed start options; Answer retryable; live retry proof (R6) |
| fn-227.6 | ⬜ todo | After .5: DESIGN.md write-up and readiness table closure (R7) |

### fn-211: Apalache backend for Umpire in Rust (model)

Captured on 2026-10-18 as a follow-up, after fn-210, fn-212 and (since 2026-10-20) fn-227, and only if their verdicts are to continue in
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

### fn-228: Runner and Run: the one Go host loop over the Model, Propose on the protocol, and the Testpilot shrink (model, temporal)

Captured on 2026-10-20, after fn-227. Files the post-verdict obligations R33 to R40 as one spec under
the owner's direction that Go is dumb and the Model holds the smarts, and retires the "Case slice"
name. Names settled: `umpire-wasm` becomes `umpire-runtime`, `umpire-tests` becomes `umpire-fixtures`,
Realization is the one word for a binding; new `Propose` on `umpire.model.v1`; Go package `runner`
with `Runner`, `Run`, `Driver` and `Shim`. The Model is the umpire; the Runtime hosts one Model; a
Realization binds it to one system; the Runner drives a Driver, asks the Model to Propose, Judge
and Answer, and records a Run. Deletes verification, evaluation, CEL, contract, casefile, campaign
and replay; one `umpire` command with five modes.
[Spec](.flow/specs/fn-228-runner-and-run-the-one-go-host-loop.md).

| Name | Status | What |
| --- | --- | --- |
| fn-228.1 | ⬜ todo | Renames alone: `umpire-runtime`, `umpire-fixtures`, Realization in prose (R1) |
| fn-228.2 | ⬜ todo | After .1: `Propose` on protocol, Runtime, `Realize`, `Host`; activity Realization proposes (R2) |
| fn-228.3 | ⬜ todo | After .2: Go `runner`: `Runner`, `Run`, `Driver`, five modes, one `umpire` command, publish (R3) |
| fn-228.4 | ⬜ todo | After .3: Temporal Driver as act-and-observe; live slice through `umpire explore` (R4) |
| fn-228.5 | ⬜ todo | After .4: Shim endpoint for non-Go SDK workers (R5) |
| fn-228.6 | ⬜ todo | After .4: delete and shrink Testpilot packages; sizes reported (R6) |
| fn-228.7 | ⬜ todo | After .5 and .6: DESIGN.md Runner and Run write-up; R33 to R40 discharged (R7) |
