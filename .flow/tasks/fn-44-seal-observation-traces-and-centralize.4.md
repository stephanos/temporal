---
satisfies: [R2, R3, R5, R6]
---
# fn-44-seal-observation-traces-and-centralize.4 Require accepted traces in Property and Implementation Link

## Description
Migrate Property verdict and Implementation Link consumers to Task 3's accepted type (R2, R3, R5), removing Observation-owned revalidation while preserving each consumer's own checks.

**Size:** M
**Files:** `model/Umpire/Observation/Verdict.lean`, `model/Umpire/ImplementationLink/Application.lean`, `model/Umpire/ImplementationLink/Tests/Application.lean`, `model/Temporal/ImplementationLinkTests/Nexus.lean`
**Touches:** [model/Umpire/Observation/Verdict.lean, model/Umpire/ImplementationLink/Application.lean, model/Umpire/ImplementationLink/Tests/Application.lean, model/Temporal/ImplementationLinkTests/Nexus.lean]

### Approach
- Remove Property verdict's accepted-trace revalidation and Evidence-bound validation branches while retaining query/property compatibility, vocabulary, capability, and logical-time checks; read the admitted bound only for verdict output.
- Replace Implementation Link's remaining coordinate helpers with the Core API and require the accepted trace type at its public application boundary.
- Remove full Observation-envelope revalidation and forged-wrapper application cases; move those negative cases to admission while retaining Link-owned source replay, mappings, Limits, Known Gaps, and translation diagnostics.
- Preserve the forward-simulation seam expected by the dependent authoring-deepening spec.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Verdict.lean:285-337` — repeated Property-side validation and remaining semantic checks.
- `model/Umpire/ImplementationLink/Application.lean:275-330` — duplicated coordinate and envelope checks.
- `model/Umpire/ImplementationLink/Application.lean:701-764` — downstream revalidation/application path.
- `model/Umpire/ImplementationLink/Tests/Application.lean:385-524` — current application/failure matrix.
- `model/Temporal/ImplementationLinkTests/Nexus.lean:409-423` — composed forged-accepted-trace fixture.

### Key context
Malformed unchecked wrappers now fail as Observation diagnostics before Property or Application. Do not preserve a raw overload merely to keep impossible later-stage statuses reachable; do preserve all Property and Link failures meaningful for an admitted trace.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.ImplementationLink.Tests Temporal.ImplementationLinkTests.Nexus
```
## Acceptance
- [ ] Property verdict and Implementation Link accept only admitted `EvidenceBackedTrace` values and contain no repeated complete Observation-envelope validation.
- [ ] Property retains query/property, vocabulary, capability, and logical-time checks after its admission branch is removed; it reads but does not revalidate the admitted Evidence bound.
- [ ] Implementation Link uses Core coordinate enumeration/lookup/kind semantics and retains source Target replay, mapping, Limit, Known Gap, and translation failures.
- [ ] Forged-wrapper coordinate/envelope cases, including malformed bounds, fail at Observation admission; no Property or Link stage runs after an Observation non-success.
- [ ] Property and Implementation Link failures for valid admitted traces retain their exact stage, status, diagnostic identity, and no-partial-result behavior.
- [ ] Focused reusable and Temporal Implementation Link suites pass without import or trust-boundary drift.
## Done summary
Made `evaluateObservationProperty` accepted-trace-only, removed its impossible Evidence-bound revalidation, and preserved Observation non-success projection without invoking Property evaluation. Removed Implementation Link envelope revalidation and local coordinate semantics in favor of Core APIs while retaining source replay, vocabulary, mappings, Limits, Known Gaps, translation diagnostics, and no-partial-result behavior; fn44.3 had already moved the forged-wrapper matrices to Observation admission.

Baseline and focused/Temporal verification passed; `make lint-model` passed. `make lint-code GOLANGCI_LINT_FIX=false` reproduced 1,386 inherited Go findings against a Lean-only task diff.

stage: impl-review - ran (Codex SHIP; receipt `/tmp/impl-review-receipt-fn-44-seal-observation-traces-and-centralize.4.json`)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 35ec1979f47eca5eed527115d11cd3d4ce0a2840
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.ImplementationLink.Tests Temporal.ImplementationLinkTests.Nexus, RED_EXPECTED: cd model && mise exec -- lake build Umpire.Observation.Tests (exit 1: EvidenceBackedTrace expected ObservationResult before accepted-only signature), cd model && mise exec -- lake build Umpire.Observation.Tests, make lint-model, INHERITED_RED: make lint-code GOLANGCI_LINT_FIX=false (exit 2: 1386 pre-existing Go findings; task diff is Lean-only), git diff --check, impl-review Codex SHIP receipt /tmp/impl-review-receipt-fn-44-seal-observation-traces-and-centralize.4.json
- PRs:
