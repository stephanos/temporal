---
satisfies: [R9, R6]
---
# fn-166-gomad-readable-package-structure-and.9 Fold gomad3integration into native qualification

## Description
Move tools/gomad3integration qualification manifests, testdata/tagged and its three manifest checks under tools/gomad3/backends/native/qualification; update Makefiles, CI workflows, manifestgen and soak references; delete the empty package. Put the root CI and gate-routing tests with internal/archcheck (amends task 1).

## Acceptance
tools/gomad3integration no longer exists; manifest checks pass from their new location; make targets and actionlint pass; R6 checks pass.

## Done summary
Temporal qualification manifests, README and manifest checks moved to backends/native/qualification. The remaining gomad3integration module (TestPublicWrappers, testdata/tagged) is removed with the library-contract checks in fn-166.7.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3450f373a8
- Tests: manifest checks in the gomad3 module, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: