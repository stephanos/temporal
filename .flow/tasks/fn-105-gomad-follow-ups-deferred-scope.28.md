---
satisfies: [R19]
---
# fn-105-gomad-follow-ups-deferred-scope.28 D19: establish fairness backlog readiness and remove skips

## Description
Apply the D19 investigation recommendation in the native fairness test. Wait for all 225 activity tasks in the matching backlog before measuring dispatch, retry the auto-enable trigger poll after an empty response, retain fairness distribution and threshold, then remove both Gomad skips after qualification.

## Acceptance
Both suites pass natively and qualify on darwin/arm64 seeds 11 and 17 with skips lifted; traced exact replay matches; strict-tick seeds 1-17 pass for each suite with 225 tasks ready at the first measured poll and unfairness below 1.0; generated manifest validates; record Linux status.

## Done summary
Applied the D19 fairness-test correction: wait for all 225 activity tasks before measuring and retry an empty auto-enable trigger poll. Removed both Test_Activity_Basic qualification skips and regenerated tests.json. On darwin/arm64 the tracked suites qualified on seeds 11 and 17; traced exact choice-tape replay matched for all four suite-seed results; strict-tick sweeps of seeds 1-17 passed for both suites at unfairness 0.45, with 225/225 transfers started before the first measured poll. The native suites passed once and changed-package lint passed. A 30-run native auto-enable leaf check had two failures at the preexisting line-468 workflow-task wait, separate from this correction. Linux was not run. Evidence: .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d19-correction-sweep.json and the adjacent tracked/traced reports. No commit was made per repository instructions.
## Evidence
- Commits:
- Tests: gomad explore --seeds 11 --clock-tick=strict TestFairnessSuite baseline (target_failure, unfairness 1.3), go test -tags test_dep ./tests -run ^(TestFairnessSuite|TestFairnessAutoEnableSuite)$ -count=1 -p 2 (pass), go test -tags test_dep ./tests -run ^TestFairnessAutoEnableSuite$/^Test_Activity_Basic$ -count=30 -p 2 (2 known line-468 failures), gomad qualify-set --manifest d19-current-traced.json (both suites qualified, four exact replays), gomad explore --seeds 1-17 --clock-tick=strict TestFairnessSuite (17 pass, 225/225 backlog), gomad explore --seeds 1-17 --clock-tick=strict TestFairnessAutoEnableSuite (17 pass, 225/225 backlog), gomad qualify-set --manifest d19-tracked.json (both suites qualified), make -C tools/gomad3 validate-qualification, GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false make lint-code-fast, flowctl validate --all, git diff --check
- PRs: