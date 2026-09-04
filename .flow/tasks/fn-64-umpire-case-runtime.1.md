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
  and closed instruction/transition unions.
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
  property-specific variants.
- [ ] Focused wire/ProtoJSON round-trip tests include source-shaped values and crossed
  kind/cardinality failures.
- [ ] `make umpire-gen-lean-api` and `go test -count=1 -tags test_dep
  ./tools/umpire/cmd/umpire-gen-lean-api` pass while legacy consumers still compile.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
