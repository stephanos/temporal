---
satisfies: [R1, R3, R8]
---
# fn-18-versioned-umpire-artifact-boundary.4 Define runtime configuration and experiment-run transports

## Description
Implement R3's exact RuntimeConfiguration and ExperimentRun schemas/canonical projections and strict Go codecs without adding execution behavior.

**Size:** M
**Files:** `model/Umpire/Artifact/Runtime.lean`, `model/Umpire/Artifact/Tests/Runtime.lean`, `tools/umpire/artifact/runtime.go`, `tools/umpire/artifact/runtime_test.go`
**Touches:** [model/Umpire/Artifact/Runtime.lean, model/Umpire/Artifact/Tests/Runtime.lean, tools/umpire/artifact/runtime.go, tools/umpire/artifact/runtime_test.go]

### Approach
- Define every exact RuntimeConfiguration and ExperimentRun field, nested record, enum, order, bound, identity view, and semantic reference from the parent normative schema.
- Prohibit authority material in RuntimeConfiguration and Property/semantic verdicts in ExperimentRun by construction.
- Implement canonical Lean encoders and strict Go decode/validate/re-encode for both formats.
- Validate the locally persisted profile-required capability projection and exact capability union, phase timestamp/status consistency, control attempts against planned occurrences/requested faults, source-closure gaps, semantic-reference resolution, and experiment/configuration binding relations.
- Add cross-language positive fixtures plus unknown field/enum, authority-field, phase/control/gap, bound, identity, and cross-binding mutations.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_COMPONENTS.md:302-328` — runtime responsibility and artifact seam
- `.plans/UMPIRE4_DSL.md:245-324` — artifact/component separation
- `model/Umpire/Artifact/Experiment.lean` — binding and provenance conventions
- parent spec `Normative v1 wire contract` — exact fields, order, identities, references, and exclusions

### Acceptance
- [ ] Runtime config cannot encode endpoints, credentials, namespaces, executables, or arbitrary options.
- [ ] Run cannot encode a Property verdict; runtime configuration cannot encode authority material.
- [ ] Profile-required capabilities are locally recomputable from the persisted projection; no profile lookup is needed for admission.
- [ ] Phase/control/source-closure and cross-binding mutations fail exactly.
- [ ] Canonical Lean and Go bytes agree for both valid formats.

## Acceptance
- [ ] R3 runtime/run formats and bindings are minimal, exact, bounded, and transport-only.
- [ ] Both strict codecs round-trip cross-language bytes.
- [ ] Focused Lean/Go tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
