---
satisfies: [R1, R2, R6, R7, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.12 Close the schema and aggregate regression matrices

## Description
Complete the matrices across the canary Case binding, the canary Profile, the policy, the provenance, the receipt beside it, versions, canonical bytes, secrets, Known Gaps and Limits, and run every gate: `cd model && lake build`, `LEAN_NUM_THREADS=1 make lint-model` at its baseline, `make umpire-check-regression` (with `canary-check-case`, the extended Profile check, `./tools/canary/...` appended to the end of its pinned package-local Go test line, and a `go test -tags 'test_dep canary_harness' ./tools/canary/testharness/` line), the same additions made to `.github/workflows/umpire.yml` in a canary job of its own with a 30-minute timeout and pinned by `tools/umpire/regression/ci_workflow_test.go`, so CI runs the canary's unit tests and checks, not only its live tests, `go test -tags test_dep` over `./tools/canary/... ./tools/umpire/...`, the live suite's selection `^TestTestpilot` (which includes `TestTestpilotCanary*`), race, formatting and `make lint-code-fast`.

### Quick commands
`make umpire-check-regression; go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/...; make lint-code-fast`

**Files:** `model/Temporal/Evaluation/CanaryTests.lean`, `tools/canary/**`, `Makefile`, `.github/workflows/umpire.yml`, `tools/umpire/regression/ci_workflow_test.go`
**Touches:** `model/Temporal/Evaluation/CanaryTests.lean`, `tools/canary/**`, `Makefile`, `.github/workflows/umpire.yml`, `tools/umpire/regression/ci_workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Cross-language canonical fixtures agree and every prior or other version rejects without an alias.
- [ ] The canary's live tests run under `umpire-check-live-tests` by the `^TestTestpilot` selection; Go tests use `-tags test_dep` and `integration` only where required.
- [ ] Existing comments stay accurate and import ownership stays one-way: the canary imports Umpire, never the reverse.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
