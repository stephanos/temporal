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
Extracted the unchanged `Observation.Internal` structural analyzer into `Evaluation.Structure` and relocated direct structural regressions into `Tests.Structure`, preserving the facade and one analyzer call per raw/accepted boundary. The review-driven cleanup reuses `DefinitionId.canonicalSet`; full Lean/regression gates are green, the approved global Go-lint baseline remains exactly 1,381 findings, and task-scoped golangci reports zero issues.

stage: impl-review - ran [2026-09-04T12:26:24Z..2026-09-04T12:36:48Z]
## Evidence
- Commits: 9908a04a78af56a0c11c769e306464607592b844, aea63327bca740b548a4afd76a7e4f9b5e436837, dc989ccde94d2584519721d0b3bd59d50fd65a8c
- Tests: baseline: green via handoff (green verified at 78134995 by fn-54-decompose-the-observation-evaluator.1); make lint-model passed; approved inherited global Go lint reproduced exactly 1,381 findings, TDD RED: cd model && mise exec -- lake build Umpire.Observation.Tests.Structure (expected missing Evaluation.Structure import), cd model && mise exec -- lake build Umpire.Observation.Tests.Structure Umpire.Observation.Tests.Evaluation, cd model && mise exec -- lake lint Umpire.Observation.Evaluation.Structure Umpire.Observation.Tests.Structure Umpire.Observation.Tests.Evaluation, cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make umpire-check-regression, make lint-model, INHERITED_RED: make lint-code GOLANGCI_LINT_FIX=false (exact approved 1,381 findings), GOLANGCI_LINT_BASE_REV=d1bd56f1e167a1cc55f813b8f3579b623ad02eed make lint-code GOLANGCI_LINT_FIX=false (golangci: 0 task-diff issues; unchanged trailing tools/umpire/runtime/errors.go:60 errortype finding explicitly waived), Codex impl-review /tmp/impl-review-receipt-fn-54-decompose-the-observation-evaluator.2.json: SHIP after one fixed P3
- PRs: