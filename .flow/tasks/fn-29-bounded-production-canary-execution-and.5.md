---
satisfies: [R2, R6, R10]
---

# fn-29-bounded-production-canary-execution-and.5 Record each iteration and admit it through fn-26

## Description
Add `tools/canary/assessment/admission.go`: each completed iteration's Run is encoded with `recordedrun.Encode` under the pinned Case's identity and the live Driver's identity, kept in memory only (a recorded Run holds whole history events and is never written), and admitted with `evaluation.Admit` against the tree's catalog; the subject's Driver identity must be the one the controller opened, and its Profile name the policy's. A lost iteration (a Run whose process ended before it closed) has no record and no subject. Isolation, authority and cleanup facts enter only the provenance; the Verdict, disposition and cleanup are the recorded ones.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/assessment/ ./tools/umpire/...`

**Files:** `tools/canary/assessment/admission.go`, `tools/canary/assessment/admission_test.go`
**Touches:** `tools/canary/assessment/admission.go`, `tools/canary/assessment/admission_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Crossed, stale, open, noncanonical, lost or foreign-identity iterations are never accepted; each is rejected by fn-26 admission or has no subject at all.
- [x] Authority, isolation and cleanup facts change the assessment only through the provenance and the Profile; the recorded Verdict is never rewritten.
- [x] No Run Evaluation, second evaluator, internal evidence reader or synthetic fact is introduced.

## Done summary
`tools/canary/assessment/admission.go`: `Admit(policy, driver identity, run)` encodes the closed Run with `recordedrun.Encode` in memory only, under the pinned Case's identity and the prepared Case's Driver identity, and admits it with fn-26's `evaluation.Admit` against the tree's catalog. A nil Run is `ErrLost` and has no subject; a Profile name other than the policy's, or a policy naming another Case, is `crossed`; everything else is fn-26's own rejection (stale catalog, crossed Case, open, inconsistent Verdict, oversized). The recorded Verdict, disposition and cleanup reach the subject unchanged. The lifecycle proof decides with `Admit` and `Assess` under `production-canary`, and both live Runs are accepted; its first Run is the tests' fixture when `UMPIRE_CANARY_RECORD` is set. Implementation review: SHIP in one round; its P3 note applied.
## Evidence
- Commits: 99e9936e029bbe7299b1742e37a67058266b5f1e, 50a278dfd44bd533d0d657069d43a777dc12d32f
- Tests: go test -count=1 -tags test_dep ./tools/canary/..., go test -count=1 -tags 'test_dep integration' ./tests/ -run '^TestTestpilotCanaryLifecycle$', GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: