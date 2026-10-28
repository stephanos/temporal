# Milestones

The owner's overview of open work, execution order and blockers.

As of 2026-10-28.

The owner resumed work on 2026-10-27: complete repository migration first, then
continue this backlog. [fn-258](.flow/specs/fn-258-complete-the-resumed-standalone-umpire.md)
replaces retired fn-257 and retains its completed export evidence. Open Flow tasks
and unvalidated candidates remain open.

The frozen `c5189bd35d` candidate passed the scheduled source-only standalone
gate: all 17 stages, with 903 tests discovered, 900 executed and three ignored.
Parent-child receipts include the separate doctests after the controller admission
repair. Its five-consumer Activity warm run passed all 148 tests and 25 compiler
controls in 444.463 s; the 60 s target remains unmet. Review fixes align CI's mise
pin, restore combined Apalache opt-in, keep formatting outside JVM admission and
make supervisor scratch private. Focused joined controls pass; follow-up review
returned SHIP and .17 is closed. Provisioning .16 is frozen and blocked in its isolated worktree. About 32 GiB of idle validation caches were reclaimed
with verified proof-output recovery. Go-host count producers and mixed-control
TLC unknowns remain with .12. macOS validation remains owner-deferred.

## Keeping this page current

- Describe the present; Flow and git retain history. Remove completed tasks and closed specs.
- Keep remaining tasks in `Name | Status | What` tables: 🔄 in progress, ⬜ todo, ⛔ blocked, ⏸️ deferred.
- Only the root agent edits this page. Update the date with every edit.
- Follow `AGENTS.md` for task tracking, committed closures and branch integration.

## Direction

Umpire uses the Rust workspace described in [model/README.md](model/README.md).
Its guest runs as stock WASM in a pure Go host.
The Scala tree under `model/archived/` is frozen reference material.

Repository extraction resumes as **fn-258**, ahead of the backlog below. The current
framework has 161 changed files since the original export checkpoint; refresh the
standalone candidate and adapt actual consumers before current-pair proof and cutover.
Preserve source candidates and recovery evidence in [MIGRATE.md](MIGRATE.md).
Validation may run. GitHub destination inspection remains blocked by SAML SSO; local
work proceeds while publication remains unverified.

Backlog order remains **fn-254 → fn-255 → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245**. fn-240 and fn-211 remain deferred.
fn-254 (fast gates, and a framework gate that runs from `model/umpire` without the Temporal tree) leads: the owner
ranks a fast feedback loop among the most important properties. The standalone workspace/driver and independent gate checkpoint are integrated, and the complete isolated proof passed.
Immediate validation path: downloads .16, then shell ports .15, budgets .12 and final proof .8, consuming the reviewed and closed .17 nextest/stage-DAG checkpoint. The consumer/cache checkpoint is reviewed and closed; verified configurations cache independently of refuted siblings, and test-only Kettle readers no longer expand unrelated libraries. fn-254 gets first admission to shared build/JVM/review capacity. Already running fn-246.9/.13 work may reach safe checkpoints, and disjoint source work may continue without delaying fn-254. Finish fn-254.7 after committed fn-246 closure.
The owner waived fresh baseline measurements; final measurements remain. fn-254.7 waits for fn-246; fn-254.13's workspace move is integrated. fn-253.2–.5 (countdown spike and library) may run as a lane when one is free; activity changes
retain their listed dependencies and follow the current priority/source-ownership order.
fn-255.3 and .4 (fuzz targets and cargo-mutants) may prepare disjoint source before fn-254 closes; execution waits for committed fn-254.12 budget admission. Shared pins, manifests, decoder seams and native command registrations land serially.
fn-256 (Scala-era tooling removal) may run as a lane when its blockers clear; shared Makefile, CI, pin and documentation edits land serially with fn-254's.
Consolidation and build prerequisites for fn-246 are satisfied. Priority does not create dependencies;
clear cross-spec blockers only after prerequisite closures are committed in `umpire`.
Parallel work requires disjoint source and code-generation ownership.

The owner lifted the validation hold on 2026-10-26. Resume focused fn-254 checks
and retain unvalidated candidates until their required evidence and reviews pass.
The fn-254.5 fixture/compiler checkpoint is reviewed and closed on `umpire`: all 24 cases,
Temporal compiler/activity checks, 61 parser rows plus 18 compiler refusals, bounded profiles
and snapshot repair controls passed. The .16 provider candidate is adapted in its own
worktree; .15/.12 candidates remain in the isolated source-assembly branch.
Task .17's shared mise task graph, batched nextest runner, supervisor and Rust/TLC
receipt producers are joined on `umpire`. Inventory, lifecycle, cross-invocation
lease/handoff and public graph controls pass. The final real scheduled standalone
passed in 990.582 s on `c5189bd35d`, including all 896 nextest tests and four doctest
bodies, with matching parent-child receipt authority and all owned processes reaped.
The current Activity warm run passed in 444.463 s with exactly five consumers,
148 tests, 23 negative/two positive compiler controls and three documentation targets;
14 verified TLC cache hits avoided model executions. Post-review fixes pass focused
CI, opt-in provisioning, formatting and supervision controls; follow-up review
returned SHIP and .17 is closed. The owner deferred macOS validation on 2026-10-27; Darwin behavior is
preserved, with no runtime acceptance claim.
Remaining work includes provisioning, native control ports, budget admission and
final combined-gate measurements. Downstream candidates are unvalidated and not integrated into `umpire`;
preparation does not advance Flow dependencies or mark tasks complete. Integrate shared CLI/configuration
changes in dependency order. fn-246.9/.13 have retained source/proof checkpoints awaiting integration and closure.
The owner-approved admission policy is one coordinator, up to four implementation
workers across the team, and one separate review lane. The coordinator owns admission,
Git/Flow integration and closure; workers own disjoint source. Admit one review round
at a time, preserving required review lenses and a fixed reviewed head. Each source
handoff gets a fresh reviewer. Retain the earlier extra-batch candidates without
restarting those extra lanes; new implementation admission uses the four-worker cap.
Measure verified Flow closures and observed cycle/review/rework/blocked time in
[the execution record](.flow/artifacts/fn-257-extract-the-standalone-umpire/agent-throughput.json),
keeping unvalidated preparation out of completed-task counts. Execution gates may now run. fn-254 retains priority for shared build/JVM capacity; retain the
fn-255 generator/fuzz/mutation and fn-253.2 spike candidates without treating preparation as completion.
Spike verdicts, performance claims and task completion still require evidence.

When validation resumes, run focused checks for each change and its changed gates. Defer unrelated broad suites and full
`model-check`/`model-validate` runs to fn-254.8. Exact TLC agreement, migration inventories, gate failure controls
and the changed standalone gate keep their task-specific checks. Deferred checks remain final requirements.

Current formats migrate together; goldens may be re-accepted after review. Go size targets remain
5,000 generic-runtime and 11,000 total non-test Testpilot lines. Each spec closure requires its
reviewed validation evidence and a committed validation-cost assessment.

## Open specs

### fn-258: Complete the resumed standalone Umpire migration

The renewed source inventory and history task is reviewed and closed on `umpire`.
The standalone preserves current integrated source and both verified histories;
1,189 mapped files have zero missing or unexpected differences and all 499 protected
records remain exact. The consumer candidate includes a regenerated guest reproduced
in two independent sibling layouts, with unchanged public ABI sections and passing
host/wire controls. These are source candidates; final gates and cutover remain open.
Standalone preparation is reviewed and closed on `umpire`. The current required-TLC validation passed all 18 parent and 17 isolated
child stages, with 912 tests/doctests executed and three ignored in each tests stage.
Its 1,679.762-second gate and 24.162-second edit loop miss their soft targets.
Proof binaries, receipts and source snapshots are retained and independently recoverable.
The in-tree framework stays authoritative until publication and verified cutover.
Destination access remains blocked by SAML/credentials. See [MIGRATE.md](MIGRATE.md).

[Spec](.flow/specs/fn-258-complete-the-resumed-standalone-umpire.md).

| Name | Status | What |
| --- | --- | --- |
| fn-258.3 | 🔄 in progress | Consumer `3f5f6135` fixes review gaps in missing-checkout errors, workspace evidence and docs; recovery and exact-pair focused checks recorded; follow-up review pending. Final paired freshness remains .4 (R4) |
| fn-258.4 | ⬜ todo | Exact final-pair fresh-target, release/parity, generation/guest and local-edit acceptance proofs with timings (R3–R5) |
| fn-258.5 | ⬜ todo | Complete candidate receiving/recovery dispositions, destination access, authorized publication, pinned CI and source cutover; then resume fn-254 (R1, R4, R6) |


### fn-254: Fast validation gates and a real inner loop

The TLC-required gate took 2,093 s at fn-247's close against a 60 s soft target; `cargo test --workspace` was 82% of it.
Historical causes included serial test binaries, one TLC JVM per property (about 1.5 s each), about 134 test
binaries linking the whole graph, and no real Model-edit inner loop. Routine gates now skip Apalache;
its opt-in cost is separate. The retained complete five-consumer activity quick run took 444.027 s, including 388.875 s in
realization tests. The corrected six-consumer Kettle quick run took 129.783 s warm (99.527 s fixtures,
20.229 s TLA tests); these are different workloads, and the 60 s target remains unmet.
The current .17 edited-source Kettle run took 173.142 s: 45.465 s in inventory/build
and 110.616 s in its test phase. It retained the existing positive TLC cache and fresh
negative executions; the 60 s target remains unmet. See [the current proof](.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/kettle-semantic-renewal/summary.json).
The current five-consumer Activity runs on `c5189bd35d` took 598.237 s for setup
and 444.463 s warm, with all 148 selected tests and 23 negative/two positive
compiler controls passing. The warm run had 14 verified TLC hits and no
model-checker executions; its test phase alone took 346.122 s. The 60 s target
remains unmet. Earlier pre-counter warm timing was 321.299 s under different
build/cache conditions; these runs do not establish a causal speed comparison.
macOS validation is owner-deferred for now; the Linux proof does not establish Darwin cleanup.
Fixture/compiler batching and nextest scheduling address gate costs.
Coverage stays: TLC remains the independent checker, and `model-validate` never uses the new TLC result cache.
cargo-nextest approved, pinned via mise. The framework's checks move into `model/umpire` and must pass on that
directory alone (R7). First in the current order; only .7 waits for fn-246's closure.

[Spec](.flow/specs/fn-254-fast-validation-gates-and-a-real-inner.md).

| Name | Status | What |
| --- | --- | --- |
| fn-254.7 | ⛔ blocked | After committed fn-246 closure: one test binary per `temporal` crate (R4) |
| fn-254.8 | ⬜ todo | After remaining implementation tasks: final measurement against the baseline, targets, docs (R6, R1) |
| fn-254.12 | ⬜ todo | Budget parser/retry candidate retained; consume .17 Rust/TLC receipts, add actual Go-host counts, then admit tables against final .15/.16 interfaces. Tables remain uncalibrated and refuse budget enforcement until admitted (R8) |
| fn-254.15 | ⬜ todo | Integrated tree has 29 active shell paths; retained source candidate reduces these to two bootstraps/four controls. Adapt ports to .17 supervision and final .16 interfaces, finish native controls and validate (R7) |
| fn-254.16 | ⛔ blocked | Focused and real provider controls pass on frozen 60f1c762. Full standalone remains unaccepted; after owner resume and ≥50 GiB headroom, renew proof/review/integration. [Retained validation state](.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/session-wrap-20261027/resource-hold-draft.json) (R7) |

### fn-256: Retire the Scala-era build and Go tooling

`model/archived` stays as read-only reference (owner). Its tooling goes: the Scala-era Go IR reader in `tools/umpire`
(about 43k lines that nothing else imports, still run in CI), the Quint export and backends targets, about 290 lines of
Scala Make targets and the `scala-cli` pin. Independent of fn-254's lanes; its root `Makefile` edits avoid fn-254's model
section, so it may run as a lane, landing serially.

[Spec](.flow/specs/fn-256-retire-the-scala-era-build-and-go.md).

| Name | Status | What |
| --- | --- | --- |
| fn-256.1 | ⛔ blocked | Source checkpoint retained; required live gate awaits fn-246.9 whole-prefix consumer repair and current guest/policy renewal (R1, R3) |
| fn-256.2 | ⬜ todo | Retirement candidate source-reviewed and assembled: Scala targets/pin removed, archive retained read-only. Prerequisite closure and execution evidence remain (R2–R4) |

### fn-255: Thorough framework testing

Owner-approved techniques beyond today's fixtures and hand-seeded mutants:
- generated Models whose compiled, interpreted and TLC results must agree;
- cargo-fuzz on the guest-dispatch and TLC decoders;
- cargo-mutants on the framework crates;
- Miri on the WASM ABI's `unsafe` code;
- cargo-llvm-cov to aim the generator and corpora.

All of it lives in `model/umpire`, stays out of `model-quick`, and pins every finding as a fixture or corpus regression.
Bounded so it cannot undo fn-254 (R7). Work counts are capped hard: Model sizes, cases, TLC launches, corpus size.
Its share is at most 30 s in `model-check` and 120 s in `model-validate`. Mutants, coverage and long runs are
scheduled-tier only. If the gates miss fn-254's targets, fn-255's counts are cut, never the targets.
Not adopted: Kani, bolero, Loom, Shuttle (a candidate once checking goes parallel), deterministic simulation.

The reviewed [source-generation design](model/design/generated-differential.md) is integrated;
its measured bounds and same-schema, Verified-only batching guide the maintained harness.
Execution of .2–.6 waits for committed fn-254.12 budget admission; disjoint source preparation may continue. Their explicit Flow blocks prevent premature admission. Final tier wiring remains .7.
The [renewed plan review](.flow/artifacts/fn-255-thorough-framework-testing-generated/plan-review-20261027/summary.json) returned SHIP: stable replay and nightly fuzz wrappers have separate ownership, and differential checks must include legitimately Refuted claims.

[Spec](.flow/specs/fn-255-thorough-framework-testing-generated.md).

| Name | Status | What |
| --- | --- | --- |
| fn-255.2 | ⛔ blocked | Adapt retained differential-harness candidate to the admitted source design; validate shrinking and fixture pinning (R1) |
| fn-255.3 | ⛔ blocked | [Stable harness/nine-wrapper source split retained on 7010baad96](.flow/artifacts/fn-255-thorough-framework-testing-generated/task3/source-preparation-7010baad96.json), unbuilt and unintegrated; budgets, pins, locks, native admission and runtime controls remain (R2) |
| fn-255.4 | ⛔ blocked | [Four-file runner candidate is unregistered](.flow/artifacts/fn-255-thorough-framework-testing-generated/task4/readiness-20261027.json); tool pin, fixed smoke manifest, budget admission and measured triage pending (R3) |
| fn-255.5 | ⛔ blocked | Miri ABI candidate prepared and under source review; .3 pin, registration and actual Miri runs pending (R4) |
| fn-255.6 | ⛔ blocked | Coverage runner/parser candidate prepared and under source review; .2/.3 integration and measured guidance pending (R5) |
| fn-255.7 | ⬜ todo | After .2–.6 and committed fn-254.6/.10/.12: tier wiring within budget, re-measured gates, docs (R6, R7) |

### fn-246: Realization as a checked refinement

Shared registry/entity/filler infrastructure, whole-prefix witnesses, compatible frontiers,
protocol refinement and Model-derived worker policies. Feature mapping must fit within twice
the Model's lines, re-measured on the workflow Model. Guest/Host admission and actual SDK
heartbeat delivery are early gates. fn-237.4 retains the worker journal, evidence/gates and
mandatory live proofs. Final required-TLC validation and live gates remain outstanding. An optional Apalache
subprocess refusal remains recorded separately.

[Spec](.flow/specs/fn-246-realization-as-a-checked-refinement-a.md).

