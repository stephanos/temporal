---
satisfies: [R19]
---
# fn-105-gomad-follow-ups-deferred-scope.19 D19: investigate activity fairness backlog readiness

## Description
Covered by the 2026-09-30 blanket investigation approval. TestFairnessSuite/Test_Activity_Basic and TestFairnessAutoEnableSuite/Test_Activity_Basic measure dispatch fairness before all workflow activity transfers are known to have reached matching. The recorded virtual-time runs initially dispatch from only 3-4 of 15 workflows. Establish whether the test setup needs an explicit backlog-readiness condition or whether there is a product fairness defect, then propose the correction.

## Acceptance
- Compare native and Gomad runs on seeds 11 and 17, retaining task-transfer, queue-readiness, dispatch, and unfairness evidence for both suites.
- Establish the intended fairness contract and whether the complete backlog is actually eligible before measurement starts.
- Evaluate a bounded readiness condition using existing queue/testcore observations; distinguish test setup bias from a matcher fairness defect.
- Record the cause, correction owner, proposed next action, and qualification criteria in fn-105 for a subsequent decision; retain any needed fix as explicit open work.
- Preserve the fairness assertion and distribution, keep the skips until verification supports removal, and do not substitute a relaxed threshold or Gomad-only source rewrite for evidence.

## Done summary
D19 investigation closed with cause, evidence, and an owned proposal; nothing is fixed and both skips stay. Report: docs/research/gomad/GOMAD_D19_FAIRNESS_BACKLOG_READINESS.md; evidence index: .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-evidence.json.

- Cause (setup bias, no matcher fairness defect found): Test_Activity_Basic polls before its 225 activity tasks reach matching. Each history shard reads its transfer queue through a rate limiter (20/s, burst 20). In the whole suite the sibling tests exhaust the burst in the first virtual instant, the next read waits on a 50 ms timer, and Gomad delivers the timer only after the test has drained the partial backlog. Unchanged suite, seeds 1-17: backlog incomplete on every seed, 7 fail at :603 under strict and 7 under forward. Alone the test passes at 0.45.
- With a complete backlog the metric was the ideal 0.45 in 191 of 191 Gomad runs and all native runs; in all 79 partial-backlog runs the first dispatches carried every key present. Native runs also start polling early (8 of 12), but the gap closes within the first dispatch.
- Second cause, auto-enable suite only: triggerAutoEnable's single activity poll can return empty when auto-enable reloads the queue (:508; 3 of 17 seeds strict, 2 of 17 forward, 0 of 34 native runs).
- Proposed correction, owner = owner of tests/priority_fairness_test.go: wait for 225 backlog tasks through the existing DescribeTaskQueuePartition count, and repeat the trigger poll (fn105-d19-variant-pt-test.diff.txt). In throwaway copies it passes both suites on seeds 1-17, qualify-set on seeds 11 and 17 with the skips lifted, and 12 native runs. Not applied; open work under R19 with qualification steps in the report.
- Ruled out: clock_tick forward. Diagnostic variant R (reader rate x1000) removes the stall with the test unchanged.
- Not done: linux/amd64, traced replay of the proposal, explanation of one native :468 failure in 30 runs.
- The checkout was replaced mid-run by a fresh clone at 6782b55f4 (another worker's faulty script deleted it); files under test are byte-identical, and the seed 11/17 runs, qualify-set, and the proposal sweep were repeated on the clone with the same outcomes.
- baseline: none (the spec defines no Quick commands). No tracked product, test, runtime, or manifest file was changed.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: gomad explore (fn105-d19-run.sh / fn105-d19-sweep.sh): 288 single-seed runs of ^TestFairnessSuite$ and ^TestFairnessAutoEnableSuite$ (suite and leaf, strict and forward, unchanged and variants R, P, P+T); outcomes in .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-gomad-runs.txt, gomad qualify-set --manifest=fn105-d19-qualify-set-unskipped-manifest.json: unchanged test exit 1 (target_failure, as expected for the reproducer); variant P+T in a throwaway copy exit 0 (both suites qualified on seeds 11 and 17), go test -tags test_dep ./tests -run '^TestFairnessSuite$' / '^TestFairnessAutoEnableSuite$' (and leaves) -count=1 -p 2 -v: 12 unchanged runs pass; -count=30 auto-enable leaf: 29 pass, 1 fail at :468; 12 runs with the variant through -overlay pass, codex exec -s read-only -m gpt-5.6-sol (3 rounds): VERDICT: SHIP in round 3, no gate suite applies: investigation only, no tracked product/test/runtime/manifest change; baseline: none
- PRs: