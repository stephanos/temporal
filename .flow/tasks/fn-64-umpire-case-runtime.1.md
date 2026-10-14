---
satisfies: [R1, R10]
---
# fn-64-umpire-case-runtime.1 Define the versioned Umpire Case IR

## Description
Define the split Case/Program/Contract/Run protobuf contracts and their generated Go and Lean
representations (R1). Keep legacy messages temporarily buildable so later tasks can migrate
consumers before the hard cutover.

**Size:** M
**Files:** `proto/internal/temporal/server/api/umpire/v1/{value,program,contract,run,case}.proto`,
generated `api/umpire/v1/*.pb.go`, generic Case IR modules under `model/Umpire/`, Lean API generator
inputs/tests, focused schema documentation
**Touches:** [proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, model/Umpire/**,
tools/umpire/cmd/umpire-gen-lean-api/**, Makefile]

## Approach
- Split values, Programs, Contracts, Runs, and Cases by ownership while keeping one version envelope
  and closed instruction/transition unions. Add declared scalar captures, capture references and
  transition assignments with source-event support and count/byte/work bounds.
- Encode entrypoint context, activation identity, typed outcomes, Slots, Observations, bounds,
  cleanup, monotonic elapsed Run coordinates, disposition, and Verdict without endpoint secrets.
- Preserve source semantic kinds, cardinalities, and identity-bearing compiler data rather than
  reconstructing them from lossy limits.
- Extend the established protobuf-to-Lean generation path; generated files remain generator-owned.

## Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto:355` — legacy message
- `proto/internal/temporal/server/api/umpire/v1/message.proto` — current shared wire types
- `proto/internal/temporal/server/api/umpire/v1/service.proto:14` — legacy service dependency
- `Makefile:509-530` — protobuf and Lean API generation wiring
- `model/Umpire/Artifact/PortableEvaluationContract.lean:415-535` — current Lean IR mirror

**Optional** (reference as needed):
- `proto/internal/buf.yaml:18-24` — Umpire proto lint exceptions
- `.flow/memory/bug/integration/portable-schemas-must-preserve-source-2026-09-03.md` —
  type/cardinality regression lesson

## Key context
The schema is the portable public contract; Lean is one Producer, not a runtime dependency. Preserve
existing comments when declarations move.

## Acceptance
- [ ] Five schema areas generate matching Go and Lean types with stable version/ID semantics and no
  Temporal endpoint or credential fields.
- [ ] Program data represents the approved context matrix, entrypoint-local DAGs, typed
  outcomes/guards, Slots/Observations, cleanup, and global/per-instruction bounds.
- [ ] Contract and Run data represents deterministic monitors, recorded monotonic horizon facts,
  supporting event references, dispositions, cleanup outcome, and Verdict precedence without
  property-specific variants. Horizons use expiry-before-transition ordering on every event;
  witnesses at the deadline are late. Capture types, references, and assignments round-trip in
  both languages.
- [ ] Focused wire/ProtoJSON round-trip tests include source-shaped values and crossed
  kind/cardinality failures.
- [ ] `make umpire-gen-lean-api` and `go test -count=1 -tags test_dep
  ./tools/umpire/cmd/umpire-gen-lean-api` pass while legacy consumers still compile.

## Done summary
Defined the five versioned Case/Program/Contract/Run/value schema areas with generated Go and Lean representations, typed captures, source semantics/presence, cleanup graph identity, and explicit event horizon ordering. Source-shaped wire/ProtoJSON and Lean checks cover the review fixes; legacy consumers remain buildable for later migration tasks.

Authored generic helpers live under `model/Umpire/**`; deterministic generator-owned mirrors live under `model/Temporal/API/**`. Preserved externally updated task/spec material and target-neutral Lean fixtures. The concurrent `dbe92e0cb164403da2b919c5e479680c13ea4560` planning/protobuf-drift commit is preserved but excluded from task commits and evidence; unrelated runtime and abandoned `tools/umpire1` fixes remain staged for their owner.

Baseline: task-scoped schema/Lean checks green on resume; default make lint-code red before edits. Default main-based lint still has 1,361 unrelated inherited findings; task-base scoped make lint-code passes completely including errortype. Initial broad Go failures came from Lean clang missing stddef.h and /var temp symlink; physical TMPDIR and CGO_ENABLED=0 yield all Umpire Go packages green.

Regression evidence combines the external full selector/prerequisite run in .flow/tmp/task1_verify_regression.log (explicit accepted legacy failure policy, no Umpire4 or unclassified failures) with an owned successful trailing recipe in /tmp/fn64-resume-regression-tail.log. The initial full run failed the 90s caller-closure harness bound; supported TEMPORAL_TEST_TIMEOUT=315s and CC=/usr/bin/clang yield a passing focused portability run. The external run subsequently exposed the generic-fixture vocabulary guard; preserved target-neutral fixture fixes pass that guard and rebuild CaseTests/UmpireTests in the owned trailing run. No full-suite green receipt was fabricated for these composed/dirty-tree checks.

All current task-1 acceptance criteria are covered by schema generation, source-shaped Go/Lean tests, legacy compilation, scoped lint, and the final SHIP review. Runtime admission/capture evaluation and legacy hard cutover remain owned by later tasks. Initial re-review preceded the final broad gates; the final re-review followed green scoped/composed gates and included current staged fixes.

stage: impl-review - ran [2026-09-04T19:25:25Z..2026-09-04T20:00:20Z]; Codex fix loop reached SHIP, final response `/tmp/fn64-resume-review-final.log`; receipt `/tmp/impl-review-receipt-fn-64-umpire-case-runtime.1.json`

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8a4b9968ce7a403992ed38a1711e5551f2214b01, 7c8f79eaecaafa5d93c987c48ea72c898f52c0e9, 979ffa17e1d14f300e1e96b0cbfb71ef3a93530b, 2ea8f9ee336e4388616a3360116436f67333631b
- Tests: make proto (passed; /tmp/fn64-resume-proto.log), make umpire-gen-lean-api (passed; /tmp/fn64-resume-generate-final.log), go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api (passed; /tmp/fn64-resume-focused.log), make umpire-build-model (passed; /tmp/fn64-resume-model.log), cd model && mise exec -- lake env lean Umpire/CaseTests.lean (passed; /tmp/fn64-resume-case-lean.log); changed fixture rebuilt by final regression recipe, go test -count=1 -tags test_dep ./tools/umpire/runtime ./tools/umpire/cmd/umpire-gen-lean-api ./tools/umpire1 (passed; /tmp/fn64-resume-error-green.log), go test -count=1 -tags test_dep ./tools/umpire/runtime (passed external-package regression; /tmp/fn64-resume-errors-final.log), CGO_ENABLED=0 TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -count=1 -tags test_dep ./tools/umpire/... (passed; /tmp/fn64-resume-go-portable.log), make fmt-imports (passed; /tmp/fn64-resume-fmt-gate.log), make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false (passed including errortype; /tmp/fn64-resume-lint-gate.log), make lint-code (default main-based gate failed on 1361 inherited branch findings; /tmp/fn64-resume-lint-final.log), make umpire-check-regression (initial live gate failed portability harness90s timeout; /tmp/fn64-resume-regression.log), CC=/usr/bin/clang TEMPORAL_TEST_TIMEOUT=315s TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp mise exec -- go test -count=1 -tags "test_dep integration" ./tests -run "^TestUmpireCallerClosurePortability$" (passed69.265s; /tmp/fn64-resume-portability-env.log), Full ^TestUmpire selector from external make umpire-check-regression: explicit accepted legacy-failure policy sentinel, no Umpire4/unclassified failures; .flow/tmp/task1_verify_regression.log. Prerequisites passed before final vocabulary guard failure; that guard was corrected and rerun below., CC=/usr/bin/clang TEMPORAL_TEST_TIMEOUT=315s TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp make umpire-check-regression -o umpire-check-regression-views -o umpire-check-generated-go-test -o umpire-check-portable-evaluation-fixtures -o umpire-check-promotion -o umpire-check-legacy-vocabulary -o umpire-check-live-tests (passed all trailing guards and Lean tests; /tmp/fn64-resume-regression-tail.log), Codex implementation review SHIP; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.1.json; /tmp/fn64-resume-review-final.log
- PRs:
