---
satisfies: [R5, R6]
---
# fn-140-one-sentence-witness-queries-with.9 Prove deferred Activity witness assessments and Case equivalence

## Description
**Recut 2026-10-18 (one-language series):** Activity Case equivalence is proven under fn-203's single regeneration and fn-204's descriptor.

**Size:** M
**Touches:** [tools/umpire/check/activity_system_test.go, tools/umpire/lower/activity_cases_test.go, tools/umpire/conformance/played_test.go, model/ir/**, model/cases/**]

Owner-deferred consumer proof split from .4. No Go command, consumer edit, managed regeneration, full gate, review or live run is admitted until the owner reopens that lane. Scala source task .4 preserves the source baseline, eligibility/reader inventory, full-state predicates, source-span and identity maps for this proof; it does not substitute native execution or scratch lifting for consumer assessment evidence.

Run the original first-Activity proof below on the integrated source and retained before-migration baseline. Re-anchor historical System paths to the current Protocol/facet files; use .4's complete mappings, not inferred suffix renames. Preserve every original differential, receipt, Case membership, expected assessment and protected assertion. Keep the capability-generated terminateSettles Case; only the named hand-written terminate Case disappears. Historical recorded Runs are immutable.

### Retained original proof contract
Apply the first real Model conversion and establish R5's semantic pin before the remaining Model migration. This task owns its baseline and regenerated proof outputs. Documentation task .5 may run beside it because it changes no production Model or managed output.

**Size:** M
**Files:** Activity lifecycle and focused subject files listed in Touches; `ActivitySubjectRegression.test.scala` and the fatal-Property references in `ActivityResetRegression.test.scala`; existing Activity check/lowering/assessment regression tests; affected generated IR/Cases.

### Approach
- Begin only after `fn-151-split-standalone-activity-into-smaller`, then `fn-155-name-the-standalone-activitys-repeated`, and then `fn-156-let-the-scala-compiler-and-lint-enforce` close with Flow closure committed in the integrated local branch. Re-read that closed source and receipt baseline. Re-anchor the file paths, owners and line references below at execution using the actual split-owner and abstraction identity maps. Inventory pinned finds and inbound Property/Scenario references before choosing conversions. Preserve the strict inherited completion, fatal-failure and pause/resume assertions and record their red results separately from migration regressions. Their correction remains in Batch 5; this task does not wait for that batch's closure or count its inherited failures as passing evidence.
- Convert only eligible unshared path/fact claims. Inventory the actual split owners and all inbound readers. Completion's actual single `completes` Property is shared by its completion Query and Pausing's pauseResume Query; Cancellation still has two Queries reading `cancelRequestedThenCanceled`. Retain a triple when its actual readers require it. Update regression references to converted declarations while preserving their independent predicates, exact receipt assertions and negative checks. Preserve invariant and monitor-only forms.
- For retry, first replay the baseline assessment with the records-only claim as a disposable differential, then retain the full terminal-state equality via ends where needed. The existing `RetryFailures.states.completedOnRetry` value includes the deadlines, so comparing only attempts is insufficient. Never alter expected assessments to fit the conversion.
- Delete the hand-written terminate Query, its otherwise-unread Property/Scenario and its Case. Keep the capability-generated terminateSettles Query and all its dependencies. Update tests whose literal inventory currently counts the duplicate.
- Record before/after complete Query receipts, tables, live expectations and Case filenames keyed by unchanged Query name. For each rewritten Property function, prove equal truth values on every applicable model row; map edited source positions explicitly to their declaration spans, never remove positions generically. Classify each regeneration difference under R5/R6; an unexplained semantic or artifact change stops this proof. Do not rewrite historical recorded Runs.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/system/System.scala` - Properties, capabilities and pinned Queries.
- The focused subject files listed in Touches - pinned claims and their current owners.
- `model/temporal/features/activity/standalone/ActivitySubjectRegression.test.scala` and `ActivityResetRegression.test.scala` - direct Property/Scenario readers and independent semantic assertions.
- `tools/umpire/check/activity_system_test.go` - Activity semantics pins.
- `tools/umpire/lower/activity_cases_test.go:138-240` - inventory and retry Case members.
- `tools/umpire/conformance/played_test.go:251-299` - independent baseline assessments.
- `model/cases/manifest.json` - Case membership and expected assessments.
**Optional:**
- `model/temporal/features/activity/standalone/system/Dispatch.scala` - neighboring monitor-only claims.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check ./tools/umpire/lower ./tools/umpire/conformance -run 'Activity|Retry|LoweredActivity'
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases
```

### Actual protected-assertion handoff from .4

The completion/fatal-failure/pause-resume path-assessment assertions are Go tests, not the Scala
Quick suites. Source task .4 preserves their bytes, identifies the inherited receipts, and freezes
its independently pinned pre/post source/scratch IR/replay inputs. Here, independently execute
those exact protected assertions on BOTH frozen .4 populations using the actual checker/assessment
route, retaining their individual failure status separately from new migration regressions.
Source bytes, an old receipt, native predicate equality or a passing unrelated native suite do not
satisfy this execution obligation. Do not rewrite expected assessments to turn inherited reds green.
The retained original source-operation prose describes .4's migration evidence; consume that frozen
migration rather than re-performing or broadening source edits outside this task's consumer Touches.

### Historical receipt availability (verified by conductor)

The surviving historical disposition is preserved in main checkout
`.flow/tmp/fn145-integrated-done-summary.md`, `fn145-integrated-done-evidence.json` and
`fn145-integration-evidence.md`, and the retained fn-145.4 Flow Done summary. The referenced
original `/tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454` detailed receipt directory is absent.
Pin the surviving summary bytes and identify that missing-original limitation; summaries
are not authenticated underlying execution receipts. They record canonical exit1, completion
failed, fatal unfinished/OOM and pause unreached in that particular run, not three observed
assertion failures. Fn-151.3's later candidate canonical has no RUN/terminal for the trio.

Preserve the actual assertion route
`tools/umpire/conformance/played_test.go: TestALoweredActivityCaseRunsLiveAndReplaysAlike`
and its existing current-source replay seam, without running Go while held. The .9 consumer
proof must execute BOTH frozen pre/post populations afresh and preserve actual per-child
results. No historical summary or unrelated generated-Case test supplies that proof.
## Acceptance
- [ ] An explicit eligibility inventory identifies each converted and retained Activity triple with its inbound readers (R5).
- [ ] Records-only/full-state retry differential is recorded; final Query answers and live assessments match the committed fn-156 closure baseline, including the closed fn-151 Activity split and subsequent fn-155 abstraction mapping. Strict inherited completion, fatal-failure and pause/resume assertions remain unchanged; their red results are recorded separately from migration regressions and remain Batch 5 obligations.
- [ ] Only hand-written terminate and its unshared declarations/Case disappear; generated terminateSettles remains and duplicate admission is tested (R6).
- [ ] Complete Query receipts, table semantics, expected assessments and filenames satisfy the recorded equivalence pin; rewritten Property functions have equal truth values on every applicable row and edited source positions have an explicit declaration-span map, with only R5/R6-authorized changes in generated outputs.

- [ ] Actual protected Go completion/fatal-failure/pause-resume assertions execute independently on both frozen .4 pre/post populations, with individual results and exact replay/input pins; source-only preservation and prior recorded receipts substitute for none of this proof.
## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: