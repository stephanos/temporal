---
satisfies: [R12]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.17 Synchronize authenticated shared broad-harness readiness

## Description
Size: M

**Touches:** `tools/gomad/rust/driver/broad_harness.py`, `tools/gomad/rust/driver/broad_stages.py`, `tools/gomad/rust/driver/tests/test_broad_harness.py`, `tools/gomad/rust/driver/tests/test_broad_stages.py`

### Files / Touches
- tools/gomad/rust/driver/broad_harness.py
- tools/gomad/rust/driver/broad_stages.py shared initialization caller
- tools/gomad/rust/driver/tests/test_broad_harness.py
- tools/gomad/rust/driver/tests/test_broad_stages.py

### Approach
Reproduce the three allowed broad stages racing shared initialization: execute_build publishes the shared root with mkdir before build/admission receipts exist, while broad_stages.run treats directory existence as readiness. Establish synchronized initialization and an authenticated ready boundary; callers may consume only the completed build validated by read_build. Preserve complete source/config identity, custody, receipt validation and refusals, and keep stage outputs separate. Define bounded failure behavior for an initializer interrupted before readiness; partial or stale roots must never be accepted as ready. Do not add libraries or weaken source authentication to resolve contention.

### Bounded required reads
Read this spec R12, broad_harness.py owned_root/read_build/execute_build/execute and producer/build helpers directly involved in readiness, broad_stages.py run, their focused tests, and the memory entry requiring full source identity and synchronization for shared artifacts. Consult existing locking/publication helpers only where needed to follow repository conventions.

### Focused tests
Run the existing broad-harness and broad-stage tests with their established invocation. Include deterministic interleaving of all three stage callers, an initializer blocked between root creation and receipt completion, failure/interruption before readiness, and source/config or receipt mismatch. No full cycle or broad build is required.
## Acceptance
- [ ] A deterministic regression reproduces the former root-exists/receipt-missing race for the three concurrently allowed stages.
- [ ] Concurrent stage callers share one completed authenticated initialization and cannot read a partial build as ready.
- [ ] Initializer failure or interruption before readiness yields bounded strict failure or authenticated recovery; stale partial roots never become accepted evidence.
- [ ] Foreign source/config identity and missing, changed or unauthenticated receipts remain refusals; per-stage output ownership stays intact.
- [ ] Focused harness/caller regressions pass without new dependencies or weakened source checks.

## Done summary
All three broad stages now enter one synchronized initialization boundary. The OS lock stays held through build completion and read_build authentication; independent stage execution proceeds after that boundary. Reuse reauthenticates source/config, raw process/command, executable and admission evidence. Partial or interrupted builds remain strict refusals, with bounded lock contention and private no-follow lock custody.

The deterministic three-caller regression failed before the fix. All 20 focused tests pass, including killed initialization, concurrent stage execution, tampered locks and invalid receipts. The combined mandatory gate passed on integration commit 63280b2da1126d50178adcf6ab25715c74ea2954, including all 576 Python driver tests. Log: /home/agent/wt/evidence/fn232-split-closure/integration-gate.log.

stage: wave-dispatch - ran (model: gpt-6.1-sol)
stage: impl-review - ran: SHIP across correctness, contracts and integration (model: gpt-6.1-sol)
stage: plan-sync - skipped(config: disabled)

Review receipt: /home/agent/wt/evidence/fn232-split-closure/harness-review.json. All source commits remain reachable from gomad. Task .16 follows this shared caller change. Task .13 still owns actual cycle evidence. Tracker sync: n/a (bridge inactive). No push or PR.
## Evidence
- Commits: 17af4f1cdf76fd34b3d4eba201451d28f09ad157, 1ed2324cf7bbc51487d39117b9c61fe107f335d9
- Tests: baseline: green; PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s /home/agent/wt/fn232-harness-ready/tools/gomad/rust/driver/tests -p 'test_broad*.py' -v; exit 0; 14 tests, RED: PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s /home/agent/wt/fn232-harness-ready/tools/gomad/rust/driver/tests -p 'test_broad_stages.py' -k three_stages -v; exit 1; both follower callers consumed partial build, PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s /home/agent/wt/fn232-harness-ready/tools/gomad/rust/driver/tests -p 'test_broad*.py' -v; exit 0; 20 tests, git diff --check; exit 0, flowctl gate classify --base b0cf3c4ddbd648a699c88a4e9a62d6d7bed5cb18; exit 1 FULL (expected code classification; full gates conductor-owned), PATH=<stock-Go1.27.1>/bin:$PATH GOCACHE=/home/agent/.cache/go-build-fn232-corpus CARGO_TARGET_DIR=<workspace>/.build-cache/fn232-composition-target CARGO_BUILD_JOBS=2 make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism; exit 0 on63280b2da1126d50178adcf6ab25715c74ea2954; 576 driver tests pass
- PRs: