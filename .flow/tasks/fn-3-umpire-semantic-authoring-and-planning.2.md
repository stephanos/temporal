---
satisfies: [R2, R5, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.2 Implement portable pure properties and evaluation

## Description
Implement the portable Property language over the semantic foundation (R2). Keep declaration data, denotation, executable evaluation, and capability-scoped trace access together so fn-4 can qualify evaluation inputs later without redefining property meaning.

**Size:** M
**Files:** `model/Temporal/Experiment/Property.lean`, `model/Temporal/Experiment/PropertyTests.lean`
**Touches:** [model/Temporal/Experiment/Property.lean, model/Temporal/Experiment/PropertyTests.lean]

### Approach
- Define inspectable property declarations for the first-slice state, transition, relation/input-output, finite ordering, and bounded-progress combinators.
- Resolve every temporal bound to a typed value/unit and include that expansion in the model-only evaluation result.
- Build capability-limited trace views from checked requirements; the evaluator must not receive the unrestricted trace when the property declares a narrower view.
- Prove evaluator/denotation agreement structurally for the entire portable core, using one induction theorem over declarations or compositional theorems that cover every constructor; theorem-backed fixtures are additional boundary examples, not a substitute.
- Return clause identity, relevant trace span, bound, and semantic provenance; leave qualification and evidence derivations to fn-4.
- Give every portable Property declaration a deterministic canonical JSON projection/digest and keep opaque Lean predicates outside portable planning, projection, and artifacts.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE_DSL.md:244-297` — Property responsibility, supported kinds, purity, and bounded semantics
- `model/NexusAutoClose.lean:740-755` — existing cancellation properties to denote rather than duplicate
- `model/NexusAutoClose.lean:870-932` — uniqueness property and positive/negative theorem fixtures
- `model/Temporal/Experiment/DSL.lean:73-99` — legacy property observation strings being replaced

**Optional** (reference as needed):
- `model/Temporal/ExperimentTests.lean:370-524` — current Nexus positive/negative fixture style

### Quick commands
```bash
cd model && mise exec -- lake env lean Temporal/Experiment/PropertyTests.lean
```
## Acceptance
- [ ] Portable declarations are inspectable typed data and cover the first-slice property kinds without arbitrary callbacks.
- [ ] A structural Lean theorem proves executable evaluation agrees with denotation for every portable Property declaration; positive, negative, and boundary fixtures exercise the theorem and evaluator.
- [ ] A property cannot access vocabulary outside its declared capability view, including when the full planner trace contains that vocabulary.
- [ ] Mixed bound units, undeclared references, and opaque declarations fail before planning with structured diagnostics.
- [ ] Model-only results identify the evaluated clause/span/bound/provenance and contain no raw-evidence or qualification fields.
- [ ] Repeated Property projection is byte-identical, canonicalizes collection order, and changes for each meaning-bearing constructor, reference, or bound mutation.
- [ ] The focused Lean test command passes and the R8 exclusion audit is clean.
## Done summary
Implemented inspectable portable Property declarations, capability-scoped trace evaluation, typed and named bounds, independent finite denotations with structural agreement proofs, model-only clause results, and deterministic canonical projections. Focused fixtures cover the authoritative cancellation uniqueness result, every portable clause kind, capability isolation, structured authoring failures, digest sensitivity, and fail-closed logical-time evaluation; Codex review reached SHIP after both findings were fixed.

baseline: red (cd model && mise exec -- lake env lean Temporal/Experiment/PropertyTests.lean failed pre-edit because the task-created test module did not yet exist)
GATE_SKIPPED:unittest:green-receipt d01538a7 - baseline reused from prior post-gate pass
GATE_SKIPPED:smoke:green-receipt d01538a7 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-25T02:17:34Z..2026-08-25T02:26:51Z]
## Evidence
- Commits: 9b7c9e5f9ed4d232687565a1b5e62811ac2c40e5, a75dd2d1d8204a827a3231f616420d302c18ce95
- Tests: baseline: red (cd model && mise exec -- lake env lean Temporal/Experiment/PropertyTests.lean failed pre-edit because the task-created test module did not yet exist), GATE_SKIPPED:unittest:green-receipt d01538a7 - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt d01538a7 - baseline reused from prior post-gate pass, cd model && mise exec -- lake build Temporal.Experiment.Property, cd model && mise exec -- lake env lean Temporal/Experiment/PropertyTests.lean, cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect, make umpire-check-regression
- PRs: