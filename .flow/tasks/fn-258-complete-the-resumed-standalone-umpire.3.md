---
satisfies: [R4]
---
# fn-258-complete-the-resumed-standalone-umpire.3 Adapt actual Temporal consumers to current standalone interfaces

## Description
Port retained consumer 91547f57ec onto current Temporal logic. Preserve supervision, admission, counts and nextest; keep xtask normal dependency. Verify sibling Cargo and Go workspace setup without replacing existing parent entries.

## Acceptance
- [ ] Actual consumer paths and imports use local standalone source.
- [ ] Current validation orchestration retained; focused checks pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
