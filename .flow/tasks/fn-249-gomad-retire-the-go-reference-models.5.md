---
satisfies: [R4]
---
# fn-249-gomad-retire-the-go-reference-models.5 Update the stage plan, Make, docs and inventories; record the stage delta and sizes

## Description
Remove or replace the cycle stages and leaf commands that built `reference-helper`/`conformance`, ran the Go model tests or ran `helper_corpus`. Write `.flow/artifacts/fn-249/stage-delta.md` mapping every removed stage to its replacement coverage for fn-246. Update the Makefile, `ARCHITECTURE.md`, `README.md`, `CLI.md` and `wasmhost/protocol.md`, plus the determinism and archcheck inventories. Record before/after line counts with exact revisions.

## Acceptance
- [ ] Stage plan golden updated with the recorded delta; every removed stage is mapped; counts recorded with revisions.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
