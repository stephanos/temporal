---
satisfies: [R7, R10]
---

# fn-29-bounded-production-canary-execution-and.7 Publish each receipt and its provenance immutably

## Description
Export the exclusive publisher as `tools/umpire/publish` (`cli.Publish` delegates to it). Add `tools/canary/publication`: after cleanup and the lease's release, for each admitted iteration in order, publish the fn-26 receipt bytes unchanged under `<receipt identity>.json` and then the provenance under `<provenance identity>.provenance.json` in the output directory that is uploaded, each exclusive and idempotent; a conflict on either is reported and never overwritten; a lost or unconstructible iteration publishes nothing, and a process lost before publication leaves its iterations for reconcile to report as lost. A publication that succeeded but could not be reported is the named ambiguity and never reruns anything.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/publication/ ./tools/umpire/...`

**Files:** `tools/umpire/publish/**`, `tools/umpire/internal/cli/publish.go`, `tools/canary/publication/**`
**Touches:** `tools/umpire/publish/**`, `tools/umpire/internal/cli/publish.go`, `tools/canary/publication/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Accepted, rejected and incomplete receipts publish with their provenance; lost or unconstructible iterations publish nothing.
- [ ] A conflicting name, a symlink, a partial file or a crossed provenance fails closed and changes nothing.
- [ ] Publishing the same content again is already published; a reporting failure after publication is named and never reruns.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
