---
satisfies: [R4, R6]
---
# fn-166-gomad-readable-package-structure-and.5 Move gomadtool under the native backend

## Description
Move cmd/gomadtool to backends/native/cmd/gomadtool with Git renames; update the Makefile, owner tables and documentation.

## Acceptance
Root cmd/ holds only cmd/gomad; make targets using gomadtool pass; R6 checks pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
