---
satisfies: [R3, R4, R5, R6, R7, R8]
---
# fn-29-bounded-production-canary-execution-and.8 Compose the canary qualification controller and closed run mode

## Description
Implement R3-R8 behind one production-fixed canary controller and the qualification binary's closed run mode.

**Size:** M
**Files:** `tools/umpire/canaryqualification/**`, `tools/umpire/qualification/**`, `tools/umpire/cmd/umpire-qualify-production-canary/**`, `model/lakefile.toml`
**Touches:** [tools/umpire/canaryqualification/**, tools/umpire/qualification/**, tools/umpire/cmd/umpire-qualify-production-canary/**, model/lakefile.toml]

### Approach
- Compose ordered input/pilot/profile/workflow-context admission, protected authority and scope preflight, lease, execution, cleanup/reconciliation, postflight, evidence/provenance closure, conformance, offline qualification, v5 construction, and exactly one publication behind a narrow API.
- Reuse environment-neutral remote transport/control seams while keeping staging and canary policy/controllers separate; production injection fixes authority, profile/checker, program, limits, action, statuses, and publisher.
- Implement the exact run arguments and canonical secret-free status 0/1/2 summary/error contract with dispatch/cleanup/publication booleans and reporting-after-publication recovery.
- Maintain the exact controller RPC ledger across phases, transfer only the 24-call reserve into cleanup, and expose narrow RemoteRecoveryRecord v2/progress hooks for Task `.9`; run mode cannot reset, weaken, or select those paths.
- Preserve every constructible post-dispatch failed/incomplete run and independent status; pre-dispatch tooling failures publish nothing, and no path redispatches, rechecks, or republishes automatically.
- Register only required sibling executables in the primary Lake package and make no model-local Make change.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.8.md` — staging controller stage/status contract
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime API and cleanup dominance
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — conformance API/tooling errors
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — pilot/policy/receipt command contract
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — sole publisher behavior

### Key context
Cleanup/postflight and isolation facts must close before conformance and qualification; only execution evidence reaches semantics. This controller is deep composition, not a second Drive DSL.

### Acceptance
- [ ] Malformed/preflight input performs no remote or publication IO.
- [ ] One valid injected run closes cleanup/postflight before conformance and publishes one admitted v5 set.
- [ ] Every cancel/failure/non-success/cleanup/reporting/publication row preserves exact facts without redispatch.
- [ ] Crash/restart stage fakes prove v2 recovery resumes the remaining reserve rather than resetting the 64-call total, while staging v1 remains outside this controller.
- [ ] Run mode accepts only set, pilot evidence, output root, and run ID and exposes no target/action/fault/claim selector.

## Acceptance
- [ ] R3-R8 controller, run mode, sibling handshake, stage order, status, and publication contracts are complete.
- [ ] Independent stage fakes and command integration tests pass.
- [ ] Existing orchestration comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
