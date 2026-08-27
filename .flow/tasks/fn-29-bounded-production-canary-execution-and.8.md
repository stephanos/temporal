---
satisfies: [R3, R4, R5, R6, R7, R8]
---
# fn-29-bounded-production-canary-execution-and.8 Compose the canary Claim Assessment controller and closed Run mode

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R3-R8 behind one production-fixed canary controller and the Claim Assessment binary's closed run mode.

**Size:** M
**Files:** `tools/umpire/canaryassessment/**`, `tools/umpire/evaluation/**`, `tools/umpire/cmd/umpire-assess-production-canary/**`, `model/lakefile.toml`
**Touches:** [tools/umpire/canaryassessment/**, tools/umpire/evaluation/**, tools/umpire/cmd/umpire-assess-production-canary/**, model/lakefile.toml]

### Approach
- Compose ordered input/pilot/profile/workflow-context admission, protected authority and scope preflight, lease, execution, cleanup/reconciliation, postflight, evidence/provenance closure, Run Evaluation, offline Claim Assessment, v6 construction, and exactly one publication behind a narrow API.
- Reuse environment-neutral remote transport/control seams while keeping staging and canary policy/controllers separate; production injection fixes authority, profile/checker, program, limits, action, statuses, and publisher.
- Implement the exact run arguments and canonical secret-free status 0/1/2 summary/error contract with dispatch/cleanup/publication booleans and reporting-after-publication recovery.
- Maintain the exact controller RPC ledger across phases, transfer only the 24-call reserve into cleanup, and expose narrow RemoteRecoveryRecord v3/progress hooks for Task `.9`; run mode cannot reset, weaken, or select those paths.
- Preserve every constructible post-dispatch failed/incomplete run and independent status; pre-dispatch tooling failures publish nothing, and no path redispatches, rechecks, or republishes automatically.
- Register only required sibling executables in the primary Lake package and make no model-local Make change.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.8.md` — staging controller stage/status contract
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime API and cleanup dominance
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — Run Evaluation API/tooling errors
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — Evaluation Profile, Evaluation Receipt, and Claim Assessment command contract
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — sole publisher behavior

### Key context
Cleanup/postflight and isolation facts must close before Run Evaluation and Claim Assessment; only execution evidence reaches semantics. This controller is deep composition, not a second Drive DSL.

### Acceptance
- [ ] Malformed/preflight input performs no remote or publication IO.
- [ ] One valid injected run closes cleanup/postflight before Run Evaluation and publishes one admitted v6 set.
- [ ] Every cancel/failure/non-success/cleanup/reporting/publication row preserves exact facts without redispatch.
- [ ] Crash/restart stage fakes prove v3 recovery resumes the remaining reserve rather than resetting the 64-call total, while staging v2 remains outside this controller.
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
