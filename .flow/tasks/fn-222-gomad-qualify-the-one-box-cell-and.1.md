---
satisfies: [R1]
---
# fn-222-gomad-qualify-the-one-box-cell-and.1 Qualify the one-box cell on the current candidate

## Description
Using the fn-220 installation, run the one-box smoke cell through every gate internal/backends/wasm/QUALIFICATION.md names for a cell: unchanged source preparation, storage/transport artifact qualification, repeated measured pass and replay on the frozen candidate. Record numeric exits, host identity, helper, compiler, module and model identities in a retained checkpoint under .flow/artifacts, and update the cell row from pending to qualified with that reference, in the style of the fixture and SQLite rows. Early proof point: run source preparation first; if it fails, stop and report. If any gate fails, the row stays pending with the gate and exit recorded.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.md

## Acceptance
- [ ] Every cell gate has a recorded numeric exit and identities in the retained checkpoint
- [ ] QUALIFICATION.md one-box row references the checkpoint, or stays pending with the failing gate recorded
- [ ] No capacity limit raised without a recorded decision

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
