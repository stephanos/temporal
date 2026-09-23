---
satisfies: [R3, R4, R5, R6, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.10 Prove the canary end to end against the test cluster

## Description
Add `tests/umpire_canary_test.go` (`test_dep integration`): provision a namespace, both queues and a Nexus endpoint with fn-83's `provision.Create`, write a test policy whose digests are the test cluster's coordinates, supply a workflow environment through the authority hook with no credential (insecure transport), and run `umpire-canary run` as a separate process: preflight passes, the lease is taken, the Case is prepared once, two serial Runs close satisfied with fresh fenced identities, each is admitted, assessed `accepted` under `production-canary` and published with its provenance, cleanup closes every fenced workflow and the lease is gone. Separate cases crash the process before the first Run and during a Run (a controller hook that exits), then run `reconcile`: no Verdict is fabricated, the lost iteration has provenance and no receipt, and a new `run` proceeds only after reconciliation. The Driver's server and worker authorities stay separate, and only public server observations reach the Contract.

### Quick commands
`go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestUmpireCanary'`

**Files:** `tests/umpire_canary_test.go`, `tools/canary/testharness/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Server and worker authority stay separate and only public server observations reach the Contract.
- [ ] A crash before or during a Run, cleanup uncertainty and recovery produce no fabricated Verdict and no accepted receipt.
- [ ] Nothing in the harness can publish or retain a receipt as a production receipt: its policy and workflow context are labeled test-only and preflight refuses them outside the harness hook.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
