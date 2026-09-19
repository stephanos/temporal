---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-63-consolidate-umpire-go-tests-into-golden.3 Consolidate Run Evaluation and Nexus scenarios

## Description
Reuse the shared scenario contract for complete admitted-set-to-Result behavior spanning Nexus Evidence and Run Evaluation (R2/R3/R4/R5/R6). Keep process, protocol, Temporal lifecycle, and concurrency guarantees in focused tests.

**Size:** M
**Files:** `tools/umpire/runevaluation/integration_test.go`, `tools/umpire/runevaluation/mutation_test.go`, `tools/umpire/runevaluation/result_test.go`, `tools/umpire/temporal/nexus/*_test.go`, `tools/umpire/temporal/nexus/testdata/**`
**Touches:** [tools/umpire/runevaluation/integration_test.go, tools/umpire/runevaluation/mutation_test.go, tools/umpire/runevaluation/result_test.go, tools/umpire/runevaluation/testdata/**, tools/umpire/temporal/nexus/*_test.go, tools/umpire/temporal/nexus/testdata/**]

### Approach
- Replace repeated accepted-set assembly and stable result assertions with shared normal and duplicate-delivery scenarios backed by the existing input/run-set manifests.
- Extend the smallest scenario set needed for missing closure and correlation conflict while reusing canonical Artifact members rather than copying payloads.
- Assert exact admitted Artifact identities and stable semantic Results; validate fresh run/workflow/task-queue/correlation/Evidence identities structurally instead of snapshotting them.
- Replace every ordinary-test `realCheckerProcess` semantic call with checked independent response fixtures or an injected checker boundary. Move real Lean checker execution exclusively into explicit generation/check commands.
- Retain process framing, byte/timeout bounds, and child cleanup against the existing Go fixture executable; keep causal mutation precedence, resource ownership, eventual source closure, and concurrent lifecycle cases focused.
- Map every removed test to a scenario or retained invariant category in the task completion summary.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/runevaluation/integration_test.go:26-175` — repeated real-checker setup and stable outcome assertions
- `tools/umpire/runevaluation/integration_test.go:269-291` — current ordinary-test Lean build dependency to remove
- `tools/umpire/runevaluation/mutation_test.go:82-605` — mutation/precedence coverage to partition carefully
- `tools/umpire/runevaluation/checker_test.go:44-498` — Go fixture protocol, bounds, and process cases to retain
- `tools/umpire/temporal/nexus/testdata/caller-closure-input-set/manifest.json` — existing admitted input set
- `tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-run-set/manifest.json` — existing violating run set

**Optional** (reference as needed):
- `tools/umpire/runevaluation/README.md:125-184` — live proof and stable/dynamic comparison contract

## Acceptance
- [ ] Normal, duplicate-delivery, missing-closure, and correlation-conflict scenarios reuse admitted set members and cover the complete stable Run Evaluation outcomes.
- [ ] Deterministic Artifacts and identities remain exact; runtime-assigned values remain non-empty, unique, correctly linked, and otherwise structurally valid without being goldenized.
- [ ] Ordinary `go test -count=1 -tags test_dep` uses checked response fixtures or an injected checker boundary and passes when Lean/mise/lake are unavailable.
- [ ] Real Lean parity runs only in explicit generation/check commands; process framing/bounds/cleanup continue against the Go fixture executable.
- [ ] Mutation precedence, source closure, Temporal resource ownership, and concurrency cases remain focused and retain their current failure classes.
- [ ] Repeated set-loading and result-assertion code is removed, with every removed test mapped in the task summary.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
