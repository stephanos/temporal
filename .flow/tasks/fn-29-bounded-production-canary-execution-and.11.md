---
satisfies: [R3, R4, R5, R7, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.11 Run the adversarial authority and containment matrices

## Description
Exercise, against the controller in-process and the harness: target drift (refused by preflight) and routing drift (a repointed endpoint: a Run that is not accepted, with its receipt), a credential planted in every output path, a lease collision and a stale fence, a duplicate dispatch, a Run crossing (another Case, another fence), a scope escape (an identity outside the fence), a crash at each phase, a lease that timed out, a lease terminated by hand with another reason, a second dispatch during a live Run, a stale or tampered recovery record, cleanup uncertainty, an iteration past the limit and a tenfold request (capped by the policy, `DefaultCeilings` and fn-26's caps, each pinned), a publication conflict and a reporting failure. Race-enabled unit tests prove no state leaks between iterations or leases.

### Quick commands
`go test -race -count=1 -tags test_dep ./tools/canary/...; go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotCanary'`

**Files:** `tools/canary/**/*_test.go`, `tests/testpilot_canary_test.go`
**Touches:** `tools/canary/**/*_test.go`, `tests/testpilot_canary_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Every mutation fails at its owning boundary and touches no unrelated resource.
- [ ] A proved violation stays violated; anything else incomplete in execution or evaluation stays inconclusive or incomplete.
- [ ] Race-enabled tests show no cross-Run or cross-lease state.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
