---
satisfies: [R1, R2]
---
# fn-224-accepted-self-loops-in-the-attribute.1 Self-loop marker in #[machine] handlers, lifter admission, and fixtures moved back to TLC

## Description
Add the accepted self-loop marker beside reject! in umpire-dsl and #[machine] (name chosen consistently with reject! and recorded in DESIGN.md; provisional stay!()). Unchanged handler arm with the marker = enabled self-loop, unit outcome or Accepted in modules using reject!; reject! takes precedence; misuse outside #[on(..)] handlers refused, pinned by trybuild. Admit it in umpire-lift through the existing authored-self-loop emission. Then rewrite the checker-only fixtures whose only gap was the self-loop (progress_self_loop and siblings, edge_checks/vacuity where their remaining reasons allow) back into shared attribute fixtures with byte-identical behaviour goldens up to fn-223's listed spellings, and update fn-223's disposition table. Verify make model-check, UMPIRE_BACKENDS=require, make model-time. Starts after fn-213 is closed in umpire's history.

## Acceptance
- [ ] Marker makes an unchanged arm an enabled self-loop; misuse refused by trybuild
- [ ] All pre-existing goldens byte-identical
- [ ] Self-loop fixtures back in the shared corpus agree with TLC; disposition table updated
- [ ] make model-check and UMPIRE_BACKENDS=require pass

## Done summary
Added `stay!()` (name recorded in DESIGN.md "Accepted self-loops (fn-224)": a short imperative in `reject!`'s style that names the edge; `accept!` would imply an outcome that unit modules lack). It records a flag in the same scoped frame as `reject!` and does not return. An unchanged arm becomes an enabled self-loop, with unit outcome, or `Accepted` in `reject!` modules, where a rejection wins. `#[machine]` rewrites `stay!()` in `#[on]` bodies to the macro's hidden arm. The plain arm is a `compile_error!`, so an effect, helper or use outside `#[machine]` refuses; trybuild pins all three (machine_stay_effect, machine_stay_helper, stay_outside_machine). Modules without `stay!` expand exactly as before: no existing golden or expansion snapshot changed.

umpire-lift strips a top-level `stay!()` from an arm and sets `preserve_self_loop`. A `stay!()` nested in a condition gets a located refusal (source.rs refusal table).

Fixtures: new shared `accepted_self_loops` (Accepted, precedence and redundant-mark rows; lift refused at `reject!`) with expansion snapshot `staying_machine`. Moved back as attribute fixtures with TLC goldens: progress_self_loop, progress_target_barrier, progress_excluded_prefix, progress_scope_exit. Their report/rows/states/paths goldens are byte-identical up to fn-223's two spellings, and TLC agrees on each (states 1/3/2/2, verdicts Refuted/Verified/Refuted/Refuted, counterexamples replayed). The trimmed trait Graph now lives in progress_action_choices.

Deviation: edge_checks, vacuity and shared_starts stay checker-only. Their remaining gap is `bool` outcomes (plus duplicate starts for shared_starts); their self-loops are now stateable. progress_multiple_starts (starts) and progress_action_choices (choice) also stay. The DESIGN table and corpus counts are updated (12 admitted, 5 refusals).

Gate: `make model-check` passed, and so did `UMPIRE_BACKENDS=require make model-check`. A concurrent writer's uncommitted suite/sample_kettle edits later broke clippy/fmt in the shared tree, so model-time ran on a clean detached worktree of HEAD (target dir shared with the main tree, so docs/core-wasm-clippy were not warm):
timing-controls 0.020s, capability-controls 0.160s, source-boundary 1.072s, protobuf-freshness 0.056s, temporal-protobuf-freshness 0.056s, fmt 0.171s, clippy 4.648s, docs 29.442s, tests 10.368s, wasm32-wasip1 0.486s, core-wasm-clippy 32.022s, wasm-guest 7.315s, go-host 1.287s, temporal-go-host 5.797s; inner loop (edit+test) 4.644s; gate total 93.426s (soft target 30s, over); inner loop 4.644s (soft target 10s).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2cfd220f92, 3aa77201dd
- Tests: make model-check, UMPIRE_BACKENDS=require make model-check, make model-time
- PRs: