---
satisfies: [R3, R4, R5, R6]
---
# fn-49-centralize-observation-field-and.4 Route raw and accepted validation through structural analysis

## Description
Complete both adapters and pin their distinct diagnostics, precedence, and mutation behavior (R3-R5).

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Tests/Evaluation.lean`, `model/Umpire/Observation/Tests/EvidenceLink.lean`, `model/Umpire/Observation/Tests/Disposition.lean`, `model/Umpire/Observation/Tests/Mutations.lean`, `model/Umpire/Observation/Tests/Verdict.lean`
**Touches:** [model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Tests/Evaluation.lean, model/Umpire/Observation/Tests/EvidenceLink.lean, model/Umpire/Observation/Tests/Disposition.lean, model/Umpire/Observation/Tests/Mutations.lean, model/Umpire/Observation/Tests/Verdict.lean]

### Approach
- Map raw findings back to the current detailed diagnostic at the same validation stage.
- Build one `Observation.Internal.StructuralLinkSupport` per accepted Evidence Link, then invoke `analyzeStructure` once with that link-scoped support and consume its normalized per-link boundaries plus rule-attributed inconsistency findings.
- Map accepted support findings only to missing-order-support or missing-closure-support at the established admission seam.
- Preserve complete first-failure matrices, related-ID order, no-partial output, and disposition/identity checks around the shared analysis.
<!-- Updated by plan-sync: fn-49-centralize-observation-field-and.3 used StructuralLinkSupport and rule-attributed normalized per-link support, not only aggregate accepted support -->

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:959-1079,1611-1669` — raw evaluation and accepted admission order
- `model/Umpire/Observation/Tests/Evaluation.lean` — detailed raw failures
- `model/Umpire/Observation/Tests/EvidenceLink.lean:20-250` — accepted support mutations
- `model/Umpire/Observation/Tests/Mutations.lean` — independent semantic-altitude oracles
- `model/Umpire/Observation/Tests/Disposition.lean` — retention/digest failure ordering
- `model/Umpire/Observation/Tests/Verdict.lean` — downstream non-invocation boundary
## Acceptance
- [ ] Raw validation retains every existing diagnostic kind, status, related-ID order, and precedence.
- [ ] Accepted validation retains missing-order-support/missing-closure-support classification and never leaks raw diagnostics.
- [ ] Accepted validation supplies one `StructuralLinkSupport` per Evidence Link, maps normalized per-link facts and closures plus rule-attributed inconsistent support findings at the accepted boundary, and retains the established related identities.
- [ ] Each adapter invokes structural analysis once and maps the returned normalized facts/findings without rebuilding or sorting them; the 10× fixture preserves this call path and outcome.
- [ ] Invalid inputs produce no partial accepted trace or downstream verdict.
- [ ] Observation aggregate and mutation suites pass with no semantic or artifact drift.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
