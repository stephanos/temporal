---
satisfies: [R3, R4, R5, R6, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.10 Prove the canary end to end against the test cluster

## Description
Add the `canary_harness` build: `tools/canary/testharness` provides, under that tag only, a policy source reading a test policy (the test cluster's digests, the `canary-harness` Evaluation Profile, authority class `harness`) and a crash hook from the environment; a regression test pins that the untagged build contains neither. Add `tests/umpire_canary_test.go` (`test_dep integration`): provision a namespace and a Nexus endpoint with fn-83's `provision.Create`, write the test policy, set the workflow environment variables and no credential (insecure transport), build `umpire-canary` with `-tags canary_harness` and run it as a separate process: preflight passes, the lease is taken, the Case is prepared once, two serial Runs close satisfied with fresh fenced identities, each is admitted, assessed `accepted` under `canary-harness` and published with its provenance after cleanup, cleanup closes every fenced workflow and the lease is gone. Separate cases crash the process before the first Run and during a Run (a controller hook that exits), then run `reconcile`: no Verdict is fabricated, the lost iteration has no receipt or provenance and is named in the reconciliation report, a `run` before reconcile refuses on the held lease, and one after it proceeds. The Driver's server and worker authorities stay separate, and only public server observations reach the Contract.

### Quick commands
`go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestUmpireCanary'`

**Files:** `tests/umpire_canary_test.go`, `tools/canary/testharness/**`, `tools/umpire/regression/canary_workflow_test.go`
**Touches:** `tests/umpire_canary_test.go`, `tools/canary/testharness/**`, `tools/umpire/regression/canary_workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Server and worker authority stay separate and only public server observations reach the Contract.
- [ ] A crash before or during a Run, cleanup uncertainty and recovery produce no fabricated Verdict and no accepted receipt.
- [ ] Nothing in the harness can publish or retain a production receipt: its receipts are under `canary-harness` with authority class `harness`, and the untagged binary has no override path.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
