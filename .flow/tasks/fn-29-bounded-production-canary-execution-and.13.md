---
satisfies: [R9, R10]
---

# fn-29-bounded-production-canary-execution-and.13 Write the canary runbook and reconcile the roadmap

## Description
Write `tools/canary/README.md`: the precondition that the `production-canary` environment restricts deployment branches to `main` and requires reviewers (the in-repo checks are defense in depth), protected invocation, what preflight checks, the Limits, how to read each iteration's Verdict, receipt and provenance, lost iterations, the held lease and reconciliation, how an operator clears an uncertain scope (close the listed workflows by hand, then dispatch again so reconcile verifies them), cleanup outcomes, that receipts are not self-authenticating and always carry `releaseEligibility: false`, how the retained artifact is handled, and that nothing here authorizes a release. Update `.plans/UMPIRE4_COMPONENTS.md` and `.plans/UMPIRE4_ORDER.md` to the implemented ownership.

### Quick commands
`make umpire-check-retired-vocabulary`

**Files:** `tools/canary/README.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`
**Touches:** `tools/canary/README.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_ORDER.md`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] An operator can tell accepted, rejected, incomplete, lost, cleanup-uncertain, published and reporting-ambiguous states apart from the runbook.
- [ ] The docs state that receipts are not self-authenticating and always carry `releaseEligibility: false`.
- [ ] No schedule, automatic rerun, rollout, customer-traffic or release-authorization guidance is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
