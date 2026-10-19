---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.20 Flat form: per-item attributes on one impl, match per actor, grouping enforced

## Description
**Owner decisions (2026-10-19, final form after reverting the nested groups and marker traits as "too much ceremony"):** the Model is `#[machine] pub mod kettle { .. }` with the state struct, the action enums and **one `impl Model`** whose methods carry per-item kind attributes, the `options/Kettle.attrs.rs` style: `#[on(User)]` / `#[on(Heater)]` (one `match` method per actor; match guards are the guards; `_ => {}` is "disabled" under .18's enabled-iff-changed rule; rustc's exhaustiveness replaces the missing-handler error when no `_` arm is written), optional `#[effect]` helpers (the only other methods allowed `&mut self`), `#[invariant]`, `#[postcondition]` and `#[postcondition(after = Heater::Failover)]`, `#[assumption]`, `#[progress]`. `#[machine]` enforces that items of one kind are contiguous and the kinds appear in the order on, effect, invariant, postcondition, assumption, progress, with a hard error at the first out-of-place item naming the kind it must follow and precede; unattributed helpers are ignored by the order check but refused if they take `&mut self` (`<method> takes &mut self; only #[on] and #[effect] methods change the state`); `#[machine]` emits `#[forbid(unsafe_code)]` on the module and takes no arguments (catalog order = the `Action` enums' declaration order). Every attribute is pass-through; nothing is rewritten.

```rust
#[machine]
pub mod kettle {
    #[derive(State)]
    pub struct Model {
        #[starts(COLDEST_TEMPERATURE..=40)]
        pub temperature: i32,
        pub heating_on: bool,
        pub beeping: bool,
    }

    #[derive(Action)]
    pub enum User { PressButton, DepressButton }

    #[derive(Action)]
    #[internal]
    pub enum Heater { Heat, Failover }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::PressButton if !self.heating_on => { self.heating_on = true; self.beeping = false; }
                User::DepressButton if self.heating_on => { self.heating_on = false; self.beeping = false; }
                _ => {}
            }
        }

        #[on(Heater)]
        fn heater(&mut self, action: Heater) {
            match action {
                Heater::Heat if self.heating_on && self.temperature < BOILING_TEMPERATURE => self.temperature += 1,
                Heater::Failover if self.heating_on && self.temperature >= BOILING_TEMPERATURE => { self.heating_on = false; self.beeping = true; }
                _ => {}
            }
        }

        #[invariant]
        fn beeps_only_when_off(&self) { if self.beeping { check!(!self.heating_on) } }

        #[postcondition]
        fn warms_only_when_on(&self, before: &Self) { if !before.heating_on { check!(self.temperature <= before.temperature) } }

        #[postcondition(after = Heater::Failover)]
        fn fails_over_at_boiling(&self) { check!(self.temperature == BOILING_TEMPERATURE); check!(self.beeping) }

        /// The user switches the kettle on but never off.
        #[assumption]
        fn never_switched_off() -> Assumption { without([User::DepressButton]) }

        /// Switched on and left on, the water boils and the kettle beeps.
        #[progress]
        fn boils(&self) -> LeadsTo { whenever(self.heating_on).eventually(self.beeping).assuming(Self::never_switched_off()) }
    }
}
```

Do: convert `#[machine]`'s registration from the group impls (.6) to per-item attributes on one impl, binding `#[on(Actor)]` by the attribute's actor and checking the parameter type matches; remove the group attributes and any marker traits; implement the order/grouping check, the `&mut self` rule and `forbid(unsafe_code)`; refusals, each pinned by a trybuild `.stderr` fixture: out-of-order item, `#[on]` parameter not the named actor, two `#[on]` for one actor, an actor without `#[on]`, `&mut self` on a helper or a claim, a leftover group attribute (`#[transitions]`, `#[invariants]`, ...) naming the per-item form; rewrite Kettle and the `machine` macro fixture; all canonical goldens byte-identical (same enabled set); `kettle_trait` untouched; DESIGN.md's comparison shows the Scala `transitions` block beside the Rust `#[on(Heater)]` match, and notes that `State: Hash` excludes interior mutability and that thread-locals/static atomics fall under the capability Clippy policy; README updated. The claim value type is `LeadsTo`; `Progress` is free again and unused as a type.

**Order:** after .18 (done); before or after .19, both touch `machine.rs` in different functions, the second to land rebases; .7 waits for both. **Touches:** `model/umpire/dsl/**`, `model/umpire/tests/fixtures/kettle/model.rs`, `model/umpire/tests/fixtures/changed_state/model.rs`, `model/README.md`, `model/DESIGN.md`. **Feedback loop (R30):** paste `make model-time` into the done summary.

Implementation continuation: the existing shared changed_state attribute fixture must migrate too, without behavior/payload changes. The open .20 review finding still applies to the flat form: qualified, nested, imported and aliased Model impls must not bypass the read-only signature rule, while unrelated helper types stay usable. Compiler-resolved declaration checks may preserve ordinary method bodies; no runtime guards or body rewriting. Keep a real actor-variant catalog for postcondition(after=..) validation. All existing .19 recorder, fixed-assumption and manual-trait compatibility requirements remain; the unchanged manual trait Kettle may retain its legacy core Progress builder, while the attribute-facing claim value is LeadsTo.

**Owner-approved path scope (2026-10-19):** keep path! in fn-215; use the existing whenever(...).eventually(...).assuming(...) builder in .20, returning the author-facing LeadsTo value. The sketch above is updated accordingly. No path! implementation is required or authorized here. Preserve repro commit25841ad439 and the compiler identity proof during continuation; .22 still owns fixture directory renames.

**Owner-approved pass-through exception (2026-10-19):** the owner selected “Allow those compile-time declarations (recommended).” The machine macro may insert compile-time-only checking declarations inside function/const lexical blocks so Rust resolves local aliases, local trait projections and impl identity. Every original authored statement and method/impl header remains unchanged; enclosing blocks may gain declarations. No runtime guards, impl-header predicates, or new authoring restrictions are authorized by this exception. Cover Model refusals and unrelated helper acceptance with regression fixtures. This narrow permission supersedes the no-body-augmentation interpretation above, not the ordinary pass-through contract. The separate path! scope decision is settled above; both owner decisions are resolved.
## Acceptance
- One `impl Model` with per-item attributes `#[on(Actor)]`, `#[effect]`, `#[invariant]`, `#[postcondition]`/`#[postcondition(after = ..)]`, `#[assumption]`, `#[progress]`; no group attributes, no marker traits; every attribute pass-through.
- `#[machine]` refuses an out-of-place item, a non-grouped kind, an `#[on]` whose parameter is not the named actor, two `#[on]` for one actor, an actor without `#[on]`, `&mut self` outside `#[on]`/`#[effect]`, and a leftover group attribute, each with the message above and a trybuild `.stderr` fixture; omitting `_ => {}` yields rustc's exhaustiveness error; the module carries `#[forbid(unsafe_code)]`.
- Kettle and the `machine` macro fixture use the flat form; all canonical goldens are byte-identical; `kettle_trait` unchanged.
- DESIGN.md and README show the form; the full local gate passes within R30.
## Done summary
CURRENT FLAT SOURCE - in_progress, integrated at 4b7924fb105e15e53b72cd17e0a8ea95153b52c7 and under the same-primary-session re-review. Source checkpoint dc56fc7199096944292522ec47137c97723f66f8 implements the approved flat per-item actor/effect/claim attributes, order checks and author-facing LeadsTo builder. Compiler-only declarations enforce readonly Model receivers across qualified, nested, imported/aliased, function-local, const-local and trait-projected impls while preserving unrelated generic/shadowed/local helpers and all original statements and method/impl headers.

Integrated verification: refreshed fixture build-script outputs and proved all four active registries refer to the root checkout, not the retained worktree (fn210-20-integrated-final-registry-paths.log). Root cargo test --workspace --no-run and make model-check both exited0 (fn210-20-integrated-final-artifacts.jsonl and fn210-20-integrated-final-model-check.log). The model tree is exact to dc56fc7199; protected payload/metadata/manual trait/options/workspace manifest/lock remain unchanged against b00. Review verdict is pending; no task completion claim.

Both owner decisions are resolved and merged normally. Compile-time-only declarations may extend lexical blocks under the recorded narrow pass-through exception; path! stays in fn-215, and .20 uses whenever/eventually/assuming. No runtime guard, header predicate, new crate/version, workspace-manifest, lockfile, unsafe or lint-policy change was added. The macro crate enables only the existing syn dependency's visit-mut feature for declaration insertion. Manual trait Kettle and its legacy core Progress compatibility remain byte-identical.

Final focused tests, workspace tests, all-target Clippy and make model-check passed on the correct assigned-worktree fixture registries. All 41 refusal cases pass. The machine expansion is the only changed existing expected artifact; every canonical behavior payload/metadata file, the manual trait source, historical options, and all 22 blobs in the conductor manifest are unchanged. The task remains in_progress for the conductor's same-primary-session review and Flow completion. Full fix details, exact red-to-green proof and gate receipts are in `fn210-20-review-fix-summary.md` and `fn210-20-review-fix-evidence.json`. Historical readiness and source/gate records are retained below and in the readiness archive.

Exactly one actual warm make model-time ran on final committed source dc56fc7199096944292522ec47137c97723f66f8, exiting 0 with no retry (`fn210-20-flat-model-time.log`):

```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.009s
capability-controls               0.504s
protobuf-freshness                0.093s
fmt                               0.140s
clippy                            0.382s
docs                              0.138s
tests                             2.130s
wasm32-wasip1                     0.101s
core-wasm-clippy                  0.074s
wasm-guest                        0.090s
go-host                           1.301s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.421s
gate total                        5.115s  (budget 30s)
inner loop                        1.421s  (budget 3s)
```

The committed worktree `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21020-actors` is clean, all worker commands and child agents are terminal, and all logs/proofs are copied to canonical root `.flow/tmp/`. Full task history is f7ec7b60f25594b1ae8b8c6dd46bb62c180dfb13..dc56fc7199096944292522ec47137c97723f66f8, including the preserved failing reproduction 25841ad439109073aaf2c518f07603d771e1ba2b and normal planning merges. Requested implementer remains gpt-6.1-sol at medium; the host exposes no executed-model identifier, so no actual-model claim is made.

HISTORICAL GROUPED-SOURCE CHECKPOINT

Implemented actor-type transition dispatch, the optional effects group, named mutable-Model-method refusals and module-level unsafe prohibition. Migrated Kettle, changed_state and DSL fixtures while preserving every existing behavior payload; README and DESIGN show that earlier approved authoring form.

Requested implementer: gpt-6.1-sol at medium, from dispatch and AGENTS. The host exposed no executed model identifier, so no actual-model claim is made. The run retains the conductor's judge-unavailable(no_key) fallback; this worker issued no judge verdict. Three disjoint implementation children finished before staging. The worker alone ran Cargo/Make and committed.

Baseline: green. `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test --manifest-path model/Cargo.toml -p umpire-dsl -p umpire-dsl-macros -p umpire-tests` exited 0 before edits (`fn210-20-baseline.log`). No handoff or gate receipt was reused.

Test-first evidence: legacy-form mutable helper, typed mutable helper and unsafe-with-local-allow controls compiled under the old macro, making the refusal suite fail with exit 101. The mutable invariant produced only the former generic E0596 diagnostic rather than the required named refusal (`fn210-20-refusals-red.log`). The first implementation compile exposed a syn 3 Receiver API mismatch, corrected using its actual ReceiverKind definition (`fn210-20-first-green.log`). Reviewed diagnostic updates preserve existing claim, assumption, async, return, legacy Scope and unknown-after errors. Actor duplicate/missing/nonactor/signature, rustc exhaustiveness, mutable methods, effects placement/duplication/orphan and unsafe controls are pinned in trybuild. The positive fixture deliberately changes actor method names and order, uses effects and an unrelated mutable helper type, and preserves its values payload.

Verification: focused tests passed (`fn210-20-focused-final.log`), followed by full workspace tests (`fn210-20-workspace-final.log`). The first full model gate caught fixture Clippy issues (`fn210-20-model-check-final.log`). Explicit Copy derives on concrete unit-only actors retain the required owned parameter form without changing derive(Action) or lint policy. The empty actor match consumes its receiver, and the existing wildcard covers NoChange. Only the reviewed machine expansion payload was accepted, first for actor dispatch and then for those Copy derives (`fn210-20-diagnostics.log`, `fn210-20-expansion-copy.log`). Final `make model-check` and final `mise exec -- cargo test --workspace` both exited 0 (`fn210-20-model-check-final2.log`, `fn210-20-workspace-final2.log`). The model gate includes formatting, all-target Clippy, docs, native fixtures, capability/protobuf controls, wasm builds and CGO-disabled Go-host parity.

The before manifest records all expected artifacts and trait source (`fn210-20-protected-before.txt`). Shared fixture tests exited 0 (`fn210-20-shared-parity.log`). All expected artifacts outside the single machine expansion remain byte-identical to the base (`fn210-20-canonical-parity.log`, exit 0). Kettle, changed_state, manual trait fixtures, dependent controls and historical options remain unchanged in their protected payloads. No fixture alias symlinks were restored. Git diff --check passes and the committed worktree is clean.

Exactly one actual warm `make model-time` ran after source commit 7bb60aa2ead81faf2e6f38c0e9ea81a53777ec62, exiting 0 (`fn210-20-model-time.log`). No timing retry or exception was needed.

```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.011s
capability-controls               0.151s
protobuf-freshness                0.087s
fmt                               0.121s
clippy                            0.138s
docs                              0.117s
tests                             0.849s
wasm32-wasip1                     0.058s
core-wasm-clippy                  0.078s
wasm-guest                        0.076s
go-host                           1.482s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.796s
gate total                        3.301s  (budget 30s)
inner loop                        1.796s  (budget 3s)
```

All named logs are copied to `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/`. Workspace `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21020-actors` remains for conductor integration and review. Base f7ec7b60f25594b1ae8b8c6dd46bb62c180dfb13, complete source range f7ec7b60f25594b1ae8b8c6dd46bb62c180dfb13..7bb60aa2ead81faf2e6f38c0e9ea81a53777ec62. Read-only flowctl show confirms in_progress. All children and commands are terminal. The conductor owns review, integration and Flow completion.

Conductor review finalized SHIP at integrated head 4b7924fb105e15e53b72cd17e0a8ea95153b52c7. The same primary session 01a1520d-2416-73f3-95f0-eabbba7af7b1 ran round2 on gpt-6.1-sol at high effort, marked the receiver-validation finding fixed, and reported no new findings. Writer and reviewer are from the same model family. Receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.20.json has SHA256 2d33963989d9342908dc7d0e0660aa60d4b00f1ae6301378d54db2b5f9aa0328. Fresh Cargo in the read-only reviewer sandbox could not acquire the build lock; six cached probes supplement the conductor's actual green integrated model-check rather than replacing it.

Memory capture after the nontrivial NEEDS_WORK-to-SHIP fix created bug/build-errors/resolve-model-impl-identity-in-its-2026-10-19. Its moderate overlap concerns a separate empty-enum derive bug; the new entry records lexical/type/trait-context validation and paired positive controls.

stage: impl-review - ran (model: gpt-6.1-sol high)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits: 147793c2abc74a9890f4915aca27ff781239df2c, 7bb60aa2ead81faf2e6f38c0e9ea81a53777ec62, b00b30df2a416d6dd4144399929713e86a671f0b, 25841ad439109073aaf2c518f07603d771e1ba2b, 4da1bda26ad652fe7ceb1c36e5b79b07a0b5ec78, 39327c67999fb9f307bf84d984929bc65b631872, 2d095df2bfa812fec0acb62350addff8ca55abd9, 0659a665278c14c3955fb0f6b1f0d32164bc3348, dbe08e9baec9998ae06a7e069ac55cec1b9ef19a, 63305f5f5fe5e9c42f9c9e52fc25d8444eedf24a, 98846ca12226f001ff593fcc9166b5349cdda49d, dc56fc7199096944292522ec47137c97723f66f8, 4b7924fb105e15e53b72cd17e0a8ea95153b52c7
- Tests: HISTORICAL GROUPED SOURCE 7bb60aa2ea: baseline/focused/workspace/model-check exit0; original test-first exit101; one warm make model-time exit0 gate3.301s inner1.796s. Original logs and summary retained., test-first qualified/nested/importalias controls unexpectedly compiled before fix (exit101, fn210-20-review-fix-red.log; repro committed25841ad439), test-first function/const/local projection controls unexpectedly compiled before lexical traversal (exit101, exactly3/41 fail, fn210-20-flat-local-projection-red.log), positive trait-context regression produced E0220 before preserving original trait context (exit101, fn210-20-flat-trait-context-red.log); final positive fixture passes, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-tests (exit 0, fn210-20-flat-focused-final2.log; all 41 macro refusals pass), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test --workspace (exit 0, fn210-20-flat-workspace-final.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo clippy --workspace --all-targets -- -D warnings (exit 0, fn210-20-flat-clippy-final.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check (exit 0, fn210-20-flat-model-check-final2.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time (single warm run after commit dc56fc7199096944292522ec47137c97723f66f8, exit 0, gate5.115s inner1.421s, fn210-20-flat-model-time.log), cargo test --workspace --no-run --message-format=json (exit 0; all four active fixture registries point to assigned worktree, fn210-20-flat-final-registry-paths.log), git diff --exit-code b00b30df2a416d6dd4144399929713e86a671f0b -- ':(glob)model/**/expected/**' ':(exclude)model/umpire/dsl/macros/tests/fixtures/machine/expected/**' model/umpire/tests/fixtures/kettle_trait/model.rs model/options model/Cargo.toml model/Cargo.lock (exit 0, fn210-20-flat-canonical-parity-final.log), git hash-object comparison against conductor fn210-22-golden-before.txt (exit 0, all22 blob IDs match, fn210-20-flat-canonical22-objects-final.log), git diff --check (exit 0); final git status --short empty; pending snapshot search empty, INCONCLUSIVE: intermediate stale-root expansion observation fn210-20-flat-diagnostics2.log was rejected; only verified-worktree snapshots were accepted, INTEGRATED ROOT HEAD4b7924fb105e15e53b72cd17e0a8ea95153b52c7: cargo test --workspace --no-run --message-format=json exit0; all4 active generated fixture registries verified ROOT not worktree (fn210-20-integrated-final-artifacts.jsonl, fn210-20-integrated-final-registry-paths.log), INTEGRATED ROOT HEAD4b7924fb105e15e53b72cd17e0a8ea95153b52c7: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check exit0 (fn210-20-integrated-final-model-check.log), Integrated model tree exact worker dc56fc7199; protected expected metadata/payloads/manualtraitsource/options/workspacemanifest/lock unchanged against b00 (git diff --exit-code exit0), Review limitation: fresh Cargo in readonly reviewer sandbox blocked by build-lock permissions; six cached probes are supplementary; root actual integrated full gate is authoritative
- PRs: