---
satisfies: [R1, R2, R3, R4]
---
# fn-220-replace-spell-with-standard-rust.1 Replace Spell end-to-end and refresh formatting goldens

## Description
# Use standard Rust formatting instead of Spell

The owner approved Display for Models/actions and standard Debug for their fields, including formatting-only golden changes such as Option::Some { 0: 1 } becoming Some(1).

### Requirements

- R1: Remove the custom Spell trait, implementations, exports and active callers. Machine state and action values use Display; domain outcomes and facts use Debug so unit, Option, Vec and tuples work without custom implementations.
- R2: State/Action derives and generated aggregate actions provide idiomatic standard formatting. State fields use Debug, supporting ordinary Rust collections and custom Debug-only fields. Preserve action naming where no field-format change is needed. Formatter errors must propagate in fmt implementations.
- R3: Reports, counterexamples, progress diagnostics and native/Wasm row rendering use standard formatting consistently. Retain ordering, claim semantics, row structure and the distinction between model verdicts and domain outcomes.
- R4: Migrate manual Models, fixtures, compile-fail tests, macro expansion snapshots and current documentation. Only formatting implementations/derives may change in the independent Temporal Model; no transition, assumption, verdict or protocol semantics changes. Update formatting-only goldens and prove native/Wasm/Go parity.

### Verification

Add or adapt executable formatting coverage first and record its expected failing result before implementation. Cover Debug-only collection/nested fields, actions including the empty action enum, and formatter error propagation. Run focused DSL/checker tests, make model-check and make umpire-check-model-slice, serializing heavy Go builds. Review golden diffs for formatting-only changes.

### Scope

**Touches:** model/umpire/**, model/temporal/**, model/README.md, model/DESIGN.md (excluding generated protocol files and all model/archived/**).

This is a standalone simplification request, not permission to resume the milestone queue or change archived code, protocol schema, Temporal adapter policy, or unrelated planning files. Current branch umpire. Local commits only; no push or PR.

## Acceptance
R1–R4 implemented; focused failing-first formatting tests recorded; model-check and dedicated real SDK slice pass; golden diffs formatting-only; no archived changes or milestone continuation.

## Done summary
Model authors now use Display for machine states/actions and Debug for fields, outcomes and facts. The derives, reports, diagnostics, native/Wasm rows, manual fixtures, compile-fail tests, expansion snapshots and current documentation use these contracts, and every fmt implementation propagates formatting errors.

R1-R4 are implemented. The nested Debug-only collection/action fixture and the report/path formatter-error test failed before production edits, with expected missing Display/Debug and obsolete Spell-bound errors. Logs are `.flow/tmp/fn220-formatting-red.log` and `.flow/tmp/fn220-report-red.log`. Both baseline gates passed before edits, recorded in `fn220-baseline-model-check.log` and `fn220-baseline-slice.log`.

The final `make model-check` passed, including 62 Rust tests, compile-fail controls, docs, Wasm builds and both Go hosts. `make umpire-check-model-slice` passed Complete, Fail, HostRefusal and Cancellation. Logs are `.flow/tmp/fn220-model-check.log` and `.flow/tmp/fn220-slice.log`. Final fmt and Clippy checks passed. Snapshot regeneration passed in `.flow/tmp/fn220-update-verified.log`; intermediate regeneration runs encountered compile-time include_str snapshots that required rebuilding consumers after payload updates.

The golden audit compared 177 existing payloads and classified all 57 changed payloads as exact formatting substitutions (51 reports, 2 rows, 2 states, 2 other). Its script and receipt are `.flow/tmp/fn220-golden-audit.pl` and `.flow/tmp/fn220-golden-audit.log`. The former leaf-Spell fixture was replaced by the nested collection/action formatting fixture. Graph counts, action/row order, paths, predicates and verdict meanings are unchanged. The Temporal transition/assumption code is unchanged; the owner's concurrent Model declaration move was preserved in their `cafdf38dee` checkpoint. All three owner wip checkpoints within the task range are retained. Archived code, generated protocols, milestone files and adapter policy are unchanged.

The three fresh same-family Codex review contexts returned SHIP with zero findings. Their sandbox prevented Cargo execution, so runtime evidence comes from the worker's gates. Canonical receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-220-replace-spell-with-standard-rust.1.json`; durable review ledger: `.flow/specs/fn-220-replace-spell-with-standard-rust.json` (`review_attempts`); review sidecars: `.flow/review-fanout/4948052fe29e43489537e8df39676d1a/`.

GATE_SKIPPED:unittest:green-receipt 9d505cfb - final verification reused this task’s green full Model gate
GATE_SKIPPED:smoke:green-receipt 9d505cfb - final verification reused this task’s green real SDK slice

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran (model: gpt-6.1-sol at high; same family; three fresh contexts; SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cafdf38deee0486abebf89699ee1e84f5926864a, 9f36f5ad5f5d921c927101407b453ece4dedf315, b56024f4fe59af873f8f2338dacae7dc996a31c6, e2dedde3a838ee873492b6e3fc2eb50a11b0f62f, 9d505cfbb2f6bc6a93a495dd413f961f2cdc0185
- Tests: baseline: green (make model-check and make umpire-check-model-slice ran before edits), expected-red: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-dsl --test fixtures (exit 101, missing Display/Debug and obsolete Spell bounds), expected-red: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-checker --test fixtures report_and_path_propagate_formatter_errors (exit 101, missing Display), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-update, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make umpire-check-model-slice, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo fmt --all --check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings, perl .flow/tmp/fn220-golden-audit.pl, GATE_SKIPPED:unittest:green-receipt 9d505cfb - final verification reused this task’s green full Model gate, GATE_SKIPPED:smoke:green-receipt 9d505cfb - final verification reused this task’s green real SDK slice
- PRs:
