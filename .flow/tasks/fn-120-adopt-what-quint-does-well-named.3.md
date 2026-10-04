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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
