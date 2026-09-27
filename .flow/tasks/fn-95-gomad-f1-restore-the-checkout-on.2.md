---
satisfies: [R2]
---
# fn-95-gomad-f1-restore-the-checkout-on.2 Pass make -C tools/gomad3 test on darwin/arm64

## Description
Run the full gate tier by tier (test-harness, test-toolchain, intercept-test, test-host, overlay-test, world-test, test-builder, test-live-capability, test-runtime, test-upstream). Fix darwin regressions introduced by the linux port (platform-pinned fakes, darwin source-set pins, fixtures). Record pre-existing unrelated failures precisely.

## Acceptance
- every tier passes on darwin/arm64, or a failure is recorded with exact output and justification in the milestone status

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
