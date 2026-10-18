---
satisfies: [R5, R14, R26, R36]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.5 #[machine] attribute: pass-through impl, generated Machine impl from #[on]/#[invariant]/#[postcondition]/#[scope]/#[progress]

## Description
**Recut 2026-10-18 (R36):** `#[machine]` is a registration attribute only. It passes the impl through and generates the `Machine` impl from `#[on]`, `#[invariant]`, `#[postcondition]`, `#[scope]` and `#[progress]`; it records no tokens for lifting and emits no liftable-subset warnings (both moved to fn-212). Ignore those instructions below.

**Integration after .3/.4:** the existing engine is world-shaped (R9/R10). Generated dispatch must clone the state, call the original transition, and turn an enabled `Option<()>` into the ordered unit-outcome/fact-free edge; `None` means disabled, not an API Refusal or rejecting outcome. Preserve the original impl and method bodies in the expansion; helper attributes must remain valid pass-through Rust attributes. Record any deliberate authoring changes from the hypothetical sketch in the .6 DESIGN comparison, not by silently rewriting bodies.

**R14 cross-kind order:** source-order metadata must retain interleaved invariant/postcondition/progress names, not merely each kind's order. A defaulted Machine metadata hook can be added within this DSL lane; .6 will connect it to checker reporting and prove actual cross-kind order after the lanes join. Do not edit checker source while .3 is active.

**R19 handoff:** keep the original machine impl tokens available for the later lift. State/action declarations and start expressions live outside that impl; preserve/expose their derive input at a minimal implementation-only seam here or document the exact seam .8 must add. Do not substitute handwritten Kettle TLA or create a stable IR/public protobuf metadata protocol. Read backend-readiness.md, fn2104-derive-seam.md and claim-ordering.md in the shared notes directory.

**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R26):** `#[machine]` registers claims, scopes and progress under the author's Rust names; outputs use them verbatim.

**Touches:** model2/umpire/dsl/build.rs, model2/umpire/dsl/macros/build.rs, model2/umpire/dsl/src/**, model2/umpire/dsl/tests/**, model2/umpire/dsl/macros/src/**, model2/umpire/dsl/macros/tests/**

**Parallel boundary:** shared Cargo wiring is prepared by .11; no manifests or lockfiles change in this lane. This lane may follow .4 while .3 remains active.

**Layout (R22):** expansion cases go in `umpire/dsl/macros/tests/fixtures/<name>/`; refusals in `dsl/macros/tests/ui/`.

**Deps 2026-10-18 (R18):** same macro stack as .4; the pass-through golden is an insta snapshot of `machine_impl(tokens)` formatted by `prettyplease` (replaces macrotest).

**Recut 2026-10-18 (R13):** `#[machine]` warns on expressions outside the liftable subset (arithmetic, comparisons, boolean logic, field access, enum `match`, `imbl` operations, helpers in the same impl), naming the expression and the reason; pin one warning with a trybuild golden.

**Note 2026-10-18 (R8, golden files):** the pass-through property is an insta snapshot (`expand/kettle.expanded.rs` contains the original impl unchanged); refusals are `trybuild` goldens.

Attribute macro on an `impl` block: emit the impl unchanged, then a `Machine` impl whose `step` dispatches by `#[on(Action::Variant)]` (a `None` return refuses), whose claim lists collect `#[invariant]` and `#[postcondition(after = …)]` functions by name, and whose progress collects `#[progress(within = …)]`. Compile errors naming the action for a missing or duplicate `#[on]` and for an unknown `after`. Confirm rust-analyzer navigates the impl (record in the write-up).

**Files:** model2/umpire/dsl/macros/src/machine.rs, tests/ui/.

**Quick commands:** `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`
## Acceptance
- Attribute output contains the original impl unchanged (insta snapshot of the prettyplease-formatted expansion).
- trybuild: missing #[on], duplicate #[on], unknown after.
## Done summary
Authors can register ordinary Rust transition and claim methods with #[machine], retaining the original impl and generating a world adapter, actor action sum, and ordered registrations. The fixture pins ordered dispatch, disabled edges, named invariants/postconditions/scopes/progress, interleaved claim metadata, and the required action diagnostics.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)
stage: impl-review - ran (codex:gpt-6.1-sol:high; correctness/contracts/integration all first-pass SHIP; same-family reviewer)

At worker handover, task fn-210-rust-prototype-kettle-on-a-trait-engine.5 remained in_progress for conductor review and completion. Its workspace was .worktrees/fn2105-machine on work/fn2105-machine. The worker range was 68131797c269426ecb4cc04f878da5f45876ab1a..7920be5c6eb17fdc76518ccf2eef1e77097df97d, containing 0c4a27e470941df4d98367f96795114779e2d434 and 7920be5c6eb17fdc76518ccf2eef1e77097df97d.

baseline: green. Before any implementation edit, both `mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros` and `make model2-check` passed with exit 0. Logs are .flow/tmp/fn2105-baseline-focused.log and .flow/tmp/fn2105-baseline-full.log in this worktree. Both receipt checks returned RUN, so no baseline or verification gate was skipped. All Cargo commands used CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target.

The first feature test failed with unresolved registration macros and missing generated CounterWorld/CounterAction, recorded in .flow/tmp/fn2105-red.log. The original annotated methods later compiled and remained directly callable. A second test-first boundary check proved that an unregistered within scope incorrectly compiled, recorded in .flow/tmp/fn2105-unknown-scope-red.log. The macro now refuses that name and pins its diagnostic in machine_unknown_scope.stderr.

R5 and R26 evidence uses the existing shared fixture infrastructure. There is one macro fixture binary, with expansion and trybuild tests, and the existing single DSL fixture binary. machine_missing_on.stderr names User::Release through the compiler's exhaustive action match. machine_duplicate_on.stderr names User::Press. machine_unknown_after.stderr names User::Missing. Helpers are exported no-op proc attributes. The machine expansion fixture invokes the actual macro implementation on the compiled DSL fixture's annotated impl and snapshots its prettyplease output; the first item is the original impl with unchanged method bodies and helper attributes. The expansion preserves method order, while the runtime catalog follows actor and enum declaration order. The final behavior golden uses Spell for states, actions, unit outcomes and claim outcomes.

The runtime fixture also compiles private and public State declarations, an empty actor catalog, direct method invocation, a None transition that mutates only the discarded clone, unit-outcome/fact-free edges, per-state and filtered/unfiltered edge predicates, named scopes, progress predicates and scope attachment. It pins cross-kind claim_order as reset_clears, bounded, reaches_two, grows_at_most_one. Machine's default scopes() and claim_order() hooks preserve manual-world defaults. Task .6 owns connecting the metadata to actual checker report ordering and its shared report golden.

Generated StateWorld and StateAction adapters are private to the author module. Task .6 can use them inside its exported fixture runner; the existing wasm guest keeps its trait Kettle source. The deliberate sketch changes are static scope functions returning Scope<StateAction>, static progress builders returning Progress<StateWorld>, explicit world generic parameters in builder chains, and actor wrapper values in scope lists. The original hypothetical scope constant cannot construct Vec in const evaluation. Full examples, validation limits, naming details and .6 ordering obligations are in /Users/stephan/Workspace/skunkworks/umpire/temporal/.git/flow-notes/fn-210-20261018T203953Z-root/fn2105-machine-seam.md. Original model2/options/Kettle.attrs.rs is unchanged.

R36 governs this code. The macro stores no token record and performs no lifting or liftable-subset warning analysis. No checker source, shared fixtures, manifest, lockfile, MILESTONES, DESIGN, Scala model or feature map changed. The second commit replaces Debug formatting in this lane's new fixture with canonical Spell output; it preserves the three observed judgment states.

Final focused tests passed with exit 0 in .flow/tmp/fn2105-focused-final-spell.log. DSL/macro all-target/all-feature Clippy with -D warnings passed in .flow/tmp/fn2105-clippy-final.log. The final full gate, including workspace warnings-denied Clippy and rustdoc, native fixtures/UI/property checks, WASI builds and checks, protobuf freshness, capability/export controls and Go-host parity, passed inside the actual make model2-time invocation below. .flow/tmp/fn2105-warmup-full.log records the preceding cache warm-up. git diff --check passed. Suite results were read from command exit codes; each invocation captured its output once.

Rust-analyzer 1.99.1 (58cf8f9 2026-10-13) with proc macros enabled answered an LSP definition request on direct.increment() with the original increment method at fixture line 48. The analyzer exited after the probe. Evidence is .flow/tmp/fn2105-ra-navigation.log and the retained client is .flow/tmp/fn2105-ra-navigation.py. This verifies original impl navigation and does not claim navigation for every generated adapter operation.

R30 passed on the first warm measurement. The actual macro edit/test loop was 2.805 s, within 3 s, and the full gate was 7.172 s, within 30 s. Command was CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target UMPIRE_INNER_CRATE=umpire-dsl-macros UMPIRE_INNER_SOURCE=umpire/dsl/macros/src/lib.rs make model2-time. Exit code was 0. Full output is .flow/tmp/fn2105-time.log and follows with terminal color escapes omitted.

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2105-machine'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.010s
capability-controls               0.141s
export-controls                   0.072s
protobuf-freshness                0.071s
fmt                               0.115s
clippy                            0.468s
docs                              0.322s
tests                             2.691s
wasm32-wasip1                     0.119s
core-wasm-clippy                  0.147s
wasm-guest                        1.450s
go-host                           1.349s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2105-machine'
inner source: umpire/dsl/macros/src/lib.rs
inner command: mise exec -- cargo test -p umpire-dsl-macros --quiet
inner loop (edit+test)            2.805s
gate total                        7.172s  (budget 30s)
inner loop                        2.805s  (budget 3s)
```

The worker issued no review verdict or flowctl done and handed over with no live commands. The conductor integrated both source commits in dc11609596058a59749b9997f4d9b9bf924ae926, then reviewed that exact head against normalized base b70de88cc6b76b129c7d3723a435038be5770b74. All three first-round draws returned SHIP with zero findings and zero unaddressed requirements. Metadata commit 99205bb891ed5c7830dde25a3c024ef4696dc26f is integrated into umpire. Canonical receipt is /tmp/impl-review-receipt-0953c42b5eee-fn-210-rust-prototype-kettle-on-a-trait-engine.5.json. Raw sidecars are archived at .flow/review-fanout/241308d289c9431ab61c09c71132fdf8/; worker logs and the analyzer probe are archived at .flow/tmp/fn2105-archived/.

The reviewers' Cargo attempts were inconclusive because their build lock was read-only. The conductor's actual focused suite at the integrated review-metadata head exited 0. Command was `cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros`; output is .flow/tmp/fn210-5-closure-focused.log. The final full combined gate remains required at rolling quiesce.

stage: wave-dispatch - ran (isolated registration lane; dependent frontier held)
stage: wave-join - ran (ancestry-preserving source and review integration; focused integrated verification passed)
stage: memory-capture - skipped(policy: first-pass SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0c4a27e470941df4d98367f96795114779e2d434, 7920be5c6eb17fdc76518ccf2eef1e77097df97d, dc11609596058a59749b9997f4d9b9bf924ae926, 99205bb891ed5c7830dde25a3c024ef4696dc26f
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target mise exec -- cargo clippy -p umpire-dsl -p umpire-dsl-macros --all-targets --all-features -- -D warnings, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target make model2-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target UMPIRE_INNER_CRATE=umpire-dsl-macros UMPIRE_INNER_SOURCE=umpire/dsl/macros/src/lib.rs make model2-time, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2104-target python3 ../.flow/tmp/fn2105-ra-navigation.py, git diff --check, Integrated focused verification at 99205bb891: cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros (exit 0; .flow/tmp/fn210-5-closure-focused.log), First-round correctness/contracts/integration codex:gpt-6.1-sol:high all SHIP, zero findings; /tmp/impl-review-receipt-0953c42b5eee-fn-210-rust-prototype-kettle-on-a-trait-engine.5.json
- PRs: