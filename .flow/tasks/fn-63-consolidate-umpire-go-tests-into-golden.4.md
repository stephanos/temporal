---
satisfies: [R2, R3, R4, R5]
---
# fn-63-consolidate-umpire-go-tests-into-golden.4 Consolidate Artifact family acceptance goldens

## Description
Collapse repeated whole-Artifact acceptance and cross-Artifact closure checks into the existing canonical corpus (R2/R3/R4/R5), while retaining granular codec, bounds, checksum, and precedence tests.

**Size:** M
**Files:** `tools/umpire/artifact/golden_test.go`, `tools/umpire/artifact/result_test.go`, `tools/umpire/artifact/runtime_test.go`, `tools/umpire/artifact/set_execution_test.go`, `tools/umpire/artifact/testdata/**`
**Touches:** [tools/umpire/artifact/golden_test.go, tools/umpire/artifact/result_test.go, tools/umpire/artifact/runtime_test.go, tools/umpire/artifact/set_execution_test.go, tools/umpire/artifact/testdata/**]

### Approach
- Extend the six-family cross-language golden and valid admitted-set corpus for repeated whole-document success/round-trip/closure cases; reuse files by reference rather than embedding copies in Go tests.
- Represent only coherent whole-document negative cases as input plus expected typed error code/precedence metadata. Keep malformed-token position, N/N+1 Limits, checksum recomputation, collection closure, and multi-fault precedence as direct mutations.
- Require every expected canonical document to admit and validate independently before comparison so the corpus cannot self-confirm invalid output.
- Preserve exact canonical bytes, field/element order, checksums, unknown-major/unknown-field rejection, and current error categories.
- Map every removed test to a golden scenario or retained invariant category in the task completion summary.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/artifact/golden_test.go:23-90` — current six-family exact-byte corpus
- `tools/umpire/artifact/result_test.go:1057-2060` — large result validation and precedence matrix
- `tools/umpire/artifact/runtime_test.go:587-1329` — runtime bounds/closure cases
- `tools/umpire/artifact/limits_test.go:42-260` — N/N+1 Limits coverage to retain
- `tools/umpire/artifact/json_test.go:36-527` — grammar, canonicality, and parser precedence to retain

**Optional** (reference as needed):
- `tools/umpire/artifact/testdata/valid-run-evaluation-set/manifest.json` — existing complete-set golden

## Acceptance
- [ ] Whole-document success, exact round-trip, and selected coherent closure/error cases use the canonical six-family or admitted-set golden corpus.
- [ ] Every expected document independently admits and validates before it serves as an oracle, and deterministic bytes/checksums remain exact.
- [ ] Grammar/fuzz, N/N+1, checksum, collection-closure, unknown-version/field, and multi-fault precedence coverage remains focused where mutation locality matters.
- [ ] Repeated literal builders and superseded broad validation tables are removed, with every removed test mapped in the task summary.
- [ ] Artifact package tests pass with `go test -count=1 -tags test_dep`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
