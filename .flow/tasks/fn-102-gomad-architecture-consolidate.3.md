---
satisfies: [R3]
---
# fn-102-gomad-architecture-consolidate.3 Consolidate retention policy without merging strategy transactions

## Description
R3. Reuse task 2 assessment. Share only matching eligibility, novelty, bounded capacity, and manifest/artifact input construction. Artifact remains the publication owner. Keep seed publication in selection-ordinal order, and exploration publication under staged round/commit logic. Do not move counters or durable journal writes into a policy helper. Preserve independent guided-corpus retention, exact-replay prerequisites, failure deduplication and expansion of distinct prefixes. Record a before/after fixed-input comparison of manifest and journal projections; built binary hashes naturally differ. Reuse current cancellation, retained-success capacity and whole-round resume tests. Extend the existing fake-executor parallelism test to 10 and 100 jobs with Parallel=2, KeepSuccesses=none and identical bounded payloads; assert maximum active executions stays at most 2. Exercise fixed success count/byte limits separately at both selection sizes and require the same capacity classification. Inspect policy data structures for new total-selection-sized state or payload duplication; this is a control-bound check, not a timing/RSS benchmark. Preserve existing comments. No new dependencies. Run focused commands from tools/gomad3 with GOWORK=off and -tags test_dep; use the patched toolchain for runtime-consumer tests.

**Size:** M

**Touches:** [tools/gomad3/runner/*.go] — WIDER: retention helpers and existing strategy/resume tests share the runner package.

**Files:** `tools/gomad3/runner/{runner.go,choice_exploration_campaign.go,simulation_exploration_campaign.go,guidance.go}`; new private retention policy/tests; existing runner and resume tests.

### Quick commands

`go test -tags test_dep ./runner ./runner/internal/campaign ./runner/internal/corpus ./artifact`

## Acceptance
- [ ] R3 rules have one private owner where semantics match.
- [ ] Capacity, incomplete transcript, publication error, first-novel ordering and duplicate failure tests retain existing outcomes.
- [ ] Seed interruption, round interruption/resume, cancellation and corpus independence remain covered.
- [ ] Fixed-identity canonical manifests/journal projections match baseline; 10/100-job bound tests preserve Parallel=2 and capacity failures; review confirms no new full-payload copies or state proportional to total selected seeds.

## Done summary
NOT IMPLEMENTED. Moved to fn-105-gomad-follow-ups-deferred-scope.2 (D2) on 2026-09-29 as a scope cut; the task text above remains the implementation brief.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: