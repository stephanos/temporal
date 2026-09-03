---
satisfies: [R1, R3, R4]
---
# fn-58-partition-the-property-language.1 Extract Property trace projection and evaluation

## Description
Move the Property-owned coordinate adapter and capability-limited trace projection into Trace, and move executable and denotational constraint, pattern, occurrence, clause, span, result, and evaluation semantics into Evaluation. Keep public consumers on the facade while checking temporarily remains in Language.

**Size:** M
**Files:** `model/Umpire/Property/Language.lean`, `model/Umpire/Property/Trace.lean`, `model/Umpire/Property/Evaluation.lean`, `model/Umpire/Property.lean`
**Touches:** [model/Umpire/Property/Language.lean, model/Umpire/Property/Trace.lean, model/Umpire/Property/Evaluation.lean, model/Umpire/Property.lean]

### Approach
- Move `PropertyTraceField.valueAt?`, capability-limited trace steps/views, and `CheckedProperty.traceView` into Trace while preserving strict Model Coordinate and prior-state behavior.
- Move constraint and pattern denotation/evaluation, occurrences, all clause interpreters and agreement proofs, spans/results, and `evaluateProperty` into Evaluation.
- Keep authoring data and inert helpers in Language; the checker may remain there until Task .2 completes the final import chain.
- Preserve every namespace, name, visibility, type, proof statement, comment, result field, evaluated Limit, and semantic provenance value.
- Keep Observation and Planning consumers unchanged through `Umpire.Property`; do not expose child modules as public contracts.
- Audit public agreement theorem axiom inventories before and after the move.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Language.lean:35-172` — coordinate adapter plus constraint/pattern meaning
- `model/Umpire/Property/Language.lean:703-1257` — trace projection, clause interpretation, proofs, and results
- `model/Umpire/Property/Tests/Evaluation.lean:21-95` — clause, span, and provenance coverage
- `model/Umpire/Property/Tests/LogicalTime.lean:9-77` — logical-time failure closure
- `model/Umpire/Observation/Tests/Verdict.lean:192-223` — prior-state coordinate compatibility

**Optional** (reference as needed):
- `model/Umpire/Observation/Verdict.lean:286-327` — facade consumer of evaluation
- `model/Umpire/Planning/Engine.lean:522-537` — facade consumer of checked Property data

### Key context
- Trace projection is a Property-specific adapter over Core coordinates, not shared coordinate authority.
- Preserve existing comments and theorem trust; no test should import past the intended module interface merely to reach a private helper.

## Acceptance
- [ ] R1 and R3 are satisfied for independently buildable Trace and Evaluation modules behind the unchanged facade.
- [ ] Capability filtering, hidden values, strict coordinate lookup, initial/prior/resulting-state edges, repeated equal values, and relation compatibility remain exact.
- [ ] Empty matches/triggers, missing, malformed, or regressing logical time, same-position/strict ordering, inclusive bounded windows, zero-distance eventuality, all clause meanings, spans, Limits, and provenance retain current results.
- [ ] Executable/denotational agreement theorems retain exact statements and axiom inventories.
- [ ] Observation and Planning consumers compile without source or import changes.
- [ ] Existing comments are preserved and no new public representation, adapter, dependency, cache, or traversal is introduced.
- [ ] `cd model && mise exec -- lake build Umpire.Property.Trace Umpire.Property.Evaluation Umpire.Property.Tests Umpire.Observation.Tests.Verdict` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
