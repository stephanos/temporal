---
satisfies: [R7, R10]
---

# fn-29-bounded-production-canary-execution-and.7 Publish each receipt and its provenance immutably

## Description
Move the exclusive publisher whole into `tools/umpire/publish`: `Publish`, `ConflictError`, `Resolve`, `Within` and the no-follow open with its platform files and tests; `tools/umpire/internal/cli` imports it, keeping aliases for every name its current callers use -- `Publish`, `Publication`, `ConflictError`, `StatusPublished`, `StatusAlreadyPublished`, `Resolve`, `Within` -- so nothing is duplicated and no cycle forms. Add `tools/canary/publication`: after the cleanup attempt, whatever its outcome, for each admitted iteration in order, publish the fn-26 receipt bytes unchanged under `<receipt identity>.json` and then the provenance under `<provenance identity>.provenance.json` in the output directory that is uploaded, each exclusive and idempotent; a conflict on either is reported and never overwritten; a lost or unconstructible iteration publishes nothing, and a process lost before publication leaves its iterations for reconcile to report as lost. A publication that succeeded but could not be reported is the named ambiguity and never reruns anything.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/publication/ ./tools/umpire/...`

**Files:** `tools/umpire/publish/**`, `tools/umpire/internal/cli/publish.go`, `tools/umpire/internal/cli/publish_test.go`, `tools/umpire/internal/cli/publish_unix_test.go`, `tools/umpire/internal/cli/publish_nofollow_unix.go`, `tools/umpire/internal/cli/publish_nofollow_other.go`, `tools/umpire/internal/cli/proposal.go`, `tools/canary/publication/**`
**Touches:** `tools/umpire/publish/**`, `tools/umpire/internal/cli/publish.go`, `tools/umpire/internal/cli/publish_test.go`, `tools/umpire/internal/cli/publish_unix_test.go`, `tools/umpire/internal/cli/publish_nofollow_unix.go`, `tools/umpire/internal/cli/publish_nofollow_other.go`, `tools/umpire/internal/cli/proposal.go`, `tools/canary/publication/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Accepted, rejected and incomplete receipts publish with their provenance; lost or unconstructible iterations publish nothing.
- [x] A conflicting name, a symlink, a partial file or a crossed provenance fails closed and changes nothing.
- [x] Publishing the same content again is already published; a reporting failure after publication is named and never reruns.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: