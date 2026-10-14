---
satisfies: [R1, R6]
---
# fn-64-umpire-case-runtime.2 Compile shared typed IR and descriptor paths

## Description
Build the reusable, immutable typed IR binder consumed by Program admission and Contract admission.
This is the foundational portion of the former preparation task; full Case admission and the root
facade are assigned to tasks 11–13, with no requirement removed.

**Size:** M
**Files:** `tools/umpire/internal/ir/{catalog,type,path,expression}.go` and focused tests
**Touches:** [tools/umpire/internal/ir/**]

## Approach
- Snapshot and validate a descriptor catalog without I/O; bind arbitrary unary methods and preserve
  exact immutable catalog identity. Reject unresolved, duplicate/conflicting and streaming methods.
- Compile scalar, enum, message, WKT and whole-Any types and literals with exact cardinality, canonical
  numeric representation/ranges, defined enum values, and bounded nested payload validation.
- Compile descriptor paths once. Ordinary segments name protobuf fields; a oneof selector names the
  oneof group in `field` and its selected member in `selected_field`. A presence selector yields a
  boolean. Optional/oneof/message traversal and literal map-key lookup retain possible absence;
  repeated wildcards retain fan-out, while whole repeated/map leaves retain collection cardinality.
  Whole typed Any copies are admitted; unpacked Any traversal and opaque capability inspection reject.
- Bind the closed expression vocabulary using a caller-supplied typed reference/presence environment.
  Program and Contract graph/dataflow analysis belong to their respective admission tasks. Do not
  infer reference availability from a declared type alone or conflate absent values with zero values.
- Keep all catalog/type/path/expression state immutable. Errors carry stable categories and bounded
  paths; shared work/depth/fan-out accounting must reject overflow and malformed nested inputs.

## Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/value.proto:28` — exact scalar and expression vocabulary
- `proto/internal/temporal/server/api/umpire/v1/program.proto:63` — assignments and response projections
- `tools/umpire/testplan/validate.go:63` — pre-I/O validation and bounded proto surface inspection
- `tools/umpire/testplan/plan.go:49` — immutable admitted clone precedent
- `tools/umpire/cmd/umpire-gen-lean-api/model.go:109` — established descriptor construction
- `.flow/memory/bug/integration/portable-schemas-must-preserve-source-2026-09-03.md` — source type/cardinality preservation

## Acceptance
- [ ] Tests bind multiple unrelated unary methods and reject missing, conflicting, malformed and
  streaming descriptors; mutation of source descriptor data cannot change the compiled catalog.
- [ ] Table-driven type/literal tests cover every scalar kind, canonical numeric limits, named enums,
  messages/WKTs, whole typed Any, repeated/map cardinality, unknown fields and crossed types.
- [ ] Path tests cover nested fields, presence/oneof selection, repeated `[*]`, literal map keys,
  whole collection leaves, WKT fields and possible absence; unknown selectors, illegal Any traversal,
  capability inspection, type/cardinality mismatch and fan-out overflow reject deterministically.
- [ ] Expression tests cover every variant, reference types, explicit presence facts, operator/type
  mismatch, unavailable unguarded references, nil/unknown nodes, depth/work bounds and overflow.
  Admission callers own graph-specific availability and capture single-assignment proofs.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/ir/...` and applicable format/lint gates
  pass. Tests use fixed descriptor fixtures and expected types, without Lean or target I/O.

## Done summary
Implemented the immutable internal IR catalog, exact types/literals, compiled descriptor paths, and closed expression binder. Contextual numeric source types, explicit presence facts, catalog ownership, malformed unions, unknown fields, collection bounds, and pre-decode work/depth accounting reject invalid inputs without target I/O.

Catalog tests cover unrelated unary methods, malformed/duplicate/unresolved descriptors, streaming rejection and source mutation. Type/path/expression tables cover every scalar and expression variant, enums/messages/WKT/Any, cardinality, optional/oneof/map/wildcard paths, immutable accessors and concurrent reuse. Focused red/green regressions fixed ambiguous numeric inference, crossed presence sources, payload collection ceilings, and pre-decode work checks including proto2 groups and packed scalars. Existing legacy clone/surface checks and generator descriptor construction informed the new module; their artifact-specific implementations were not reusable.

Baseline: green; owned broad Umpire Go and scoped no-fix lint checks passed before edits. Final focused unit verification, race tests, make fmt-imports, and scoped make lint-code passed. The repository's inherited main-based lint backlog remains outside this task. Logs: .flow/tmp/fn64-task2-baseline{,-lint}.log, fn64-task2-{verify,race-final,format-final,lint-final}.log. Focused commands do not mint full-suite receipts.

Original task 2 was split across tasks 2/11/12/13 while retaining all R1–R10 coverage. Fresh plan review SHIP: /tmp/plan-review-receipt-fn-64-umpire-case-runtime.json. Program/policy and reservation admission, Contract capture analysis, and root composition remain assigned to later tasks; this task adds no runtime evaluator or public Run stub.

No commits authored: changes and receipts remain staged for the user. Review round one returned NEEDS_WORK because the installed wrapper supplied an empty commit range. The conductor-approved task-local launcher overrides only snapshot capture and uses an immutable git write-tree object; normal read-only review, artifact hashing, retry caps, and verdict persistence remain intact. The corrected review returned SHIP; the reviewer could not run tests inside its read-only sandbox, so the owned green gates provide execution evidence.

Reviewed staged tree: fb6a1f31183460d9bd9b63d88fa46248d4ca573f (not a commit); actual HEAD: 54042ae673e80c6c290ecc7ff74ac55304792d6b. Snapshot metadata: .flow/tmp/fn64-task2-review-snapshot.json. Review logs: .flow/tmp/fn64-task2-review{,-staged}.log. Receipt: /tmp/impl-review-receipt-fn-64-umpire-case-runtime.2.json. The final code matches the reviewed tree.

stage: plan-review - ran (model:gpt-5.6-sol)
stage: impl-review - ran; configured codex:gpt-5.6-sol:high, terminal SHIP after correcting staged-tree scope
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green; broad Umpire Go and scoped no-fix lint passed before implementation, CGO_ENABLED=0 TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -count=1 -tags test_dep ./tools/umpire/..., CGO_ENABLED=0 TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -count=1 -tags test_dep ./tools/umpire/internal/ir/..., CC=/usr/bin/clang TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go test -race -count=1 -tags test_dep ./tools/umpire/internal/ir/..., CC=/usr/bin/clang TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp make fmt-imports, CC=/usr/bin/clang TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, git diff --quiet fb6a1f31183460d9bd9b63d88fa46248d4ca573f -- tools/umpire/internal/ir
- PRs: