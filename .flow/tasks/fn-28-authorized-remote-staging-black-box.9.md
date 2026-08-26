---
satisfies: [R3, R4, R5, R8, R9]
---
# fn-28-authorized-remote-staging-black-box.9 Add workflow-only recovery and the protected execution workflow

## Description
Implement R5/R8/R9's ephemeral recovery protocol, reconcile mode, bounded progress channel, root UX, and manual protected workflow.

**Size:** M
**Files:** `Makefile`, `.github/workflows/umpire-remote-staging-qualification.yml`, `tools/umpire/staging/recovery.go`, `tools/umpire/staging/recovery_test.go`, `tools/umpire/staging/progress.go`, `tools/umpire/staging/progress_test.go`, `tools/umpire/cmd/umpire-qualify-remote-staging/**`
**Touches:** [Makefile, .github/workflows/umpire-remote-staging-qualification.yml, tools/umpire/staging/recovery.go, tools/umpire/staging/recovery_test.go, tools/umpire/staging/progress.go, tools/umpire/staging/progress_test.go, tools/umpire/cmd/umpire-qualify-remote-staging/**]

### Approach
- Atomically create/update the closed mode-0600 RemoteRecoveryRecord v1 at the fixed runner-temp path after lease acquisition; bind exact invocation, lease workflow/run fence, deterministic caller identity, dispatch state, target digest, and expiry while excluding target coordinates, credentials, payloads, and artifact claims.
- Add the same binary's `reconcile --run-id` mode: re-acquire fixed protected authority, validate the recovery record and exact live fence, terminate/verify only recorded resources, and never dispatch, conform, qualify, construct, or publish; missing record is a canonical no-op and invalid/stale/mismatched state fails without mutation.
- Add the exact 256-event/64-KiB RemoteProgress v1 JSONL sink with closed phases/states/message codes and at most one heartbeat per 30 seconds; keep terminal stdout/stderr contracts unchanged.
- Add only repository-root run-mode Make wiring with required set/pilot/output/run inputs and no target/profile/credential or reconciliation selector.
- Add one workflow-dispatch-only job bound to the fixed protected environment, pinned actions, read-only repository permission, fixed concurrency, hard timeout, runner-temp authority/recovery/progress/output roots, live progress streaming, and an always-run reconcile/evidence step; never upload the recovery record.

### Investigation targets
**Required** (read before coding):
- `.github/workflows/umpire-model-verification.yml:1-28` — pinned-action, permission, timeout, and concurrency conventions
- `.github/workflows/docker-build-manual.yml:1-35` — manual dispatch pattern
- `common/testing/umpire/canary/canary.go:109-175` — recovery-safe cleanup record and fresh cleanup context
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.4.md` — fenced lifecycle and server-timeout backstop
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.8.md` — run controller, recovery hook, and terminal reporting

### Key context
The workflow reconciler handles ordinary process failure when its runner-temp record survives. Complete runner loss may skip the step, so server execution timeouts remain the only structural backstop and no receipt may claim verified cleanup in that case.

### Acceptance
- [ ] Recovery record creation/update/removal is atomic, mode-0600, physically runner-temp-contained, bounded, secret-scanned, and never uploaded or admitted as an artifact.
- [ ] Reconcile mode can affect only the exact validated fence and has no dispatch/conformance/qualification/publication capability.
- [ ] Progress events are bounded, secret-free, visible during every remote phase, and separate from terminal records and qualification identities.
- [ ] The only Make change is in the repository-root Makefile and exposes run mode only.
- [ ] Workflow policy tests prove manual protected isolation, least privilege, pinning, progress, always-run reconciliation/evidence, and no default/deploy/release coupling.

## Acceptance
- [ ] R3-R5/R8/R9 recovery, reconcile, progress, root UX, and workflow contracts are complete.
- [ ] Focused record/path/fence/no-capability/workflow policy tests pass.
- [ ] Existing Make/workflow/orchestration comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