| Name | Status | What |
| --- | --- | --- |
| fn-246.13 | 🔄 in progress | Protocol/approved cancellation correction and guest publication checkpoints prepared with native/TLC proofs; root integration/review/closure pending (R5) |
| fn-246.9 | 🔄 in progress | Whole-prefix recording/replay and consumer repair checkpoint prepared; 59-row Model-slice proof retained. Root integration/review/closure pending; full live umbrella remains separate (R1, R5) |
| fn-246.10 | ⬜ todo | After .9: bounded alternatives and coherent explanation publication (R1, R5) |
| fn-246.2 | ⬜ todo | shared entity lens and filler-driven probes with equivalence pins (R3) |
| fn-246.3 | ⬜ todo | After .2, .13: activity migration, exact probe parity and mapping count (R4, R7) |
| fn-246.4 | ⬜ todo | After .3, .13: generic native Refinement consumer and activity declaration (R5) |
| fn-246.5 | ⬜ todo | After .3, .4: complete observable-worker policies and derived programs (R6) |
| fn-246.6 | ⬜ todo | After .3, .4, .5: integrated evidence, count/timing gates and author docs (R1–R7) |

### fn-237: Activity timeouts, heartbeat and cooperative cancellation

The independent Model supports deadlines, heartbeat and cooperative cancellation. Live constructors
still use atomic cancellation with absent timeouts. Remaining work must prove actual raw/SDK
worker evidence, hold-based expiry and cancellation before the final design write-up.

[Spec](.flow/specs/fn-237-activity-timeouts-heartbeat-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-237.4 | ⬜ todo | After committed fn-246: live timeout admission and cooperative cancellation evidence (R4) |
| fn-237.5 | ⛔ blocked | After .4 and committed fn-243.5: final DESIGN.md timeout and cancellation write-up (R5) |

### fn-243: Programmed workers

Programs are constant per Model/World and use feature-owned keys in a feature-neutral envelope.
The Run records the raw or SDK executor; worker actions are observed, never proposed.
Unheld cancellation of an open activity before Cancel RPC is currently refused; held-before-start
and terminal cancellation remain supported. This spec stays open for the unheld live proof.
Design: [programmed-workers.md](model/design/programmed-workers.md).

[Spec](.flow/specs/fn-243-programmed-workers-the-model-issues.md).

| Name | Status | What |
| --- | --- | --- |
| fn-243.5 | ⛔ blocked | After committed fn-237.4: restore justified unheld admission, raw/SDK race and replay proof, final R40 live evidence (R2, R5) |

### fn-245: Finite collections

Add finite `imbl::OrdSet`, `imbl::OrdMap` and `imbl::Vector` through the shared IR, interpreter and
TLC path. Shared value/schema/replay and IR traversal prerequisites are satisfied. The three
fixtures are `finite_sets`, `finite_maps` and `bounded_queue`; the set fixture also supplies the
outstanding performance comparison of the interpreter and compiled checker.

[Spec](.flow/specs/fn-245-finite-collections-sets-maps-and.md).

| Name | Status | What |
| --- | --- | --- |
| fn-245.1 | ⬜ todo | After fn-237.5: IR set/map types, canonical values, expression nodes (R1) |
| fn-245.2 | ⬜ todo | After .1: front-end admission and method table, native value export, interpreter evaluation (R2, R3) |
| fn-245.3 | ⬜ todo | After .1: TLA+ emission (subsets, functions with `Absent`, quantifiers, Head/Tail) and typed TLC decoding (R4) |
| fn-245.4 | ⬜ todo | After .2, .3: the three fixtures with goldens, differential and TLC agreement, sets performance measurement, DESIGN and README (R3 to R6) |

### fn-253: Reusable deadline countdowns for Models

A generic countdown (`umpire_dsl::countdown`: pure arm/tick/expired functions and within/descends laws,
lowered from source like `closable`) plus Temporal deadline semantics in `model/temporal`, so the
activity and later workflow, Nexus and timer Models stop hand-writing each countdown. A spike on the
activity settles the surface and the lifter gaps first. The spike and `umpire-dsl` work (.2–.5) are
independent of fn-246. Activity edits consume committed fn-246.15/.13 corrections and run serially as .1 → .7 → .9 → .10; .8 validates their final combined result.
Whether fn-246.13's protocol Model may share the Temporal layer is an open owner question.

[Spec](.flow/specs/fn-253-reusable-deadline-countdowns-for-models.md).

| Name | Status | What |
| --- | --- | --- |
| fn-253.1 | ⬜ todo | After committed fn-246.13: Option equality, named expiry effects/progress helpers, deduplicated Tick postconditions, split `state_bounds` (R4) |
| fn-253.2 | ⬜ todo | Spike: activity against the proposed `countdown` surface; verdict in `model/design/deadline-countdowns.md` (R1) |
| fn-253.3 | ⬜ todo | Countdown source candidate reviewed and assembled; .2 verdict, encoding/lowering admission and execution evidence remain (R2) |
| fn-253.4 | ⬜ todo | After .3: front end lowers countdown calls in handlers, effects and claims (R2) |
| fn-253.5 | ⬜ todo | After .4: shared `countdowns` fixture with refuting mutants (R2) |
| fn-253.6 | ⬜ todo | After .2, .5: Temporal deadline semantics as a named pattern (R3) |
| fn-253.7 | ⬜ todo | After .1, .6: activity uses countdowns and the deadline pattern, behaviour pinned (R4) |
| fn-253.8 | ⬜ todo | After .7, .9 and .10: final combined docs, parameter/refinement checks and full gates (R5) |
| fn-253.9 | ⬜ todo | After .7: assess `maximum_attempts` as a `Finite` wrapper; preserve native forged-value controls and record admission refusal if unsupported (R4) |
| fn-253.10 | ⬜ todo | After .9 and an owner decision on the fn-237 R3 compatibility control: remove `asynchronous_cancel`, always cooperative; coordinated with fn-237.4's live binding (R6) |

### fn-240: Symbolic back end (deferred)

BTOR2/rIC3 safety checking follows finite collections and uses shared replay. Witnesses must replay
through `checked_step` with native agreement; TLC owns liveness. The
[native probe](model/design/native-symbolic-backend.md) remains available for measurements.

[Spec](.flow/specs/fn-240-symbolic-back-end-models-as-btor2.md).

| Name | Status | What |
| --- | --- | --- |
| fn-240.1 | ⏸️ deferred | Rust width/signedness metadata, then Kettle-shaped BTOR2 emission, checked arithmetic, one-step history and goldens (R1) |
| fn-240.2 | ⏸️ deferred | After .1: pinned rIC3 under the framework build root; standalone admission, IC3 proofs, bounded BMC and lifecycle (R2) |
| fn-240.3 | ⏸️ deferred | After .1, .2: bounded whole-trace completion and fn-250.3 shared exact-edge replay, claim confirmation and mutation controls (R3, R4) |
| fn-240.4 | ⏸️ deferred | After .3: `expected/symbolic.txt` consumer, require-mode gating, ownership, Kettle goldens, DESIGN.md and MILESTONES (R4, R5, R8) |
| fn-240.5 | ⏸️ deferred | After .4: symbolic Worlds over declared finite domains, computed starts, decoded World replay, `parameterized_world` golden (R6) |
| fn-240.6 | ⏸️ deferred | After .4: finite action inputs, outcomes, sequences and fn-245 sets/maps/queue operations with symbolic agreement goldens (R7) |

### fn-211: Apalache back end (deferred)

Fixture integration and symbolic Worlds remain deferred until set-valued state is admitted.
Counterexamples require typed decoding and native replay; TLC owns liveness. The unfinished
consumer is preserved on `work/fn2114-deferred`; Apalache retains its separate `UMPIRE_APALACHE=require` opt-in.

[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.4 | ⏸️ deferred | Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |
| fn-211.5 | ⏸️ deferred | After .4 and the deferred R6 ownership decision: symbolic World constants, checked construction/catalogs and decoded-World replay (R6) |
