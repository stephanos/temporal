---
satisfies: [R3, R5, R8, R9]
---
# fn-29-bounded-production-canary-execution-and.9 Reuse remote recovery/progress and add the protected canary workflow

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R5/R8/R9's recovery, reconcile, progress, root UX, and manual protected production-canary workflow without duplicating the staging control protocol.

**Size:** M
**Files:** `Makefile`, `.github/workflows/umpire-production-canary-claim-assessment.yml`, `tools/umpire/remoteassessment/**`, `tools/umpire/staging/**`, `tools/umpire/canaryassessment/**`, `tools/umpire/cmd/umpire-assess-production-canary/**`
**Touches:** [Makefile, .github/workflows/umpire-production-canary-claim-assessment.yml, tools/umpire/remoteassessment/**, tools/umpire/staging/**, tools/umpire/canaryassessment/**, tools/umpire/cmd/umpire-assess-production-canary/**]

### Approach
- Extract the environment-neutral RemoteRecoveryRecord and RemoteProgress implementations from the staging control layer only as needed; preserve staging's v2 record/reader, command behavior, wire protocol, tests, comments, and Artifact Checksums.
- Add closed RemoteRecoveryRecord v3 for production canary with the v2 identity/fence fields plus remaining cleanup/reconcile RPC reserve. Reconcile validates the fixed workflow invocation, target digest, lease/caller fences, dispatch state, expiry, and reserve, then may only spend that reserve to terminate/verify exact resources; missing record is no-op and malformed/stale/mismatched state performs no mutation.
- Keep the record atomic, mode-0600, runner-temp-contained, secret-free, bounded, and never uploaded/admitted. Keep progress closed at 256 events/64 KiB and separate from terminal records/artifacts.
- Add only a repository-root run-mode Make target with required set/pilot/output/run inputs and no target/profile/credential/reconcile selector.
- Add a credential-free preflight job that accepts only the protected default ref and records its immutable SHA, then one workflow-dispatch-only protected job whose externally configured environment deployment-branch rule admits only that branch and whose pinned checkout uses exactly the admitted SHA. Keep read-only repository permission, fixed concurrency/timeout, runner-temp paths, progress streaming, always-run reconcile/evidence handling, and no authority/recovery upload.
- Assert the workflow is unreachable from PR, push, schedule, deployment, promotion, rollout, and release paths and cannot emit a release-eligibility output.
- Test every repository-controlled ref/SHA guard and document that the external protected-environment branch rule must be verified before secrets are provisioned or a run is approved.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.9.md` — recovery/progress/workflow contract to preserve and reuse
- `.github/workflows/umpire-model-verification.yml` — pinned-action/permission/timeout conventions
- `.github/workflows/docker-build-manual.yml` — manual dispatch pattern
- `common/testing/umpire/canary/canary.go` — fresh cleanup context concepts only
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.8.md` — controller hooks and terminal reporting

### Key context
Only the external protected-environment branch rule prevents modified code on a selected untrusted branch from receiving credentials; an in-workflow check alone is insufficient. Retained provenance still declares that approver/build identity is not independently authenticated. Total runner loss is contained by server timeouts, not an overclaimed reconcile guarantee.

### Acceptance
- [ ] Shared recovery/progress extraction preserves staging v2 bytes/reader/behavior, adds strict canary v3 dispatch, and introduces no staging dependency into canary.
- [ ] Reconcile has no dispatch/Run Evaluation/Claim Assessment/construction/publication capability and touches only exact fences.
- [ ] Recovery/progress are bounded, secret-free, runner-temp-contained, and excluded from Claim Assessment identity; v3 persists the exact remaining reserve and cannot reset it after restart.
- [ ] The only Make change is repository-root run-mode wiring.
- [ ] Workflow policy tests prove the credential-free default-ref guard, exact-SHA checkout, manual protected isolation, pinning, least privilege, non-release output, progress, and always-run reconciliation/evidence; the runbook gates provisioning on the external deployment-branch rule.
## Acceptance
- [ ] R3/R5/R8/R9 shared control, closed recovery, progress, root UX, and protected workflow are complete.
- [ ] Focused path/fence/no-capability/workflow policy tests pass.
- [ ] Existing Make, workflow, and orchestration comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
