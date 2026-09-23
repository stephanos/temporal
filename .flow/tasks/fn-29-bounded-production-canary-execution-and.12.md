---
satisfies: [R1, R2, R6, R7, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.12 Close the schema and aggregate regression matrices

## Description
Complete the matrices across the canary Case binding, the canary Profile, the policy, the provenance, the receipt beside it, versions, canonical bytes, secrets, Known Gaps and Limits, and run every gate: `cd model && lake build`, `LEAN_NUM_THREADS=1 make lint-model` at its baseline, `make umpire-check-regression` (with `canary-check-case` and the extended Profile check), `go test -tags test_dep` over `./tools/canary/... ./tools/umpire/...`, the integration selection `^TestUmpireCanary`, race, formatting and `make lint-code-fast`.

### Quick commands
`make umpire-check-regression; go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/...; make lint-code-fast`

**Files:** `model/Temporal/Evaluation/CanaryTests.lean`, `tools/canary/**`, `Makefile`
**Touches:** `model/Temporal/Evaluation/CanaryTests.lean`, `tools/canary/**`, `Makefile`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Cross-language canonical fixtures agree and every prior or other version rejects without an alias.
- [ ] The integration selection is `^TestUmpireCanary`; Go tests use `-tags test_dep` and `integration` only where required.
- [ ] Existing comments stay accurate and import ownership stays one-way: the canary imports Umpire, never the reverse.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
