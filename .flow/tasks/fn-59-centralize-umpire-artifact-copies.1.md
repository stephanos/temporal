---
satisfies: [R1, R2, R4]
---
# fn-59-centralize-umpire-artifact-copies.1 Establish the internal artifact copy authority

## Description
Create the cohesive internal copy surface that implements R1-R2 before changing its callers. Keep the surface root-oriented so artifact and runtime can copy owned values without learning the nested representation.

**Size:** S
**Files:** `tools/umpire/internal/artifactv2/clone.go`, `tools/umpire/internal/artifactv2/clone_test.go`
**Touches:** [tools/umpire/internal/artifactv2/clone.go, tools/umpire/internal/artifactv2/clone_test.go]

### Approach

- Add type-specific root copy operations for the artifact documents currently copied by admission or runtime, and compose private helpers for the nested plan, provenance, Known Gap, pointer, and slice shapes.
- Keep leaf helpers package-private; expose only the roots required across the internal-package boundary and give those roots concise Go documentation.
- Preserve nil versus empty, zero values, order, and every admitted scalar while cloning every mutable descendant in the schema-valid graph exactly once.
- Build table-driven direct tests that mutate sources and copies in both directions and cover Raw Evidence's admitted field-value types: nil, Boolean, string, and canonical integer `json.Number`.
- Keep the package dependency-free beyond the standard library and free of artifact- or runtime-package imports.

### Investigation targets

**Required** (read before coding):
- `tools/umpire/internal/artifactv2/artifact.go:20-142` — artifact roots and nested model ownership
- `tools/umpire/internal/artifactv2/evidence.go:42-54,287-335` — Raw Evidence dynamic field type and admitted scalar domain
- `tools/umpire/artifact/set.go:700-887` — complete existing type-aware copy implementation to consolidate
- `tools/umpire/runtime/engine.go:75-133` — independently duplicated execution/evidence subset

**Optional** (reference as needed):
- `tools/umpire/internal/artifactv2/artifact_test.go` — internal package test conventions
- `tools/umpire/artifact/set_test.go:107-149` — existing mutation-isolation expectations

### Key context

This is a pure value-copy seam over schema-valid artifacts, not validation or normalization. `RawEvidenceField.Value` is dynamically typed, but admitted values are immutable scalars only; composite or custom programmatic values are invalid and outside the isolation guarantee. Do not introduce errors, generic composite copying, reflection, serialization round trips, unsafe operations, generated code, synchronization, or another abstraction layer. Preserve existing comments in every changed file.

### Acceptance

- [ ] R1 is satisfied by one acyclic internal root-copy surface; leaf representation helpers remain hidden and the internal package imports neither artifact nor runtime.
- [ ] R2 is directly tested for nil and empty collections, zero-valued roots, order, independent nested pointer/slice storage, and unchanged nil/Boolean/string/canonical-`json.Number` field values across Experiment, DrivePlan, RuntimeConfiguration, ExperimentRun, RawEvidence, Provenance, and Known Gaps.
- [ ] Tests mutate every current schema-valid mutable descendant family, including property requirements, plan preconditions and checkpoints, authority and participant capabilities, run outcomes/control attempts/cleanup/limits, Evidence causal IDs and field slices, provenance, and Known Gaps.
- [ ] Invalid composite or custom `RawEvidenceField.Value` inputs are not promoted into the copy contract, newly validated, normalized, or generically cloned.
- [ ] No public Umpire API, artifact schema, validation path, generated file, third-party dependency, or existing comment changes.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2` passes.
- [ ] `make fmt-imports` and `make lint-code` are run; any inherited lint failure is recorded against the pre-edit baseline and the task introduces zero scoped findings.
## Acceptance
- [ ] Internal artifact copy authority and direct mutation-isolation tests satisfy R1-R2.
- [ ] Focused tests, formatting, and code-lint verification complete with no task-scoped regressions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
