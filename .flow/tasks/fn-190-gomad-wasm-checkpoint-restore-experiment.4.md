---
satisfies: [R5]
---
# fn-190-gomad-wasm-checkpoint-restore-experiment.4 Temporal one-box boot-probe measurement

## Description
Run the one-box cell for TestFrontendSystemInfo and one smoke suite with an explicit test-only probe marker right after testcore setup and a controlled handoff to an inspectable decision; record marker, ordinal and source location. Same instrumented module for fresh and restored runs. Report boot wall time to the first test-body boundary, image bytes, capture, restore and continuation wall times as medians and ranges of at least five warm runs on one host in one window, with host identity, and both continuation modes matched or first divergence named. Depends on .3.

## Acceptance
Measurement artifact with medians, ranges, host identity and window; both modes reported; no retained failure claimed from a restored run without fresh replay.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
