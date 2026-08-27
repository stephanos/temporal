---
satisfies: [R4]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.4 Rename Qualification to Observation Evaluation

## Description
Apply R4 after fn-4 has settled the behavior. Rename the Observation module and result family around what the code actually does: evaluating Evidence into a complete Evidence-backed Model Trace with auditable Evidence Links.

**Size:** M
**Files:** `model/Umpire/Observation/Qualification.lean`, `model/Umpire/Observation/Tests/Qualification.lean`, Observation facade/import/tests, Umpire aggregate roots
**Touches:** [model/Umpire/Observation/**, model/Umpire/ObservationTests.lean, model/UmpireTests.lean]

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
No current `model/Temporal/**` source consumes the Qualification API, so this task is intentionally confined to Observation and its Umpire aggregate roots. Do not rename this layer to Claim Assessment: it establishes Model Facts from Evidence but does not issue an environment Claim or evaluate Properties over a Run.
## Acceptance
- [ ] The Evaluation module and Observation result family compile with no Qualification forwarding path.
- [ ] Accepted results carry a complete EvidenceBackedTrace and coordinate-complete Evidence Links.
- [ ] Unknown, conflict, and unsupported remain exhaustive and fail closed for their existing cases.
- [ ] Raw, redacted, rejected, missing, contradictory, and causally unsupported Evidence cannot establish a Model Fact.
- [ ] No Observation API or diagnostic implies Run Evaluation or Claim Assessment.

## Done summary
Hard-cut the Umpire Observation Qualification vocabulary to Evaluation, Evidence Link, Model Coordinate, and accepted Observation Result terminology across the facade, tests, verdicts, and Nexus aggregate consumer. The implementation preserves fail-closed behavior exactly: accepted observations carry a complete `EvidenceBackedTrace` with `EvidenceLink`s, while unknown, conflict, and unsupported outcomes remain rejected.

Baseline was green via build, unittest, and smoke receipts at `d4dc228e`. Focused evaluator, aggregate, import, and Nexus tests passed; the full Lean build, pinned Go tests, and `mise exec -- make umpire-check-regression` passed. The final generated-view target remains the declared fn37.6 sequencing gap; its current projection equivalent passed through the regression target.

GATE_SKIPPED:build:green-receipt e142a860 - verify reused from pre-review post-gate pass
GATE_SKIPPED:unittest:green-receipt e142a860 - verify reused from pre-review post-gate pass
GATE_SKIPPED:smoke:green-receipt e142a860 - verify reused from pre-review post-gate pass
GATE_SKIPPED:generated-view-smoke:inherited-sequencing-gap - fn37.6 creates the final umpire-check-regression-views target; the current projection check passed through umpire-check-regression

stage: impl-review - ran (first-pass SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e142a86031b0ce2238af8ba334fb3738e6685d81
- Tests: GATE_SKIPPED:build:green-receipt d4dc228e - baseline reused from prior post-gate pass, GATE_SKIPPED:unittest:green-receipt d4dc228e - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt d4dc228e - baseline reused from prior post-gate pass, RED: cd model && mise exec -- lake build Umpire.Observation.Evaluation - renamed module absent before edit, cd model && mise exec -- lake build Umpire.Observation.Evaluation, cd model && mise exec -- lake build Umpire.Observation.Tests, cd model && mise exec -- lake build Umpire.Observation.ImportTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.ObservationTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, mise exec -- go test ./tools/umpire/..., mise exec -- make umpire-check-regression, GATE_SKIPPED:build:green-receipt e142a860 - verify reused from pre-review post-gate pass, GATE_SKIPPED:unittest:green-receipt e142a860 - verify reused from pre-review post-gate pass, GATE_SKIPPED:smoke:green-receipt e142a860 - verify reused from pre-review post-gate pass, GATE_SKIPPED:generated-view-smoke:inherited-sequencing-gap - fn37.6 creates the final umpire-check-regression-views target; the current projection check passed through umpire-check-regression
- PRs:
