---
satisfies: [R20]
---
# fn-105-gomad-follow-ups-deferred-scope.29 D20: make heartbeat rejection deadlines explicit and remove skip

## Description
Apply the D20 recommended test correction: explicitly configure the heartbeat timeout, send the two rejected heartbeats after the chain deadline, and assert every heartbeat outcome while preserving timeout counts and expected history. Remove the qualification skip after verification.

## Acceptance
Native leaf count 20 and suite pass; Gomad leaf seeds 1-64 pass under strict and forward; suite qualifies on seeds 11 and 17 with exact replay; tracked manifest validates and qualifies after skip removal; retain baseline and report Linux status.

## Done summary
Applied D20 recommended correction: pin the 5s heartbeat timeout, send both rejected heartbeats after a local upper-bound deadline plus 100ms, assert each outcome, preserve the 47-event history and two timeout/recovery cycles, and remove the skip. Baseline strict seed 11 fails expected 2/actual 1. The correction passes strict and forward leaf seeds 1-64 and 20 native repetitions. After changing only the new assertion to Require.Equal, the native suite, tracked qualification, and traced exact replay pass on seeds 11 and 17. Retained source-bound evidence and reports use fn105-d20-correction-*; Linux not run. Root lint/vet passes with ./tests scoped explicitly because lint-code-fast includes nested Gomad-module paths in the root module. No commits: user owns commits.
## Evidence
- Commits:
- Tests: go test -tags test_dep ./tests -run ^TestWorkflowTaskTestSuite$/^TestWorkflowTaskHeartbeatingWithEmptyResult$ -count=20 -p 2, go test -tags test_dep ./tests -run ^TestWorkflowTaskTestSuite$ -count=1 -p 2, gomad explore --seeds 1-64 --clock-tick=strict (D20 leaf), gomad explore --seeds 1-64 --clock-tick=forward (D20 leaf), gomad qualify-set D20 final traced and tracked manifests: seeds 11,17 qualified; traced exact replay matched, make -C tools/gomad3 tests-qualification-generate validate-qualification, GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false make lint-code LINT_CODE_TARGETS=./tests
- PRs: