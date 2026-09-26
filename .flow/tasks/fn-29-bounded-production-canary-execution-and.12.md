---
satisfies: [R1, R2, R6, R7, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.12 Close the schema and aggregate regression matrices

## Description
Complete the matrices across the canary Case binding, the canary Profile, the policy, the provenance, the receipt beside it, versions, canonical bytes, secrets, Known Gaps and Limits, and run every gate: `cd model && lake build`, `LEAN_NUM_THREADS=1 make lint-model` at its baseline, `make umpire-check-regression` (with the extended Profile check, `canary-check-case` as .2 added it, `./tools/canary/...` appended to the end of its pinned package-local Go test line, and a `go test -tags 'test_dep canary_harness' ./tools/canary/testharness/` line, and an explicit 30-minute `-timeout` on the live gate's Go test), the same additions made to `.github/workflows/umpire.yml` in a canary job of its own with a 30-minute timeout and pinned by `tools/umpire/regression/ci_workflow_test.go`, so CI runs the canary's unit tests and checks, not only its live tests, `go test -tags test_dep` over `./tools/canary/... ./tools/umpire/...`, the live suite's selection `^TestTestpilot` (which includes `TestTestpilotCanary*`), race, formatting and `make lint-code-fast`.

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
The canary's schema matrices are closed, and the canary now runs in the one regression gate and in CI.

- `tools/canary/schema_test.go` covers six documents: the policy, both rendered canary Profiles, a receipt, a provenance and the recovery record. Each decodes from its canonical bytes and rejects version 0, version 2, and every alias of 1: a string, a float, a negative, a missing field and a case-folded key.
- `TestBindRefusesEveryOtherCaseVersion` re-pins the policy to each re-versioned Case, so the refusal it proves is Testpilot's `unsupported Case version`, not the identity check.
- Lean and Go pin the same Profile identities. The harness test now pins the `canary-harness` identity from `CanaryTests.lean`, beside the existing `production-canary` pin.
- `umpire-check-regression` appends `./tools/canary/...` to its package-local Go test line. It adds a `test_dep canary_harness` line for `./tools/canary/testharness/` and gives the live gate's Go test `-timeout 30m`.
- `.github/workflows/umpire.yml` gains a 30-minute `canary` job that runs the canary's unit tests, the harness build's tests and `make canary-check-case umpire-check-evaluation-profiles`. The portability job goes to 40 minutes so the live suite's Go timeout can fire. `ci_workflow_test.go` pins all of it.

Carried follow-ups:
- The harness Redactor now comes from `authority.Coordinates.Redactor`, the one coordinate list production's `Load` also uses. A mutation that dropped a coordinate from that list failed the harness test.
- A deterministic test now drives `run` (`controller.Invoke`) to `publication-unreported`, exit 3. It uses an unexported `publish` hook on `Invocation`, as the existing `service` and `prepare` hooks do, so no exported surface was added. The recovery record stays in `publishing`.

stage: impl-review - ran [2026-09-26T22:43..2026-09-26T22:55] SHIP (claude, round 2; round 1 NEEDS_WORK: CI job timeout shorter than the new Go timeout, and a Redactor comment, both fixed)
## Evidence
- Commits: dc5023f466cf614738005be51ec5002dfb3b06c7, 13b4835183617299dcd1a45b68caced65011b8f1, 673b60c203c9238d58293063ea2e3f7297a652e4
- Tests: baseline: green (go test -tags test_dep ./tools/canary/... ./tools/umpire/...; make umpire-check-regression, 1792s); lint-code-fast red pre-edit (inherited typecheck in tools/umpire/cmd/umpire-export-proto-descriptors/testdata), go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/... (pass), go test -race -count=1 -tags test_dep ./tools/canary/... (pass), go test -race -count=1 -tags 'test_dep canary_harness' ./tools/canary/testharness/ (pass), cd model && lake build (pass, 599 jobs, 2s warm), LEAN_NUM_THREADS=1 make lint-model (3 inherited diagnostics in Temporal/API/Proto.lean and Umpire/Command/Refinement.lean; no model file changed by this task), make umpire-check-regression (pass, 869s; live: empty failure set across 45 passing identities, 13 TestTestpilotCanary* passes; ./tools/canary/... and the canary_harness line ran), make lint-code-fast (32 staticcheck findings, all in tests/*.go outside this task; none in changed files), mutation checks: harness redactor list, publication-unreported mapping (both red, reverted), impl-review: SHIP (claude, round 2)
- PRs: