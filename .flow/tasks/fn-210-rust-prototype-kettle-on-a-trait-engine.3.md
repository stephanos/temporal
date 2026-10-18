---
satisfies: [R3]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.3 Progress: whenever/eventually within a scope, weak fairness on internal steps, lasso counterexamples

## Description
**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner. Keep the progress oracle's case count bounded by default.

**Shared fixtures 2026-10-18 (R29):** end-to-end fixtures live in `umpire/tests/fixtures/<name>/` after .13; their expected files select the consumers (checker, TLC, Apalache).

**Conventions 2026-10-18 (R25-R27):** new fixtures implement `Spell` by hand until .4's derives land; an unknown scope action is a `Refusal`, not a panic; progress results and lassos are spelled through `Spell`.

**Touches:** model2/umpire/checker/src/**, model2/umpire/checker/tests/**, model2/umpire/tests/fixtures/**

**Parallel boundary:** shared Cargo wiring is prepared by .11 and scope-refusal protocol wiring by .14; no manifests, lockfiles, protobuf schema, generated Go or wasm adapter change in this lane. Read the shared progress-readiness.md note for SCC/witness/cut analysis.

**Layout (R22):** progress cases go in `umpire/checker/tests/fixtures/<name>/{model.rs,expected/}`; Kettle's `boils` result lands in `umpire/tests/fixtures/kettle/expected/report.txt`.

**Deps 2026-10-18 (R18):** build the scoped graph as a `petgraph` `DiGraph` over `StateId`s and use `tarjan_scc`; a fair cycle is a non-trivial SCC inside the scope where every weakly-fair internal step enabled throughout is taken; the lasso is a BFS prefix to the SCC plus a cycle within it.

**Recut 2026-10-18 (R8, golden files):** progress results land in `report.txt` and lasso counterexamples in `paths/<claim>.txt`; the no-heater control Model is a case of its own. Extend the oracle explorer with the same progress check.

Implement progress checking over the explored graph restricted to the scope's action set: from every state where `whenever` holds, every fair path reaches a state where `eventually` holds; weak fairness on internal steps; a refutation is a lasso (prefix plus cycle) or a dead end. Verify Kettle's `boils` within `without(DepressButton)`; refute it on a control Model whose heater never fires.

**Files:** model2/umpire/checker/src/progress.rs, tests.

**Quick commands:** `mise exec -- cargo test -p umpire-checker -p umpire-tests`
## Acceptance
- Kettle's `report.txt` lists `boils` verified. The actual no-heater Kettle control refutes with a dead-end path; a separate cyclic no-progress Model has a lasso golden with prefix and cycle marked. No implicit stuttering is added.
- The oracle test covers progress.
- An unknown scope action returns the pre-wired `Refusal::ScopeAction` with its canonical name in a runtime fixture golden; catalog membership is not a compile-time property.
## Done summary
Scoped progress now checks every globally reached trigger, follows only allowed actions until the target, and refutes actual dead ends or weakly fair closed walks. Deterministic witnesses reuse ordered graph edges, retain the obligation prefix, and spell explicit prefix/cycle boundaries through Spell; unknown scope catalog actions return Refusal::ScopeAction with their canonical spelling.

Kettle and trait Kettle report boils verified. Their state/row goldens and all exploration counts are unchanged; only their report claims (plus the existing negative report and cut report) gained boils. The actual no-heater Kettle control refutes with a dead-end path, and separate cyclic Models have marked lasso goldens. No stuttering was introduced. Declared progress precedes dynamic effect-failure rows.

Fairness uses petgraph DiGraph<StateId, ()> and tarjan_scc with discovery-order component selection and catalog/choice-order witness paths. Enabledness includes scoped exits to target/outside the candidate. The closed walk visits disabling witnesses and takes required internal actions; fairness applies to actions rather than individual choices. Completed per-action expansion distinguishes actual disabled actions from cut or panicking evaluations. Observed cycles can still refute partial expansions when taken edges already certify every fair action, including the internal/external self-loop-before-cut controls. Independent definite refutations survive other holes or cuts; progress predicate panics are Model refutations with paths, while effect panics leave affected progress unknown.

R3/R8/R11/R12/R14/R27/R29 coverage: shared progress fixtures cover continuous and intermittent enablement, excluded fair exits, action choices, two-lobe fair walks, globally excluded prefixes, multiple starts, target barriers, singleton loops, immediate fulfillment and a trigger that stops before the target. Checker fixtures cover start/admission/queued/mid-choice cuts, known refutations before/through cuts, trigger/target holes and panics, an independent known refutation alongside holes, effect panic without a fabricated dead end, and the runtime unknown-action refusal. The existing oracle retains 256 bounded generated worlds and safety/row checks; independently generated internal flags, scope, triggers and targets feed a mask-augmented closed-walk reference without any SCC calls, and every returned progress witness is checked against reference edges, target avoidance, closure and weak fairness.

baseline: green via handoff at 44580891599e0e37ceffe7ffee4dc21fddf589b4; the supplied exact focused baseline log was read at .flow/tmp/fn210-3-rolling-baseline.log. No source path changed between the handoff SHA and worker start. Child baseline runs also passed. Both child regressions first failed because the checker omitted declared progress (exit 101); copied logs are .flow/tmp/fn2103-oracle-red.log and .flow/tmp/fn2103-fixtures-red.log.

Verification passed (all commands exited 0): final focused cargo test -p umpire-checker -p umpire-tests, focused Clippy with warnings denied, the complete make model2-check (native fixtures, capability/export controls, protobuf freshness, wasm and CGO-disabled Go parity), and warm make model2-time. Logs: .flow/tmp/fn2103-final-focused.log, fn2103-clippy.log, fn2103-full-warmup.log and fn2103-time.log in the conductor checkout. The initial distinct-target compilation was excluded from warm measurements. The 256-world oracle took 0.02 s. Focused green receipt: worker .flow/tmp/green-receipts/59c789c3-unittest.json. gate classify required full execution; no fabricated or reused GATE_SKIPPED evidence is claimed.

stage: impl-review - ran (model: gpt-6.1-sol)
Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)

Task stays in_progress. Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2103-progress, branch work/fn2103-progress. Range: 44580891599e0e37ceffe7ffee4dc21fddf589b4..59c789c3cea42e5e76f38a2e89c3c4b75d4fe9fb. Child lanes were isolated and committed; their commits were cherry-picked before final validation. Their clean inactive worktrees remain under the conductor's integration/cleanup boundary: .worktrees/fn2103-oracle (work/fn2103-oracle, 62cfa7fcf37355a1fe7d42d2a08f051bc55e9bf3) and .worktrees/fn2103-fixtures (work/fn2103-fixtures, 280fc46eab71e446df59422ff823a45864005b63). No manifest, lockfile, protocol, Go, wasm adapter, macro/DSL or Scala source changed.

make model2-time (exit 0):

```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
timing-controls                   0.010s
capability-controls               0.200s
export-controls                   0.091s
protobuf-freshness                0.180s
fmt                               0.086s
clippy                            0.310s
docs                              0.241s
tests                             2.048s
wasm32-wasip1                     0.162s
core-wasm-clippy                  0.129s
wasm-guest                        1.180s
go-host                           1.206s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            0.840s
gate total                        5.985s  (budget 30s)
inner loop                        0.840s  (budget 3s)
```

Conductor integration and review: source commits remain reachable after integration merge 992be90d25820d90ad705c8c46bbb23a112a2a25. Review ran against that exact isolated head with normalized base a98a532b0b8d7b46e9b69a70f3df637c3aeb4371, then its metadata commit 400bd9062cffd32f1fa7558dccc2a6f1eaf7d0fc was integrated in 87de245123a945175dbb6dc4fed9db585b7d18c2. All three codex:gpt-6.1-sol:high lenses returned SHIP with no findings; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.3.json. Reviewers' fresh Cargo writes were inconclusive; actual focused tests on the combined progress/derive target passed (root .flow/tmp/fn210-3-integrated-focused.log), and no executable file changed after that verification. Full combined-suite verification runs at rolling quiesce. No memory capture on first-pass SHIP.

stage: wave-dispatch - ran (isolated progress implementation alongside derive lane)
stage: wave-join - ran (ancestry-preserving task integration; focused combined-target verification passed)

Child cleanup: exact patch IDs match originals to their cherry-picked integrated commits, but original child branches are not ancestry-merged. A no-code ancestry merge attempt conflicted with later lead edits and was aborted cleanly without resolving or changing the target. The two clean inactive child worktrees and branches are retained pending the user's explicit cleanup choice; no source work was lost.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: edf6b080056edc6e5fa23eec3db976cc5fc56930, e3ca5a8ba155fff0f6b67c1d9773ee8fcc774d31, 59c789c3cea42e5e76f38a2e89c3c4b75d4fe9fb, 992be90d25820d90ad705c8c46bbb23a112a2a25, 400bd9062cffd32f1fa7558dccc2a6f1eaf7d0fc, 87de245123a945175dbb6dc4fed9db585b7d18c2
- Tests: mise exec -- cargo test -p umpire-checker -p umpire-tests, mise exec -- cargo clippy -p umpire-checker -p umpire-tests --all-targets -- -D warnings, make model2-check, make model2-time, combined target: cd model2 && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test -p umpire-checker -p umpire-tests (passed; .flow/tmp/fn210-3-integrated-focused.log)
- PRs: