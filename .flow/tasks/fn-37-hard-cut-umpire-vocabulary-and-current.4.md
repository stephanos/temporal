---
satisfies: [R4]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.4 Rename Qualification to Observation Evaluation

## Description
Apply R4 after fn-4 has settled the behavior. Rename the module and result family around what the code actually does: evaluating Evidence into a complete Evidence-backed Model Trace with auditable Evidence Links.

**Size:** M
**Files:** `model/Umpire/Observation/Qualification.lean`, `model/Umpire/Observation/Tests/Qualification.lean`, Observation facade/import/tests, current consumers
**Touches:** [model/Umpire/Observation/**, model/Umpire/ObservationTests.lean, model/UmpireTests.lean, model/Temporal/**/*.lean]

### Approach
- Rename the module/file path from `Qualification` to `Evaluation` and update aggregate imports and import-boundary tests without a forwarding module.
- Rename `QualificationStatus`, failure kinds, diagnostics, result, and `qualifyEvidence` to the Observation Evaluation family.
- Use `accepted`, `unknown`, `conflict`, and `unsupported`; only accepted carries a complete `EvidenceBackedTrace`.
- Rename `SemanticCoordinate` to `ModelCoordinate` and `SemanticDerivation` to `EvidenceLink`, preserving every current derivation field and validation.
- Rewrite comments and diagnostics to distinguish Observation Evaluation from downstream Run Evaluation and Claim Assessment.
- Preserve all fail-closed behavior and independently supplied wrapper validation from fn-4.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Qualification.lean:1-220` — result, diagnostic, trace, and derivation vocabulary.
- `model/Umpire/Observation/Language.lean` — checked Observation plan consumed by evaluation.
- `model/Umpire/Observation/Tests/Qualification.lean` — outcome coverage.
- `model/Umpire/Observation/Tests/Derivation.lean` — Evidence Link completeness cases.
- `model/Umpire/Observation/Tests/Disposition.lean` — raw/redacted/rejected leakage guards.
- `.flow/specs/fn-4-umpire-observation-and-semantic-verdicts.md` — settled behavior contract; rename without semantic drift.

### Key context
Do not rename this layer to Claim Assessment. It does not issue an environment claim or evaluate Properties; it only establishes Model Facts and a Model Trace from Evidence.

## Acceptance
- [ ] The Evaluation module and Observation result family compile with no Qualification forwarding path.
- [ ] Accepted results carry a complete EvidenceBackedTrace and coordinate-complete Evidence Links.
- [ ] Unknown, conflict, and unsupported remain exhaustive and fail closed for their existing cases.
- [ ] Raw, redacted, rejected, missing, contradictory, and causally unsupported Evidence cannot establish a Model Fact.
- [ ] No Observation API or diagnostic implies Run Evaluation or Claim Assessment.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
