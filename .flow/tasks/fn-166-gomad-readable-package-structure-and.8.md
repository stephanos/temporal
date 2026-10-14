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
No network-v1 model or choice-v1 decoder remains; trace.go has explicit named old-version refusal and Rust/Go controls reject retired network configuration. README model table retires v1. diagnostic-profile remains in Cargo.toml; frozen locked-release Rust default/diagnostic suites (33+48 and38+49 tests), fmt and both all-target Clippy configurations pass.

Full actual WASM qualification is frozen960c966cdc9fba84736a268d8949f30214c16ac4 at /Users/stephan/Workspace/skunkworks/gomad-current-qualification-20261014.kKv0jE, source manifest8c60541c8de0e71bef350444594814be67e38e6dc282268d6d4497961dafa278. All budgets/corpus dispositions are retained: five primary successes/four selections qualified; five expected-failure canaries not qualified. Raw stages.log terminal PASS includes ordinary Rust/Go, 800recordings/32replays, artifact/operator/storage/network/cluster, ten cells/aggregate, selections/catalogue/preservation. Exact three-file947 gate-only correction joins affected corrected-source checks under MILESTONES' bounded-fix rule; no retained result is relabelled current-HEAD execution.

Formal shared implementation receipt .flow/tmp/impl-review-rotation-integrated-20261014.json is SHIP, retained primarysession01a138cd-892e-74d3-96f1-0b60220f7c22, both prior findings fixed and no new blockers. Reviewer gpt-6.1-sol/high shares writer's GPT family. Native execution remains unverified/deferred fn128/fn149. Fn166.11 future atomic rename and complete post-rename qualification are excluded and remain mandatory; wholefn166 is not complete.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4, ea9ca7c35cc46db340b648126cb8edd8affbd571
- Tests: No network-v1 model or choice-v1 decoder remains; trace.go has explicit named old-version refusal and Rust/Go controls reject retired network configuration. README model table retires v1. diagnostic-profile remains in Cargo.toml; frozen locked-release Rust default/diagnostic suites (33+48 and38+49 tests), fmt and both all-target Clippy configurations pass., Complete frozen qualification stages.log PASS, retained source/corpus binding unchanged, Formal shared implementation SHIP .flow/tmp/impl-review-rotation-integrated-20261014.json, Corrected-source affected architecture/lint/validate-retry/clock/actionlint PASS; native execution deferred; fn166.11 excluded
- PRs: