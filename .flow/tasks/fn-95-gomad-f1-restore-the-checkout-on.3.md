---
satisfies: [R3, R4]
---
# fn-95-gomad-f1-restore-the-checkout-on.3 Pass gomad3-integration-test and the core qualification set on darwin/arm64

## Description
Run `make gomad3-integration-test` and `make -C tools/gomad3 compatibility-pack-qualification core-qualification-set`; verify 5/5 supported with exact choice replay.

## Acceptance
- integration test passes
- core set: selected 5, supported 5, unsupported 0, all choice_replay_exact

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
