---
satisfies: [R1, R3, R6]
---
# fn-54-decompose-the-observation-evaluator.2 Extract the Observation structural analyzer and focused tests

## Description
Move the complete existing internal structural-analysis implementation intact into its own child module and relocate direct structural examples into a focused test module. Keep raw and accepted diagnostic translation with their owning boundaries.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation/Structure.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Tests/Structure.lean`, `model/Umpire/Observation/Tests/Evaluation.lean`, `model/Umpire/Observation/Tests.lean`
**Touches:** [model/Umpire/Observation/Evaluation/Structure.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Tests/Structure.lean, model/Umpire/Observation/Tests/Evaluation.lean, model/Umpire/Observation/Tests.lean]

### Approach
- Move the fn-49 `Observation.Internal` structural records, findings, normalization, comparators, and `analyzeStructure` implementation without renaming or changing finding order.
- Retain existing fully qualified internal names so raw and accepted adapters keep the same seam.
- Reuse `DefinitionId.canonicalSet` only where it is exactly equivalent; keep Observation-specific order and closure calculations with the structural owner.
- Move direct structural examples from the mixed evaluation test file into `Tests.Structure`, preserving comments and expected normalized values.
- Keep raw and accepted diagnostic mapping outside this module and continue to call structural analysis once per path.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:335-767` — structural facts, findings, normalization, and analyzer
- `model/Umpire/Observation/Tests/Evaluation.lean:12-355` — direct structural and mixed evaluator examples
- `model/Umpire/Core.lean:33-38` — canonical Definition ID set helper
- `.flow/tasks/fn-49-centralize-observation-field-and.3.md` — established structural ownership
- `.flow/tasks/fn-49-centralize-observation-field-and.4.md` — raw/accepted adapter distinction

**Optional** (reference as needed):
- `model/Umpire/ARCHITECTURE.md:240-248` — documented internal structural authority

### Key context
- Move the established module; do not redesign it or collapse raw and accepted diagnostics.
- Finding order is observable through first-diagnostic translation and must remain exact.

## Acceptance
- [ ] R3 is satisfied by one unchanged `Observation.Internal.analyzeStructure` authority in `Evaluation.Structure`.
- [ ] Empty, single-source, multi-source, duplicate identity/sequence/closure, mixed-origin, gap, missing-parent, cycle, reverse-edge, required-kind, closure count/byte, and per-link support cases retain exact findings and order.
- [ ] Raw and accepted adapters retain distinct diagnostic vocabularies outside the structural module.
- [ ] Direct structural examples live in `Tests.Structure`, retain their comments and expected values, and remain aggregated by Observation tests.
- [ ] No public helper, generic graph framework, duplicate normalization, or second traversal is introduced.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests.Structure Umpire.Observation.Tests.Evaluation` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
