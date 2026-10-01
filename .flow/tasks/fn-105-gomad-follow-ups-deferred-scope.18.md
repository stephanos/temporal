---
satisfies: [R18]
---
# fn-105-gomad-follow-ups-deferred-scope.18 D18: investigate worker cancellation delivery and timeout budgets

## Description
Investigation approved on 2026-09-30 for TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout, following the seed-11 walkthrough and blanket approval of remaining investigations. The cancellation command is not received before the 90-second test context has less than the two-second long-poll minimum remaining, while Awaitf permits 120 seconds. Seed 17 passes. Identify why delivery is late and propose the correction; implementation remains a subsequent decision.

## Acceptance
- Reproduce the unskipped test on seeds 11 and 17 and compare native Go with Gomad; retain commands, identities, platform, virtual-time observations, and outcomes.
- Trace workflow timeout, cancellation generation, persistence/transfer, worker-control-queue delivery, and polling to locate the causal delay.
- Account for the 90-second parent context, five-second child polls, two-second server minimum, and 120-second Awaitf; establish which budget and delivery assumptions are valid.
- Record the cause, correction owner, proposed next action, and regression/qualification criteria in fn-105. Retain any needed fix as explicit open work for a subsequent decision.
- Preserve the cancellation-delivery assertion and keep the skip until verification supports removal; a longer timeout or classification alone does not resolve the issue.

## Done summary
Investigation only. Nothing is fixed, the skip stays, and the correction is open work in fn-105 for a subsequent decision.

Cause: `TestWorkerCommandsTaskSuite/TestDispatchCancelOnWorkflowTimeout` fails under Gomad because the server never creates the cancel command. Under the strict tick the workflow start and the activity schedule share one virtual instant. The server caps the activity's timeouts at the 3 s run timeout (`chasm/lib/activity/validator.go:197-212`), so the activity deadline equals the run expiration, and `CreateNextActivityTimer` skips an activity timer only when it is strictly later (`service/history/workflow/timer_sequence.go:127-132`). The `ActivityTimeoutTask` and the `WorkflowRunTimeoutTask` are due at the same timestamp and the seed picks the order. When the activity task runs first the activity closes as timed out, the workflow closes with no pending activity, and `GenerateActivityCancelCommandsForClose` generates no `WorkerCommandsTask`. No timeout budget and no delivery step is at fault; a longer timeout cannot help.

Evidence (darwin/arm64, toolchain key 8d28bd44, linux/amd64 not run):
- Gomad suite, strict tick: seed 11 fails at 120.00 s virtual in three runs, seed 17 passes at 3.00 s in two. Leaf alone: the roles swap (seed 11 passes, seed 17 fails, two runs). Leaf sweep seeds 1 to 16: 9 fail. Every failure processed the activity timer task first and created no WorkerCommands task; every pass processed the run-timeout task first and created one.
- Native: leaf 5 of 5 and suite 21 of 21 pass. The activity is scheduled 5 to 49 ms after the workflow starts, so the server creates no activity timer at all.
- Three interventions each remove the failure: `clock_tick: forward` (leaf 64 of 64 seeds; suite `qualified` on seeds 11 and 17 with the skip lifted in scratch manifests, untraced at repeat 2 and traced with exact replay), scheduling the activity 1 s into the run (throwaway copy, 16 of 16 strict, native 5 of 5), and skipping the activity timer on an equal deadline (throwaway copy, 16 of 16 strict).
- Budgets: `Awaitf` extends the 90 s default context to its 2 minute ceiling; 29 polls of 4 s run from 0.0 to 114.8 s; the 2 s long-poll minimum refuses 11 polls from 118.9 s; await and context expire together at 120.0 s. The 120 s await plus its 10 s reserve cannot fit under the 2 minute ceiling, which does not affect the outcome.

Proposed correction, owner Gomad qualification configuration (`tools/gomad3integration`): run the suite under `clock_tick: forward`, verify per the report's seven-step list, then remove the skip. Alternatives recorded with owners: a test change (test owner) and a server tie-break (history service owner, production review). The skip's recorded reason is inaccurate and should be replaced when the decision is taken.

Report: `docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md`. Evidence index: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d18-evidence.json`. Milestones D18 bullet and row and the research index updated. No product, test, runtime, or manifest file changed; no commit (commits forbidden for this task).

Left behind: two prepared-target cache entries of about 147 MB each under `tools/gomad3/.toolchain/builds/8d28bd44.../prepared-targets` (created 23:22 and 23:25 on 2026-09-30) probably belong to the deleted throwaway copy; they were not removed because the cache is shared.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) - 3 rounds, final verdict SHIP; one round-3 citation nit applied after the verdict and not re-reviewed
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: none (the spec defines no Quick commands), platform darwin/arm64, macOS 26.6.2, gomad toolchain go1.27.1 build 8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc; linux/amd64 not run (unavailable), gomad explore --seeds 11,17 --clock-tick=strict ... go-test ./tests -- -test.run=^TestWorkerCommandsTaskSuite$ -test.parallel=8 -test.v [exit 1: seed 11 FAIL 120.00s virtual (3 runs), seed 17 PASS 3.00s (2 runs)], gomad explore --seeds 11,17 --clock-tick=strict ... -test.run=^TestWorkerCommandsTaskSuite$/^TestDispatchCancelOnWorkflowTimeout$ [exit 1: seed 11 PASS, seed 17 FAIL; 2 runs], gomad explore --seeds 1-16 --clock-tick=strict ... leaf [exit 1: 7 pass, 9 fail], gomad explore --seeds 11,17 --clock-tick=forward ... suite [exit 0: 2 pass], gomad explore --seeds 1-64 --clock-tick=forward ... leaf [exit 0: 64 pass], gomad qualify-set scratch manifest, skip lifted, strict [exit 1: seed 11 target_failure, seed 17 qualified], gomad qualify-set scratch manifest, skip lifted, clock_tick forward [exit 0: seeds 11 and 17 qualified, repeat 2], gomad qualify-set scratch manifest, skip lifted, clock_tick forward, traced [exit 0: seeds 11 and 17 qualified, exact choice replay], go test -tags test_dep ./tests -run '^TestWorkerCommandsTaskSuite$/^TestDispatchCancelOnWorkflowTimeout$' -count=5 -p 2 -timeout 20m -v [exit 0: 5 of 5 PASS], go test -tags test_dep ./tests -run '^TestWorkerCommandsTaskSuite$' -count=3 -p 2 -timeout 20m -v [exit 0: 21 of 21 PASS], throwaway copy, variant B (activity scheduled 1 s into the run): gomad leaf seeds 1-16 strict [exit 0: 16 pass]; native leaf -count=5 [exit 0], throwaway copy, variant C (activity timer skipped on an equal deadline): gomad leaf seeds 1-16 strict [exit 0: 16 pass], investigation receipt: no fix applied, the tracked skip is unchanged; make lint-code-fast not run (no Go source changed), flowctl gate classify --base 7ef3800a2c: FULL, triggered by another task's uncommitted .github/workflows/gomad3-smoke.yml; this task changed only docs and .flow artifacts, and the spec defines no gate commands to run
- PRs: