---
satisfies: [R1, R4, R6]
---
# fn-54-decompose-the-observation-evaluator.3 Extract raw Observation Evidence evaluation

## Description
Move raw bundle validation, expression and disposition evaluation, structural-finding translation, emission ordering, Evidence Link construction, ambiguity handling, and unchecked-trace assembly into one internal child module. Keep `evaluateEvidence` as the sole public entry point.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation/Raw.lean`, `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Tests/Evaluation.lean`
**Touches:** [model/Umpire/Observation/Evaluation/Raw.lean, model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Tests/Evaluation.lean]

### Approach
- Move record/profile validation, checked-expression evaluation, digest and disposition handling, emission construction/order, raw structural finding translation, alternative handling, and unchecked-trace assembly.
- Keep `syntheticDigestToken` at its current root name and put the unchecked evaluator plus shared private helpers under `Observation.Internal`.
- Do not export a second evaluation entry point or move accepted admission into the raw module.
- Preserve operation order exactly: bound, source closure, Known Gaps, empty/profile checks, structural analysis, bindings/digests, alternatives, dispositions, and emission assembly.
- Continue calling `analyzeStructure` exactly once on the raw path and return no partial unchecked trace on failure.
- Add a raw failure table with complete expected diagnostics and combined-invalid bundles that pin bound, source-closure, Known Gap, empty, profile, and structural first-failure precedence.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:768-1609` — raw validation, evaluation, and unchecked-trace assembly
- `model/Umpire/Observation/Tests/Evaluation.lean:357-634` — raw failure matrix and precedence checks
- `model/Umpire/Observation/Tests/Disposition.lean:49-162` — disposition and forbidden-material behavior
- `model/Umpire/Observation/Tests/Fixtures.lean:237-350` — checked plan and Evidence bundle fixtures
- `model/Umpire/Observation/Tests/EvidenceLink.lean:232-316` — emitted support and link invariants
- `model/Umpire/Core.lean:33-38` — canonical identity helper

### Key context
- Preserve exact failure precedence and related-identity ordering, not only diagnostic kinds.
- Raw evaluation is pure and internal; no callback, I/O, cache, or new trust path belongs here.

## Acceptance
- [ ] R4 is satisfied for the existing accepted and failure fixture matrix with exact unchecked semantic content or first diagnostic.
- [ ] Bound, open source, Known Gap, empty, profile/version/kind/field, binding, digest, disposition, alternative, fault-target, closure, and multi-source cases retain exact status, kind, fields, related identities, and precedence.
- [ ] Combined-invalid raw bundles assert the complete diagnostic, including related identities and Limit/count fields, and prove that no accepted or partial unchecked trace is returned.
- [ ] Raw failures return no partial trace, and rejected, redacted, or raw field material cannot leak into an unchecked success.
- [ ] The raw path invokes structural analysis once and introduces no duplicate normalization or sorting pass.
- [ ] `evaluateEvidence` remains the sole public entry point; the child module exposes no new public evaluator.
- [ ] Existing comments are preserved.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests.Evaluation Umpire.Observation.Tests.Disposition` passes.

## Done summary
Extracted raw Observation Evidence validation, expression/disposition evaluation, structural translation, emission assembly, and unchecked-trace construction into `Evaluation.Raw` behind the unchanged `evaluateEvidence` facade. Added a table-driven combined-invalid regression covering complete bound, source-closure, Known Gap, empty, profile, and structural diagnostics with no partial accepted trace; full Lean/regression/model-lint gates pass, global Go lint remains at the approved 1,381-finding baseline, and diff-scoped golangci reports zero issues (the unchanged trailing errortype finding is waived).

stage: impl-review - ran [2026-09-04T13:05:19Z..2026-09-04T13:09:56Z]
## Evidence
- Commits: 64735f28ad9af16d6f1092b03660c79667ee7032, 425d18bd819687f88930a166d3409c020d12f66c
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests.Evaluation Umpire.Observation.Tests.Disposition, cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.ImportTests Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make umpire-check-regression, make lint-model, make lint-code GOLANGCI_LINT_FIX=false (approved inherited baseline: exactly 1,381 findings), GOLANGCI_LINT_BASE_REV=bfc6dd1cfa2a2329033f390593408c6b26835773 make lint-code GOLANGCI_LINT_FIX=false (golangci: 0 issues; unchanged tools/umpire/runtime/errors.go:60 errortype finding waived)
- PRs: