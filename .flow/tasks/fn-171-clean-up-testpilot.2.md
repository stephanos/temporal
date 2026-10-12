---
satisfies: [R2]
---
# fn-171-clean-up-testpilot.2 One spelling per Driver type: the adapter imports contract; shrink the root alias block

## Description
From the Testpilot research (2026-10-12). `contract` (54 names) has no importer outside Testpilot, yet `contract.go` re-exports them as 60 aliases in the root, so every Driver type has two spellings. Make `temporal/*` import `contract` directly, and shrink the root's alias block to what external callers use (`Driver`/`Session` and the few others the importers need). Behaviour unchanged.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
