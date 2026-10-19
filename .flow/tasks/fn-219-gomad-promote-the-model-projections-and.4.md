---
satisfies: [R5, R6]
---
# fn-219-gomad-promote-the-model-projections-and.4 Record the identity ledger, requalify rotated identities and refresh documentation

## Description
Write .flow/artifacts/fn219-identity-ledger-<date>.md listing the helper source digest, volume implementation identity, choice identity, model identities and WASM overlay identity as unchanged or rotated with the cause. Re-run the current-candidate WASM gates named in internal/backends/wasm/QUALIFICATION.md for every rotated identity and record numeric exits. Update tools/gomad/README.md, ARCHITECTURE.md and internal/backends/wasm/README.md: models are owned source, generator lives in internal/model/cmd/protocolgen, native tree is a frozen consumer that still validates under its selection; native execution stays deferred under fn-128/fn-149 and is neither deleted nor qualified.

**Size:** S
**Files:** .flow/artifacts/fn219-*.md, tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, tools/gomad/internal/backends/wasm/README.md

## Acceptance
- [ ] Ledger names every retained identity as unchanged or rotated with cause
- [ ] Rotated identities requalified with recorded numeric exits on the frozen candidate
- [ ] README, ARCHITECTURE and WASM README describe owned models, the generator home and the frozen native tree without claiming deletion or qualification

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
