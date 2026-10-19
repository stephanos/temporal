---
satisfies: [R6, R14, R44]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.6 Kettle in surface C with its test; parity with the Scala suite

## Description
**R14 joined integration:** consume the .5 source-order metadata in checker reporting, if the separate registry hooks otherwise group claims by kind. Add a shared Model with interleaved invariant/postcondition/progress declarations and a report golden proving actual order, while keeping manual-world defaults and dynamic effect failures deterministic. This checker integration belongs here after .3/.5, not in the parallel DSL lane.

**Touches:** `model/umpire/tests/**`, `model/umpire/checker/src/**`, `model/umpire/checker/tests/**`, `model/umpire/dsl/**`, `model/DESIGN.md`

**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model-check` at 30 s or less, warm; paste `make model-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Location (R29 amended):** there is no `samples` crate; write Kettle in surface C as `umpire/tests/fixtures/kettle/model.rs`, replacing the trait-shaped source there. Shared end-to-end fixtures live in `umpire/tests/fixtures/<name>/` and their expected files select the consumers.

**Owner-approved authoring decision (2026-10-18):** keep the sketch's separate `User` and `Heater` enums. This supersedes this task's earlier byte-identical canonical-goldens requirement only for the four author action spellings: `Act::UserPressButton` becomes `User::PressButton`, `Act::UserDepressButton` becomes `User::DepressButton`, `Act::HeaterHeat` becomes `Heater::Heat`, and `Act::HeaterFailover` becomes `Heater::Failover`. Counts, state spelling and discovery order, action order, claims, verdicts, vacuity and counterexample structure must not change. Record the exception in DESIGN.md. Preserve the separate trait Kettle and all its goldens unchanged as the WASM/native reference. Names produced by surface C must remain the actual author Rust names (R25/R26), not aliases chosen to fit old goldens.

Write `umpire/tests/fixtures/kettle/model.rs` in surface C from `options/Kettle.attrs.rs`; the shared fixture runner asks the Scala suite's questions: explore counts, verify_all verified, `boils` verified, and `!beeping` refuted. Prove parity against the existing canonical goldens with only the four permitted name substitutions, and retain executable coverage of the unchanged trait reference. Record every deliberate deviation from the hypothetical sketch, including static scope/progress builders and generated world/action adapters.

**Files:** model/umpire/tests/fixtures/kettle/model.rs and shared fixture runner; checker ordering integration and shared interleaved-claim fixture; model/DESIGN.md for deliberate sketch differences. Do not change workspace manifests, the ABI, archived Scala, or the separate trait fixture and its goldens.

**Quick commands:** `cd model && mise exec -- cargo test --workspace`

**Target surface (owner approved 2026-10-18, nested form; R44, supersedes the sketch where they differ):** the Model is one inline `#[machine] pub mod kettle { .. }` with one `impl Kettle` per part kind, in this order and each at most once: `#[transitions]`, `#[invariants]`, `#[postconditions]`, `#[scopes]`, `#[progress]`. The block is the group: no `#[invariant]`/`#[postcondition]`/`#[progress]` per item; only the per-claim qualifiers `#[on(..)]`, `#[after(..)]`, `#[within(..)]`. `#[machine]` takes no arguments; catalog order is the `#[derive(Action)]` enums' declaration order. `#[machine]` registers from the module's items (R36), errors at the first out-of-place or duplicated group, ignores unmarked impls and helpers; trybuild `.stderr` fixtures `machine_out_of_order` and `machine_duplicate_group` pin the messages. Rewrite the `machine` macro fixture to this form. The R14 interleaved-claim fixture must be a manual trait world, since the attribute surface cannot interleave kinds.

```rust
#[machine]
pub mod kettle {
    #[derive(State)]
    pub struct Model {
        #[starts(COLDEST_TEMPERATURE..=40)]
        temperature: i32,
        heating_on: bool,
        beeping: bool,
    }

    #[derive(Action)]
    pub enum User { PressButton, DepressButton }

    #[derive(Action)]
    #[internal]
    pub enum Heater { Heat, Failover }

    #[transitions]
    impl Model {
        #[on(User::PressButton)]
        fn press_button(&mut self) -> Option<()> { (!self.heating_on).then(|| { self.heating_on = true; self.beeping = false; }) }
        #[on(Heater::Failover)]
        fn failover(&mut self) -> Option<()> { .. }
        #[on(Heater::Heat)]
        fn heat(&mut self) -> Option<()> { .. }
        #[on(User::DepressButton)]
        fn depress_button(&mut self) -> Option<()> { .. }
    }

    #[invariants]
    impl Model {
        fn beeps_only_when_off(&self) -> bool { !self.beeping || !self.heating_on }
    }

    #[postconditions]
    impl Model {
        fn warms_only_when_on(&self, before: &Self) -> bool { before.heating_on || self.temperature <= before.temperature }
        #[after(Heater::Failover)]
        fn fails_over_at_boiling(&self) -> bool { self.temperature == BOILING_TEMPERATURE && self.beeping }
    }

    #[scopes]
    impl Model {
        /// The user switches the kettle on but never off.
        fn never_switched_off() -> Scope { without([User::DepressButton]) }
    }

    #[progress]
    impl Model {
        /// Switched on and left on, the water boils and the kettle beeps.
        #[within(Self::never_switched_off)]
        fn boils(&self) -> Progress { whenever(self.heating_on).eventually(self.beeping) }
    }
}
```

Details fixed with it: actor-level `#[internal]` only (as Scala's `extends Internal`), no per-variant form; `before` optional in postconditions (macro reads the arity); `Scope` and `Progress` are plain non-generic values, `Progress` = `{ trigger: bool, target: bool }` evaluated per state, expanded as `|_, s| s.boils().trigger` / `.target`, `Scope` holds actor-level action values mapped into `kettle::Action` by the expansion; the author never names the generated `kettle::World`/`kettle::Action` (module-scoped, no prefix). Kept: `#[on]` handlers returning `Option<()>` (empty-vs-enabled-no-op matters for liveness) and named `#[scopes]`. The `Machine` trait, checker, goldens and wasm host do not move; consumers (shared fixture runner, `umpire-wasm`'s `#[path]` include) refer to `kettle::Model` and the `pub` items. **Naming (owner 2026-10-18):** the state struct is `Model` (Scala's `object Model`), so the canonical goldens carry a fifth approved substitution, state spelling `Kettle { .. }` to `Model { .. }` in `states.txt`, `rows.txt` and `paths/`; record it next to the four action substitutions in DESIGN.md. `kettle_trait` stays unchanged. Record any remaining difference from the sketch in DESIGN.md, one sentence each.
## Acceptance
- The canonical Kettle uses the attribute surface with separate `User` and `Heater` enums; its fixture asks all Scala-suite questions.
- Canonical expected payloads differ from the previous trait-shaped Kettle only by the four owner-approved action-name substitutions and the state spelling `Kettle { .. }` to `Model { .. }`; report counts/verdicts, states, catalog/discovery order and counterexample structure remain identical.
- The separate `kettle_trait` source and all its expected files remain unchanged and its native/WASM consumers still pass.
- `#[machine]` on the Model module rejects a duplicated or out-of-order group impl (`#[transitions]`, `#[invariants]`, `#[postconditions]`, `#[scopes]`, `#[progress]`) with an error at the first offending block; trybuild `.stderr` fixtures pin both messages and every attribute fixture complies (R44).
- A shared interleaved invariant/postcondition/progress fixture on a manual trait world proves actual checker report declaration order; manual-world defaults and dynamic effect-failure ordering remain deterministic.
- Kettle is written in the approved nested surface: a `#[machine] pub mod kettle` whose state struct is `Model`, with one impl per part kind, actor-level `#[internal]`, optional `before`, `#[after(..)]`/`#[within(..)]` qualifiers, `fn never_switched_off() -> Scope { without([User::DepressButton]) }` and `fn boils(&self) -> Progress { whenever(self.heating_on).eventually(self.beeping) }`, with no generated world or action type named by the author; remaining differences from the sketch are recorded in DESIGN.md.
- The workspace test command and full local gate pass; actual warm model-time output satisfies R30 or records an owner-approved exception.
## Done summary
Canonical Kettle now uses the approved nested Model module, actor-level internal actions and ordinary Rust scope/progress methods. Checker reports honor declared ordering while supplemental query slots and dynamic effect failures remain deterministic.

Task: fn-210-rust-prototype-kettle-on-a-trait-engine.6
Status: in_progress; conductor owns integration, review and completion.
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2106-kettle
Base: 85eb697c5ce5b135ba6d981c05d2bac66602b77f
Source commits: e56786fbfc3eb901208946e3fa4c79d3f0f9fa0b; f42ad5da3c87ad99c911e06445ea0284ea30e149
Final HEAD: f742f1203f844489745fe8357c86ff0702193dc6
Tier: project implementer gpt-6.1-sol medium; judge unavailable no_key retainprojecttier.
Delegation: 4 unique host child implementers, explicitly dispatched as gpt-6.1-sol at medium, with disjoint written-file ownership. Existing children were reused for follow-ups. All children are terminal. Executed model metadata was unavailable, so actual_model is not inferred from dispatch settings.
stage: impl-review - ran [85eb697c5ce5b135ba6d981c05d2bac66602b77f..702e7a724e3ae310e0709fd424162348b4d30863] (model: gpt-6.1-sol high)

### Final implementation

This record supersedes the provisional handover. The owner expanded .6 after the first source checkpoint, then approved Model naming and the fifth golden substitution. The worker merged the authoritative f97c8fbb6f and 4f35013f4d planning checkpoints normally, at d71da8a7b0 and f742f1203f. Those commits carry the owner's Flow/MILESTONES changes; the worker wrote implementation only inside the expanded task Touches. Later owner fn-213/MILESTONES changes were left untouched.

The machine attribute now accepts one inline module with a derived state and action enums in catalog declaration order. Its single group-order table enforces transitions, invariants, postconditions, scopes, progress, with each group at most once and an error at the first offending block. Unmarked impls and transition helpers remain ordinary Rust. Original methods, bodies and doc comments pass through after registration markers are removed. The replaced impl-based entry and standalone marker exports were deleted.

Private module-scoped World and Action adapters preserve the Machine protocol. The documented public world() constructor returns an opaque Machine with public state and actor From bounds. Local typed Scope/without and boolean Progress/whenever/eventually helpers preserve the approved author syntax without type erasure or generated names in handwritten code. Unknown actor exclusions fail the conversion bound. Actor-level internal applies to the whole enum, variant markers refuse, and postconditions accept optional before parameters.

Canonical kettle::Model uses all five groups and the exact approved scope/progress spelling. Cut uses world(); no-heat is generic over the public machine interface and actor conversions. DESIGN.md records every remaining sketch difference. The shared interleaved_claims Model is now a manual trait world with interleaved typed claim constructors and explicit declaration metadata; its existing report/state/row/path payloads remain unchanged.

The checker reorders registered claim slots after evaluation, preserving duplicate-name multiplicity and cached reports. Checker::refuted carries the supplemental invariant's origin index so even a name collision cannot move it. Effect failures remain after declared/query claims in discovery order. The claim_order fixture pins manual defaults, duplicates, two dynamic failures, declared/dynamic name collision, repeated cached reporting and supplemental name collision.

### Golden preservation

fn210-6-golden-parity.log compares 40 original-base artifacts, including canonical, cut, no-heat and the complete trait reference. Exactly eight payloads differ after only these five approved substitutions:

- Act::UserPressButton -> User::PressButton
- Act::UserDepressButton -> User::DepressButton
- Act::HeaterHeat -> Heater::Heat
- Act::HeaterFailover -> Heater::Failover
- Kettle {  -> Model {  (state prefix)

Counts, state values, catalog/discovery order, claims, verdicts, vacuity and counterexample structure remain identical after those substitutions. The canonical negative report remains byte-identical. All snapshot metadata, symlink targets, kettle_trait source/goldens and original option sketches remain unchanged. Explicit protected-path comparisons also preserve host, wasm/proto ABI, archive, manifests and lockfile.

The deliberate DSL snapshot changes are separate from canonical parity. The machine expansion now represents the nested surface. The runtime DSL values golden changes only its order line to the newly mandated kind grouping. Existing missing-on, duplicate-on, unknown-after and unknown-scope controls were rewritten into the new surface and still fail for their intended reasons. New pins cover first out-of-order group, first duplicate group, unknown actor conversion and forbidden variant-level internal.

### Verification and observations

All logs named below exist in both the worktree's .flow/tmp/ and /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/. All Cargo/Make commands used CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target. No gate was skipped or receipted by the worker; receipts and review remain conductor-owned. Classification requires FULL (unmatched MILESTONES.md after the owner planning merge).

- baseline: green. Original pre-edit cargo test --workspace exited 0 (fn210-6-baseline.log). The revised-scope baseline after merging the nested plan also exited 0 before revised edits (fn210-6-revised-baseline.log).
- Ordering red/green: checker fixtures exited 101 for grouped rather than declared order, then 0 (fn210-6-order-red-checker.log, fn210-6-order-green.log). The initial sort's movement of never_beeps was captured and corrected (fn210-6-kettle-red-actual.log).
- Supplemental collision red/green: a duplicate=Refuted query moved from slot 2 to slot 1 before its origin-index fix (exit 101, fn210-6-collision-red.log); the joined focused checker/shared fixtures then passed.
- Actor red: the macro expansion's internal() returned false instead of true for an actor-marked enum (exit 101, fn210-6-actor-red-actual.log). After the fix, the final macro expansion and runtime suites pass.
- Model spelling red: shared fixtures exited 101 at the first state-spelled counterexample (fn210-6-model-name-red-actual.log). Before changing expected files, the worker independently compared the generated pending bytes with the old payload plus only Kettle { -> Model {. The five-transform proof was then refreshed against the original base.
- Grouped values red: DSL fixtures exited 101; pending output differed only in the task-required order line (fn210-6-group-order-values-red.log). That deliberate pin was updated.
- UI observations: fn210-6-module-focused-first.log records exact intended errors for all retained/new controls. Source relocation changed old stderr locations; new groups and From<Other> refusals gained reviewed pins. No control was accepted merely for an expected-module parsing error.
- Final focused: cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-tests -p umpire-checker exited 0, including all 10 macro refusal controls, two missing-Spell controls, normal fixtures and the checker property test (fn210-6-nested-focused.log).
- Final exact Quick command: cargo test --workspace exited 0 with nonzero suites (fn210-6-nested-workspace.log).
- Final full gate: make model-check exited 0 (fn210-6-nested-model-check-first.log), including fmt, all-target Clippy with warnings denied, docs, native tests, capability/protobuf controls, wasm core/guest and CGO-disabled Go parity against unchanged trait goldens.
- Final make model-time exited 0 on the final joined HEAD, running the complete actual gate again (fn210-6-nested-model-time.log). Both R30 bounds pass; no timing retry or exception was needed.
- git diff --check, protected-path comparisons and clean git status passed. flowctl show remains in_progress. Every command and child has terminated.

Earlier inconclusive observations are retained, not counted as regressions or green gates. fn210-6-kettle-red.log saw a fixture mid-write without expected/; fn210-6-order-red.log saw old no-heat imports; fn210-6-actor-red.log saw the intentionally changed expansion fixture input before the module producer accepted ItemMod; fn210-6-model-name-red.log saw missing Self type arguments in the manual conversion, subsequently fixed. Actual gate/lint failures were fixed without suppressions or weakening checks: initial checker function length and test idioms (fn210-6-model-check*.log), then module collapsible-if, function length and needless value passing (fn210-6-module-clippy-first.log). Pending snapshots were removed after comparison/acceptance.

Actual final warm model-time output:

~~~text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.023s
capability-controls               0.267s
protobuf-freshness                0.073s
fmt                               0.115s
clippy                            0.107s
docs                              0.118s
tests                             0.718s
wasm32-wasip1                     0.056s
core-wasm-clippy                  0.075s
wasm-guest                        0.083s
go-host                           1.241s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            0.879s
gate total                        3.106s  (budget 30s)
inner loop                        0.879s  (budget 3s)
~~~

No review verdict, Flow lifecycle mutation, shared gate receipt, push, history rewrite, repository extraction or worktree teardown was performed. Keep the worktree for conductor review.

### Conductor integration and review

The preceding paragraph records the worker handoff, not the final conductor state. Integrated normally at 702e7a724e3ae310e0709fd424162348b4d30863; the model tree exactly matches worker f742f1203f. The separate approved fn-213 planning checkpoint is retained and no fn-213 work was started.

All three read-only correctness/contracts/integration draws returned SHIP, with zero findings or failed draws. Actual reviewer model: gpt-6.1-sol, high effort, same family as the writer. RID: f2c489b29a93447a83ecc9ce52709e8b. Receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.6.json; SHA256: 9c5caa876b53d6e23f53edb2fe0f4ec3b9b789955b62f5f02c257f56308a3d92. Raw sidecars remain under .flow/review-fanout/f2c489b29a93447a83ecc9ce52709e8b/. Reviewer attempts to run Cargo were blocked by their read-only build lock; those attempts are inconclusive, not passing tests. The conductor's actual gate is authoritative.

Integrated make model-check exited 0 (.flow/tmp/fn210-6-integrated-model-check.log). An earlier identical integrated invocation completed but its exit observation was lost; its log fn210-6-integrated-before-review.log is retained as inconclusive, not counted as a green run. Final make model-time exited 0 (.flow/tmp/fn210-6-integrated-model-time.log), without timing retries or an owner exception:

~~~text
timing-controls                   0.012s
capability-controls               0.718s
protobuf-freshness                0.116s
fmt                               0.142s
clippy                            0.296s
docs                              0.134s
tests                             1.522s
wasm32-wasip1                     0.077s
core-wasm-clippy                  0.081s
wasm-guest                        0.103s
go-host                           1.983s
inner loop (edit+test)            2.945s
gate total                        5.586s  (budget 30s)
inner loop                        2.945s  (budget 3s)
~~~

Memory capture skipped: memory.enabled is true, but first-pass SHIP had no NEEDS_WORK-to-SHIP fix. The new owner-approved handler-shape task .18 is explicitly a follow-up: this checkpoint intentionally retains the .6 Option-returning handlers until .18 changes them. No source change was made during review.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f91f6b787e30c0414c8f75a2adaee4bdc3a470b1, e56786fbfc3eb901208946e3fa4c79d3f0f9fa0b, 858fce89b7304efda7e37876eddd641752fef9ac, f97c8fbb6fc06b8e0f66d7a3b464103e0c26cd34, d71da8a7b0572ec197cc256068b2ac7e2596b566, 4f35013f4d1b32832a37137fbc4ebc0e866fff86, f42ad5da3c87ad99c911e06445ea0284ea30e149, f742f1203f844489745fe8357c86ff0702193dc6, 702e7a724e3ae310e0709fd424162348b4d30863
- Tests: baseline: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test --workspace (exit 0; fn210-6-baseline.log), revised baseline: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test --workspace (exit 0; fn210-6-revised-baseline.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-checker --test fixtures (ordering red exit 101; green exit 0; fn210-6-order-red-checker.log, fn210-6-order-green.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test -p umpire-checker --test fixtures (supplemental collision red exit 101; fn210-6-collision-red.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test -p umpire-dsl-macros --test fixtures fixtures (actor internal red exit 101; fn210-6-actor-red-actual.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test -p umpire-tests --test fixtures (Model spelling red exit 101; fn210-6-model-name-red-actual.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test -p umpire-dsl --test fixtures (intentional grouped order red exit 101; fn210-6-group-order-values-red.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test -p umpire-dsl-macros -p umpire-dsl -p umpire-tests -p umpire-checker (exit 0; fn210-6-nested-focused.log), cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s mise exec -- cargo test --workspace (exit 0; fn210-6-nested-workspace.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s make model-check (exit 0; fn210-6-nested-model-check-first.log), CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s make model-time (exit 0; inner 0.879s, gate 3.106s; fn210-6-nested-model-time.log), exact original-base golden comparison: 40 artifacts, eight payloads changed through only five approved substitutions; fn210-6-golden-parity.log, integrated CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s make model-check (exit 0; fn210-6-integrated-model-check.log), integrated CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target timeout 600s make model-time (exit 0; inner 2.945s, gate 5.586s; fn210-6-integrated-model-time.log), implementation review: three gpt-6.1-sol high draws SHIP, zero findings; RID f2c489b29a93447a83ecc9ce52709e8b; receipt SHA256 9c5caa876b53d6e23f53edb2fe0f4ec3b9b789955b62f5f02c257f56308a3d92, git diff --check (exit 0), git diff --exit-code 85eb697c5ce5b135ba6d981c05d2bac66602b77f -- model/umpire/tests/fixtures/kettle_trait model/options model/umpire/hosts model/archived model/Cargo.toml model/Cargo.lock (exit 0)
- PRs: