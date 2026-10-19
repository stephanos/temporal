---
satisfies: [R1]
---
# fn-223-one-lifted-source-form-the-lifter-reads.1 Trait-form shared fixtures: attribute rewrites or checker-only moves, with a disposition table

## Description
Survey the eight trait-form shared fixtures under model/umpire/tests/fixtures (sample_kettle_trait, choices, edge_checks, vacuity, shared_starts, progress_graph, deep_chain, interleaved_claims). For each, decide attribute rewrite, checker-only move to umpire/checker/tests/fixtures, or removal (only if another fixture pins the same behaviour on both backends). Rewrite the expressible ones as #[machine] mod Models keeping behaviour goldens byte-identical up to explicitly listed spelling substitutions; move the rest with checker goldens unchanged; sample_kettle_trait becomes checker-only as the independent engine reference. Record the disposition table (fixture, disposition, reason or surface gap) in model/DESIGN.md's TLC section. Verify make model-check and the required TLC run (UMPIRE_BACKENDS=require) and paste both results into the done summary. Does not extend the attribute surface; gaps are recorded only. Starts after fn-215 is closed in umpire's history.

## Acceptance
- [ ] Every trait-form shared fixture has a disposition row in model/DESIGN.md
- [ ] Attribute rewrites keep report/rows/states/paths/tlc goldens byte-identical except for listed spellings
- [ ] Checker-only moves keep checker goldens byte-identical
- [ ] make model-check and UMPIRE_BACKENDS=require runs pass

## Done summary
Every trait-form shared fixture now has a disposition row in model/DESIGN.md's TLC section (fn-223 table). There are 19 rows, not 8: the eleven `progress_*` fixtures each selected a mode of the trait `progress_graph` Model, so they were trait-form too.

- Attribute (6): progress_graph (cycle), progress_continuous, progress_intermittent, progress_two_lobes, progress_fulfilled, progress_trigger_stops. These are `#[machine] mod graph` Models. Behaviour goldens (report/rows/states/paths) are byte-identical apart from the listed spellings `State { 0: n }` -> `Model { value: n }` and `Action::X` -> `User::X` / `Environment::X`. tlc.txt goldens are byte-identical. TLA+/cfg were regenerated (`State.*` -> `Graph.*`). In two_lobes the catalog order is Enter, Escape, Left, Right; Enter and Escape never fire, so the goldens are unchanged.
- Checker-only (13), moved to umpire/checker/tests/fixtures with checker goldens byte-identical (pure renames). Their lift/tla/cfg/tlc goldens were dropped and each hook was merged into model.rs.
  - sample_kettle_trait: the engine reference. Its Model is now `kettle.rs`, which the wasm guest includes. The Go host and wasm dispatch read its goldens from the new path. Its TLC coverage stays pinned by sample_kettle (report byte-identical).
  - choices: bool outcomes, string facts and nondeterministic choice.
  - edge_checks, vacuity: bool outcomes and an authored self-loop.
  - shared_starts: duplicate starts, plus bool outcomes and a self-loop.
  - deep_chain: Rc clone counter.
  - interleaved_claims: interleaved claim_order and a supplemental claim.
  - progress_self_loop, progress_multiple_starts, progress_target_barrier, progress_excluded_prefix, progress_scope_exit, progress_action_choices: self-loops, choice and non-range starts. A trimmed trait Graph lives in progress_self_loop/model.rs, following the progress_unknown precedent.
- Removed: none.
- TLC coverage lost, recorded as surface gaps: self-loops, nondeterministic choice, non-unit outcomes, facts, and starts outside a field range. Refuted and vacuous after-postconditions stay on TLC through changed_state.
- README updated: 12 shared fixtures; checker root holds the manual-trait Models; admitted-subset sentence. second-opinion.md links repointed.

Deviations
- The eleven progress_* fixtures were added beyond the spec's eight.
- umpire-lift's manual tests were repointed to the checker paths. `wrappers_keep_their_selected_mode_and_dependency` was deleted because no trait wrapper fixture remains; R2 deletes the manual lowering anyway.
- progress_fulfilled and progress_two_lobes carry a file-level `#![allow(clippy::unused_self)]`: their User actions are never enabled, and the lifter refuses a method-level `allow`.
- The work landed in commit 3c2fd273ce, titled "wip". A concurrent commit picked up the staged index, so no conventional-message commit of my own exists for the code. History was not rewritten.

Verification
- make model-check: exit 0.
- UMPIRE_BACKENDS=require make model-check: exit 0. All 8 admitted shared fixtures agree with TLC; tlc goldens unchanged.
- make model-time:
  gate total 12.153s (soft target 30s); inner loop 1.962s (soft target 10s)
  timing-controls 0.020s, capability-controls 0.142s, source-boundary 0.159s, protobuf-freshness 0.058s, temporal-protobuf-freshness 0.065s, fmt 0.283s, clippy 0.092s, docs 0.170s, tests 2.590s, wasm32-wasip1 0.052s, core-wasm-clippy 0.086s, wasm-guest 0.068s, go-host 1.325s, temporal-go-host 6.818s

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3c2fd273ce
- Tests: make model-check, UMPIRE_BACKENDS=require make model-check, make model-time
- PRs: