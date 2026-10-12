---
satisfies: [R3, R6]
---
# fn-166-gomad-readable-package-structure-and.4 Deepen runner and qualification packages

## Description
Merge runner/internal/exploration/* into one package; move inspection types and Inspect to runner/inspect; narrow runner/internal/campaign exports; merge qualification/workload into qualification. Update archcheck owner and edge tables.

## Acceptance
runner exported identifiers drop by at least half; exploration is one package; package count drops by at least five; R6 checks pass with compiled model and runtime identities unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
