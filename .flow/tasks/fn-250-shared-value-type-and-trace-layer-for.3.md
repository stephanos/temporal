---
satisfies: [R3, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.3 Observer names in one module and one trace-replay core for TLC and Apalache

## Description
Implement **Observers and replay (A5)**: `umpire_tla::observers` owns the `__umpire_*` names and `EdgeObservers::decode(&TraceState)`; one step-replay core (input ordinal → catalog entry with `Concrete` identity, choice, `checked_step`, state/outcome/facts comparison) serves `dev/testing/src/backend.rs` (TLC) and `apalache/src/replay.rs`; each keeps only its stutter/tick handling. The dsl macro that emits the names uses the same constants (or a shared constant crate if the macro crate cannot depend on umpire-tla).

## Acceptance
- [ ] No `__umpire_*` literal outside the observers module (and its generated uses).
- [ ] TLC and Apalache replays share one core; divergences still reported per step.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
`umpire_tla::observers` now owns trace observer names and `EdgeObservers::decode`; `umpire_tla::replay::step` selects the canonical concrete action, invokes checked native execution once, and compares typed successor fields, outcome and ordered facts for TLC and Apalache. TLC retains cycle/tick/progress projection, while Apalache retains tagged stutters, before-state history, arithmetic boundaries and specific-property confirmation; its non-unit-fact refusal is removed after an observed red-to-green pinned-tool control.

Tier: session (jev-unavailable(no_key)).
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Task: fn-250-shared-value-type-and-trace-layer-for.3
Status: in_progress; no lifecycle state was mutated.
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2503-replay
Base: 6f030b6204bfc05161c02540c41cd8972667ca67
Commits (oldest first): bda6cddad775161a2621f859a450b9b1d0fc721e, 5d6215c23647d4a3963981df02933846f21a5c7c

Validation: pre-edit scoped baseline green under mise; the initial plain-Cargo invocation failed only because protoc was not on PATH. Final five-crate tests passed 254 tests (one existing ignored), strict all-target Clippy and cargo fmt check passed, and required TLC backend controls passed all 9 tests. Final required Apalache controls passed real fact vectors/stutters/source-only fact mutation (24.53s), an arithmetic refusal after a valid prefix (2.92s), and typed payload rejection/selected postconditions (7.87s). The fact control first failed on the intended unit_facts restriction before implementation, recorded in the reproduction commit. Real TLC A/B/C/D and clearing traces, missing/wrong/duplicate facts, omitted observers and the nonempty-fact activation refusal remain covered. Apalache's existing replay tests still confirm typed identity despite colliding displays, canonical first ordinals, retained starts/catalog, and exactly one checked_step invocation.

Observer sweep: all wire field names, including dynamic predecessor fields, are centralized in producers, schemas, decoders, replay readers, TLA assertion/mutant snippets and the world_fairness/action_inputs/phased_rejection fixture tests. DSL macro `__umpire_input_<field>` temporaries and `__umpire_input_domain` calls are Rust helpers, not emitted wire names; `__umpire_outcome`, `__umpire_get/set`, related generated Rust methods, hygienic TLA locals/World constants, and arithmetic helper operators are also internal helpers. No macro emits a wire observer name, and no macro/backend dependency or manifest/lockfile change was added. The common core uses callbacks and accepts the native step iterator without introducing a DSL/checker dependency or a new step-vector allocation in umpire-tla.

Goldens: no emitted bytes or snapshots changed; the retained fixture goldens passed without regeneration. Gate classification reports FULL; the dispatch explicitly reserves the joined `UMPIRE_APALACHE=require make model-validate` / `make model-check` obligation for the conductor. Required remaining actions are independent implementation review, integration of both commits, joined validation and Flow closure. No review verdict is claimed, no full gate/receipt was fabricated, and no feature-map or MILESTONES file was edited.

Evidence logs and actual Apalache TLA/output/ITF artifacts: /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/milestones-resume-20261024/fn2503-proof
Task-owned build cache: /home/agent/.cache/codex-build/umpire-milestones-20261024/fn2503. Pinned TLC/Apalache caches were copied into this worktree's ignored model/build from the existing admitted root caches; they can be removed with the worktree after integration once the retained proof directory is verified. The worktree remains available for conductor review and integration.


Supplemental joined-gate fix

The joined gate exposed that fixture! emits a backend check hook even for manual/native-only Worlds. The hook now uses the existing autoref Probe pattern: only Worlds with Lifts and typed state/outcome/fact exports instantiate typed replay; the fallback preserves the original located lift/source refusal before executing the native producer and explicitly refuses missing typed projection if lifting succeeds.

Tier: session (jev-unavailable(no_key)).
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Supplemental base: 3821cab6f910e54041974339f9caaff392439c8a
Supplemental commit: aed8b902c03fa9193efde1591cef96031507de5f
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2503-replay

The regression integration test includes the existing native format_values Model unchanged. It first failed to compile on the intended missing Values/IntoValue bounds, then passed exact native state/outcome/fact formatting and equality between lifted/check source refusals; a producer that panics verifies the refused backend never invokes it. A positive admitted-World control verifies that probe dispatch still selects typed backend execution. It uses a child process from the workspace root to resolve rustc's workspace-relative embedded source path without changing the concurrent test process's working directory.

Pre-fix checker all-features Clippy reproduced all 42 E0277 errors. Focused checker/testing/TLA/Apalache/fixtures strict all-target/all-features Clippy passed. All-features tests for those five crates passed 266 tests with one existing ignored test. Full workspace all-target/all-features strict Clippy passed (19.74s), covering the original failing native-only call sites and other downstream callers. Format and diff checks passed. Logs: /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/milestones-resume-20261024/fn2503-proof/supplement

Only dev/testing/src/lib.rs and a new dev/testing integration test changed. Typed replay admission/comparisons and native formatting semantics remain intact; no native-only export impls, fixture/model changes, guest assets, manifests, gate changes or lifecycle mutations were made. The conductor owns independent supplemental review, integration, and the joined full model-validate rerun; this worker claims no review verdict.

Conductor acceptance: integrated in umpire at code checkpoint d30d679fc1; task implementation review reached actual SHIP (including supplemental fixes). The joined UMPIRE_APALACHE=require make model-validate exited 0 on the unchanged model tree, including required TLC/Apalache controls, strict workspace clippy/docs/tests, embedded guest freshness, all standard wasm/native controls and all three Go hosts. Gate 1308.170s exceeds the 60s soft target; edit/test 8.882s meets 10s. Logs: consolidation-retry-model-validate.log and .exit. Final boundary/snapshot checks and reviewed golden/guest updates are integrated.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bda6cddad775161a2621f859a450b9b1d0fc721e, 5d6215c23647d4a3963981df02933846f21a5c7c, 3821cab6f910e54041974339f9caaff392439c8a, aed8b902c03fa9193efde1591cef96031507de5f, d30d679fc1
- Tests: mise exec -- cargo test -p umpire-tla -p umpire-tlc -p umpire-apalache -p umpire-testing (pre-edit baseline: green; plain Cargo first failed to find protoc, corrected with mise), UMPIRE_APALACHE=require mise exec -- cargo test -p umpire-apalache --test controls real_fact_witnesses -- --nocapture (red reproduction: fails on the intended non-unit-facts refusal), mise exec -- cargo test -p umpire-tla -p umpire-tlc -p umpire-apalache -p umpire-testing -p umpire-fixtures (green: 254 passed, 1 existing ignored), mise exec -- cargo clippy -p umpire-tla -p umpire-tlc -p umpire-apalache -p umpire-testing -p umpire-fixtures --all-targets -- -D warnings (green), mise exec -- cargo fmt --all --check (green), git diff --check (green), UMPIRE_BACKENDS=require UMPIRE_APALACHE=require mise exec -- cargo test -p umpire-testing --lib backend::tests -- --nocapture (green: 9 passed), UMPIRE_APALACHE=require UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache --test controls real_fact_witnesses_replay_exact_vectors_and_stutters -- --nocapture (green: 1 passed), UMPIRE_APALACHE=require UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache --test controls real_arithmetic_refusal_follows_a_replayed_valid_prefix -- --nocapture (green: 1 passed), UMPIRE_APALACHE=require UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache --test controls real_typed_payload_rejection_and_after_claims_replay -- --nocapture (green: 1 passed), mise exec -- cargo clippy -p umpire-checker --all-targets --all-features -- -D warnings (pre-fix red: reproduced the 42 E0277 downstream export-bound failures), mise exec -- cargo test -p umpire-testing --test native_backend (pre-fix red: manual State/Decision/Fact require exports; post-fix green: 2 tests), mise exec -- cargo clippy -p umpire-checker -p umpire-testing -p umpire-tla -p umpire-apalache -p umpire-fixtures --all-targets --all-features -- -D warnings (green), mise exec -- cargo test -p umpire-checker -p umpire-testing -p umpire-tla -p umpire-apalache -p umpire-fixtures --all-features (green: 266 passed, 1 existing ignored), mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings (green, 19.74s), mise exec -- cargo fmt --all --check (green), git diff --check (green), UMPIRE_APALACHE=require make model-validate (exit 0; TLC and Apalache required; 1308.170s full gate, 8.882s inner loop; unchanged model tree), Actual Codex gpt-6.1-sol:high SHIP task implementation review and supplemental re-review; same-family CLI reviewer
- PRs: