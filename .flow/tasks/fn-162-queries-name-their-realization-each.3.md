---
satisfies: [R4, R7]
---
# fn-162-queries-name-their-realization-each.3 Migrate the standalone activity off claim-container subjects

## Description
Migrate the standalone activity off its 17 claim-container derivations (7 `rebind()` + 10 `unmonitored`, listed in the spec's Resolved via Codebase). Each Property is declared once, in its facet. Each Query sits on the base machine, and each realization lists the Queries it runs. Each realization realizes the base machine.

**Size:** L
**Files:** the activity's system claim containers (`Completion`, `RetryFailures`, `Cancellation`, `Pausing`, `Timeouts`, `CompetingTimeouts`, `Heartbeat*`, `ByID*`, `Reset*`, `DispatchEligibility`, …), their realizations, `ActivitySubjectRegression` and other tests that pin these objects, regenerated `model/ir/*`, `model/cases/*`, and the pinned fixture trees (in Batch 5b)
**Touches:** [model/temporal/features/activity/standalone/**/*.scala, model/ir/*, model/cases/*, tests/testcore/testpilot/testdata/generated/*, tools/canary/casebinding/testdata/*]

### Approach
- **Properties:** move each container's Properties to their facet (fn-161), or to `ActivitySystem`'s properties if facets haven't landed. Merge the duplicates into one declaration each, with the owner's names (2026-10-10): `completes` ×6 → one `completes` in Completion; `scheduleToStartFires` ×2 and the scheduleToClose timeout ×2 → `scheduleToStartFires` and `scheduleToCloseTimesOut` in Timeouts; the cancelRequested predicate ×2 → `cancelRequested` in Cancellation.
- **Queries:** move each Query next to its Property on the base machine, unchanged in scenario, limits, total and expectation. List it in the `runs` of the realization that runs it today. Where its container had no realization (`ResetSettlement`, `ResetKeepingPause`, `CompetingTimeouts`), no realization names it, so it stays model-only.
- **Realizations:** each realizes `ActivitySystem` (or the behaviour-changing derivation it needs). Keep the controller, worker and evidence bodies unchanged. Follow fn-160.4's naming rule if it has landed.
- **Drop the containers.** Dropping the `unmonitored` derivation shouldn't change answers or totals. fn-162.1 inferred this without verifying; check it with fn-160.5's step tables.
- **Tests:** replace the subject regression test (which pinned the 7 `rebind()` subjects' 14 Property truth tables) with one pinning each Query's realization binding and each merged Property's truth table.
- **Rename map:** record every moved id for fn-160.5: Property and Query ids, and the target, state, action and fact ids that include a removed machine's name.

### Batching
- Batch 1d (see MILESTONES.md): the Scala migration lands now, verified by the Scala build, model tests and fn-160.5's step tables under the rename map. The IR and Cases are regenerated once in Batch 5b, whose shared close supplies the gate evidence. Ideally lands before fn-160.3 splits files.

### Quick commands
```sh
make umpire-check-model
make umpire-check-lint
make umpire-check-fixtures
```

### Current strict native closure debt

The fn-140.4 independently frozen pre-edit source campaign observes an actual failure of
ActivitySubjectRegression's same-when all-row predicate distinctness assertion. The reported
pair is Cancellation's cancelRequested (phase and statusCancelRequested fact) versus explicit
activityProtocol.cancelIsRequested (statusCancelRequested fact), both on client.requestCancel.
The assertion compares truth over the complete native row population; blank native val names
alone do not explain its failure. Confirm the exact pair and independent row semantics before
repairing this task's already-required cancelRequested deduplication. Preserve Query names,
paths/limits/live expectations and realization memberships, map the actual retained Property
identity and all affected readers explicitly, and retain an independent duplicate-pair control.
Do not exclude a Property or weaken the distinctness assertion to obtain green.

Fn-140.4 deliberately leaves this earlier obligation unchanged while proving its own seven
witness conversions. Its baseline/post logs and frozen inventories are provenance, not this
task's closure. Repair and verify the Scala portion before fn-140.10's final source seal;
managed IR/Cases/Go checks remain held for Batch 5b. Neither an unchanged inherited red nor
native evidence closes the deferred consumer/Case proof.
## Acceptance
- [ ] The standalone activity has no claim-container derivations; each Property is declared once; each Query is run by the realization that ran it before (listed in its `runs`), or by none (model-only) as before.
- [ ] Every Query answer and total and every Case expected outcome is unchanged; differences are limited to the ids in the recorded rename map (Property, Query, realization binding, derived Case names, machine-scoped target/state/action/fact ids).
- [ ] The subject regression test is replaced by realization-binding and merged-Property pins; the gate checks pass at Batch 5b's close.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
