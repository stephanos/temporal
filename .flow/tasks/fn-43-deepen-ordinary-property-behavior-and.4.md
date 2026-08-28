---
satisfies: [R4, R7]
---
# fn-43-deepen-ordinary-property-behavior-and.4 Extract deterministic DefinitionGraph validation

## Description
Extract the shared structural graph mechanics required by R4 after the authoring files have converged on shared identity primitives. Behavior and Observation remain the owners of domain terminology, validation staging, cycle-witness selection, and errors.

**Size:** M
**Files:** `model/Umpire/Shared/DefinitionGraph.lean`, `model/Umpire/Behavior/Language.lean`, `model/Umpire/Observation/Language.lean`, `model/Umpire/Behavior/Tests/Validation.lean`, `model/Umpire/Observation/Tests/Compilation.lean`
**Touches:** [model/Umpire/Shared/**, model/Umpire/Behavior/Language.lean, model/Umpire/Observation/Language.lean, model/Umpire/Behavior/Tests/Validation.lean, model/Umpire/Observation/Tests/Compilation.lean]

### Approach
- Create a documented shared module that owns canonical node/edge normalization, duplicate detection, self/unknown-endpoint analysis, topological order, path existence, and cycle evidence.
- Return a total staged analysis whose node, edge, and cycle findings can be consumed separately; do not impose one global error precedence or expose Behavior/Observation error constructors.
- Keep duplicate-node findings at each language's current pre-reference validation point and consume edge/cycle findings at its current post-reference point, so mixed graph/non-graph declarations keep the same winning error.
- Let each adapter derive its historical cycle witness from shared cycle evidence; do not force Behavior and Observation to adopt one witness-selection algorithm.
- Add mixed graph/non-graph, multiple-graph-fault, and divergent-cycle-witness fixtures for both languages before removing private algorithms.
- Remove superseded private graph code only after both language suites compile; move/preserve useful comments at the new ownership boundary.
- Keep the module behind focused language imports unless a real external consumer requires public umbrella exposure.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Behavior/Language.lean:273-290` — duplicate edge scan and canonical edge order.
- `model/Umpire/Behavior/Language.lean:356-451` — current validation staging, topological ordering, and cycle witness.
- `model/Umpire/Observation/Language.lean:480-503` — corresponding duplicate ordering mechanics.
- `model/Umpire/Observation/Language.lean:984-1018` — independent path/cycle validator and witness choice.
- `model/Umpire/Behavior/Tests/Validation.lean` — domain-specific failure regression style.
- `model/Umpire/Observation/Tests/Compilation.lean` — checked Observation compile/failure matrix.

### Key context
- The generic normal form is deterministic Definition-ID structure, not a general graph library. Shared analysis removes algorithm duplication; language adapters retain validation timing, witness policy, and error schemas.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Behavior.Tests Umpire.Observation.Tests
```
## Acceptance
- [ ] One documented DefinitionGraph module owns canonical node/edge structure and returns separately consumable node, edge, canonical-order, and cycle findings.
- [ ] Empty, disconnected, and acyclic graphs validate; mixed graph/non-graph and multiple-graph-fault fixtures preserve each language's historical winning error.
- [ ] Behavior and Observation derive their previous deterministic cycle witnesses from shared evidence, including a fixture where their prior witness policies would diverge.
- [ ] Behavior and Observation retain existing public error kinds, offending/related Definition IDs, canonical metadata, and fingerprints.
- [ ] Superseded private graph code is removed without losing existing comments, and no public umbrella import is added without a demonstrated consumer.
- [ ] Focused Behavior and Observation suites pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
