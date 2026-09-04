---
satisfies: [R1, R2, R6]
---
# fn-54-decompose-the-observation-evaluator.1 Extract Observation Evaluation contract types

## Description
Move the inert raw Evidence, diagnostic, support, Evidence Link, and unchecked-carrier declarations into a data-only child module while keeping accepted construction and results co-located until the admission task. Preserve every root name, field, constructor, instance, comment, and facade import.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation/Types.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/ImportTests.lean`
**Touches:** [model/Umpire/Observation/Evaluation/Types.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/ImportTests.lean]

### Approach
- Create `Umpire.Observation.Evaluation.Types` with the raw Evidence values and bundle, statuses and diagnostics, structural support, Evidence Link, and unchecked accepted-trace carrier.
- Preserve exact namespaces, fully qualified names, constructor and field order/defaults, derived instances, classifiers, accessors, and existing comments/docstrings.
- Leave `EvidenceBackedTrace`, its private construction helper, and `ObservationResult` in the evaluator until Task .4 so no construction bypass is exported.
- Keep the stable evaluator as the only intended importer-facing module; do not migrate direct consumers to `Types`.
- Strengthen the public import check only where needed to pin root names, constructors, projections, and instances.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:14-334` — existing data contracts and accepted carrier adjacency
- `model/Umpire/Observation/Language.lean:18-143` — upstream value, field, and expression contracts
- `model/Umpire/Observation/Language.lean:373-451` — checked-plan contracts consumed by evaluation
- `model/Umpire/Observation/ImportTests.lean:1-45` — current public facade compile checks
- `model/Umpire/SemanticInventory/KnownGaps.lean:508-552` — direct evaluator consumer surface
- `.flow/specs/fn-44-seal-observation-traces-and-centralize.md` — opaque accepted-trace ownership to preserve

**Optional** (reference as needed):
- `model/Umpire/Artifact/Result.lean:105-150` — downstream result projection

### Key context
- This is a physical data-contract extraction, not a new public interface.
- Co-locate the opaque accepted type with its private constructor later in Admission; never expose a constructor merely to cross a file seam.
- Preserve all existing comments and add only a concise child-module docstring.

## Acceptance
- [ ] R1-R2 are satisfied for the extracted contracts with exact names, constructors, field order/defaults, derived instances, rendering, classifiers, statuses, and projections.
- [ ] `EvidenceBackedTrace`, its private constructor, and `ObservationResult` remain co-located and no accepted construction bypass is exposed.
- [ ] Existing direct consumers retain `import Umpire.Observation.Evaluation` or `import Umpire.Observation`; none imports the child module.
- [ ] Public import checks pin representative contract construction, equality/rendering instances, result classification, and inaccessible accepted construction.
- [ ] Existing comments/docstrings are preserved and the new module has an accurate module docstring.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.ImportTests Umpire.SemanticInventory.KnownGaps` passes.
- [ ] The inherited `make lint-code GOLANGCI_LINT_FIX=false` baseline does not exceed 1,381 findings, and a run scoped to this task's base commit reports zero findings.

## Done summary
Extracted the inert Observation Evaluation contracts into `Evaluation.Types` without changing their declarations, facade surface, or accepted-trace construction boundary. Public import checks now pin representative construction, instances, projections, result classification, and private accepted construction; the approved Go lint baseline remained exact with zero task-scoped golangci findings and one unchanged unscoped errortype finding.

stage: impl-review - ran [2026-09-04T11:52:09Z..2026-09-04T11:55:12Z]
## Evidence
- Commits: 78134995d52ce68ec661f057db4661862425fbc9
- Tests: baseline: focused, aggregate, model build, regression, and model lint green; INHERITED_RED: make lint-code GOLANGCI_LINT_FIX=false reported exactly 1,381 findings pre-edit (approved waiver), TDD RED: cd model && mise exec -- lake build Umpire.Observation.Evaluation.Types (expected missing module), cd model && mise exec -- lake build Umpire.Observation.Evaluation.Types, cd model && mise exec -- lake build Umpire.Observation.ImportTests Umpire.SemanticInventory.KnownGaps, cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make umpire-check-regression, make lint-model, INHERITED_RED: make lint-code GOLANGCI_LINT_FIX=false (exactly 1,381 findings, unchanged from approved baseline), make lint-code GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=054da224022fe5c5776308e5e5ea828c819360a4 (golangci: 0 issues; trailing unscoped errortype finding at tools/umpire/runtime/errors.go:60 is unchanged from base and explicitly waived), Codex impl-review /tmp/impl-review-receipt-fn-54-decompose-the-observation-evaluator.1.json: SHIP, 0 findings
- PRs: