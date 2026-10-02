# fn-105.19 (D19) review — raw codex bridge on the working-tree diff (commits forbidden), model gpt-5.6-sol, reasoning effort high, read-only

## Round 1

## Findings

- **SHOULD-FIX:** Several causal statements exceed the retained evidence’s scope. The report says incomplete backlogs contain “2 to 6 workflows,” yet unchanged runs include 0, 1, 7–14 workflows ([report](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md:8), [runs](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-gomad-runs.txt:52)). “On any seed” should be limited to unchanged seeds 1–17, and “the matcher still served every key that was present first” is demonstrated only for the inspected seed-11 timeline, not all 79 incomplete runs. Likewise, “observations hold in every row” conflicts with passing whole-suite runs ([report](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md:113)). These should be scoped precisely; the P/R interventions still support the overall setup-bias conclusion.

- **SHOULD-FIX:** The proposed patch uses `assert.NoError` inside `EventuallyWithT` ([variant](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-variant-pt-test.diff.txt:17)), contrary to the repository instruction to prefer `require`, especially in Eventually callbacks. Correct the retained proposal before recommending it for direct application.

- **NIT:** “Lists all 246 Gomad runs per seed” incorrectly suggests 246 runs for each seed; evidence records 246 total runs ([report](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md:108)).

All cited source line references are correct. The reported totals, seed outcomes, unfairness values, per-shard counts, native counts, qualification results, and evidence hashes agree. Each acceptance criterion is substantively covered: native/Gomad comparison, fairness contract and eligibility, bounded queue-count readiness, cause/owner/next action/qualification criteria, preserved workload/assertion, unchanged skips, and explicit open-work status. No fix is claimed, and the stated limitations are otherwise honest.

VERDICT: NEEDS_WORK

## Round 2

- **SHOULD-FIX:** [Report line 203](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md:203) claims missing native transfers belonged to one workflow and that workflow started last in 7/8 runs. The retained native evidence contains only aggregate counts, while [the evidence index](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-evidence.json:551) says raw logs were not retained. Remove this detail or retain its derivation evidence.
- **NIT:** The burst-equals-rate citation at [report line 169](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md:169) should also cite `service/history/transfer_queue_factory.go:195` and `common/quotas/rate_burst.go:102-130`.
- **NIT:** “3 of 17 seeds” at [report line 25](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md:25) and [milestone line 97](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:97) should say “under the strict tick”; forward observed 2/17.

All Round 1 corrections otherwise verify. The run arithmetic, causal chain, bounded P+T experiment, setup-bias conclusion, open-work status, unchanged skips, preserved distribution/assertion, owner/next action, and qualification criteria satisfy R19.

VERDICT: NEEDS_WORK

## Round 3

No BLOCKER, SHOULD-FIX, or NIT findings.

- Round-2 fixes are present and supported, including the native per-workflow derivation.
- All run counts, unfairness values, shard counts, seeds, and qualification results reconcile with retained evidence.
- Source citations correctly establish the limiter/timer mechanism, readiness gap, fairness contract, and trigger-poll race.
- All acceptance criteria pass. The proposal preserves the workload and `< 1.0` assertion, assigns ownership, defines qualification gates, keeps both skips, and claims no implemented fix.
- Limits are stated honestly, including no Linux run, inferred unchanged-test readiness, deleted raw logs, and no traced successful qualification replay.

VERDICT: SHIP

## Disposition

Round 1 and round 2 findings were all applied and re-reviewed. Round 3 returned SHIP with no findings.
