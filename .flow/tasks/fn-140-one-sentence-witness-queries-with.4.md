---
satisfies: [R5, R6]
---
# fn-140-one-sentence-witness-queries-with.4 Prove the first Activity witness migration and remove duplicate terminate

Touches: [model/temporal/features/activity/standalone/system/System.scala, tools/umpire/check/activity_system_test.go, tools/umpire/lower/activity_cases_test.go, tools/umpire/conformance/played_test.go, model/ir/**, model/cases/**]

## Description
Apply the first real Model conversion and establish R5's semantic pin before the remaining Model migration. This task owns its baseline and regenerated proof outputs. Documentation task .5 may run beside it because it changes no production Model or managed output.

**Size:** M
**Files:** Activity `system/System.scala`; existing Activity check/lowering/assessment regression tests; affected generated IR/Cases.

### Approach
- Begin only after `fn-151-split-standalone-activity-into-smaller` and then `fn-155-name-the-standalone-activitys-repeated` close. Re-read that closed source and receipt baseline. Re-anchor the file paths, owners and line references below at execution using the actual split-owner and abstraction identity maps. Inventory pinned finds and inbound Property/Scenario references before choosing conversions. Preserve the strict inherited completion, fatal-failure and pause/resume assertions and record their red results separately from migration regressions. Their correction remains in Batch 5; this task does not wait for that batch's closure or count its inherited failures as passing evidence.
- Convert only eligible unshared path/fact claims. Shared completion/pauseResume Properties and cancellation Scenarios retain their triple until all their readers make them unshared. Preserve invariant and monitor-only forms.
- For retry, first replay the baseline assessment with the records-only claim as a disposable differential, then retain the full terminal-state equality via ends where needed. The existing `completedOnRetry` value includes the deadlines, so comparing only attempts is insufficient. Never alter expected assessments to fit the conversion.
- Delete the hand-written terminate Query, its otherwise-unread Property/Scenario and its Case. Keep the capability-generated terminateSettles Query and all its dependencies. Update tests whose literal inventory currently counts the duplicate.
- Record before/after complete Query receipts, tables, live expectations and Case filenames keyed by unchanged Query name. For each rewritten Property function, prove equal truth values on every applicable model row; map edited source positions explicitly to their declaration spans, never remove positions generically. Classify each regeneration difference under R5/R6; an unexplained semantic or artifact change stops this proof. Do not rewrite historical recorded Runs.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/system/System.scala:280-440` - Properties, capabilities and pinned Queries.
- `tools/umpire/check/activity_system_test.go` - Activity semantics pins.
- `tools/umpire/lower/activity_cases_test.go:138-240` - inventory and retry Case members.
- `tools/umpire/conformance/played_test.go:251-299` - independent baseline assessments.
- `model/cases/manifest.json` - Case membership and expected assessments.
**Optional:**
- `model/temporal/features/activity/standalone/system/Record.scala` - neighboring monitor-only claims.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check ./tools/umpire/lower ./tools/umpire/conformance -run 'Activity|Retry|LoweredActivity'
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases
```
## Acceptance
- [ ] An explicit eligibility inventory identifies each converted and retained Activity triple with its inbound readers (R5).
- [ ] Records-only/full-state retry differential is recorded; final Query answers and live assessments match the closed fn-151 Activity split and subsequent fn-155 abstraction baseline. Strict inherited completion, fatal-failure and pause/resume assertions remain unchanged; their red results are recorded separately from migration regressions and remain Batch 5 obligations.
- [ ] Only hand-written terminate and its unshared declarations/Case disappear; generated terminateSettles remains and duplicate admission is tested (R6).
- [ ] Complete Query receipts, table semantics, expected assessments and filenames satisfy the recorded equivalence pin; rewritten Property functions have equal truth values on every applicable row and edited source positions have an explicit declaration-span map, with only R5/R6-authorized changes in generated outputs.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
