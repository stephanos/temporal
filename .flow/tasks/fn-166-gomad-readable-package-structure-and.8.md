---
satisfies: [R8, R6]
---
# fn-166-gomad-readable-package-structure-and.8 Delete network v1 and choice v1 (rotation batch)

## Description
Delete WASM network v1 (Go and Rust) and the legacy choice-trace v1 decoder as specified in R8. Keep the helper `diagnostic-profile` feature (R8 amendment, 2026-10-13). Land in the same integrated candidate as fn-153, fn-154 and fn-166 tasks 6-7.

## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

No network v1 model or choice v1 decoder remains; the `diagnostic-profile` feature still builds and its tests pass; README model table drops Network v1; v1 traces are refused by name; Rust fmt/clippy/tests and the full WASM gate set pass on the integrated candidate.

### Revival (2026-10-13)

Resume under the owner's active instruction to complete the WASM section. Remove both retired paths together, retaining NetworkV2, broker networking and the diagnostic measurement feature. Integrate with fn-153/fn-154 and the remaining fn-166 identity rotations before final qualification; focused source checks do not complete the task.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
