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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
