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
- Run focused suites, aggregate model builds, exact regression, trust/import, and lint checks.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:139-160,216-273` — Observation language, evidence, and link lifecycle
- `model/README.md:232-272` — public pipeline description
- `model/ARCHITECTURE.md:185-260` — cross-altitude Observation boundary

## Acceptance
- [ ] Documentation names one field authority and one internal structural analysis while preserving distinct public diagnostics.
- [ ] Existing comments remain present unless their ownership statement changed.
- [ ] Focused and aggregate builds, exact regression, import/trust checks, `make lint-model`, and `make lint-code` pass.
- [ ] No artifact, generated view, fingerprint, or persisted-byte drift remains.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
