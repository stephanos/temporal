---
satisfies: [R3, R4]
---
# fn-223-gomad-delete-the-native-backend.3 Record unchanged WASM identities, close fn-128 and fn-149, refresh docs

## Description
Rerun the identity probes used in fn-184 and fn-219 and record helper digest, overlay identity, choice, model and volume identities as unchanged in .flow/artifacts/fn223-closeout-<date>.md; run make -C tools/gomad test-wasm on the current candidate and record the exit. Close fn-128 and fn-149 as won't-do with the owner decision of 2026-10-19 written into their Decision Context and the close-out. Update tools/gomad/README.md, ARCHITECTURE.md and MILESTONES.md so native appears only as removed history. Nothing under .flow/artifacts is deleted.

**Size:** S
**Files:** .flow/artifacts/fn223-*.md, .flow/specs/fn-128-*.md, .flow/specs/fn-149-*.md, tools/gomad/README.md, tools/gomad/ARCHITECTURE.md, MILESTONES.md

## Acceptance
- [ ] Close-out artifact lists every WASM identity as unchanged and test-wasm exit 0
- [ ] fn-128 and fn-149 closed as won't-do with the recorded owner decision
- [ ] README, ARCHITECTURE and MILESTONES describe native only as removed history

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
