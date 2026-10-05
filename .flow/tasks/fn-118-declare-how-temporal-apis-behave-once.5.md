---
satisfies: [R4, R6, R8]
---
# fn-118-declare-how-temporal-apis-behave-once.5 Migrate realizations to derived waits and close fn-118 with Contract-freeze evidence

## Description
Delete the hand-written waits from both realizations, regenerate Cases, prove every Contract unchanged with the Program differences listed, and run the closing gates.

**Size:** M
**Files:** `model/temporal/features/standaloneactivity/Realization.scala` (`awaitStatus` and its callers), `model/temporal/features/nexuscaller/Realization.scala` (the two 250 ms polls and the two `timeoutMs = 5000` commands; line numbers in task 1's inventory predate fn-112.9 and fn-114.3, so re-locate them), `model/ir/**`, `model/cases/**`, lowering migration goldens' allowed-Program-delta list, `model/README.md`, `model/SEMANTICS.md`, `.plans/API_BEHAVIOR_HINTS.md` (after-numbers), `.flow/tmp/fn118-5/**`.
**Touches:** [model/temporal/features/standaloneactivity/Realization.scala, model/temporal/features/nexuscaller/Realization.scala, model/temporal/realize/**, model/ir/**, model/cases/**, tools/umpire/lower/testdata/**, tools/umpire/model/testdata/**, model/README.md, model/SEMANTICS.md, .plans/API_BEHAVIOR_HINTS.md, .flow/tmp/fn118-5/**]

### Approach
- Replace each explicit poll/interval/timeout covered by a hint with the plain read + condition; a wait no hint covers keeps its explicit form and is listed with its reason (R4). Keep task 4's final rule for explicit polls (accepted only with a recorded reason).
- Regenerate (`make umpire-gen-model`); compare against the baseline goldens: every Contract byte-identical, every Program difference listed (R6). A changed Contract stops the task.
- Recount polls and total declared wait budget with task 1's command; list candidate hints not adopted with the Case that would need each (R8).
- Run model gate, lint-model, Umpire and Testpilot Go tests, lint-code-fast and the live Case tests once (`make umpire-check-live-tests`).

### Investigation targets
**Required:**
- `.plans/API_BEHAVIOR_HINTS.md`
- both realization files above
- `tools/umpire/lower/migration_golden_test.go`

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model && make umpire-check-live-tests
```

### Execution constraints
- Contracts frozen; only Program waiting may change.
## Acceptance
- [ ] No realization contains a literal interval or timeout or an explicit poll a hint covers; remaining explicit waits are listed with reasons.
- [ ] Every existing Contract is unchanged per the baseline goldens; Program differences are listed.
- [ ] Done summary gives polls and wait budget before/after and the non-adopted candidates with the Case that would need each.
- [ ] Model gate, lint-model, Go tests, lint-code-fast and live Case tests pass; README/SEMANTICS document hints.
## Done summary
Behavior phase, closing task (R4, R6, R8): the realizations write no interval, no deadline and no explicit poll, and every Contract is unchanged.

What changed
- Kit: `await` writes no interval (`intervalMs = 0`), so every Temporal read derives its wait. The 250 ms `pollIntervalMs` is gone.
- Nexus caller: the two inert `timeoutMs = 5000` are deleted (handler replies are bare `NexusReply`s; `finish-workflow` writes no limit).
- activity-retry: `ServerStep(backoff, CauseKind.timer, firstRetryBackoffMs)`. The kit value is the server's default first retry interval, 1 s, cited. The read waits 14,000 ms.
- New visibility RequestCancelActivityExecution -> DescribeActivityExecution, at once, cited. With it the `cancel` and `cancelRequest` Queries lower to their recorded gaps instead of a refusal. Its R7 removal test refuses `cancel`.
- Recorded reason for an explicit poll: lint kind `explicit-wait` (`tools/umpire/lint`). A poll that writes its own interval, in a realization that declares a behavior, is a finding unless `<file>.lint.json` accepts it with a `because`. None exists.
- Regenerated model/ir, model/cases, the functional tree and the lifter fixtures' expected IR. The canary Case identity moved (58fb49bc… -> 32ec0ce1…): the policy is re-pinned, and the canary pinned Run was re-recorded live. The control record stayed current.
- Golden harnesses (R6): `original.json` gains `derived_waits` (23 commands).
  - The IR comparison (original baseline and migration goldens) drops exactly those commands' interval and timeout on both sides.
  - The Case comparison drops those instructions' `limits`, `waitHints`, `pollIntervalMilliseconds` and `once`, and re-derives a Query Case's identity from the projected bytes. Everything else, the Contract included, is compared byte for byte.
  - Tests show each entry is needed, no other command could be listed, and a Contract, provenance, listed-read or unlisted-limit change still fails.
- Docs: README, SEMANTICS (hints sections), `.plans/API_BEHAVIOR_HINTS.md` "As built by task 5", and the spec's API Contracts.

Contract freeze: every Case's `.contract` and every byte outside `.program` is identical to base. Program differences (`.flow/tmp/fn118-5/program-diff.txt`):
- Five reads become single reads: await-terminated x3, await-paused x2.
- await-completed and await-failed: 5 s (retry: 14 s).
- await-timed-out: 5 s.
- Eight await-scheduled: 5 s each.
- pending-attempts: 7 s.
- Sixteen 5,000 ms reply/finish limits are removed.

R8, over the 20 Cases (same jq as task 1, now skipping read-once nodes):
- Before: 19 polls, at most 779 poll RPCs, 54 waits, 460,000 ms budget (80,000 ms of it the explicit limits).
- After: 14 polls plus 5 single reads, at most 338 poll RPCs, 33 waits, 271,000 ms budget, 0 explicit.

Remaining explicit waits, all non-reads with reasons:
- `await-close` long poll (blocking-read candidate).
- `AwaitLearned` and `AwaitCommand` (Driver waits that read no API).
- Fault, Hold and Release controls.
- `wait_new_event` (W-18) is left for the owner.

Candidates not adopted, with the Case that would need each:
- Not-yet errors: none.
- Per-command timeout: none.
- `retry` cause: a Nexus Case that reads across backoff.
- Repeatable call: none.
- Blocking read: the 8 Nexus `await-close` reads, and the activity status polls.
- Read-only call: none.
- Cost: none.
- Driver-await bounds: Nexus `await-completion-authority` and `await-nexus-operation`.

Decisions, and why
- Backoff is declared a timer, not a delivery (fn-118.4 suggested a delivery, 13 s), because the Model says it is a timer and the server schedules the retry by time. The delivery bound's citation no longer covers the backoff, and its value is unchanged.
- The RequestCancel pair is declared even though no Case needs it: an existing Query's path does, and without it the Model's lowering fails instead of reporting gaps.
- The explicit-poll reason uses the existing lint-acceptance mechanism, not an IR field: no proto change for a value no realization sets. The lowering keeps an explicit poll as written; lint refuses it at the gate.

Live tests (`make umpire-check-live-tests`, exit 2):
- The four ShutdownWorker-race Cases are INCONCLUSIVE, as known (MILESTONES "Open for the owner"): activity-activityProtocol.cancelIsRequested, activity-scheduleToStartTimeout, nexus-caller-scheduleToStartTimeout/hsm, WorkerOutageCase.
- Also failing, pre-existing and unrelated: TestTestpilotAssessRecordedRuns. The pinned control record now also gets `known-gap-blocking`. This reproduces offline with umpire-assess on unchanged inputs; umpire-assess depends on no changed package, and its offline test (run_test.go:128) already expects the reason. The live expectation at tests/testpilot_assess_test.go:103 is stale and was left for its owner.
- All other live tests pass.

Review: claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1: SHIP with 2 P3s, both applied in 801e5daa64:
- The comparative-Case test takes its derived set from `derived_waits`.
- The R7 table asserts each Query's standing exactly.
FYI left as is: ProjectCurrent clones twice, which is harmless.

Subagent: one opus subagent built the golden-harness extension (tools/umpire/internal/golden/**, the four migration/original tests).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b9f95a33a9, e7367ad941, 7a40c3d0dc, 6c6f875a23, 801e5daa64
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 108 s), make lint-model (exit 0, 76 s), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 235 s; .flow/tmp/fn118-5/go-suite.json), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|Migration|Original' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 140 s), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), make umpire-rerecord-pinned-runs (exit 0), make umpire-check-live-tests (exit 2: four known ShutdownWorker-race INCONCLUSIVE Cases plus pre-existing TestTestpilotAssessRecordedRuns; all others pass; .flow/tmp/fn118-5/live.log), flowctl claude impl-review --spec claude:claude-opus-5-5:high (VERDICT=SHIP, round 1)
- PRs: