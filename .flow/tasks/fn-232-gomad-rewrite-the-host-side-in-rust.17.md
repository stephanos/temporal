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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
