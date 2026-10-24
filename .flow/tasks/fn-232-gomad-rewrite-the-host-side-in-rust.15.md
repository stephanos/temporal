---
satisfies: [R11, R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.15 Align static Go inventory with the executed selector

## Description
Size: S

**Touches:** `tools/gomad/rust/driver/static_stages.py`, `tools/gomad/rust/driver/tests/test_static_stages.py`

### Files / Touches
- tools/gomad/rust/driver/static_stages.py
- tools/gomad/rust/driver/tests/test_static_stages.py

### Approach
Make Execution.go and authenticate_result reconstruct the same Go-owned listing selector as the actual action. The replay-regeneration action selects ^TestReplayPlanConformance$, but both listing paths currently inject -list .; the actual C4 run lists 36 roots and correctly executes one, then strict verification rejects the other 35. Preserve platform-aware Go enumeration, package/build-tag scope, repetitions, exclusions, command custody and independent raw verification. Do not weaken verify_go or infer selection by filtering observed execution after the fact.

### Bounded required reads
Read this spec R11/R12, task .13 owner acceptance, static_stages.py recipe/replay action, parse_go_inventory, go_scope_exclusions, verify_go, Execution.go and authenticate_result Go branch, and their existing tests; consult only the platform-aware Go enumeration memory relevant to these functions.

### Focused tests
Run the existing test_static_stages.py suite with its established driver test invocation, including producer and independent authenticator regressions for restricted and unrestricted selection, missing selected execution and duplicate roots. No full cycle or build is required in this fix task.
## Acceptance
- [ ] Replay regeneration lists only its Go-selected TestReplayPlanConformance inventory and accepts the authentic one-root pass without demanding 35 unrelated roots.
- [ ] Unrestricted Go actions retain complete platform-aware inventory, package/tag scope, repetitions and existing exclusions.
- [ ] Producer and independent reader agree on exact listing/run argv and still reject missing selected execution, duplicate listed/executed roots and changed command receipts.
- [ ] Focused static-stage regressions pass; no corpus expectations or .13 cycle/evidence acceptance are weakened.

## Done summary
Static Go test inventory now uses the executed root selector in both producer and independent authenticator. Go owns regex matching; unrestricted inventory, package/build flags, repetitions, exclusions and strict raw verification remain intact. Unsupported slash/subtest selectors refuse before execution.

The regression failed before the fix. All 19 focused tests pass, covering restricted/unrestricted selection, missing/duplicate roots and changed command receipts. The combined mandatory gate passed on integration commit 63280b2da1126d50178adcf6ab25715c74ea2954, including all 576 Python driver tests. Log: /home/agent/wt/evidence/fn232-split-closure/integration-gate.log.

stage: wave-dispatch - ran (model: gpt-6.1-sol)
stage: impl-review - ran: SHIP across correctness, contracts and integration (model: gpt-6.1-sol)
stage: plan-sync - skipped(config: disabled)

Review receipt: /home/agent/wt/evidence/fn232-split-closure/selector-review.json. All source commits remain reachable from gomad. Task .13 still owns the actual 96-stage qualification cycle; this fix supplies no cycle pass. Tracker sync: n/a (bridge inactive). No push or PR.
## Evidence
- Commits: f8135e89caf23bd2d732b8641e9c274dd07f4fec, 66dad56f36c006e0ddc072a4acdbb08d548f5dc9
- Tests: PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_static_stages.py -v, PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -p test_static_stages.py -k subtest_selection -v, git diff --check, PATH=<stock-Go1.27.1>/bin:$PATH GOCACHE=/home/agent/.cache/go-build-fn232-corpus CARGO_TARGET_DIR=<workspace>/.build-cache/fn232-composition-target CARGO_BUILD_JOBS=2 make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism; exit 0 on63280b2da1126d50178adcf6ab25715c74ea2954; 576 driver tests pass
- PRs: