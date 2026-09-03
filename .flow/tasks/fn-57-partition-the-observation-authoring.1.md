---
satisfies: [R1, R5, R6]
---
# fn-57-partition-the-observation-authoring.1 Extract inert Observation declarations

## Description
Move the authored value types, profiles, bounds, expressions, dispositions, bindings, rules, ordering, closures, mappings, and existing field projections intact into a low-dependency Declaration module. Narrow the portable contract to that declaration-only import.

**Size:** M
**Files:** `model/Umpire/Observation/Declaration.lean`, `model/Umpire/Observation/Language.lean`, `model/Umpire/Artifact/PortableEvaluationContract.lean`, `model/Umpire/Observation/Tests/Compilation.lean`
**Touches:** [model/Umpire/Observation/Declaration.lean, model/Umpire/Observation/Language.lean, model/Umpire/Artifact/PortableEvaluationContract.lean, model/Umpire/Observation/Tests/Compilation.lean]

### Approach
- Move all inert authored declarations and their existing comments into `Umpire.Observation.Declaration`, retaining namespace `Umpire` and every fully qualified name, field, default, coercion, derivation, and record layout.
- Keep `ObservationFieldSpec.declaration`, `.reference`, `.expression`, and `.disposition` beside the vocabulary they project.
- Import only the lowest sufficient reusable substrate; do not pull Target, DefinitionGraph, Evaluation, or Temporal into Declaration.
- Change Portable Evaluation Contract to import Declaration directly because it consumes declaration data rather than checker mechanics.
- Preserve the expert raw-record path and prove the existing field projections remain exact.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Language.lean:18-190` — declaration and field-projection ownership
- `model/Umpire/Artifact/PortableEvaluationContract.lean:1-11` — current broad language import
- `model/Umpire/Artifact/PortableEvaluationContract.lean:95-180` — declaration-only types consumed by portable data
- `model/Umpire/Observation/Tests/Compilation.lean:10-32` — exact field-projection equality
- `model/Umpire/Observation/Tests/Compilation.lean:185-236` — checker-owned projected-field failures
- `model/Umpire/Observation/Tests/Fixtures.lean:20-137` — ordinary reusable declarations

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus/Observation.lean:28-173` — public-facade authoring consumer

### Key context
- Declaration data remains inert; this task must not validate, normalize, register, or choose defaults.
- Preserve existing comments and all public facade imports even though one internal consumer becomes narrower.

## Acceptance
- [ ] R1 is satisfied with exact fully qualified names, constructors, field order/defaults, coercions, derivations, comments, and documented meaning.
- [ ] Field-specification declaration, reference, expression, and disposition projections remain exactly equal to the existing inert records.
- [ ] Invalid raw declarations remain representable and are still rejected only by `checkObservation`; no validation, normalization, registration, or default disposition moves into Declaration.
- [ ] Declaration has no Target, DefinitionGraph, Evaluation, Temporal, callback, registry, or new dependency.
- [ ] Portable Evaluation Contract compiles against Declaration directly without public-surface or artifact-byte change.
- [ ] `cd model && mise exec -- lake build Umpire.Artifact.PortableEvaluationContract Umpire.Observation.Tests.Compilation` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
