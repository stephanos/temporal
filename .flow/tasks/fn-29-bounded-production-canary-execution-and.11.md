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
The adversarial matrix is now covered by tests. In process, `tools/canary/controller/matrix_test.go` proves these, and its header points to the existing tests for the other matrix cases:
- a planted credential or coordinate reaches no output;
- a Run crossing its fence fails at publication or at the fenced Session;
- a second dispatch and its reconcile during a live Run only read;
- a tenfold request is capped by the pinned policy limits, the Temporal Profile's `DefaultCeilings` (150s iteration bound) and fn-26's caps;
- a proved violation stays rejected under every other failure, and an inconclusive Run stays incomplete;
- a stale recovery record never touches a later lease;
- concurrent and serial invocations share no Run or lease state, under `-race`.

Live, `TestTestpilotCanaryHarnessRunsARepointedEndpointIncomplete` shows routing drift ends as an incomplete Run with its receipt and exit 1. `TestTestpilotCanaryHarnessRecoversALostProcess` now crashes at all four phases. The cmd test pins a report stdout cannot take as exit 3.

Each new in-process test failed under a mutation of the code it guards; every mutation was reverted.

Follow-up: the harness redactor in `tools/canary/testharness` still lists the coordinates by hand. The fix belongs in `tools/canary/authority`, outside this task. Environment note: on this host, Lean's bundled clang shadows `/usr/bin/clang` on PATH and breaks cgo. The gates ran with `CC=/usr/bin/clang`.

stage: impl-review - ran [2026-09-26T22:20..2026-09-26T22:24] SHIP (claude:opus:high, round 1; its four P3 notes applied)
## Evidence
- Commits: d463834cce612befd662ea8647f5327f6c221d08, 89433664bb05a162577afdfec4fd2074e8e2a8f6
- Tests: baseline: green (CC=/usr/bin/clang; Lean's bundled clang on PATH lacks stddef.h), go test -race -count=1 -tags test_dep ./tools/canary/..., go test -race -count=1 -tags "test_dep canary_harness" ./tools/canary/..., go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotCanary', make lint-code-fast
- PRs: