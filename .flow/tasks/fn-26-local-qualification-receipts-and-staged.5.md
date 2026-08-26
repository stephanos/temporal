---
satisfies: [R2, R5, R6]
---
# fn-26-local-qualification-receipts-and-staged.5 Expose the exact local qualification CLI and root Make target

## Description
**Size:** M
**Files:** tools/umpire/cmd/umpire-qualify-local/**, model/lakefile.toml, repository-root Makefile
**Touches:** fixed CLI/profile handshake and root Make UX

Add umpire-qualify-local and the fixed profile sibling executable registration plus only the repository-root make umpire-qualify-local target. Implement exact required arguments, fixed sibling resolution, canonical summary/error/status 0/1/2 contract, publication/reporting booleans, immutable destination recovery, and no hidden/default IO.

## Acceptance
Direct and root commands accept only SET, PILOT_EVIDENCE, and OUTPUT_ROOT, produce exact deterministic summaries/errors, publish through fn-18 only, handle reporting-after-publication without rerun ambiguity, and leave stdout empty on tooling failure. No model-local Makefile, optional authority/checker/profile flag, default target, CI workflow, or repository write outside publication is added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
