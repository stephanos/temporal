---
satisfies: [R2]
---
# fn-128-close-the-activitys-precision-gaps.2 Rejections are rows: failedPrecondition and invalidArgument

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R2 (comparison P1-3, P1-4). `Outcome` gains `failedPrecondition` and `invalidArgument`. Every (state, action class) the server answers with one of them becomes a rule with that outcome, citing the server code, in both `ActivityProduct` and `ActivitySystem`; the refinement maps outcomes by name. The seven `silent-rejection` acceptances in `model/ir/activity-standalone*.lint.json` are removed. `closedIsRejectedUniformly` keeps `notFound` for closed activities. A repeated `RequestCancel` in `cancelRequested` answers `failedPrecondition` (`model.go:201-202`).

Each realized control declares the gRPC code of each outcome through fn-133.1's `answers(…)` (read from the Run's `InstructionOutcome.protocol_code`), so the evidence confirms the rejection, not only that a call returned. Runs after task 1 so the rows are written against the dispatch field.
## Acceptance
- [ ] Each formerly silent pair is a row with its outcome and a citation in both levels; `make lint-model` reports no `silent-rejection` for the activity.
- [ ] The repeated-`RequestCancel` divergence is a Query or fixture answering `failedPrecondition`.
- [ ] `closedIsRejectedUniformly` still holds with `notFound`.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
Activity controls now answer forbidden live-state requests with `failedPrecondition`, and a worker answering canceled without a cancellation request receives `invalidArgument`. A repeated RequestCancel rejects without changing any state field or recording another request; closed controls retain `notFound`.

Tier: session (judge unavailable/no_key), AGENTS explicit override.
stage: impl-review - skipped(config: REVIEW_MODE=none)
Mandatory independent review, production IR/Cases generation, full gates and live verification remain conductor-owned at the fn128 -> fn138 -> fn129 batch boundary. No review verdict or full-gate receipt is claimed.

R2 evidence:
- Both ActivityProduct and ActivitySystem have cited rejecting rows for pause of paused/cancelRequested, unpause of scheduled/started/cancelRequested, repeated RequestCancel of cancelRequested, and respondCanceled of started. System also rejects pause of pauseRequested and respondCanceled of pauseRequested. The former backingOff/unpause pair is the scheduled/backoff state after fn128.1. Authority is chasm/lib/activity/model/model.go:171,201-202,232,251 and operator_commands.go:249-255,306,356.
- Product started merges concrete started and pauseRequested. Its pause row retains the accepted transition and adds the named pauseAlreadyRequested rejection alongside pauseApplied. This carries System's new pauseRequested rejection by matching outcome and projected state. System's accepted unpause of pauseRequested remains a refinement stutter. No refinement mapping or visibility declaration changed.
- ActivityRejectionRegression exercises the repeated-RequestCancel fixture from a held attempt through the first accepted request to the rejected repeat. It checks unchanged full states and empty facts across every finite assignment of the rejected System phases, pins both Product pause alternatives, and evaluates closedIsRejectedUniformly with NotFound and a deliberately wrong FailedPrecondition outcome.
- StandaloneActivityPins keeps full step equality over every state and class. Its explicit expectations add the intended rejecting rows and preserve the accepted pause result. The shared umpire.outcomes enums and Temporal rejectionCodes table already provide FAILED_PRECONDITION and INVALID_ARGUMENT; no duplicate enum, answers API, or code map was introduced, and no Go consumer changed.
- The three activity sidecars drop Product's live-control silent-rejection acceptance and remove the now-enabled respondCanceled phases from worker subjects. Both System sidecars remove the live pause/unpause subjects, including the historical unpause/backingOff subject. Acceptances for unstarted controls, repeat starts, stale worker responses, worker stopping and unrelated queue/admission behavior remain. Their fn128.1 stale phase/class spellings are reconciled at batch generation.

Expected batch artifact delta for R6 (source expectations, not regenerated IR proof):
- activity-standalone.json and activity-standalone-record.json gain 576 ActivitySystem rows (8 new phase/action pairs times 72 assignments of dispatch, attempts and deadlines). The 72 cancelRequested/RequestCancel rows replace accepted + statusCancelRequested with rejected(failedPrecondition) and empty facts. Product in all three activity IR files gains 6 rows and one rejected alternative in its existing started/pause row; its cancelRequested/RequestCancel row changes outcome and facts. Catalogs, action classes, state maps and static Query totals are unchanged from fn128.1. Linked tables/fingerprints/source positions change accordingly.
- Source Query names, schedules, Properties and expectations are unchanged. Their answers are expected to retain their current standings because the added rejections are state-preserving; this expectation is checked by the batch gates. No Query or Case is added by R2. The repeated-cancel refusal is pinned by the Scala fixture above.
- Existing standalone Cases completion, nonRetryableFailure, retry, pauseResume, scheduleToStartTimeout, terminate, activitySystem.cancelIsRequested and activitySystem.terminateSettles will carry regenerated linked table fingerprints and provenance. fn128.1's startDelayedCompletion Case will incorporate the same System change. The race Cases heldDispatch.staleDelivery and lostStartAnswer.committed carry the changed Product linkage/source positions while retaining their action paths. Manifest and lifter goldens are reconciled once at the batch boundary.
- Generic conformance already compares each realized rejection's protocol code through the exported exhaustive RejectionCodes table. The existing realization bindings remain unchanged; production artifacts will carry the new rows through that contract at generation.

Verification:
- baseline: green via fn1281 handoff at c0a3c09b33. The sole path committed between that checked revision and this task's c2ea6eda6ff73594e80ac6c09ce48073d8daefbe base was the prior .flow task receipt. The inherited framework command's 44 tests retain unchanged inputs. Row edits invalidated the inherited Activity checks, which were run again.
- The new suite at reproduction commit 6ee88fb703 failed with four intended behavioral failures (missing live-control rows, missing Product rejection alternative, missing canceled-response rejection, accepted repeated cancel); its closed NotFound test passed. An earlier compile error in the new test was corrected before this behavioral red observation and is not counted as defect evidence.
- `timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*Activity*' --require-tests` exited 0 with 15 tests. Log: .flow/tmp/activity-batch/fn1282-focused.log.
- `timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.RoleRefinements --require-tests` exited 0 with 3 tests. Log: .flow/tmp/activity-batch/fn1282-refinements.log.
- Touched Scala files passed focused scalafmt --check, and git diff --check passed. Gate classification returned FULL. Model lint against regenerated IR, the full Model/Go/fixture/canary gates and live Cases have not run and remain batch-deferred. No Go tests were needed for unchanged consumers. IR lint acceptance removals are source edits, not a claim that stale checked-in IR now passes lint.

Defect route:
- prior fixes: local activity history has no competing rejection-row fix; the other agent branches contain isolated scheduling/move work. Memory search/read found the admission-evidence note, unrelated to these RPC refusal paths. PR/issue checks were unchecked because gh returned HTTP 401 Bad credentials.
- diagnosis: executing the bound rows confirmed that forbidden controls/respondCanceled returned Nil and repeated cancellation returned accepted with another statusCancelRequested fact. The same tests now observe explicit unchanged-state rejections.
- introduced by: skipped because no known-good revision for these Model precision gaps was identified.
- base: 6ee88fb703 fails four rejection regressions; head: a10f745d1b passes all 15 Activity tests and 3 role-refinement tests.
- live: not run because MILESTONES.md assigns one live run to the batch boundary.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6ee88fb703, a10f745d1b
- Tests: baseline: green via fn1281 handoff at c0a3c09b33 (only .flow task receipt changed before this task), timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.ActivityRejectionRegression --require-tests (RED: four intended behavioral failures before fix), timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*Activity*' --require-tests (exit 0, 15 tests), timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.RoleRefinements --require-tests (exit 0, 3 tests), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/temporal/features/activity/standalone/product/Product.scala model/temporal/features/activity/standalone/system/System.scala model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala model/temporal/features/activity/standalone/ActivityRejectionRegression.test.scala (exit 0), git diff --check (exit 0), DEFERRED: production IR/Cases generation, mandatory independent review, full Model/Go/lint/fixture/canary gates and live Cases at fn128 -> fn138 -> fn129 batch boundary
- PRs: