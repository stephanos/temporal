---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.18 Handlers as fn(&mut self): enabled iff the state changed

## Description
**Owner decision (2026-10-18):** a transition handler is `fn name(&mut self)` with no return type; the action is enabled on a state iff the handler changed it. `#[machine]`'s dispatch already clones `next` from `state`, so it emits the step iff `next != *state` (`State: Eq`). This replaces `-> Option<()>` with `.then(|| ..)` or `Some(())`, and it is TLA+'s stuttering semantics: an unchanged state is indistinguishable from a disabled action (relevant to fn-212's agreement). Consequence, accepted: an action that writes the same value counts as disabled on that state, so an `#[after(..)]` postcondition is not checked on that stutter; Kettle has no such action; the trait form can still express an explicit no-change step, and no escape hatch is added now.

```rust
#[transitions]
impl Model {
    #[on(Heater::Heat)]
    fn heat(&mut self) {
        if self.heating_on && self.temperature < BOILING_TEMPERATURE {
            self.temperature += 1;
        }
    }
}
```

Do: change the dispatch in `dsl/macros/src/machine.rs`; refuse a handler with a return type (`#[on] handlers return nothing; enabledness is a changed state`) with a trybuild `.stderr` fixture `machine_handler_returns`; rewrite Kettle's four handlers and the `machine` macro fixture; the canonical goldens must not change (same enabled set for Kettle); update .6's "kept" note in DESIGN.md. `kettle_trait` is untouched.

**Touches:** `model/umpire/dsl/macros/**`, `model/umpire/dsl/tests/fixtures/machine/**`, `model/umpire/tests/fixtures/kettle/model.rs`, `model/umpire/tests/fixtures/changed_state/**`, `model/DESIGN.md`

**Integration clarification:** the macro expansion fixture reads the runtime DSL fixture at `dsl/tests/fixtures/machine/model.rs`; adapt that fixture and its values golden to the changed-state rule too. All existing refusal fixtures must continue to pin their intended error after handler signatures change. Canonical Kettle and trait-reference goldens remain byte-identical; DSL expansion/runtime fixture deltas are reviewed separately.

**Feedback loop (R30):** paste `make model-time` into the done summary.
**Checker-level control:** add a shared `changed_state` fixture using existing consumers: no-op, same-value assignment and mutate-then-restore actions emit no edges and leave matching false postconditions untriggered; a real state change emits an edge and triggers a false postcondition as a positive control. Keep the existing trait `edge_checks` self-edge/refutation goldens unchanged.
## Acceptance
- Transition handlers are `fn name(&mut self)` with no return type; `#[machine]` emits a step iff the handler changed the state.
- A handler with a return type is refused by `#[machine]` with a message naming the rule, pinned by a trybuild `.stderr` fixture.
- Kettle's handlers and the `machine` macro fixture use the new form; all canonical goldens are byte-identical to before this task; `kettle_trait` is unchanged.
- DESIGN.md records the changed-state enabledness and its stutter consequence in one sentence each; the full local gate passes within R30.


## Done summary
Implemented fn-210.18 in `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21018-handlers`, committed as `3fae17bccb2245a938fb1e855042275d2350e26c`. Registered handlers have no return type; dispatch invokes them on a clone and emits only when the final state differs. The return-type refusal pins the exact task message, Kettle uses guarded unit methods, and DESIGN records enabledness and the stutter consequence.

Tier: session (jev-unavailable(no_key)), preserve project routing.
Requested implementer: gpt-6.1-sol at medium. The worker host did not expose an executed model identifier; no actual-model claim is made. Two independent implementer lanes were dispatched with gpt-6.1-sol at medium for disjoint runtime-fixture and Kettle/control surfaces; both terminated before staging. The worker alone ran Cargo/Make and committed.

Baseline: green — `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-tests`, before any edit, exit 0 (`fn210-18-baseline.log`). No baseline handoff or gate receipt was used.

Test-first evidence: the new `machine_handler_returns` compile-fail control failed because the old macro accepted it (`fn210-18-return-red.log`, suite exit 101). Converted unit handlers then failed against the old Option-requiring dispatch (`fn210-18-unit-red.log`, suite exit 101). The return-pin run created the expected diagnostic pending file (`fn210-18-return-pin.log`, suite exit 101); that exact diagnostic was reviewed and installed. Existing refusal controls retain their original stderr bytes and intended errors after migrating their old handler signatures.

DSL values and expansion snapshots were reviewed individually (`fn210-18-goldens-red.log`, `fn210-18-expansion-red.log`) and accepted with exact `cargo insta accept --snapshot` paths. Tick's mutation now emits; Reset at zero emits nothing; Increment remains guarded; MutateRestore emits no edges at either start. The new shared changed_state fixture has two states and one ActualChange edge, three vacuous stutter after claims, and a refuted actual-change after claim with its counterexample. Its newly seeded goldens were generated through the existing shared runner (`fn210-18-shared-generate.log`); only those new snapshots were created. A before-update base-tree manifest is preserved in `fn210-18-protected-before.txt`. Before and after generation, `git diff --exit-code 6e519b63b3fb7cce26a582531a7c4eeeaecd2dcd --` passed for canonical Kettle expected files, kettle_cut, kettle_no_heat, kettle_trait and edge_checks. The trait source, independent goldens and explicit same-state edge control remain byte-identical.

Final gates: focused DSL/macros/shared tests exit 0 (`fn210-18-focused-final.log`); full `make model-check` exit 0 (`fn210-18-model-check-final.log`), including workspace tests, fmt, all-target Clippy, docs, capability/protobuf controls, stock wasm core/guest controls and CGO-disabled Go-host parity. The first full gate failed on unused receivers in the new control (`fn210-18-model-check.log`); constant predicates were replaced by before/after comparisons, and the deliberately empty no-op handler has a scoped Clippy expectation explaining its required receiver. No gate or assertion was weakened. `git diff --check` passes, no pending snapshots remain, and git status is clean.

Exactly one actual warm `make model-time` ran, on the committed source, exit 0 (`fn210-18-model-time.log`); no timing retry or exception was needed:

```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.011s
capability-controls               0.157s
protobuf-freshness                0.080s
fmt                               0.116s
clippy                            0.106s
docs                              0.114s
tests                             0.777s
wasm32-wasip1                     0.057s
core-wasm-clippy                  0.093s
wasm-guest                        0.072s
go-host                           1.281s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.681s
gate total                        3.004s  (budget 30s)
inner loop                        1.681s  (budget 3s)
```

All logs named above are copied to canonical `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/`. Base commit: `6e519b63b3fb7cce26a582531a7c4eeeaecd2dcd`; complete source range: `6e519b63b3fb7cce26a582531a7c4eeeaecd2dcd..3fae17bccb2245a938fb1e855042275d2350e26c`. Read-only flowctl show confirms in_progress. No review verdict, Flow lifecycle mutation, shared gate receipt, plan-sync, push, history rewrite or worktree teardown was performed. Keep the worktree for conductor integration and review. Task .19 and fn-214 were not implemented. All commands and child lanes have terminated.

stage: impl-review - ran [6e519b63b3fb7cce26a582531a7c4eeeaecd2dcd..7e3720aca4e8a8ab49bd1f36221071ff54e1436c] (model: gpt-6.1-sol high)

### Conductor integration and review

The conductor integrated the original source at 0a16296794a82b561a331b21f59ccab214a58f15, then integrated review fix e007ac10dab8040a922f1e1111c040e8e4361f3d at 7e3720aca4e8a8ab49bd1f36221071ff54e1436c. The model tree matches the final worker branch. Integrated make model-check passed before review and after the fix (fn210-18-integrated-model-check.log and fn210-18-async-integrated-model-check.log).

The first three review draws ran on gpt-6.1-sol at high effort. Correctness and contracts returned SHIP; integration found one introduced async-handler defect. An async method with no written return type produced an unpolled future that dispatch discarded. The new machine_handler_async trybuild control first compiled successfully against the defective producer, then passed after explicit asyncness rejection. The root conductor made this three-file fix in the isolated worktree because the implementer model was at capacity. The focused suite, full gate and one worktree timing run passed (full 3.261 s, inner 1.144 s). Protected canonical and trait artifacts remained unchanged. Details are in fn210-18-review-fix-summary.md and its evidence JSON.

The same primary reviewer session resumed and returned SHIP with the finding fixed and no new blockers. Initial RID 9edb4c5d4ba74d1084659e9b287ca0ee; resumed reservation b50feb3472ae49ffa3406c75b482864d. Actual reviewer model was gpt-6.1-sol high throughout, in the same family as the writer. Receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.18.json has SHA256 59bdca4a675de74cdaa6401d45063d7e56c679077c04d79a97fb7fd7c8e796c7. Read-only Cargo attempts were blocked by the build lock and count as inconclusive; the actual conductor gates passed.

Final integrated make model-time passed without a retry or exception (fn210-18-integrated-model-time.log):

```text
timing-controls                   0.011s
capability-controls               0.126s
protobuf-freshness                0.082s
fmt                               0.149s
clippy                            0.136s
docs                              0.103s
tests                             0.678s
wasm32-wasip1                     0.051s
core-wasm-clippy                  0.068s
wasm-guest                        0.061s
go-host                           1.186s
inner loop (edit+test)            2.443s
gate total                        2.965s  (budget 30s)
inner loop                        2.443s  (budget 3s)
```

Memory capture after NEEDS_WORK-to-SHIP created bug/runtime-errors/reject-async-methods-before-discarding-2026-10-19 with low overlap and no matches. All implementation and review commands are terminal. The later claims and progress-vocabulary tasks remain separate work.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3fae17bccb2245a938fb1e855042275d2350e26c, 0a16296794a82b561a331b21f59ccab214a58f15, e007ac10dab8040a922f1e1111c040e8e4361f3d, 7e3720aca4e8a8ab49bd1f36221071ff54e1436c
- Tests: baseline: green; cd model && mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-tests; exit 0; .flow/tmp/fn210-18-baseline.log, red: cd model && mise exec -- cargo test -p umpire-dsl-macros --test fixtures refusals; suite exit 101, old macro accepted the new compile-fail control; .flow/tmp/fn210-18-return-red.log, red: cd model && mise exec -- cargo test -p umpire-dsl --test fixtures; suite exit 101, old dispatch expected Option<()> from unit handlers; .flow/tmp/fn210-18-unit-red.log, cd model && mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-tests; exit 0; .flow/tmp/fn210-18-focused-final.log, make model-check; exit 0; includes workspace test suite, fmt, Clippy, docs, capability controls, protobuf freshness, stock wasm core/guest and Go parity; .flow/tmp/fn210-18-model-check-final.log, make model-time; exit 0, exactly one warm invocation; full gate 3.004s <=30s, inner loop 1.681s <=3s; .flow/tmp/fn210-18-model-time.log, git diff --exit-code 6e519b63b3fb7cce26a582531a7c4eeeaecd2dcd -- model/umpire/tests/fixtures/kettle/expected model/umpire/tests/fixtures/kettle_cut model/umpire/tests/fixtures/kettle_no_heat model/umpire/tests/fixtures/kettle_trait model/umpire/tests/fixtures/edge_checks; exit 0 before and after golden generation; .flow/tmp/fn210-18-protected-before.txt, git diff --check; exit 0; git status --short empty; no pending snapshots; all Cargo/Make calls exported CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target, async regression: old producer unexpectedly compiled fixture, exit 101; explicit synchronous-method refusal then passed; fn210-18-async-red.log and fn210-18-async-focused.log, post-fix integrated make model-check: exit 0; fn210-18-async-integrated-model-check.log, final integrated make model-time: exit 0; inner 2.443s, full 2.965s; fn210-18-integrated-model-time.log, implementation review: NEEDS_WORK then SHIP in the same primary session; actual gpt-6.1-sol high; receipt SHA256 59bdca4a675de74cdaa6401d45063d7e56c679077c04d79a97fb7fd7c8e796c7
- PRs: