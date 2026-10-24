---
satisfies: [R11, R12, R13]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.21 Restore strict lint for the current Gomad workflow inventory

Touches: tools/gomad/rust/driver/static_stages.py, tools/gomad/rust/driver/tests/test_static_stages.py

## Description
Size S. Integrate and review isolated e41dd9e47a645d0a4e4bf7cf6f661df50de28b3a. Cycle5 C8d0a is terminal INCOMPLETE/1:16PASS,1workflow-lintFAIL,79unstarted,zero mismatches. The obsolete main-plus-native-smoke selector refuses before actionlint; native smoke deletion was intentional. Reuse the shared current-workflow selector in producer and independent reader, preserving pinned actionlint and exact command/hash/exit authentication. The current inventory is gomad.yml plus gomad-behavioral-parity.yml; missing, stale, foreign and ambiguous inventory must refuse. Preserve the failed attempt and diagnosis. Root owns integration gate/review; fn-232.13 alone owns the subsequent cycle. This is a launch prerequisite, while the other receiving tasks are closure prerequisites only.

## Acceptance
- [ ] Both producer and reader accept the exact current workflow inventory and refuse missing, stale native-smoke/legacy, foreign-module, alternate-extension and extra routing workflows; reader still refuses altered argv, tool hash and nonzero exits.
- [ ] Retain regression RED/GREEN, all focused static-stage tests, actual pinned actionlint exit0 on both workflows, independent review and the mandatory integration gate. No lint skip, historical fallback, weakened parsing or workflow-content change.
- [ ] Integrate the reviewed fix with exact commit/evidence references; cycle5 remains failed history and only a new actual cycle can establish qualification.

## Done summary
Restored strict selection of the two current Gomad workflows in the producer and independent reader. Missing, extra, stale, foreign and ambiguous inventories refuse; exact module-directory routing works with or without a trailing slash while similarly named modules remain unrelated. Pinned actionlint, raw argv/tool hashes and numeric exit checks remain strict. The deliberately deleted native smoke workflow is not restored.

Source commits e41dd9e47a645d0a4e4bf7cf6f661df50de28b3a and f29f4abc42389a6f686e1254be9ad29759cb265d are integrated on gomad at0337bda4bbdd67a5d185807845ec6ba7100d58da. Regression RED/GREEN,24 focused static tests, the two workflow tests and real pinned actionlint pass. First three-axis review retained correctness:1; the path-boundary regression reproduced it, and the same-session rereview returned SHIP with no surviving findings. Final mandatory gate on clean0337 exited0 in205.441690s, including589 driver tests plus Rust and WASM determinism checks; source/head remained unchanged.

Evidence: /home/agent/wt/evidence/fn232-workflow-pair-fix/{handoff.md,rereview-verification.json}; /home/agent/wt/evidence/fn232-split-closure/workflow-closeout-{review-finalize.json,rereview.json,final-gate.process.json,final-gate.log}. Same-family reviewer/writer pairing is disclosed. Cycle5 remains terminal INCOMPLETE/1:16PASS,1workflow-lintFAIL,79unstarted. No qualification pass is claimed; .13 remains sole cycle/final-composition owner, .22-.25 own receiving/closeout and .14 owns uncredited reach gaps.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e41dd9e47a645d0a4e4bf7cf6f661df50de28b3a, f29f4abc42389a6f686e1254be9ad29759cb265d, 0337bda4bbdd67a5d185807845ec6ba7100d58da
- Tests: 24 focused static-stage tests exit0; two workflow producer/reader regressions exit0 after actual RED; pinned actionlint v1.7.7 on both current workflows exit0; exact commands/hashes: /home/agent/wt/evidence/fn232-workflow-pair-fix/rereview-verification.json, make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism: exit0 on clean0337bda4bbdd67a5d185807845ec6ba7100d58da,589 driver tests; process/log /home/agent/wt/evidence/fn232-split-closure/workflow-closeout-final-gate.process.json and .log, Three-axis implementation review retained one correctness finding; same-session rereview SHIP zero findings: /home/agent/wt/evidence/fn232-split-closure/workflow-closeout-rereview.json
- PRs: