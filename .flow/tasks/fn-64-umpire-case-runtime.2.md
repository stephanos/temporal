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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: