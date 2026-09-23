---
satisfies: [R2, R6, R10]
---

# fn-29-bounded-production-canary-execution-and.5 Record each iteration and admit it through fn-26

## Description
Export `tools/umpire/internal/recordedrun` as `tools/umpire/recordedrun` (the internal path aliases it for its current importers). Add `tools/canary/assessment/admission.go`: each completed iteration's Run is encoded with `recordedrun.Encode` under the pinned Case's identity and the live Driver's identity, written into the runner-local directory (never the uploaded one: a recorded Run holds whole history events), and admitted with `evaluation.Admit` against the tree's catalog; the subject's Driver identity must be the one the controller opened, and its Profile name the policy's. A lost iteration (a Run whose process ended before it closed) has no record and no subject. Isolation, authority and cleanup facts enter only the provenance; the Verdict, disposition and cleanup are the recorded ones.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/assessment/ ./tools/umpire/...`

**Files:** `tools/umpire/recordedrun/**`, `tools/umpire/internal/recordedrun/**`, `tools/umpire/evaluation/**`, `tools/umpire/replay/recorded.go`, `tools/canary/assessment/admission.go`, `tools/canary/assessment/admission_test.go`
**Touches:** `tools/umpire/recordedrun/**`, `tools/umpire/internal/recordedrun/**`, `tools/umpire/evaluation/**`, `tools/umpire/replay/recorded.go`, `tools/canary/assessment/admission.go`, `tools/canary/assessment/admission_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Crossed, stale, open, noncanonical, lost or foreign-identity iterations are never accepted; each is rejected by fn-26 admission or has no subject at all.
- [ ] Authority, isolation and cleanup facts change the assessment only through the provenance and the Profile; the recorded Verdict is never rewritten.
- [ ] No Run Evaluation, second evaluator, internal evidence reader or synthetic fact is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
