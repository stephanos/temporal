# Milestones

The owner's overview of open work, execution order and blockers.

As of 2026-10-25.

## Keeping this page current

- Describe the present; Flow and git retain history. Remove completed tasks and closed specs.
- Keep remaining tasks in `Name | Status | What` tables: 🔄 in progress, ⬜ todo, ⛔ blocked, ⏸️ deferred.
- Only the root agent edits this page. Update the date with every edit.
- Follow `AGENTS.md` for task tracking, committed closures and branch integration.

## Direction

Umpire uses the Rust workspace described in [model/README.md](model/README.md).
Its guest runs as stock WASM in a pure Go host.
The Scala tree under `model/archived/` is frozen reference material.

Current order: **fn-254 → fn-255 → fn-246 → fn-237.4 → fn-243.5 → fn-237.5 → fn-245**. fn-240 and fn-211 remain deferred.
fn-254 (fast gates, and a framework gate that runs from `model/umpire` without the Temporal tree) leads: the owner
ranks a fast feedback loop among the most important properties. fn-246's in-flight work is wrapped;
fn-254.13 is extracting the standalone framework workspace and simplifying its native tools. Deliver fn-254's independent framework/gate checkpoint,
resume fn-246 to closure, then finish fn-254.7/.8.
The owner waived fresh baseline measurements; final measurements remain. fn-254.7 waits for fn-246, and fn-254.13's workspace move lands at an
integration boundary. fn-253.2–.5 (countdown spike and library) may run as a lane when one is free; activity changes
retain their listed dependencies and follow the current priority/source-ownership order.
fn-255.1, .3 and .4 (generator spike, fuzz targets, cargo-mutants) touch no gate files and may run as lanes before fn-254 closes.
Consolidation and build prerequisites for fn-246 are satisfied. Priority does not create dependencies;
clear cross-spec blockers only after prerequisite closures are committed in `umpire`.
Parallel work requires disjoint source and code-generation ownership.

During fn-254, run focused checks for each change and its changed gates. Defer unrelated broad suites and full
`model-check`/`model-validate` runs to fn-254.8. Exact TLC agreement, migration inventories, gate failure controls
and the changed standalone gate keep their task-specific checks. Deferred checks remain final requirements.

Current formats migrate together; goldens may be re-accepted after review. Go size targets remain
5,000 generic-runtime and 11,000 total non-test Testpilot lines. Each spec closure requires its
reviewed validation evidence and a committed validation-cost assessment.

## Open specs

### fn-254: Fast validation gates and a real inner loop

The TLC-required gate took 2,093 s at fn-247's close against a 60 s soft target; `cargo test --workspace` was 82% of it.
Causes: test binaries run one after another, TLC starts one JVM per property (about 1.5 s each), Apalache runs
although deferred, about 134 test binaries each link the whole graph, and no inner loop covers a real Model edit.
Coverage stays: TLC remains the independent checker, and `model-validate` never uses the new TLC result cache.
cargo-nextest approved, pinned via mise. The framework's checks move into `model/umpire` and must pass on that
directory alone (R7). First in the current order; only .7 waits for fn-246's closure.

[Spec](.flow/specs/fn-254-fast-validation-gates-and-a-real-inner.md).

| Name | Status | What |
| --- | --- | --- |
| fn-254.3 | ⬜ todo | After .10: cargo-nextest across binaries with a JVM test group; identical logical test inventory (R4) |
| fn-254.5 | ⬜ todo | After .10: Apalache out of routine gates, overlapping stages, trybuild trim, dead `model-check-changed` removed (R5) |
| fn-254.6 | ⬜ todo | After .2, .10: `make model-quick` with affected crates and a cached TLC result for unchanged modules (R2) |
| fn-254.7 | ⛔ blocked | After .4 and committed fn-246 closure: one test binary per `temporal` crate (R4) |
| fn-254.8 | ⬜ todo | After .2–.7, .11, .12: final measurement against the baseline, targets, docs (R6, R1) |
| fn-254.10 | ⬜ todo | After .13: framework gate and root delegation; prove CLI changes preserve guest caches (R7) |
| fn-254.11 | ⬜ todo | After .10: framework gate passes on `model/umpire` alone; planted Temporal coupling fails (R7) |
| fn-254.12 | ⬜ todo | After framework optimizations and controls: committed stage budgets; hard work counts, bounded timing retry, unbudgeted stages fail (R8) |
| fn-254.13 | 🔄 in progress | Separate framework workspace, dedicated guest driver, shared root/context/remap handling (R7) |

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

[Spec](.flow/specs/fn-255-thorough-framework-testing-generated.md).

| Name | Status | What |
| --- | --- | --- |
| fn-255.1 | ⬜ todo | Spike: generate admitted Models at tree and/or source level, measured, seeded bug caught (R1) |
| fn-255.2 | ⬜ todo | After .1: differential harness with shrinking and fixture pinning (R1) |
| fn-255.3 | ⬜ todo | Pinned nightly; cargo-fuzz targets for guest dispatch and TLC decoders, seeded corpora (R2) |
| fn-255.4 | ⬜ todo | cargo-mutants per core crate; triage every missed mutant (R3) |
| fn-255.5 | ⬜ todo | After .3: Miri on ABI buffer logic; syscall exclusion recorded (R4) |
| fn-255.6 | ⬜ todo | After .2, .3: coverage per crate; aim generator and corpora (R5) |
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
| fn-246.13 | ⬜ todo | After committed .15: independent standalone-activity protocol Model and native/TLC invariants (R5) |
| fn-246.9 | ⬜ todo | After .8: current-format whole-prefix Run recording, replay and presentation (R1, R5) |
| fn-246.10 | ⬜ todo | After .9: bounded alternatives and coherent explanation publication (R1, R5) |
| fn-246.2 | ⬜ todo | After .10, .11: shared entity lens and filler-driven probes with equivalence pins (R3) |
| fn-246.3 | ⬜ todo | After .2, .13: activity migration, exact probe parity and mapping count (R4, R7) |
| fn-246.4 | ⬜ todo | After .3, .11, .13: generic native Refinement consumer and activity declaration (R5) |
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
independent of fn-246. The activity clarity task is pending; later activity changes retain their listed dependencies.
Whether fn-246.13's protocol Model may share the Temporal layer is an open owner question.

[Spec](.flow/specs/fn-253-reusable-deadline-countdowns-for-models.md).

| Name | Status | What |
| --- | --- | --- |
| fn-253.1 | ⬜ todo | Option equality in `deadline` and `tick_priority`, named expiry effects and progress helpers, deduplicated Tick postconditions, split `state_bounds` (R4) |
| fn-253.2 | ⬜ todo | Spike: activity against the proposed `countdown` surface; verdict in `model/design/deadline-countdowns.md` (R1) |
| fn-253.3 | ⬜ todo | After .2: `umpire_dsl::countdown` functions and laws in plain Rust (R2) |
| fn-253.4 | ⬜ todo | After .3: front end lowers countdown calls in handlers, effects and claims (R2) |
| fn-253.5 | ⬜ todo | After .4: shared `countdowns` fixture with refuting mutants (R2) |
| fn-253.6 | ⬜ todo | After .2, .5: Temporal deadline semantics as a named pattern (R3) |
| fn-253.7 | ⬜ todo | After .1, .6: activity uses countdowns and the deadline pattern, behaviour pinned (R4) |
| fn-253.8 | ⬜ todo | After .7: docs and full gates (R5) |
| fn-253.9 | ⬜ todo | After .1, coordinated with fn-246.13: `maximum_attempts` as a `Finite` wrapper (R4) |
| fn-253.10 | ⬜ todo | After .1 and an owner decision on the fn-237 R3 compatibility control: remove `asynchronous_cancel`, always cooperative; coordinated with fn-237.4's live binding (R6) |

### fn-240: Symbolic back end (deferred)

BTOR2/rIC3 safety checking follows finite collections and uses shared replay. Witnesses must replay
through `checked_step` with native agreement; TLC owns liveness. The
[native probe](model/design/native-symbolic-backend.md) remains available for measurements.

[Spec](.flow/specs/fn-240-symbolic-back-end-models-as-btor2.md).

| Name | Status | What |
| --- | --- | --- |
| fn-240.1 | ⏸️ deferred | `umpire-symbolic` crate: BTOR2 emitter over the embedded admitted tree for Kettle-shaped Models, checked arithmetic latch, one-step history, emission goldens (R1) |
| fn-240.2 | ⏸️ deferred | After .1: pinned rIC3 built into `model/build/ric3`; IC3 proofs, BMC to native depth plus one, lifecycle, verdict kinds (R2) |
| fn-240.3 | ⏸️ deferred | After .1, .2: witness decoding by declaration index, `checked_step` replay, claim re-judged, replay types shared with `umpire-apalache`, mutation controls (R3, R4) |
| fn-240.4 | ⏸️ deferred | After .3: `expected/symbolic.txt` consumer, require-mode gating, ownership, Kettle goldens, DESIGN.md and MILESTONES (R4, R5, R8) |
| fn-240.5 | ⏸️ deferred | After .4: symbolic Worlds over declared finite domains, computed starts, decoded World replay, `parameterized_world` golden (R6) |
| fn-240.6 | ⏸️ deferred | After .4: action inputs, non-unit outcomes, bounded sequences, Rust widths on `Type::Int`; `action_inputs` and `shared_returns` goldens (R7) |

### fn-211: Apalache back end (deferred)

Fixture integration and symbolic Worlds remain deferred until set-valued state is admitted.
Counterexamples require typed decoding and native replay; TLC owns liveness. The unfinished
consumer is preserved on `work/fn2114-deferred`; resumption restores Apalache's required gate.

[Spec](.flow/specs/fn-211-apalache-backend-for-umpire-in-rust.md).

| Name | Status | What |
| --- | --- | --- |
| fn-211.4 | ⏸️ deferred | Apalache consumer; Kettle's `apalache.txt` agreement golden (R4, R5) |
| fn-211.5 | ⏸️ deferred | symbolic World constants via `ConstInit`, one run per property across the declared World domain, replayed under the decoded World (R6) |
