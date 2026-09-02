---
satisfies: [R3, R4, R8]
---

# fn-28-portable-evaluation-contract-and.4 Interpret portable Observation, link, and Property clauses in Go
## Description

Implement the generic bounded Go interpreter that consumes one admitted contract plus closed Raw Evidence and produces the existing detailed Run Evaluation dimensions without invoking Lean.

**Size:** L
**Files:** `tools/umpire/portableevaluation/**`
**Touches:** [`tools/umpire/portableevaluation/**`]

### Approach
- Normalize only contract-declared fields, validate source/correlation/causal and source-local order, enforce closure, construct the exact System trace, apply the bundled link, and evaluate the bundled Property clauses.
- Preserve `satisfied`, `violated`, `unknown`, `conflict`, and `unsupported`; never turn missing closure, ambiguity, unsupported data, or deadline into success or a false violation.
- Implement only the parent spec's exact version-one operator table, tagged types, diagnostic mapping, canonical evaluation order, and precharged work accounting; keep every operator switch exhaustive and fail closed on unknown values.

### Investigation targets

**Required** (read before coding):
- Parent spec, contract proto/admission, and existing `runevaluation` protocol/result validation.
- `Umpire.Observation.Evaluation`, `Umpire.Observation.Check`, and caller-closure Run Evaluation tests.
- Existing Evidence Link, disposition, causal-order, source-closure, and Limit representations.

## Acceptance
- [ ] Focused fixtures cover every version-one operator's success/failure/type/missing/N/N+1 branches plus accepted/satisfied, accepted/violated, unknown, conflict, unsupported, and incomplete closure without Lean.
- [ ] Every accepted Model Fact and clause retains auditable Evidence Links and exact contract bindings.
- [ ] Mutation, N/N+1, cancellation, race, fuzz, and lint checks pass without adding a second model registry.

## Done summary
Finalized the previously implemented bounded portable Go evaluator after auditing implementation commits `b26171754eb445bc4bce8877a4e00533e68f7dc0`, `de6c6824f4deddb51933ccad03310e797dd08d41`, and `4b987fa54b76ee86aa9900221b86fc12d95a4499`, receipt commit `7c16a3aad7cafd46ca2ebd44bf42d471b512a20c`, plan-sync record `cf098fb1f84dd14252a0b73192c879798c916590`, and the authoritative Codex SHIP review over `9ddc5d8ed8eeca85fd5e2ddc2cb82007077a260d..4b987fa54b76ee86aa9900221b86fc12d95a4499`. Focused unit, race, fuzz, vet, and exact reviewed-range lint verification is green; no product edit was warranted, and the unrelated user-owned config/schema modifications remain untouched.

The package-wide lint command now reports three `revive` findings solely in `parity_test.go`, which was introduced by later task `.5`; task `.4`'s exact reviewed patch remains lint-clean. The initial fuzz tool invocation hit a Go telemetry mmap `SIGBUS` in the constrained host environment before compilation; isolating `TEST_TELEMETRY_DIR` under the workspace produced a green fuzz run.

baseline: green (focused portable evaluator unit/race/fuzz/vet and exact task-patch lint; no task code changed during finalization)

GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification

stage: impl-review - ran [2026-09-01T20:07:48Z..2026-09-01T20:23:48Z] (authoritative SHIP receipt reused after empty finalization diff)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green - focused portable evaluator unit/race/fuzz/vet and exact task-patch lint; no task code changed during finalization, TMPDIR=$PWD/.flow/tmp/fn28_4_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_4_tmp go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... (pass), TMPDIR=$PWD/.flow/tmp/fn28_4_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_4_tmp go test -race -count=1 -tags test_dep ./tools/umpire/portableevaluation/... (pass), TMPDIR=$PWD/.flow/tmp/fn28_4_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_4_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_4_tmp/go-telemetry go test -tags test_dep ./tools/umpire/portableevaluation -run '^$' -fuzz '^FuzzEvaluateFailsClosed$' -fuzztime=3s (pass), TMPDIR=$PWD/.flow/tmp/fn28_4_tmp GOTMPDIR=$PWD/.flow/tmp/fn28_4_tmp TEST_TELEMETRY_DIR=$PWD/.flow/tmp/fn28_4_tmp/go-telemetry go vet -tags test_dep ./tools/umpire/portableevaluation/... (pass), git diff --binary 9ddc5d8ed8eeca85fd5e2ddc2cb82007077a260d..4b987fa54b76ee86aa9900221b86fc12d95a4499 -- tools/umpire/portableevaluation > $PWD/.flow/tmp/fn28_4_tmp/task4.patch && .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml --new-from-patch $PWD/.flow/tmp/fn28_4_tmp/task4.patch ./tools/umpire/portableevaluation/... (pass: 0 issues), INHERITED_RED: .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/portableevaluation/... - three revive findings only in later task .5 file tools/umpire/portableevaluation/parity_test.go, TOOLING_RECOVERY: initial fuzz invocation hit Go toolchain telemetry mmap SIGBUS before package compilation; workspace-isolated TEST_TELEMETRY_DIR retry passed, GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification, AUTHORITATIVE_REVIEW_SHIP: 9ddc5d8ed8eeca85fd5e2ddc2cb82007077a260d..4b987fa54b76ee86aa9900221b86fc12d95a4499
- PRs: