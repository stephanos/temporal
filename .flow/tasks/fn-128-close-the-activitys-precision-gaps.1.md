---
satisfies: [R1]
---
# fn-128-close-the-activitys-precision-gaps.1 Dispatch is a field: replace the backingOff phase; start delay

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R1 (comparison P1-1). Current source after fn-132.2: `model/temporal/features/activity/standalone/Standalone.scala` owns the form; `product/Product.scala` and `system/System.scala` each own their local `Phase`, `State` and `Fact`. Actions are `client.start` and `client.control` on the `activity` entity. All three System realizations remain in `system/Realization.scala`. The kind header `features/activity/Activity.scala` declares `Timeout`, `TimeoutType`, `AttemptResult`, the worker's `poll` and `respond`, `timers` and `deadline` (fn-132.6); the form binds the worker actions to its `activity` entity. IR stems are `activity-standalone`, `activity-standalone-record` and `activity-standalone-race`. Machine names stay `ActivitySystem` and `ActivityProduct`; worker actions stay `worker.poll`/`respond`.

`system.State` gains `dispatch: Dispatch` (`now`, `startDelay`, `backoff`) and loses the `backingOff` phase. `client.start` gains a `startDelay: Timeout` input; `timers.startDelay` sets `dispatch = now`. A retryable failure sets `dispatch = backoff` and stays `scheduled`; `timers.backoff` sets `now`. Pause and unpause keep `dispatch` (`chasm/lib/activity/model/model.go:239-244`). `worker.poll` is enabled only while `dispatch = now`. Schedule-to-start is armed only while dispatchable and waiting (`model.go:312-322`). The refinement reads every waiting state as the Product's `scheduled`, whatever `dispatch` holds.

The realization sets the start request's start delay when the input is `expires`. Recompute the state count and any Limits/`total` the change moves.

Add a Query or fixture per divergence fixed: an unpause after a pause in backoff does not dispatch before the backoff timer; schedule-to-start does not fire while backing off.
## Acceptance
- [ ] No `backingOff` phase remains; `dispatch` is a `system.State` field and the refinement checks.
- [ ] Each of the two divergences is a Query or fixture that now refuses what the server refuses, citing `model.go`.
- [ ] `startDelay` is a start input with its timer, realized in the start request.
- [ ] Every table, Query answer and Case that changed is listed with why (for R6).
- [ ] The spec's Verification gates pass.
## Done summary
An unpause now retains an activity's pending dispatch delay, and schedule-to-start cannot fire during start delay or retry backoff. The start request exposes and realizes `startDelay`; delay timers can expire while paused, and a retry following a pause request keeps its backoff.

Tier: session (judge unavailable/no_key) with AGENTS implementer override.
stage: impl-review - skipped(config: REVIEW_MODE=none)
The conductor owns mandatory review, generation, full gates and live verification at the activity batch boundary in MILESTONES.md. No review verdict or full-gate receipt is claimed here.

R1 evidence:
- `system.State.dispatch` has `now`, `startDelay` and `backoff`. `Phase.backingOff` is removed; all waiting states refine to Product `scheduled`.
- `worker.poll` requires scheduled and dispatch now. `effects.backOff` sets scheduled/backoff; pause and unpause keep dispatch. `effects.backOffPaused` retains backoff when a held attempt yields to a pause request. Source authority is `chasm/lib/activity/model/model.go:130-137,239-244,355-376`.
- Schedule-to-start requires scheduled and dispatch now (`model.go:312-322`). Schedule-to-close excludes start delay because its clock starts at first dispatch (`model.go:325-335`).
- The realization's existing `deadlines` module sets `StartActivityExecutionRequest.start_delay` for the expiring input. Its derived server-step rule recognizes `timers.startDelay` by the input's name and uses the existing deadline bound. Both delay timers are unobservable refinement stutters.
- `ActivityDispatchRegression` covers both reported refusals, start-delay dispatch/deadline arming, delay expiry while paused, retry backoff after a requested pause, and the waiting-state projection. `StandaloneActivityPins` retains full equality over every state and action class with the deliberate R1 expectations updated. `RoleRefinements` checks closedness of ActivitySystem, ActivityRecord and HeldDispatch.
- Four typed action inputs required extending the existing positional calls, core binding and rule binding overloads in Action.scala, Machine.scala and Syntax.scala. The four-input named/positional test and the activity's 16 start classes cover that support. No lifter or kit interpretation was changed.

Expected batch artifact delta for R6 (not regenerated or measured from IR yet):
- `activity-standalone.json` and the ActivitySystem carried by `activity-standalone-record.json` change their System table, class IDs, dispatch-delay rows, deadline guards, refinement map and fingerprints. ActivitySystem has 792 states (11 phases x 3 dispatch values x 3 attempt counts x 8 deadline assignments), previously 288. Its start has 16 classes, previously 8; with the new timer its bound class catalog has 31, previously 22. StandaloneActivity has 1584 catalog states, previously 576. Product and admission/queue/worker tables retain their behavior.
- System pinned Query totals become 792 times their scheduled slots. Existing three-slot Queries total 2376, four-slot Queries 3168, `retry` 4752, and the five-slot `pauseResume` 3960. The record's two `competingTimers` Queries total 1584 each, and the composed `stoppedWorkerStartsNothing` totals 9504. Existing Limits remain sufficient and no source total literal required editing.
- New verify Queries `delayedAttemptsAreNotDispatched` and `scheduleToStartWaitsForDispatch` should answer verified-within-limits at eight steps; each static total is 196416. New find Query `startDelayedCompletion` should answer found at four steps, total 3168. Existing Query answers and live expectations are expected to retain their standings.
- New Case `activity-standalone-startDelayedCompletion-case.json` should include start_delay, the derived startDelay timer bound and its declared unobservable gap. Existing Cases `activity-standalone-{activitySystem.cancelIsRequested,activitySystem.terminateSettles,completion,nonRetryableFailure,pauseResume,retry,scheduleToStartTimeout,terminate}-case.json` change start class IDs, state/table bindings, provenance/fingerprints and totals; their scheduled action paths remain the same.
- `activity-standalone-race.json` and its Cases `activity-standalone-race-{heldDispatch.staleDelivery,lostStartAnswer.committed}-case.json` may change source positions from the shared realization's inserted field, with their behavior retained. The manifest and lifter goldens `hints.json` / `hintsRefused.json` carry the resulting source/Model changes. Lint subjects containing backingOff or three-input start classes must be reconciled at batch generation; fn128.2 owns rejection-row changes. Fixture consumers are checked and regenerated once at that boundary.

Verification:
- baseline: green. `mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.StandaloneActivityPins --require-tests` exited 0 with 4 tests before edits. An earlier pipe-separated glob ran no tests and was discarded as inconclusive; repeating --test-only was rejected by the CLI. Neither observation was treated as a pass.
- The committed reproduction at dafe9fea1f exited 1, with both tests failing at the expected Nil comparison. Final activity test command with `--test-only "*Activity*" --require-tests` exited 0 with 10 tests.
- Framework test command over model/project.scala and model/umpire with `--require-tests` exited 0 with 44 tests. RoleRefinements exited 0 with 3 tests. Changed Scala files passed the focused scalafmt --check, and git diff --check exited 0.
- Gate classification returned FULL. Generation, model lint/full gates, fixtures, live cases and review remain batch-deferred by explicit milestone policy. Lifter golden tests depend on the deferred packaged Model/golden update and were not represented as green.
- Logs live under `.flow/tmp/activity-batch/fn1281-{baseline-pins,regression-base,focused,framework,refinements}.log`.

Defect route:
- prior fixes: local activity history and branches have no competing dispatch fix; memory search found an admission-evidence note unrelated to these refusal paths. GitHub PR/issue checks are unchecked because gh returned HTTP 401 Bad credentials.
- diagnosis: the unchanged rules immediately poll the resumed scheduled state and arm schedule-to-start for the Waiting backingOff phase. Both were confirmed by executing their bindings; orthogonal dispatch removes those permissions while keeping timer expiry available.
- introduced by: skipped because no known-good revision for either Model precision gap was identified.
- base: dafe9fea1f fails both refusal fixtures; head: c0a3c09b33 passes those fixtures and the remaining focused checks.
- live: not run because MILESTONES.md assigns one live run to the batch boundary.

Remaining batch work includes the full IR refinement check and inspection of the expected generated changes above. The next precision task adds rejection rows; the retry-policy task will require a fifth typed input if it follows the current start signature.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: dafe9fea1f4b94dfd82845acb36a36a7d514a552, c0a3c09b333294d699ad07d7ac00a09de32d333a
- Tests: mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.StandaloneActivityPins --require-tests (baseline exit 0; 4 tests), mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.ActivityDispatchRegression --require-tests (reproduction on dafe9fea1f exit 1; 2 intended failures), flock /tmp/umpire-heavy-gates.lock mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only "*Activity*" --require-tests (exit 0; 10 tests), flock /tmp/umpire-heavy-gates.lock mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire --require-tests (exit 0; 44 tests), flock /tmp/umpire-heavy-gates.lock mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only umpire.RoleRefinements --require-tests (exit 0; 3 tests), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/umpire/Action.scala model/umpire/Machine.scala model/umpire/Syntax.scala model/umpire/Inputs.test.scala model/temporal/features/activity/Activity.scala model/temporal/features/activity/standalone/Standalone.scala model/temporal/features/activity/standalone/system/System.scala model/temporal/features/activity/standalone/system/Realization.scala model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala model/temporal/features/activity/standalone/ActivityDispatchRegression.test.scala (exit 0), git diff --check (exit 0), BATCH_DEFERRED: generation, full model/Go/lint/fixture/case gates, review and live run; MILESTONES.md activity batch fn128→fn138→fn129
- PRs: