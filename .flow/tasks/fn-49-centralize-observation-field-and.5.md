---
satisfies: [R6]
---
# fn-49-centralize-observation-field-and.5 Document and verify Observation contract ownership

## Description
Update public architecture guidance and run complete model compatibility gates (R6).

**Size:** S
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/README.md, model/ARCHITECTURE.md]

### Approach
- Document field specs as typed projections and structural analysis as internal shared mechanics with boundary-owned diagnostics.
- Preserve raw-evidence non-retention and accepted-trace ownership language from predecessor work.
- Make the documented closure contract explicit: a global empty required kind closes at sequence zero, while nonempty global closures and source-local closures retain their existing structural checks.
- State that accepted provenance evaluates closures against the checked plan before accepting closure support, preserving its boundary-owned failure precedence.
- Run the adapter-focused Observation, mutation, and verdict suites before aggregate model builds, exact regression, trust/import, and lint checks.
<!-- Updated by plan-sync: fn-49-centralize-observation-field-and.4 preserved zero-record global closures and checked-plan closure precedence in the shared adapter -->

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:139-160,216-273` — Observation language, evidence, and link lifecycle
- `model/README.md:232-272` — public pipeline description
- `model/ARCHITECTURE.md:185-260` — cross-altitude Observation boundary

## Acceptance
- [ ] Documentation names one field authority and one internal structural analysis while preserving distinct public diagnostics.
- [ ] Documentation preserves global zero-record closure compatibility and explains that accepted closure support is checked against the checked plan without exposing raw diagnostics.
- [ ] Existing comments remain present unless their ownership statement changed.
- [ ] Adapter-focused Observation, mutation, and verdict suites, aggregate builds, exact regression, import/trust checks, `make lint-model`, and `make lint-code` pass.
- [ ] No artifact, generated view, fingerprint, or persisted-byte drift remains.

## Done summary
Documented ObservationFieldSpec as the sole typed field projection authority and Observation.Internal.analyzeStructure as shared internal mechanics with raw/accepted boundary-owned diagnostics. Clarified sequence-zero closure for empty required kinds, checked-plan closure precedence, raw evidence non-retention, and opaque accepted-trace ownership across the three architecture guides.
## Evidence
- Commits: a75c520917f70cab61d1ec7d16878075ed1a2f66
- Tests: cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests Temporal.System.Nexus.Tests, cd model && mise exec -- lake build Umpire.Observation.Tests Umpire.Observation.Tests.Mutations Umpire.Observation.Tests.Verdict Umpire.Observation.ImportTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model, make lint-code (exact inherited 1,379-finding baseline)
- PRs: