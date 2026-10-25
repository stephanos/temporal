# Milestones

The owner's overview of what is open, in what order, and why. One page, describing the present.

As of 2026-10-25.

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
2026-10-21; source tasks retain attributable gates; independent preparation and reviews run in parallel.
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
| After the Rust verdict | fn-217 → fn-230 → fn-231 → fn-227 → fn-211 → fn-228 → fn-238 → fn-237 | Owner priority order (fn-231 after fn-217 and fn-229, before fn-228.2 if possible; fn-229 is complete; fn-230 has no dependency and must land before fn-228.4; fn-227 placed before fn-211 on 2026-10-20 so Apalache arrives to a parameterized Model; fn-228 depends on fn-227 only, its place relative to fn-211 is the owner's). fn-233 (Kettle service, filed 2026-10-21) depends on fn-217 alone: on 2026-10-22 the owner dropped its fn-227 dependency, since its task .1 needs only fn-227.12's request evidence, which is complete. fn-233 is complete. Remaining order is (fn-250 ∥ fn-251 ∥ fn-252) → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245, ordered for throughput on 2026-10-24: consolidation (fn-248, fn-250, fn-251, fn-252, from the owner's crate and duplication review) lands before the feature work that would otherwise repeat each change in every copy, and the three consolidation specs run as parallel lanes (the owner placed all of fn-244 before fn-243 on 2026-10-24; the programmed-worker implementation precedes fn-228.7 so R40 is written once, and fn-243.4 checks the write-up before fn-237 uses the hold fault; fn-242 is complete before fn-237.1); fn-246 (realization shape, filed 2026-10-24) is placed by the owner after fn-243.4 and before fn-237.4; fn-228 closed on 2026-10-24 with R40 discharged by pointer to fn-243 (owner decision), its live unheld-cancellation proof owned by fn-243.5; fn-245 (finite collections) follows fn-237.5 and fn-250 and precedes any fn-240 revival; fn-240 and fn-211 also wait for fn-250; fn-211 is deferred at the owner’s request on 2026-10-22; fn-240 (symbolic back end, filed 2026-10-23 after the native symbolic spike) is deferred at the owner’s request on 2026-10-23. On 2026-10-23 the owner reactivated fn-238.5/.6 and all five fn-237 tasks at the end of that queue; their existing acceptance and measurement gates remain in force. fn-210, fn-212, fn-213, fn-214, fn-215, fn-216, fn-218, fn-221, fn-223, fn-224, fn-225, fn-226 and fn-229 are complete locally with their scoped reviews and gates passed; fn-225 and fn-226's single-task reviews cover their full specs. fn-216's broad umbrella retains seven documented pre-existing failures. The recommendation is Rust under stock WASM/Go parity alone. Preserve each spec's prerequisites; priority does not add dependency edges or mark deferred work ready. fn-212 supplies lifting and TLC for its admitted source subset; fn-211 adds Apalache over its emitter. The former "Case slice" is filed as fn-228 (Runner and Run): the Model as umpire, one Model per Run, Propose on the protocol, one Go host loop, Driver and Shim, observations plus judgment and the never-overwrite publish store. Targets are at most 5,000 generic-runtime and 11,000 total non-test Testpilot Go lines. Standalone extraction remains a plan in `MIGRATE.md`. |

Spec completion now requires a committed validation-cost assessment (fn-249): reuse existing timings and review evidence, record safe savings or a justified decision to retain the gates, and close with the content-addressed report. The hook adds no test or model run.

## Specs

The 2026-10-24 cancellation counterexample adds a checkpoint boundary without moving the
independently planned Model work forward. The remaining order is fn-250, fn-251 and fn-252 as parallel lanes, then fn-246, fn-237.4, fn-243.5, fn-237.5 and fn-245.
Throughput rule (owner, 2026-10-24): consolidate before building on the duplicated code, so each later change is made once.
fn-251 (one probe engine) and fn-252 (one search, shared test harness) precede fn-246, which rewrites the same probes and
reuses the search; fn-250 (one value, schema and replay layer) precedes fn-245, which would otherwise add sets and maps to every
conversion copy. Byte-for-byte compatibility requirements are dropped across open specs; goldens are re-accepted after review. fn-243 remains
open until fn-243.5 supplies the deferred unheld-cancellation proof (fn-228 closed on 2026-10-24 with R40
discharged by pointer; the live proof is this spec's).
The current atomic Model explicitly refuses unheld cancellation of an open activity before
a Cancel RPC. Held-before-start and terminal cancellation remain supported. The CLI cannot
encode cross-spec task dependencies, so fn-243.5 and fn-237.5 have explicit blocked reasons and fn-245.1 names fn-248.3 in its description;
the conductor clears each only after the named prerequisite's closure is committed in `umpire`.

Sections follow the remaining execution order (first open task first); deferred specs come last.

fn-247 closed locally on 2026-10-25: nine tasks and R1–R6 reviewed complete, native Model tooling with no active Python, reproducible regenerated guest, and one complete TLC/Apalache-required validation plus benchmark. The actual combined run took 34m53s; its soft timing targets were exceeded and recorded. The durable validation-cost review is bound by content hash in the committed closure. Native execution evidence is Linux/aarch64; no native macOS certification is claimed.

fn-248 closed locally on 2026-10-25: crates replace hand-written code (insta goldens, heck, darling, quote, thiserror, prost-types, serde ITF envelope; enum-iterator removed), repeated helpers exist once, IR child traversal is shared. The parent completion review's findings (a stale expansion snapshot, generated calls that forced an author import) were fixed. The regenerated Temporal guest reproduces from a clean clone. One TLC/Apalache-required `make model-validate` passed at `9b3e26654a` (gate 1316 s against the 60 s soft target; inner loop 9.95 s). That run also covers fn-250.4 and fn-250.5, which the owner allowed to run ahead of fn-248.

fn-250.1/.2, fn-251.1/.2 and fn-252.1/.2 are complete locally: their scoped reviews and the joined `UMPIRE_APALACHE=require make model-validate` passed at `1c1e7fbf07`, including TLC, Apalache, workspace tests, guest freshness and all three Go hosts. The gate took 1549.246s against the 60s soft target; the inner loop took 7.623s against 10s. This used a cold task-owned cache with other independent work active, so it is not a controlled speed comparison. Apalache still refuses Models with non-unit edge facts until fn-250.3 supplies the required real-tool replay control.

fn-251.6 and fn-252.8 are complete locally and integrated: three-draw SHIP reviews, joined focused Rust tests and strict clippy passed; the regenerated Temporal guest is fresh, all standard wasm guests/native controls built, and Temporal/runtime/Kettle Go host suites passed on the same model tree. The consolidation parents retain their final full-gate obligations. fn-250.3, fn-251.3 and fn-252.5 are integrated with SHIP implementation reviews and scoped checks. The replay hook preserves native-only fixtures; moved tests pass source-boundary and snapshot ownership checks; Kettle accepted the reviewed single-line report ordering change. The current guest is regenerated and fresh. Their joined TLC/Apalache-required gate passed at `d30d679fc1` (1308.170s full gate, 8.882s edit/test loop). These three tasks are complete locally; fn-250 has an actual SHIP parent review; its remaining test-only observer literals are now centralized by fn-252.6. Supplemental review and the final joined full gate remain pending. fn-251.5 is implemented in its isolated lane with 187 focused tests and strict clippy passed; joined staging tests, three-draw SHIP review, fresh guest and all wasm/native/Go host checks passed; its full required gate is running alongside tooling. fn-252.7 is integrated with a three-draw SHIP review, compiler-enforced unsafe protection at all 149 safe Cargo roots and focused Rust/wasm/Go checks passed; its joined required gate is running.

### fn-250: Shared value, type and trace layer for the TLA+ back ends (model)

Filed on 2026-10-24 from the owner's duplication review. The back ends re-derive the same things from the IR: a wire
`Type` mirroring `ir::Type` (lowered five times), `Schema` from `Module` (five times), a `Literal` mirroring the DSL
`Value` (converted in about eight places), the observer names as literals, and one trace replay each for TLC and
Apalache. One layer replaces them (about 600–800 lines). A parallel lane after fn-248; fn-245, fn-240 and fn-211 wait for it.
[Spec](.flow/specs/fn-250-shared-value-type-and-trace-layer-for.md).

| Name | Status | What |
| --- | --- | --- |

### fn-251: One probe engine and one catalog build path for Kettle and Temporal (model, temporal)

Filed on 2026-10-24 from the owner's duplication review. Kettle and Temporal each implement the probe engine, the
descriptor-to-Rust graph builder, annotation/probe transport dispatch and build-configuration layers; Temporal also
hand-writes a `Wire` enum over catalogs the derived `Inspect` already walks, and keeps two facade crates. Each moves
to one system-neutral place in `model/umpire` (about 850–950 lines and two crates). A parallel lane after fn-248,
before fn-246, which rewrites the same probes.
[Spec](.flow/specs/fn-251-one-probe-engine-and-one-catalog-build.md).

| Name | Status | What |
| --- | --- | --- |
| fn-251.5 | 🔄 in progress | Generated annotation/probe transport integrated; SHIP review and focused/current guest/Go checks passed; full gate running (R5) |
| fn-251.4 | 🔄 in progress | After committed .3: one generic probe engine for Kettle and Temporal (R4) |

### fn-252: Quick consolidation wins in model code, tests and tooling (model)

Filed on 2026-10-24 from the owner's duplication review: independent small consolidations (about 1,100–1,300 lines,
plus ~12.9k lines of committed spike output). A parallel lane after fn-248, before fn-246, which reuses the search
and migrates the activity tests.
On 2026-10-25 the owner authorized .3 and .4 ahead of the whole fn-248 closure in separate isolated lanes; other fn-252 work still waits for fn-248.
Both tasks completed locally with focused checks and scoped SHIP reviews: .3 removed regenerable spike output and shared the authored Kettle source; .4 consolidated native fixtures and SpaceBinding. The parent R9 full validation remains open.
[Spec](.flow/specs/fn-252-quick-consolidation-wins-in-model-code.md).

| Name | Status | What |
| --- | --- | --- |
| fn-252.6 | 🔄 in progress | Shared typed Activity replay integrated; 33 real-TLC policy tests passed; review/final gate pending (R6) |
| fn-252.7 | 🔄 in progress | One proto script/wasm environment setting; shared guest lints with compiler forbids on all previously safe target roots (R7) |

### fn-246: Realization as a checked refinement: shared Temporal layer, action registry, Reach, derived strategies (model, temporal)

Captured on 2026-10-24 after the owner found the activity binding (about 3,600 non-test lines beside a
156-line Model) "way too much" and asked for abstractions, not only cuts. Three decisions taken one at a
time: the activity protocol machine becomes a real Model checked natively and by TLC, with the feature
Model declared as its refinement; `Projection::Reach` (the Model finds the path) lives in `umpire-runtime`,
system-agnostic; worker strategies enumerated from the Model replace the hand-named scenarios after fn-243
lands. Folded in: the action registry on the `Action` enum, reusable fillers, the entity lens, capability
traits, evidence kinds and a descriptor-generated WorkflowService catalog. Opaque-field inference is
removed with exact per-probe accounting; independently justified name and retry-ordering rules remain.
Target: per-feature mapping at most twice the Model's lines, re-measured on the workflow Model.
The fourteen-task plan is committed and has passed primary review. The reviewed proof staging, shared search scope and dispatch exclusions are integrated in `umpire`; implementation and live proofs remain pending.
Ambiguous timer paths retain full state and evidence cursors; one current whole-prefix witness contract
keeps later explanations consistent. Legacy readers, delta-mode fallback and Wasm marker machinery are dropped.
Current guest/Host admission and actual SDK heartbeat delivery remain early implementation gates.
The worker codec capability moves here; fn-237.4 retains all original actual live proofs in the amended plan.
Placed after committed fn-243.4, fn-228.7, fn-237.3, all of fn-247, and (for throughput, 2026-10-24) all of fn-251 and fn-252, before fn-237.4.
Reach reuses the suite's search core; strategy enumeration may use `itertools` under a budget.
The obsolete whole-fn-243 dependency is removed; the whole-fn-247 dependency remains.
[Spec](.flow/specs/fn-246-realization-as-a-checked-refinement-a.md).

| Name | Status | What |
| --- | --- | --- |
| fn-246.1 | ⬜ todo | After committed prerequisites: transactional bounded Reach and Kettle proof point (R1) |
| fn-246.12 | ⬜ todo | After committed prerequisites: shared descriptor catalog, claims and exact probe accounting (R3, R4) |
| fn-246.13 | ⬜ todo | After committed prerequisites: independent activity protocol Model and native/TLC invariants (R5) |
| fn-246.14 | ⬜ todo | After committed prerequisites: current heartbeat key and actual raw/SDK continuation capability (R6) |
| fn-246.7 | ⬜ todo | After .1: typed capabilities, fillers and internal evidence adapters (R2) |
| fn-246.8 | ⬜ todo | After .7: current default whole-prefix guest/Host contract (R1, R2) |
| fn-246.11 | ⬜ todo | After .7: variant metadata and Temporal action/entity/read registry (R2, R3) |
| fn-246.9 | ⬜ todo | After .8: current-format whole-prefix Run recording, replay and presentation (R1, R5) |
| fn-246.10 | ⬜ todo | After .7 and .9: bounded alternatives and coherent explanation publication (R1, R5) |
| fn-246.2 | ⬜ todo | After .7, .10, .11, .12: shared entity lens and filler-driven probes with equivalence pins (R3) |
| fn-246.3 | ⬜ todo | After .2, .13, .14: activity migration, exact probe parity and mapping count (R4, R7) |
| fn-246.4 | ⬜ todo | After .3, .11, .13: generic native Refinement consumer and activity declaration (R5) |
| fn-246.5 | ⬜ todo | After .3, .4, .14: complete observable-worker policies and derived programs (R6) |
| fn-246.6 | ⬜ todo | After .3, .4, .5: integrated evidence, count/timing gates and author docs (R1–R7) |

### fn-237: Activity timeouts, heartbeat and cooperative cancellation (model, temporal)

The amended remaining plan has passed fresh primary review and is integrated. Its committed-prerequisite admission and every original live/SDK proof remain required.

Split from fn-227 on 2026-10-21 by owner decision; depends on fn-227. Carries the four optional
timeouts, Heartbeat, Tick and cooperative cancellation with their native and source support
(optional handler facts, bounded optional values), the final one-tick compatibility refinement,
live timeout admission and cancellation evidence, and their write-up. Reactivated by the owner
on 2026-10-23, scheduled after fn-238. On 2026-10-24 the spec was aligned with fn-243: .4's worker evidence
is program rows (heartbeat, cancellation acknowledgement) judged from server history, the held attempt and
schedule-to-start expiry use the hold fault, and task paths carry the fn-244 layout mapping.
Facts, bounded optional values and World-selected fairness support are complete: native/interpreted/TLC agreement, mutation controls, full gates and exact old-artifact preservation passed. The independent deadline, heartbeat, Tick, retry and cooperative-cancellation Model policy is complete, with three-draw SHIP review, native/TLC replay, exact retained compatibility and full gates. Live constructors still use atomic cancellation with absent timeouts; actual timeout and cooperative worker evidence remains in .4.
[Spec](.flow/specs/fn-237-activity-timeouts-heartbeat-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-237.4 | ⬜ todo | After .3 and committed fn-246: live timeout admission and cooperative cancellation evidence (R4) |
| fn-237.5 | ⏸️ blocked | After .4 and committed fn-243.5: final DESIGN.md timeout and cancellation write-up (R5) |

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
fn-228.6 and fn-244.2 and before fn-228.7's architecture write-up; fn-237 needs the
hold fault and follows the completed fn-243.4 documentation checkpoint. The frozen atomic Model cannot
explain acknowledged cancellation that remains pending on a started attempt. The current
checkpoint refuses that unheld scope; fn-243.5 consumes fn-237's independent cooperative
semantics and proves restored admission. This parent stays open until that follow-up passes.
[Spec](.flow/specs/fn-243-programmed-workers-the-model-issues.md).

| Name | Status | What |
| --- | --- | --- |
| fn-243.5 | ⏸️ blocked | After .4, fn-246 and committed fn-237.4: restore justified unheld admission, raw/SDK race and replay proof, final R40 live evidence (R2, R5) |

### fn-245: Finite collections: sets, maps and bounded lists in Models (model)

The refreshed four-task plan passed fresh primary review after correcting collection projection, read-only captures, ordered short-circuit validity, empty sequence guards and shared testing consumers. Its committed-prerequisite admission remains unchanged.

Captured on 2026-10-24 after the owner answered four questions one at a time: shared fixtures are the
consumer now and the workflow Model later; finite element domains now with the IR shaped so unbounded
forms are additive later (Apalache or SMT arrays own those); authors write `imbl::OrdSet`, `imbl::OrdMap`
and `imbl::Vector` with a plain method table, no DSL vocabulary; placed after fn-237.5 and before fn-240
is revived, since it edits the same front-end, IR, interpreter and TLC crates as fn-237.1/.2. Three
fixtures (`finite_sets`, `finite_maps`, `bounded_queue`) carry lift, interpreter and TLC goldens; the set
fixture supplies the measurement fn-238.6 could not take. Depends on fn-237 and fn-250 (sets and maps are added once to the
shared `Literal` and `Schema`), and its .1 starts after fn-248.3's IR traversal helpers, which it extends.
[Spec](.flow/specs/fn-245-finite-collections-sets-maps-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-245.1 | ⬜ todo | After fn-237.5, fn-250 and fn-248.3: IR set/map types, canonical values, expression nodes (R1) |
| fn-245.2 | ⬜ todo | After .1: front-end admission and method table, native value export, interpreter evaluation (R2, R3) |
| fn-245.3 | ⬜ todo | After .1: TLA+ emission (subsets, functions with `Absent`, quantifiers, Head/Tail) and typed TLC decoding (R4) |
| fn-245.4 | ⬜ todo | After .2, .3: the three fixtures with goldens, differential and TLC agreement, fn-238.6 sets measurement, DESIGN and README (R3 to R6) |

### fn-240: Symbolic back end: Models as BTOR2 transition systems checked by rIC3, witnesses replayed natively (model) — deferred

Captured on 2026-10-23 from the positive native symbolic spike (`model/design/native-symbolic-backend.md`;
probe preserved at `model/design/native-symbolic/`). Turns the probe into `umpire-symbolic` beside the TLC
and Apalache back ends: the compiled World's embedded admitted tree emitted as BTOR2 with one `bad` per
property, a pinned rIC3 built into `model/build/ric3` running IC3 for unbounded proofs and BMC to the native
depth plus one for violations, witnesses replayed through `checked_step` with the claim re-judged natively,
a consumer selected by `expected/symbolic.txt` under `UMPIRE_BACKENDS=require`, then symbolic Worlds (fn-211's
R6 moves here) and the rest of the admitted subset. A proof counts as evidence only with native agreement.
No liveness; TLC owns progress claims. Depends on fn-245 and fn-250 (shared replay core); the emitter hash-conses every node and
runs rIC3 through the process crate. fn-211 stays deferred until set-valued state is admitted. Deferred at
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
(`UMPIRE_APALACHE=require` runs its real-tool controls); resuming fn-211 restores it. Since 2026-10-24 it depends on fn-250 and resumes on the
shared `Literal`, `Schema` and replay core.
[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.4 | ⏸️ deferred | After .3: Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |
| fn-211.5 | ⏸️ deferred | After .3: symbolic World constants via `ConstInit`, one run per property across the declared World domain, replayed under the decoded World (R6) |
