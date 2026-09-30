---
satisfies: [R9]
---
# fn-105-gomad-follow-ups-deferred-scope.9 D9: linux/amd64 downstream packs and qualification

## Description
Origin: fn-104. Deferred 2026-09-29 because F9 was Darwin-only. Revived 2026-09-30 by fn-107 R8/R10, which requires a Linux gate with exact replay. Follow fn-105.8 and consume the fn-107.4 final target. Author consumer-owned darwin/arm64 and linux/amd64 pack requests through discover/review/generate on each actual host. Retain reviewed facts and generation inputs, negative identity/platform tests and the bounded qualification manifest/driver. Execute seeds 11 and 17 twice each per platform and replay each retained success; retain exact tool/source/pack identities and evidence. Files/Touches: downstream localcell/gomad pack inputs, manifest, driver and retained qualification reports. Quick: pack checks, negative contract tests, both platform analyses and qualification CLI runs.
## Acceptance
Revival is recorded. Final downstream target is supported in closure and linked analysis on both platforms and qualifies with matching same-seed execution evidence and exact replay for all required observations. A classified failure does not satisfy the fn-107 consuming gate. Downstream owns reproducible packs and reports, with invalid/missing/drifted identities rejected. fn-107.5 consumes the evidence.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
