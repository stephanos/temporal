---
satisfies: [R12, R13]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.27 Handle tracked Gitlinks in onebox source inventory

## Description
Touches: tools/gomad/testdata/parity/onebox-gates.py and the existing Python qualification test suite.

Repair the actual C8 temporal-inspection/fn222 inventory initialization failure: tracked mode160000 dashboards Gitlink is opened as a regular file. Follow existing canonical source-inventory semantics, omitting only Gitlink content reads; preserve ordinary tracked source hashing, existing symlink behavior and fail-closed handling of replaced/missing regular files. Do not silently filter arbitrary filesystem directories. Outer frozen HEAD/index guards already bind Gitlink metadata; do not introduce an unrelated metadata format or recurse into submodule contents.

Diagnosis: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/cycle8-onebox-gitlink-inventory-diagnosis.{json,md}. Preserve C9fc08 and original raw exit1 as history. This task repairs the narrow source inventory defect only; diagnostic fixtures and primary capacity belong to sibling tasks. Parent .13 retains actual full96/final145 and all receiving acceptance.

## Acceptance
- [x] Retain actual RED using a real tiny Git index with mode160000 and ordinary tracked source; demonstrate old inventory tries reading the directory. Test checked-out and absent Gitlink content without network.
- [x] Minimal mode-aware repair excludes only Gitlink content reads and retains every ordinary tracked source file. Prove normal content changes remain detected and regular-file-to-directory replacement still refuses; preserve existing symlink contract and NUL-safe paths.
- [x] Focused regression suite, full current driver suite and mandatory worker/conductor merge gates pass; independent green-tree implementation review reaches SHIP. No bounds, assertions, roles, ignored cases, declarations or full qualification gates weakened.
- [x] Integrate reviewed repair with actual command/exit/source provenance. No qualification pass is inferred; .13 remains open for a fresh current-source cycle.
## Done summary
Repaired mode-aware onebox source inventory: skips only Gitlink content, preserves tracked symlink hashing and refuses regular source type changes/unmerged entries. Actual real-index RED,17 onebox regressions,589 driver tests, worker gate and conductor gate0. Integrated commits0010345096/ce29dc0e24. Combined gate on33121111dd0 (424.663s); currentc3e7a89 adds only diagnosis documentation. Three independent Codexgpt-6.1-sol/high draws SHIP with zero findings, same-family disclosed; reviewer tempfile execution was sandbox-blocked, actual worker/conductor executions passed. Durable review /home/agent/wt/evidence/fn232-onebox-gitlink-fix/formal-review/impl-review-receipt-657da2bc4466-fn-232-gomad-rewrite-the-host-side-in-rust.27.json. OriginalC8 remains failed history; .13 qualification pending.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 001034509677f5c0d09887352323c07e206a27f5, ce29dc0e24352121ec17c5e5cdf5d9323ecda545
- Tests: make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism, worker handoff: /home/agent/wt/evidence/fn232-onebox-gitlink-fix/handoff.json, conductor gate: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/conductor-repair27-gate.json, combined gate: /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/conductor-repaircombined-gate.json, independent SHIP: /home/agent/wt/evidence/fn232-onebox-gitlink-fix/formal-review/impl-review-receipt-657da2bc4466-fn-232-gomad-rewrite-the-host-side-in-rust.27.json
- PRs: