---
satisfies: [R5, R6, R7, R15]
---
# fn-120-adopt-what-quint-does-well-named.3 Add model lint after the Scala root inventory stabilizes

Touches: [tools/umpire/model/**, tools/umpire/lower/**, tools/umpire/cmd/**, Makefile, model/gate/**, model/ir/**, model/README.md]

## Description
Implement Part B against fn-114's final Scala-owned IR roots and fn-120.2's named alternatives, including the specification-hole kinds and the per-operation modality table of `.plans/MODALITIES.md`. Inventory findings only after those roots are stable; do not accept a finding merely because an earlier inventory named it.

**Size:** M
**Files:** Go IR reader/checker, lint command and fixtures, gate integration, checked-in finding acceptances.

### Approach
- Reuse reader tables, Query and realization indexes rather than adding a second evaluator.
- The lint command lives under `tools/umpire/cmd/`, with a Makefile target that runs it (`TestEveryToolingPackageHasALiveCaller`); the reader may not import `lower`. Reader-side kinds, counts and the modality views live in `tools/umpire/model`. `lower` exports the Query standing decision of `ask` and its evidence-field descriptor resolution (`descriptor.go`) for the two kinds that read them.
- Emit the R5 finding kinds with locations, fixtures for presence and absence, and reasoned checked-in acceptances.
- Fail the gate for new findings and stale acceptances.
- Compute each R15 count with the function that emits its R5 kind, so a count's difference is that kind's findings. Read the manifest standing's `no-realization` function (`tools/umpire/lower/lower.go`) instead of recomputing it; do not restate Known Gaps or the exploration ledger.
- Unmodeled API values: for each realization, collect the `Equal` and `Present` tests in poll `until` conditions and Run Event guards over a `temporal.api.*` field (resolved through the evidence element descriptor `lower` already builds). The field's descriptor gives the denominator, without the zero value; a value is mapped when a test of it belongs to an evidence kind whose `records` names a fact. Skip the `HistoryEvent` attributes oneof. Add no IR field.
- Hole kinds (R5; `.plans/MODALITIES.md` section 3): H1 `disabled-by-default`, H2 `silent-rejection`, H3 `unconstrained-result`, H4 `witness-only`, and H5 `must-not-pinned` behind a flag, off by default. They join the table, the decision trace this task builds and R9 reports (the last decision of an empty result: a `match` case whose `pattern` is `wildcard`, or an `if` naming no state field), the action's `party`/`timer`/`internal` flags, `Property.when_class`/`when_action`/`transition`, `Query.form` with the Scenario's `free`, `Progress.from/to` and the lifted named predicates, all in the IR today; no second evaluator and no IR field. Report by class and by the state record's first enum-typed field, never per state. `attemptStart`-style worker actions whose delivery the system decides are the known H2 exception, accepted with a reason.
- Per-operation table (R5): print, per machine and class, the table grouped by the named predicates the step function evaluated (from the decision trace) and by the machine's capability parameters where fn-122 has declared them, each cell MAY with results, MUST NOT with its guard and line, or `?` for H1/H2, with the Properties, progress claims and fn-122 laws that pin it (fn-122 R8 renders laws as the cells they pin and lists unpinned cells). Share the view code with task 4's explorer.
- First run on the activity IR: the 2026-10-03 measurement was 7 H1, 7 H2, 15 H3 and 8 H4; fn-112.6 removes the wildcard arms before this task, so expect the seven server-rejected pause/unpause pairs (H2) and `terminated`/`cancelRequestedWhileStarted` (H4) to remain, each accepted with fn-112's freeze as the reason (fn-112 Decision Context "Model gaps") until a later spec takes them.
## Acceptance
- [ ] R5 kinds report stable kind, machine, message and Scala location; malformed IR produces reader errors only.
- [ ] R6 each kind has positive and negative fixtures, including a written-only request field for the unmodeled-API-value kind.
- [ ] H1-H4 (and flagged H5) report class, state set by that field and position, each with a triggering and a non-triggering fixture, computed from the table, the decision trace and the claim index with no second evaluator; the activity IR's first run lists its hole findings, each fixed or accepted with a reason, with the seven pause/unpause H2 pairs and the two H4 Properties recorded against fn-112's freeze.
- [ ] The per-operation modality table prints per machine and class with MAY/MUST NOT/`?` cells, the predicates grouping them and the claims pinning them; a fixture Model with a wildcard arm shows `?`.
- [ ] R7 gate covers every checked-in IR root and never fails on a count; first-run findings, fixes, accepted reasons and the coverage summary are recorded.
- [ ] R15 summary is byte-stable, pinned by a fixture golden, printed by the gate, and every count agrees with its kind's findings.
## Done summary
Model lint (fn-120 Part B: R5, R6, R7, R15) ships as `tools/umpire/lint`, which may import only the reader, with the command `tools/umpire/cmd/umpire-lint`. The command is run by `make umpire-check-lint` and by a new model-gate step, "lint every IR file and print its coverage". Commits: 9167bbceda (reader decision trace, lowering exports), a19ddba016 (lint, command, gate, docs, acceptances), 166baa2b81 (revive splits), 0832eb61c7 (review P3s).

**What changed**
- **Reader.**
  - `Interpreter.Why(machine, state, class)` evaluates a step function again and records every `if` and `match` decision. Each decision carries its position, the case taken, whether it was a wildcard, whether it read the state, and the named predicates it called. A tracing copy of the interpreter does this; the evaluator is unchanged otherwise.
  - `Interpreter.Reads` says which arguments a claim's evaluation read.
  - Both are the shared basis for task 4's explorer.
- **Lowering.** It exports `Realizable`, which `ask` now calls, so the no-realization standing is decided in one place. It also exports `EvidenceElement` and `FieldAt`.
- **Lint kinds.** All of R5's kinds are implemented:
  - `unreachable-value`, `never-enabled`, `unproduced`, `untaken-choice`, `unasked-property`, `unfired-verify`, `unevidenced-fact`
  - `unperformed-action`, `unrealized-find`, `unread-refinement`, `unread-observation`, `unmodeled-api-value`
  - The holes: H1 `disabled-by-default`, H2 `silent-rejection`, H3 `unconstrained-result`, H4 `witness-only`, and H5 `must-not-pinned` behind `--must-not-pinned`, off by default.
- **Modality table.** `--tables` prints each machine's per-operation table by class, grouped by the state's first enum field. Each cell is MAY with its results, MUST NOT with its guard and line, or `?` for H1/H2. Each cell also lists its pinning claims (including laws) and the predicates the decision trace called.
- **Coverage summary.** It has R15's eight counts. Each count is its kind's tally (population, findings), and a test holds every difference to that kind's findings. A golden pins the summary over the lifter fixtures plus activity and nexus-control.
- **Accepted findings.** They live in `model/ir/<file>.lint.json` as {kind, owner, subjects, because}. They are matched by kind, owner and subject, never by position. A stale acceptance, a missing reason or an orphan file fails the run. The gate's `settle`, `IRPaths`, golden `IRFiles` and the original-baseline archive all skip these files. fn-122.5 can record law waivers as entries under its law kinds, using the waiver's reason.

**Decisions (own recommendation, pre-authorized)**
- **Gate start.** The conductor started this task before fn-114 closed. IrFiles roots (fn-114.1) are stable, and fn-114's remaining tasks are cleanup.
- **Package placement.** Lint lives in its own package, depending on the reader alone (conductor, for fn-124's split). Lowering is injected as `lint.Lowering`. The ownership map gains `lint: {model}`.
- **Hole granularity.** A hole finding is one class plus the field values its pairs are in. Its subject is, for example, `control-pause in unstarted, paused, pauseRequested, cancelRequested`: never per state, and not one finding per (class, phase).
- **H3, "constrains".** A transition Property or monitor constrains a row when its evaluation at that step reads the step (`Reads`). The same holds for disabled pins (H5) and for product claims read through refinement carriers. A composition's same-step Property names its member actions.
- **H1, the `if` rule.** "An `if` naming no state field" is taint-tracked: the condition read no value computed from the state.
- **Unreachable values.** They are read down to leaves: enum cases, Booleans and ranges. A record recurses into its fields; channel contents are a combination and are never reported.
- **Unmodeled API values.** The history attributes oneof is excluded structurally: it is the oneof of a member a `history` kind lifts, and no type name is used. A Run Event guard does not count fields of the payload's own package, the run's own record. This leaves activity at "7 tested, 6 mapped", as the spec's example shows.
- **Verify receipts.** `unfired-verify` reads `Check` over a copy of the Model with its verify Queries alone. On nexus-close that cut the time from about 59 s to 25 s.

**First run** (`.flow/tmp/fn120-3/findings.json`, coverage in `first-run-coverage.txt`)
- **Totals.** 691 findings, all accepted with reasons; none were fixed, because Models are under fn-112's freeze. Per file:

  | File | Findings |
  |---|---|
  | activity | 58 |
  | activity-race | 55 |
  | activity-system | 181 |
  | nexus-caller | 61 |
  | nexus-close | 262 |
  | nexus-control | 43 |
  | nexus-operation | 31 |

- **activity H2.**
  - The pause/unpause pairs (control-pause in paused, pauseRequested, cancelRequested; control-unpause in scheduled, backingOff, started, cancelRequested) are accepted against fn-112's freeze.
  - So are the other caller rejections: start-*, and control-* in unstarted.
  - Worker attemptStart/attemptResult are accepted as R5's known exception.
- **activity H4.** All 10 same-step Properties are witness-only. `terminated` and `cancelRequestedWhileStarted` are recorded against the freeze, because of the notFound-stutter falsity.
- **activity H1.** None; fn-112.6 removed the wildcards. nexus-caller has one H1: `if retryable then disabled`, an input decision, accepted with the Model's comment as the reason.
- **Other reasons.** Design Models without realizations; the admission design's out-of-scope controls; close-policy-specific states; and refinements no Query reads through yet.

**Verification** (`.flow/tmp/fn120-3/gates.status`, under the heavy lock)
- **Model gate.** `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` passed in 459 s, including the gate's Scala tests. Its new lint step took 14 s and printed the coverage for every IR file.
- **lint-model.** Passed in 165 s.
- **Full Go suite.** `-json -p 2` passed in 479 s. The slowest tests were TestMigrationProjectionPreservesSemantics (62 s) and TestOriginalBaselineCases (61 s).
- **lint-code-fast.** It first found 9 revive issues and then one import-shadowing issue; both were fixed in 166baa2b81 and it now passes on HEAD.
- **Reruns after the refactor.** model+lower tests passed in 325 s, and the lint and command tests pass. `umpire-lint` exits 0 on model/ir.

**Review.** claude-opus-5-5 at high, via `--spec claude:claude-opus-5-5:high`. The writer and the reviewer are the same family (Opus).
- Round 1: SHIP with five P3s, all fixed in 0832eb61c7:
  - drop the unused law sidecar and the waiver sentence;
  - keep H5 off end states, in the table and in the findings;
  - define the `.lint.json` suffix once, in the reader;
  - key refinement carriers by row;
  - fail the command test instead of skipping it.
- Round 2: SHIP, no findings.

**Deferred**
- Grouping the table by fn-122 capability parameters is left to fn-122 R8 and task 4. The table already shows laws as pins.
- The per-state view and `rules <class>` gap/overlap lines are R8, task 4.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9167bbceda, a19ddba016, 166baa2b81, 0832eb61c7
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), go test -tags test_dep -count=1 -p 2 ./tools/umpire/model/ ./tools/umpire/lower/ (exit 0, after refactor), go test -tags test_dep -count=1 ./tools/umpire/lint/ ./tools/umpire/cmd/umpire-lint (exit 0), go run ./tools/umpire/cmd/umpire-lint (exit 0), make lint-code-fast (exit 0)
- PRs: