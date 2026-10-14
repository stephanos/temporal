---
satisfies: [R3, R4, R5, R6, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.10 Prove the canary end to end against the test cluster

## Description
Add the `canary_harness` build: `tools/canary/testharness` and a tagged file in `tools/canary/cmd/umpire-canary` fill .8's seams, under that tag only, with a policy source reading a test policy (the test cluster's digests, the `canary-harness` Evaluation Profile, authority class `harness`) and a crash hook from the environment, rejecting a test policy that names any Evaluation Profile but `canary-harness` or any authority class but `harness`, and supplying the plaintext `Transport` the untagged build never builds; `tools/canary/build_test.go` pins that the untagged build contains neither and that no untagged, non-test package builds a plaintext transport; `tools/canary/testharness`'s own tests, run with `-tags 'test_dep canary_harness'`, pin its refusals, and a live case runs the harness binary with a production-naming policy and expects the named refusal. Add `tests/testpilot_canary_test.go` (`test_dep integration`, tests named `TestTestpilotCanary*` so `umpire-check-live-tests` runs them): provision a namespace and a Nexus endpoint with fn-83's `provision.Create`, write the test policy (with a short lease run timeout), set the workflow environment variables and no credential (insecure transport), build `umpire-canary` with `-tags canary_harness` and run it as a separate process: preflight passes, the lease is taken, the Case is prepared once, two serial Runs close satisfied with fresh fenced identities, each is admitted, assessed `accepted` under `canary-harness` and published with its provenance after cleanup, cleanup closes every fenced workflow and the lease is gone. Separate cases crash the process before the first Run and during a Run (a controller hook that exits), then run `reconcile` at once, as the workflow does: the job's own lease is reconciled, no Verdict is fabricated, the lost iteration has no receipt or provenance and is named in the reconciliation report, a `run` before reconcile refuses on the held lease, and one after it proceeds; a second `run` and `reconcile` started during a live first invocation touch nothing of it; and a lease left to time out (a short harness timeout) is refused by the next `run` until reconcile closes its orphans. The Driver's server and worker authorities stay separate, and only public server observations reach the Contract. Every case scans the process's whole stdout and stderr for each planted coordinate.

### Quick commands
`go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotCanary'`

**Files:** `tests/testpilot_canary_test.go`, `tools/canary/cmd/umpire-canary/**`, `tools/canary/testharness/**`, `tools/canary/build_test.go`
**Touches:** `tests/testpilot_canary_test.go`, `tools/canary/cmd/umpire-canary/**`, `tools/canary/testharness/**`, `tools/canary/build_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Server and worker authority stay separate and only public server observations reach the Contract.
- [ ] A crash before or during a Run, cleanup uncertainty and recovery produce no fabricated Verdict and no accepted receipt.
- [ ] Nothing in the harness can publish or retain a production receipt: the harness source refuses a policy naming `production-canary` or `protected-workflow`, its receipts are under `canary-harness` with authority class `harness`, and the untagged binary has no override path (`build_test.go`).

## Done summary
The canary_harness build (testharness policy/transport/hook seams, seams_harness.go), build_test.go and the TestTestpilotCanary* live tests (landed in b4c5a175d6/f498c7eefd) were verified against the Acceptance and run green against the test cluster. Two gaps were fixed. The untagged-only credential test now lives in seams_test.go (`!canary_harness`), so the harness-tagged `./tools/canary/...` test line is green. The live production-policy refusal now covers both `production-canary` and `protected-workflow`, asserts the named refusal, and checks that nothing was published. The review's P3 notes on the pause-hook test and the redundant build env are applied.

Follow-ups: the harness redactor repeats authority's coordinate list by hand (export it from tools/canary/authority, which is outside this task's files); no CI line runs the canary_harness-tagged tests yet (task .12).

stage: impl-review - ran [2026-09-26T12:11..2026-09-26T12:13] SHIP (claude:opus:high, round 1)
## Evidence
- Commits: b4c5a175d6dfe130e8daf31b420d1056cd53f41a, f498c7eefddc9f96022279e482932dc6ebd649a2, 8d4ccd4c3d52caeb1d87171933d31763b611c031, f22f18ffc407fae20ab83cc2bbb69139bae54f8d
- Tests: baseline: green (Quick command, pre-edit at 707b472591), CGO_ENABLED=0 go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotCanary' (green at f22f18ffc4, live against the in-process test cluster; CGO_ENABLED=0 because local clang lacks its headers), CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/canary/..., CGO_ENABLED=0 go test -count=1 -tags 'test_dep canary_harness' ./tools/canary/..., golangci-lint v2.13.1 run --build-tags test_dep,canary_harness ./tools/canary/... (0 issues); --build-tags test_dep,integration ./tests/... ./tools/canary/... (no canary issues), note: a verify run at 12:13 overlapped a user branch switch and was discarded as inconclusive
- PRs: