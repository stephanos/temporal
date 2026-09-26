---
satisfies: [R9, R10]
---

# fn-29-bounded-production-canary-execution-and.13 Write the canary runbook and reconcile the roadmap

## Description
Write `tools/canary/README.md`: the credential's privilege (a namespace writer on the canary namespace only), the operator preconditions -- the coordinate digests computed from the protected environment's values and committed to the policy in a reviewed pull request to `main` (until then preflight refuses as `policy-unconfigured`), the Nexus endpoint targets the canary namespace and handler queue (and allows the canary namespace as a caller where the deployment has that setting), the canary namespace's retention is at least 30 days, and the `production-canary` environment restricts deployment branches to `main` and requires reviewers (the in-repo checks are defense in depth), protected invocation, what preflight checks, the Limits, how to read each iteration's Verdict, receipt and provenance, lost iterations, the held lease and reconciliation, how an operator clears an uncertain scope (close the listed workflows by hand, then dispatch again so reconcile verifies them), cleanup outcomes, that receipts are not self-authenticating and always carry `releaseEligibility: false`, how the retained artifact is handled, and that nothing here authorizes a release. Update `.plans/UMPIRE4_COMPONENTS.md` and `.plans/UMPIRE4_ORDER.md` to the implemented ownership.

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
`tools/canary/README.md` is the production canary's operator runbook: the credential's privilege (namespace writer on the canary namespace only), the preconditions an operator owns (protected `production-canary` environment restricted to `main` with reviewers, the Nexus endpoint's target and allowed caller namespace, 30-day retention, coordinate digests committed in a reviewed pull request to `main`, else `policy-unconfigured`), invocation, preflight, the Limits, every `run` and `reconcile` status and exit, lost iterations, the held lease and reconciliation, how to clear an uncertain scope, the retained artifact, and that receipts are not self-authenticating, provenance always carries `releaseEligibility: false`, and nothing here authorizes a release. `.plans/UMPIRE4_COMPONENTS.md` records fn-29's implemented ownership; `.plans/UMPIRE4_ORDER.md`'s fn-29 row records tasks .1 to .13 with .12's gate notes. The `lint-model` note says the `Refinement.lean` diagnostic is outside the recorded baseline and is not fn-29's, and that the baseline was not re-measured.

Review: round 1 NEEDS_WORK (P2: `lost` could name a published Run after `publication-unreported`; P3: `lease-in-use` when clearing a scope; P3: the second live-test file), all fixed; round 2 SHIP.

stage: impl-review - ran [2026-09-26] SHIP (claude:opus:high, round 2; deterministic triage-skip overridden because the runbook's claims needed checking against the code)
## Evidence
- Commits: 24fc66b900d860d977a1674186681d9c893242d8, 5fe6539628b27675e993925ab1d853645f252771
- Tests: baseline: green (make umpire-check-retired-vocabulary, pre-edit), make umpire-check-retired-vocabulary (exit 0 at HEAD, 196s), gate classify: FULL (.plans/UMPIRE4_*.md is scanned by the retired-vocabulary gate), impl-review: SHIP (claude:opus:high, round 2; triage-skip overridden with a full review)
- PRs: