---
satisfies: [R1, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.1 One shortest-path search in the suite

## Description
Implement **One search (A3)**: `space.rs:254` and `run.rs:123` use `pattern_search::Graph::shortest` over `Path::alternatives`; delete `search.rs`'s `Along`/`Through`/`arrive`/`trace`/`settled`. Reconcile the cut-frontier rule and `Unbridged.waypoint` mapping; measure speed (keep the old path if the default gate slows by more than 25 percent).

## Acceptance
- [ ] `search.rs`'s parallel search is gone; witnesses are shortest valid Model paths that replay, and Unbridged reports identify the correct waypoint/cut-frontier failure. Equal-length path choice, numbering and report bytes need not match an old run.
- [ ] Timing before/after recorded.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
One shortest-path search in the suite: the old `Along`/`Through` search is deleted and the pattern search (in search.rs) serves `find` and `Run::along` through `Graph::shortest(&alternative, Anchor::{Starts, Anywhere})`. Review NEEDS_WORK → 171292fae5/4572cc518e: dense label table sized by the explored graph (a state-only find over 700k states now succeeds under the default budget; new test), honest admitted-start wording when the budget cuts. Default gate timing +1.3% (spec threshold 25%); the 700k-state case is 21–25% slower and peaks at ~410 MB vs ~321 MB, recorded as a known cost. Two goldens reworded (admitted start), no witness changed.

Joined validation completed at 1c1e7fbf0767088a4c820adb64da75325af1f007: `UMPIRE_APALACHE=require make model-validate` with TLC required exited 0. All named stages, guest freshness, workspace tests, wasm controls and all three Go host modules passed. Gate 1549.246s exceeds the 60s soft target; inner loop 7.623s meets 10s. This was a cold task-owned cache with concurrent independent work, not a controlled performance comparison. Full log: `.flow/tmp/milestones-resume-20261024/joined-model-validate.log`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c1e7fbf0767088a4c820adb64da75325af1f007
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/joined CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=4 UMPIRE_APALACHE=require make model-validate
- PRs: