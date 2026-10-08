---
satisfies: [R4]
---
# fn-129-activity-coverage.4 Exploration on the activity's find Queries

## Description

Measure every activity FIND against the integrated heartbeat, by-ID and reset source after .3. Enable bounded `.explore` only for added witnesses; retain pinned Scenarios when there is no additional evidence. Commit the focused source decision and inventory, not production generated artifacts.

**Size:** M
**Files:** activity Query/Scenario authoring and co-located bounded-search tests; focused lifter/lowering refusal fixtures only if the new candidates expose a coverage gap; Model documentation and retained evidence.
**Touches:** [model/temporal/features/activity/standalone/**, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/realizationRefusals/**, tools/umpire/realization/carriers_test.go, tools/umpire/lower/activity_cases_test.go, tools/umpire/lower/waits_test.go, model/README.md, .flow/tmp/activity-batch/fn129-exploration/**]

## Approach

- Inventory every current activity FIND by Query/Property/Scenario identity after .3, including new heartbeat/by-ID/reset rows. Compare its pinned witnesses with a finite, explicitly bounded exploratory search. Retain the original final action and authored expected assessment/reason. Preserve ordinary fatal FIND and all old retry Property-only explanationsDisagree expectations exactly.
- Record candidate count, added distinct witnesses, visited/search work, elapsed time, finite variation/state/depth/candidate bounds and per-candidate realization/lowering result. Distinguish a rejected or bound-exhausted candidate from runnable evidence; retain located refusal reasons. Test that variation cannot lose final action or cross the declared domain/identity.
- Add `.explore` only where the measurement finds useful missed witnesses within the existing explorer. Otherwise leave the Query pinned and retain the measurement. Do not widen default Profiles, rewrite the explorer, add dependencies or force coverage by changing Property semantics/expectations. R4's final live mapping may use an added admitted candidate or the deliberately retained pinned Query in the same shared suite.
- Scratch lift/lower the exact final current source, keeping unfiltered IR, ordered receipts and admitted Case/Program/Contract/Query identities. Produce the new-query/case and source-delta inventory for the conductor. Preserve fn-138's sealed original/adopted baseline as a separate proof. Focused recording/replay fixtures preserve exact support/status/reason; no native fixture is represented as a new live Case.
- Hand the final integrated source and measured inventory to parallel fn-128.7/.8 corrections. Both must rescan new reset/by-ID/heartbeat rows before fatal classification and final Profile-charge proof; they join before fn-128.6/fn-129.5. No cross-spec source close or extra full/live pass is required at this handoff.

## Investigation targets

**Required** (read before coding):
- `model/temporal/features/activity/standalone/system/System.scala` (Properties/Scenarios/Queries) and `model/temporal/features/activity/standalone/system/Realization.scala`.
- Activity FIND Query declarations and `.explore` call sites found in `model/temporal/features/activity/standalone/**`.
- `model/umpire` finite search/variation/final-action admission and co-located tests.
- `model/irgen/Realizations.scala`, `tools/umpire/realization/carriers.go` and `tools/umpire/lower/activity_cases_test.go`.
- Retained `.1-.3` original commits, scratch receipts, new Cases and declared refusal inventories.

## Quick commands

```bash
mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*Activity*' --require-tests
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/realization ./tools/umpire/lower -run 'Activity|Carriers|ReadsWait'
```

Run bounded search measurements and targeted explorer/variation fixtures with actual commands/nonzero counts; serialize heavy scratch commands with the shared flock. No production generation, full suites or live acceptance here.

## Acceptance

- [ ] Every activity FIND is listed as exploring with its added witnesses or pinned with measured absence of added evidence. Search work, elapsed time, bounds, final action and candidate refusal standings are retained.
- [ ] No invalid/unrealizable/bound-exhausted candidate supplies coverage credit. Finite search/final-action/identity negative fixtures pass, and original fatal/retry expected outcomes/reasons remain unchanged.
- [ ] Final-source scratch lift/lower and focused record/replay retain exact Query/Case/Program/Contract identities, new Case inventory and original commit. Production artifacts/full/live acceptance stays at .5; fn-128.7/.8 receive the new rows and exact final Profile/source inputs.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
