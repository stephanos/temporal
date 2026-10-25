---
satisfies: [R2, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.2 Path stores only a Pattern

## Description
Implement **Path stores a Pattern (A4)**: drop `first`/`leg`/`more` in `dsl/src/path.rs:686-870` and `IntoWorldPath<kind::Legacy>`; derive `claim()` from the pattern.

## Acceptance
- [ ] No panicking `Path` accessors; legacy representation gone.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
`Path` stores only its `Pattern`; the inline state form, panicking accessors, `is_state_only`, `into_world`, `into_pattern`, `IntoWorldPath` and path_input.rs are gone; `claim()` reads trigger/gap/target from the three-part pattern. Suite entry points take `WorldPath<S>`. Author impact: only a standalone `let p = path![..]` with no consumer needs `Path<S>`; all documented examples infer as before (DESIGN.md notes it). Review NEEDS_WORK → 44a221cc0c (trybuild pass fixture, dead code). No golden changed; changes the Temporal guest (regenerated at the join, 7259e9b989).

Joined validation completed at 1c1e7fbf0767088a4c820adb64da75325af1f007: `UMPIRE_APALACHE=require make model-validate` with TLC required exited 0. All named stages, guest freshness, workspace tests, wasm controls and all three Go host modules passed. Gate 1549.246s exceeds the 60s soft target; inner loop 7.623s meets 10s. This was a cold task-owned cache with concurrent independent work, not a controlled performance comparison. Full log: `.flow/tmp/milestones-resume-20261024/joined-model-validate.log`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1c1e7fbf0767088a4c820adb64da75325af1f007
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-milestones-20261024/joined CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 CARGO_BUILD_JOBS=4 UMPIRE_APALACHE=require make model-validate
- PRs: