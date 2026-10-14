---
satisfies: [R4, R7]
---
# fn-162-queries-name-their-realization-each.3 Claim containers removed: mini feature and shared code (standalone activity proof at the sweep)

## Description
**Scope (2026-10-12, Scala DSL batch rules; supersedes the text below where they conflict):** re-scoped to the mini feature and the shared code. The standalone activity's claim-container removal and cancellation deduplication are already integrated in source (3afd8b9f74 and follow-ups, Batch 1d); its identity/source proof, its strict native closure debt below, the `ActivitySubjectRegression` replacement and the regenerated IR/Cases/fixture trees wait for the owner's sweep and Batch 5b. Consumer closure stays deferred. This task: remove any claim container (a `Derived(m.unmonitored)` or bare `m.rebind()` holding only Properties and Queries) that `features/activity_mini` or the shared code (foundations, actors, capabilities, realize, `Bounds`) still has, or prove there are none; ship a scalafix rule under `model/migrate/rules/` if a source change is needed; and prove the mini identity and step tables equivalent to the mini baseline under this task's recorded rename map.

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
- [ ] `features/activity_mini` and the shared code have no claim-container derivations (removed, or proven absent with every derivation classified); each mini Property is declared once (the lifter's duplicate refusal passes on the whole mini lift); each mini find Query keeps its realization binding or model-only standing.
- [ ] The mini step tables are equivalent to `.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/` under the recorded rename map; any source change ships its scalafix rule, proven on copied mini sources.
- [ ] Deferred, not claimed: the standalone activity's identity/source proof, strict native closure debt and subject-regression replacement (owner's sweep), regenerated IR/Cases/fixtures and gate checks (Batch 5b), consumer closure.
## Done summary
Re-scoped to the mini feature and the shared code (chore commit 403bdad4fe). There are no claim containers there, so no source change, rename or scalafix rule was needed. The mini identity and step tables are equivalent to the mini baseline under an empty rename map.

baseline: no mini source change in this task; the tree is the fn-162.2 receipt state (a0a29ac3ef).
stage: impl-review - skipped (REVIEW_MODE=none; independent review runs once at Scala batch end)

Requirement evidence (R4/R7, mini reading):
- Inventory (`grep` for `Derived(`, `.rebind()`, `.unmonitored`, `DerivesFrom(` over features/activity_mini, foundations, actors, capabilities, realize, Bounds and framework, tests excluded). Every derivation changes behaviour. `DispatchProgress` restricts. `ScheduledQueue` restricts and extends. `TrustingAnswers`, `VolatileQueue` and `ForgetfulQueue` rebind an effect (NegativeControl). `LossyMatchingQueue` extends, refines and assumes. `TaskQueueProductUnderStorageLoss` extends and assumes. The one `unmonitored` derivation is `ProtocolMember`, a composition member with no sections, used by `Queue`. fn-162.1 classed its real-feature twins `RecordMember` and `TrustingRecordMember` as "not containers" on the same ground. No bare `.rebind()` exists. So there was nothing to remove, and no scalafix rule ships.
- Declared once: the whole-feature mini lift (`MiniRealizationFixtures`, the "activity-mini" lift, exit 0 in fn-162.2's suite-6.log on this same source tree) passes the lifter's duplicate-Property refusal. No two author Properties of one mini machine share a selector and predicate, cancellation included.
- Bindings and standings kept: `MiniActivityRun` still runs exactly `protocolCompleted` and `retryCompleted` (pinned by the existing "mini live expectations" test). The other mini finds (`completed`, `failedAnswer`, `staleAnswerFound`, `starts`) stay model-only, unchanged.
- Rename map: `.flow/tmp/fn162/task3/mini.map`, empty with its reason. No Property, Query, binding, Case name or machine-scoped id moves.
- Equivalence: `FEATURES=activity_mini ... --check-equivalence <abs mini-selected-baseline-00d22597d2>/ .flow/tmp/fn162/task3/mini.map --steps-only` exit 0, 16 step tables, "equivalent to the baseline" (equivalence.log). Baseline integrity.mjs PASS (16 keys, 11,325 lines, frozen jar; integrity.log).

Verification: the equivalence above; integrity.mjs; fn-162.2's MiniRealizationFixtures 8/8 on the unchanged tree.

Failures retained: none.

Open obligations, deferred and not claimed: the standalone activity's identity/source proof under the batch-1d rename map, the task's strict native closure debt (cancelRequested versus activityProtocol.cancelIsRequested in ActivitySubjectRegression) and the subject-regression replacement all go to the owner's sweep. Regenerated IR/Cases/fixture trees and gate checks go to Batch 5b. Consumer closure stays deferred. Go realization-binding consumers are in fn-162.4.

Deferred gates: Go/proto/regeneration, make gates, real-feature checks, broad validation and independent review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 403bdad4fe
- Tests: FEATURES=activity_mini timeout 1800 mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/check -- --check-equivalence /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/ /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn162/task3/mini.map --steps-only (equivalent, 16 tables), node .flow/tmp/scala-dsl/mini-selected-baseline-00d22597d2/integrity.mjs (PASS), derivation inventory grep over features/activity_mini and shared code (no claim containers)
- PRs: